package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	goredis "github.com/redis/go-redis/v9"
)

// UniversalClient 是 go-redis 的 UniversalClient 接口别名，方便外部引用。
type UniversalClient = goredis.UniversalClient

// clientHolder lets us store UniversalClient (an interface) in an atomic.Pointer,
// which requires a concrete pointer type.
type clientHolder struct {
	c UniversalClient
}

var (
	globalClient atomic.Pointer[clientHolder]
	initOnce     sync.Once
)

// Init initializes the global Redis client. Only the first call takes effect.
func Init(cfg *Config) error {
	if cfg == nil {
		return errors.New("redis: config must not be nil")
	}

	var initErr error
	initOnce.Do(func() {
		c, err := NewClient(cfg)
		if err != nil {
			initErr = fmt.Errorf("redis: init failed: %w", err)
			return
		}
		globalClient.Store(&clientHolder{c: c})
	})
	return initErr
}

// GetClient returns the global client.
// Panics if Init has not been called — this is a programming error.
func GetClient() UniversalClient {
	h := globalClient.Load()
	if h == nil {
		panic("redis: not initialized, call Init first")
	}
	return h.c
}

// NewClient creates a Redis client from cfg without touching any global state.
// Use this when you need multiple independent Redis connections.
func NewClient(cfg *Config) (UniversalClient, error) {
	if cfg == nil {
		return nil, errors.New("redis: config must not be nil")
	}
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("redis: at least one address is required")
	}

	var client UniversalClient

	switch cfg.Mode {
	case "single":
		client = goredis.NewClient(&goredis.Options{
			Addr:            cfg.Addresses[0],
			Password:        cfg.Password,
			PoolSize:        cfg.PoolSize,
			MinIdleConns:    cfg.MinIdleConns,
			ConnMaxIdleTime: cfg.IdleTimeout,
		})
	case "cluster":
		client = goredis.NewClusterClient(&goredis.ClusterOptions{
			Addrs:           cfg.Addresses,
			Password:        cfg.Password,
			PoolSize:        cfg.PoolSize,
			MinIdleConns:    cfg.MinIdleConns,
			ConnMaxIdleTime: cfg.IdleTimeout,
		})
	default:
		return nil, fmt.Errorf("redis: unsupported mode %q (want \"single\" or \"cluster\")", cfg.Mode)
	}

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping %v: %w", cfg.Addresses, err)
	}

	return client, nil
}
