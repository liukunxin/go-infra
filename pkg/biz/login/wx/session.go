package wx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiMiniProgramSession = "https://api.weixin.qq.com/sns/jscode2session"
	apiOAuthToken         = "https://api.weixin.qq.com/sns/oauth2/access_token"
	apiUserInfo           = "https://api.weixin.qq.com/sns/userinfo"
)

// Config 微信客户端配置。
// 注入 HTTPClient 时 HTTPTimeout 字段会被忽略，以注入实例的 Timeout、Transport 为准。
type Config struct {
	AppID       string
	AppSecret   string
	HTTPTimeout time.Duration // 默认 10s
	HTTPClient  *http.Client  // 可注入 pkg/http_client 的底层客户端
}

// WxError 微信 API 业务错误。
type WxError struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (e *WxError) Error() string {
	return fmt.Sprintf("wx api error: errcode=%d errmsg=%s", e.ErrCode, e.ErrMsg)
}

// Client 微信登录客户端，支持小程序、网页 OAuth、APP 三端。
type Client struct {
	appID      string
	appSecret  string
	httpClient *http.Client
}

// NewClient 创建微信登录客户端。
func NewClient(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		timeout := cfg.HTTPTimeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		hc = &http.Client{Timeout: timeout}
	}
	return &Client{
		appID:      cfg.AppID,
		appSecret:  cfg.AppSecret,
		httpClient: hc,
	}
}

// MiniProgramSession 小程序登录会话信息。
type MiniProgramSession struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
}

type miniProgramResp struct {
	MiniProgramSession
	WxError
}

// GetMiniProgramSession 用小程序 code 换取 openid、session_key 等会话信息（jscode2session）。
func (c *Client) GetMiniProgramSession(ctx context.Context, code string) (*MiniProgramSession, error) {
	url := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		apiMiniProgramSession, c.appID, c.appSecret, code)
	var resp miniProgramResp
	if err := c.get(ctx, url, &resp); err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 {
		return nil, &resp.WxError
	}
	return &resp.MiniProgramSession, nil
}

// OAuthAccessToken OAuth2 授权令牌，网页与 APP 登录共用。
type OAuthAccessToken struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
}

type oauthTokenResp struct {
	OAuthAccessToken
	WxError
}

// GetWebAccessToken 用网页 OAuth2 code 换取 access_token（公众号网页授权场景）。
func (c *Client) GetWebAccessToken(ctx context.Context, code string) (*OAuthAccessToken, error) {
	return c.getOAuthToken(ctx, code)
}

// GetAppAccessToken 用微信开放平台 APP 登录 code 换取 access_token（移动应用场景）。
func (c *Client) GetAppAccessToken(ctx context.Context, code string) (*OAuthAccessToken, error) {
	return c.getOAuthToken(ctx, code)
}

func (c *Client) getOAuthToken(ctx context.Context, code string) (*OAuthAccessToken, error) {
	url := fmt.Sprintf("%s?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		apiOAuthToken, c.appID, c.appSecret, code)
	var resp oauthTokenResp
	if err := c.get(ctx, url, &resp); err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 {
		return nil, &resp.WxError
	}
	return &resp.OAuthAccessToken, nil
}

// UserInfo 微信用户信息，含跨端唯一身份 UnionID。
type UserInfo struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionID    string   `json:"unionid"`
}

type userInfoResp struct {
	UserInfo
	WxError
}

// GetUserInfo 通过 access_token 获取微信用户信息（含 UnionID）。
// 需要用户已授权 snsapi_userinfo scope。
func (c *Client) GetUserInfo(ctx context.Context, accessToken, openID string) (*UserInfo, error) {
	url := fmt.Sprintf("%s?access_token=%s&openid=%s&lang=zh_CN",
		apiUserInfo, accessToken, openID)
	var resp userInfoResp
	if err := c.get(ctx, url, &resp); err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 {
		return nil, &resp.WxError
	}
	return &resp.UserInfo, nil
}

func (c *Client) get(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
