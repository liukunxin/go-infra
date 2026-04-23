package http_client

import (
	"sync"
	"sync/atomic"
)

var (
	globalHTTPClient atomic.Pointer[Client] // atomic read/write; safe for concurrent Init + GetHTTPClient
	initOnce         sync.Once
)

// Init initializes the global HTTP client. The configuration is applied only on
// the first call; subsequent calls are no-ops (safe to call from multiple goroutines).
// Should be called once during program startup (main or wire).
func Init(cfg Config) {
	initOnce.Do(func() {
		globalHTTPClient.Store(NewClient(cfg))
	})
}

// GetHTTPClient returns the global client.
// Panics if Init has not been called — this is a programming error that should
// be caught at startup, not silently swallowed at runtime.
func GetHTTPClient() *Client {
	c := globalHTTPClient.Load()
	if c == nil {
		panic("http_client: not initialized, call Init first")
	}
	return c
}
