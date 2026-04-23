package applepay

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Client App Store Server API 客户端。
// 使用前请确保已在 App Store Connect 创建 API 密钥，并下载 .p8 格式私钥。
type Client struct {
	cfg        Config
	httpClient *http.Client
	privKey    *ecdsa.PrivateKey
}

// NewClient 根据配置创建客户端。
// 解析 PKCS#8 EC 私钥，验证必填字段，失败时返回描述性错误。
func NewClient(cfg Config) (*Client, error) {
	cfg = cfg.normalized()
	if cfg.KeyID == "" || cfg.IssuerID == "" || cfg.BundleID == "" || cfg.PrivateKey == "" {
		return nil, ErrInvalidConfig
	}
	privKey, err := parseECPrivateKey(cfg.PrivateKey)
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
		privKey:    privKey,
	}, nil
}

// parseECPrivateKey 解析 PKCS#8 PEM 格式的 EC 私钥（从 App Store Connect 下载的 .p8 文件）。
func parseECPrivateKey(pemStr string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("applepay: private key PEM decode failed")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("applepay: parse private key: %w", err)
	}
	ec, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("applepay: private key must be ECDSA (P-256), got %T", key)
	}
	return ec, nil
}

// generateJWT 生成每次 API 调用所需的 Bearer Token（ES256，有效期 30 分钟）。
//
// App Store Server API 鉴权流程：
//   - Header: alg=ES256, kid=<KeyID>
//   - Payload: iss=<IssuerID>, iat=now, exp=now+30m, aud="appstoreconnect-v1", bid=<BundleID>
//
// 生产建议：可缓存 token 至临近过期前重新生成，当前实现每次请求重新生成以保持简洁。
func (c *Client) generateJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": c.cfg.IssuerID,
		"iat": now.Unix(),
		"exp": now.Add(30 * time.Minute).Unix(),
		"aud": "appstoreconnect-v1",
		"bid": c.cfg.BundleID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = c.cfg.KeyID
	return token.SignedString(c.privKey)
}

// do 发送 HTTP 请求，自动附加 Bearer JWT，限制响应体最大 4 MB。
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) ([]byte, int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.baseURL()+path, body)
	if err != nil {
		return nil, 0, err
	}
	jwtStr, err := c.generateJWT()
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+jwtStr)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "go-infra-pay/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

// newDefaultHTTPClient 创建专用于 App Store Server API 的独立 HTTP 客户端。
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
