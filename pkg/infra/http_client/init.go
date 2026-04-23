package http_client

import "sync"

var (
	globalHTTPClient *Client
	once             sync.Once
)

// Init 初始化全局 HTTP 客户端，整个进程生命周期内只生效一次。
// 应在程序启动时（main 或 wire）调用，调用前请确保 cfg 已就绪。
func Init(cfg Config) {
	once.Do(func() {
		globalHTTPClient = NewClient(cfg)
	})
}

// GetHTTPClient 返回全局客户端。若 Init 尚未调用，直接 panic（属于编程错误，应在启动阶段发现）。
func GetHTTPClient() *Client {
	if globalHTTPClient == nil {
		panic("http_client: not initialized, call Init first")
	}
	return globalHTTPClient
}
