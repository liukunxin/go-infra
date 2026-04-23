package v8

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-redis/redis/v8"
)

// clientHolder lets us store redis.UniversalClient (an interface) in an atomic.Pointer,
// which requires a concrete pointer type.
type clientHolder struct {
	c redis.UniversalClient
}

var (
	globalClient atomic.Pointer[clientHolder] // atomic read; safe for concurrent Init + GetClient
	initOnce     sync.Once
)

// Init initializes the global Redis client. Only the first call takes effect.
// Returns an error so the caller controls failure handling instead of calling log.Fatal.
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
func GetClient() redis.UniversalClient {
	h := globalClient.Load()
	if h == nil {
		panic("redis: not initialized, call Init first")
	}
	return h.c
}

// NewClient creates a Redis client from cfg without touching any global state.
// Use this when you need multiple independent Redis connections.
func NewClient(cfg *Config) (redis.UniversalClient, error) {
	if cfg == nil {
		return nil, errors.New("redis: config must not be nil")
	}
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("redis: at least one address is required")
	}

	var client redis.UniversalClient

	switch cfg.Mode {
	case "single":
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.Addresses[0],
			Password:     cfg.Password,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			IdleTimeout:  cfg.IdleTimeout, // already time.Duration; do NOT multiply by time.Second
		})
	case "cluster":
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.Addresses,
			Password:     cfg.Password,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			IdleTimeout:  cfg.IdleTimeout,
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
