package wx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type WeChatSession struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type Client struct {
	appID     string
	appSecret string
}

func NewClient(appID, appSecret string) *Client {
	return &Client{
		appID:     appID,
		appSecret: appSecret,
	}
}

// GetWeChatSession 调用微信接口
func (wx *Client) GetWeChatSession(code string) (*WeChatSession, error) {
	appID := wx.appID
	appSecret := wx.appSecret

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		appID, appSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var session WeChatSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, err
	}

	return &session, nil
}
