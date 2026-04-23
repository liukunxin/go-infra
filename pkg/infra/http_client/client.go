package http_client

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

// Config 定义 HTTP 客户端连接池与超时参数。
// 所有字段均有合理默认值，零值时由 normalized() 填充。
type Config struct {
	Timeout             time.Duration `yaml:"timeout"                json:"timeout"`                  // 请求超时，默认 30s
	MaxIdleConns        int           `yaml:"max_idle_conns"         json:"max_idle_conns"`           // 最大空闲连接总数，默认 100
	MaxIdleConnsPerHost int           `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"` // 每 host 最大空闲连接，默认 10
	MaxConnsPerHost     int           `yaml:"max_conns_per_host"     json:"max_conns_per_host"`       // 每 host 最大连接数，0 表示不限
	IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout"      json:"idle_conn_timeout"`        // 空闲连接回收时间，默认 90s
	TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout"  json:"tls_handshake_timeout"`    // TLS 握手超时，默认 10s
}

func (c Config) normalized() Config {
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 100
	}
	if c.MaxIdleConnsPerHost <= 0 {
		c.MaxIdleConnsPerHost = 10
	}
	if c.IdleConnTimeout <= 0 {
		c.IdleConnTimeout = 90 * time.Second
	}
	if c.TLSHandshakeTimeout <= 0 {
		c.TLSHandshakeTimeout = 10 * time.Second
	}
	return c
}

// Client 是带连接池的 HTTP 客户端封装。
type Client struct {
	httpClient *http.Client
}

// NewClient 创建带连接池的 HTTP 客户端，Config 零值字段使用内置默认值。
func NewClient(cfg Config) *Client {
	cfg = cfg.normalized()

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
	}
}

// HTTPClient 返回底层 *http.Client，便于注入依赖原生客户端的模块（如 pkg/pay）。
func (c *Client) HTTPClient() *http.Client {
	if c == nil {
		return nil
	}
	return c.httpClient
}

// Get 发送 GET 请求，ctx 用于超时与取消控制。
// 响应体最多读取 32 MB，超出部分被截断。
func (c *Client) Get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.do(req)
}

// Post 发送 POST 请求，ctx 用于超时与取消控制。
// 响应体最多读取 32 MB，超出部分被截断。
func (c *Client) Post(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.do(req)
}

// do 执行请求并读取响应体（上限 32 MB）。
func (c *Client) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, 0, err
	}
	return respBody, resp.StatusCode, nil
}
