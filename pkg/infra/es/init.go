package es

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	globalClient atomic.Pointer[Client]
	initOnce     sync.Once
)

// Init initializes the global Elasticsearch client. Only the first call takes effect.
// Returns an error so the caller controls failure handling.
func Init(cfg Config) error {
	var initErr error
	initOnce.Do(func() {
		c, err := NewClient(cfg)
		if err != nil {
			initErr = fmt.Errorf("es: init failed: %w", err)
			return
		}
		globalClient.Store(c)
	})
	return initErr
}

// GetClient returns the global client.
// Panics if Init has not been called — this is a programming error.
func GetClient() *Client {
	c := globalClient.Load()
	if c == nil {
		panic("es: not initialized, call Init first")
	}
	return c
}
