package alipay

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client 支付宝 OpenAPI 客户端（RSA2）。
type Client struct {
	cfg        Config
	httpClient *http.Client
	priv       *rsa.PrivateKey
	aliPub     *rsa.PublicKey
}

// NewClient 创建客户端。
func NewClient(cfg Config) (*Client, error) {
	cfg = cfg.normalized()
	if cfg.AppID == "" || cfg.PrivateKeyPEM == "" || cfg.AlipayPublicKeyPEM == "" {
		return nil, ErrInvalidConfig
	}
	priv, err := parseRSAPrivateKey(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}
	aliPub, err := parseRSAPublicKey(cfg.AlipayPublicKeyPEM)
	if err != nil {
		return nil, err
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = newDefaultHTTPClient(cfg.HTTPTimeout)
	}
	return &Client{
		cfg:        cfg,
		httpClient: hc,
		priv:       priv,
		aliPub:     aliPub,
	}, nil
}

func (c *Client) signedForm(apiMethod string, biz map[string]any, extra url.Values) (url.Values, error) {
	if extra == nil {
		extra = url.Values{}
	}
	bizJSON, err := json.Marshal(biz)
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Set("app_id", c.cfg.AppID)
	v.Set("method", apiMethod)
	v.Set("format", "JSON")
	v.Set("charset", c.cfg.Charset)
	v.Set("sign_type", "RSA2")
	v.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	v.Set("version", "1.0")
	v.Set("biz_content", string(bizJSON))
	for k, vs := range extra {
		for _, one := range vs {
			v.Add(k, one)
		}
	}
	signContent := buildSignContent(v)
	sig, err := rsaSignPKCS1v15SHA256Base64(c.priv, signContent)
	if err != nil {
		return nil, err
	}
	v.Set("sign", sig)
	return v, nil
}

func (c *Client) postGateway(ctx context.Context, apiMethod string, biz map[string]any, extra url.Values) (map[string]json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	v, err := c.signedForm(apiMethod, biz, extra)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.gatewayURL(), strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset="+c.cfg.Charset)
	req.Header.Set("User-Agent", "go-infra-pay/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 限制最大读取 4 MB，防止异常超大响应撑爆内存。
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	// verifyAndParseJSONResponse 验签并返回已解析的 root map，避免二次 json.Unmarshal。
	root, err := verifyAndParseJSONResponse(c.aliPub, raw)
	if err != nil {
		return nil, err
	}
	return root, nil
}

// newDefaultHTTPClient 创建专用于支付宝网关的独立 HTTP 客户端。
// 使用私有 Transport，不受 http.DefaultTransport 全局配置影响。
func newDefaultHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   4,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}
