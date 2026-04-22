package alipay

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
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
		hc = &http.Client{Timeout: cfg.HTTPTimeout}
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
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := verifyJSONResponse(c.aliPub, raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVerifySign, err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	return root, nil
}
