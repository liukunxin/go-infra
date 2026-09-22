package traffic

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitController is a built-in token-bucket rate limiter that implements
// Controller. It can be used directly without any external dependency:
//
//	traffic.Init(traffic.WithController(
//	    traffic.NewRateLimitController(100, 10), // 100 req/s, burst 10
//	))
//
// For production circuit-breaking + adaptive rate limiting, consider wiring
// in sentinel-golang (https://github.com/alibaba/sentinel-golang) via WithController.
type RateLimitController struct {
	mu       sync.Mutex
	limiters map[string]*resourceLimiter
	r        rate.Limit // tokens per second
	b        int        // burst size

	idleEvictAfter    time.Duration
	idleEvictInterval time.Duration

	stopEvict chan struct{}
	closeOnce sync.Once
}

type resourceLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// RateLimitOption configures optional RateLimitController behavior.
type RateLimitOption func(*RateLimitController)

// WithIdleEviction overrides the default idle-key eviction timings
// (IdleEvictAfter=10m, IdleEvictInterval=1m).
func WithIdleEviction(after, interval time.Duration) RateLimitOption {
	return func(c *RateLimitController) {
		if after > 0 {
			c.idleEvictAfter = after
		}
		if interval > 0 {
			c.idleEvictInterval = interval
		}
	}
}

// NewRateLimitController creates a controller that allows r requests per second
// per resource, with a burst capacity of b.
// r == rate.Inf means no limit; b must be > 0.
// A background goroutine periodically evicts idle limiters.
// Call Close to stop it.
func NewRateLimitController(r rate.Limit, b int, opts ...RateLimitOption) *RateLimitController {
	if b <= 0 {
		b = 1
	}
	c := &RateLimitController{
		limiters:          make(map[string]*resourceLimiter),
		r:                 r,
		b:                 b,
		idleEvictAfter:    10 * time.Minute,
		idleEvictInterval: time.Minute,
		stopEvict:         make(chan struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	go c.evictLoop()
	return c
}

// Close stops the background eviction goroutine. Safe to call multiple times.
func (c *RateLimitController) Close() {
	c.closeOnce.Do(func() {
		close(c.stopEvict)
	})
}

func (c *RateLimitController) evictLoop() {
	ticker := time.NewTicker(c.idleEvictInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopEvict:
			return
		case <-ticker.C:
			c.evictIdle()
		}
	}
}

func (c *RateLimitController) evictIdle() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for resource, rl := range c.limiters {
		if now.Sub(rl.lastAccess) >= c.idleEvictAfter {
			delete(c.limiters, resource)
		}
	}
}

func (c *RateLimitController) limiterFor(resource string) *rate.Limiter {
	c.mu.Lock()
	defer c.mu.Unlock()
	if rl, ok := c.limiters[resource]; ok {
		rl.lastAccess = time.Now()
		return rl.limiter
	}
	l := rate.NewLimiter(c.r, c.b)
	c.limiters[resource] = &resourceLimiter{limiter: l, lastAccess: time.Now()}
	return l
}

func (c *RateLimitController) TryPass(resource string) (Pass, BlockError) {
	if c.limiterFor(resource).Allow() {
		return &rateLimitPass{}, nil
	}
	return nil, &rateLimitBlockError{resource: resource, limit: c.r}
}

// rateLimitPass is the Pass token returned when the request is allowed.
type rateLimitPass struct{}

func (p *rateLimitPass) Error(err error) {}
func (p *rateLimitPass) Done()           {}

// rateLimitBlockError is returned when the resource has exceeded its rate limit.
type rateLimitBlockError struct {
	resource string
	limit    rate.Limit
}

func (e *rateLimitBlockError) Error() string {
	return fmt.Sprintf("rate limit exceeded for resource %q (limit=%.2f req/s)", e.resource, float64(e.limit))
}

func (e *rateLimitBlockError) BlockType() BlockType { return BlockTypeLimit }
func (e *rateLimitBlockError) BlockMsg() string     { return e.Error() }
