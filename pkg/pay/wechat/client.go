package wechat

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.mch.weixin.qq.com"

// Client 微信支付 APIv3 客户端。
type Client struct {
	cfg        Config
	httpClient *http.Client
	priv       *rsa.PrivateKey
	platPub    *rsa.PublicKey
}

// NewClient 根据配置创建客户端。
func NewClient(cfg Config) (*Client, error) {
	cfg = cfg.normalized()
	if cfg.AppID == "" || cfg.MchID == "" || cfg.CertificateSerialNo == "" || cfg.PrivateKeyPEM == "" || cfg.APIv3Key == "" {
		return nil, ErrInvalidConfig
	}
	priv, err := parseRSAPrivateKey(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}
	var platPub *rsa.PublicKey
	if strings.TrimSpace(cfg.PlatformCertPEM) != "" {
		platPub, err = parseRSAPublicKey(cfg.PlatformCertPEM)
		if err != nil {
			return nil, err
		}
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = newDefaultHTTPClient(cfg.HTTPTimeout)
	}
	return &Client{
		cfg:        cfg,
		httpClient: hc,
		priv:       priv,
		platPub:    platPub,
	}, nil
}

func (c *Client) authorization(method, urlPath string, body []byte) (string, error) {
	nonce := randomNonce()
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	bodyStr := string(body)
	msg := method + "\n" + urlPath + "\n" + ts + "\n" + nonce + "\n" + bodyStr + "\n"
	sig, err := signSHA256WithRSA(c.priv, msg)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		c.cfg.MchID, nonce, sig, ts, c.cfg.CertificateSerialNo,
	), nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	fullURL := baseURL + path
	var bodyReader io.Reader = http.NoBody
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	auth, err := c.authorization(method, path, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "go-infra-pay/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	// 限制最大读取 4 MB，防止异常超大响应撑爆内存。
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

func randomNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// newDefaultHTTPClient 创建专用于微信支付的独立 HTTP 客户端。
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
