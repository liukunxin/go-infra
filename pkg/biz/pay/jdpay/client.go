package jdpay

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Client 京东支付客户端。
type Client struct {
	cfg        Config
	httpClient *http.Client
	priv       *rsa.PrivateKey
	jdPub      *rsa.PublicKey
}

// NewClient 根据配置创建客户端，解析商户私钥和京东平台公钥。
func NewClient(cfg Config) (*Client, error) {
	cfg = cfg.normalized()
	if cfg.MerchantNo == "" || cfg.AppKey == "" || cfg.PrivateKeyPEM == "" || cfg.JDPublicKeyPEM == "" {
		return nil, ErrInvalidConfig
	}
	priv, err := parseRSAPrivateKey(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}
	jdPub, err := parseRSAPublicKey(cfg.JDPublicKeyPEM)
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
		jdPub:      jdPub,
	}, nil
}

// post 发送 JSON POST 请求到京东支付网关。
// params 为请求参数 map（不含 sign）；方法内部自动签名后附加 sign 字段。
// 成功时返回已解析的响应 map（已验签）。
func (c *Client) post(ctx context.Context, path string, params map[string]string) (map[string]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 自动注入公共参数。
	params["merchantNo"] = c.cfg.MerchantNo
	params["appkey"] = c.cfg.AppKey

	// 签名。
	signContent := buildSignContent(params)
	sig, err := rsaSignSHA256Base64(c.priv, signContent)
	if err != nil {
		return nil, fmt.Errorf("jdpay sign: %w", err)
	}
	params["sign"] = sig

	raw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.baseURL()+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-infra-pay/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("jdpay: unmarshal response: %w", err)
	}

	// 验证响应签名（有签名字段时进行）。
	if respSign, ok := result["sign"]; ok && respSign != "" {
		respSignContent := buildSignContent(result)
		if err := rsaVerifySHA256Base64(c.jdPub, respSignContent, respSign); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrVerifySign, err)
		}
	}

	// 检查业务结果码。
	if result["resultCode"] != resultCodeOK {
		return nil, &APIError{
			ResultCode: result["resultCode"],
			ResultMsg:  result["resultMsg"],
		}
	}

	return result, nil
}

// verifyNotifySign 验证京东支付异步通知中的签名。
func (c *Client) verifyNotifySign(params map[string]string) error {
	sig, ok := params["sign"]
	if !ok || sig == "" {
		return fmt.Errorf("%w: missing sign in notify", ErrVerifySign)
	}
	content := buildSignContent(params)
	if err := rsaVerifySHA256Base64(c.jdPub, content, sig); err != nil {
		return fmt.Errorf("%w: %v", ErrVerifySign, err)
	}
	return nil
}

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
