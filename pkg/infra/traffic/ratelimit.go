package traffic

import (
	"fmt"
	"sync"

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
	limiters map[string]*rate.Limiter
	r        rate.Limit // tokens per second
	b        int        // burst size
}

// NewRateLimitController creates a controller that allows r requests per second
// per resource, with a burst capacity of b.
// r == rate.Inf means no limit; b must be > 0.
func NewRateLimitController(r rate.Limit, b int) *RateLimitController {
	if b <= 0 {
		b = 1
	}
	return &RateLimitController{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (c *RateLimitController) limiterFor(resource string) *rate.Limiter {
	c.mu.Lock()
	defer c.mu.Unlock()
	if l, ok := c.limiters[resource]; ok {
		return l
	}
	l := rate.NewLimiter(c.r, c.b)
	c.limiters[resource] = l
	return l
}

func (c *RateLimitController) TryPass(resource string, opts ...TryPassOption) (Pass, BlockError) {
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
