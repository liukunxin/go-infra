package http_client

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// Config 定义 HTTP Client 配置
type Config struct {
	Timeout             time.Duration `yaml:"timeout" json:"timeout"`                                 // 请求超时
	MaxIdleConns        int           `json:"max_idle_conns" yaml:"max_idle_conns"`                   // 最大空闲连接数
	MaxIdleConnsPerHost int           `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"` // 每个 host 最大空闲连接数
	IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout" json:"idle_conn_timeout"`             // 空闲连接超时时间
}

// Client 封装的 HTTP 客户端
type Client struct {
	httpClient *http.Client
	mu         sync.Mutex
}

// NewClient 创建一个带连接池的 HTTP 客户端
func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	return &Client{
		httpClient: client,
	}
}

// Get 发送 GET 请求
func (c *Client) Get(url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return respBody, resp.StatusCode, nil
}

// Post 发送 POST 请求
func (c *Client) Post(url string, body []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, 0, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return respBody, resp.StatusCode, nil
}
