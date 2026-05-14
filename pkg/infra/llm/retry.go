package llm

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"time"
)

// RetryConfig controls provider request retries.
type RetryConfig struct {
	Enabled     bool          `yaml:"enabled" json:"enabled"`
	MaxAttempts int           `yaml:"max_attempts" json:"max_attempts"`
	BaseBackoff time.Duration `yaml:"base_backoff" json:"base_backoff"`
	MaxBackoff  time.Duration `yaml:"max_backoff" json:"max_backoff"`
	JitterRatio float64       `yaml:"jitter_ratio" json:"jitter_ratio"`
	RetryOn429  bool          `yaml:"retry_on_429" json:"retry_on_429"`
	RetryOn5xx  bool          `yaml:"retry_on_5xx" json:"retry_on_5xx"`
}

func (c RetryConfig) normalized() RetryConfig {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 1
	}
	if c.BaseBackoff <= 0 {
		c.BaseBackoff = 200 * time.Millisecond
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = 3 * time.Second
	}
	if c.JitterRatio < 0 {
		c.JitterRatio = 0
	}
	if c.JitterRatio > 1 {
		c.JitterRatio = 1
	}
	// If user configures max attempts > 1 but forgets enabled=true, auto-enable retries.
	if c.MaxAttempts > 1 && !c.Enabled {
		c.Enabled = true
	}
	return c
}

func (c RetryConfig) maxAttempts() int {
	n := c.normalized()
	if !n.Enabled {
		return 1
	}
	return n.MaxAttempts
}

func (c RetryConfig) shouldRetryHTTPStatus(status int) bool {
	n := c.normalized()
	if !n.Enabled {
		return false
	}
	if n.RetryOn429 && status == http.StatusTooManyRequests {
		return true
	}
	if n.RetryOn5xx && status >= http.StatusInternalServerError {
		return true
	}
	return false
}

func (c RetryConfig) backoff(attempt int) time.Duration {
	// attempt starts from 1, backoff applies before next attempt.
	n := c.normalized()
	if !n.Enabled || attempt <= 0 {
		return 0
	}
	base := n.BaseBackoff
	for i := 1; i < attempt; i++ {
		base *= 2
		if base >= n.MaxBackoff {
			base = n.MaxBackoff
			break
		}
	}
	if n.JitterRatio <= 0 {
		return base
	}
	factor := 1 + ((rand.Float64()*2 - 1) * n.JitterRatio)
	if factor < 0 {
		factor = 0
	}
	return time.Duration(float64(base) * factor)
}

func shouldRetryTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}
