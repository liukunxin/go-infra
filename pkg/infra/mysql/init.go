package mysql

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	globalClient atomic.Pointer[Client] // atomic read; safe for concurrent Init + GetClient
	initOnce     sync.Once
)

// Init initializes the global MySQL client. Only the first call takes effect.
// Returns an error instead of calling log.Fatal so the caller controls failure handling.
func Init(cfg Config) error {
	var initErr error
	initOnce.Do(func() {
		c, err := NewClient(cfg)
		if err != nil {
			initErr = fmt.Errorf("mysql: init failed: %w", err)
			return
		}
		globalClient.Store(c)
	})
	return initErr
}

// GetClient returns the global client.
// Panics if Init has not been called — this is a programming error that should
// be caught at startup, not silently swallowed at runtime.
func GetClient() *Client {
	c := globalClient.Load()
	if c == nil {
		panic("mysql: not initialized, call Init first")
	}
	return c
}
