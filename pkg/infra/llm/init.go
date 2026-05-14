package llm

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	globalClient atomic.Pointer[Client]
	initOnce     sync.Once
)

// Init initializes the global LLM client. Only the first call takes effect.
func Init(opts ...Option) error {
	var initErr error
	initOnce.Do(func() {
		c, err := New(opts...)
		if err != nil {
			initErr = fmt.Errorf("llm: init failed: %w", err)
			return
		}
		globalClient.Store(c)
	})
	return initErr
}

// GetClient returns the global client. Panics if Init has not been called.
func GetClient() *Client {
	c := globalClient.Load()
	if c == nil {
		panic("llm: not initialized, call Init first")
	}
	return c
}
