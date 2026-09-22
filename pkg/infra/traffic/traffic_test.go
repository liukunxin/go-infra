package traffic

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestHTTPStatus(t *testing.T) {
	cases := []struct {
		bt   BlockType
		want int
	}{
		{BlockTypeLimit, http.StatusTooManyRequests},
		{BlockTypeCircuitBreaking, http.StatusServiceUnavailable},
		{BlockTypeInternal, http.StatusInternalServerError},
		{BlockTypeUnknown, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		if got := HTTPStatus(tc.bt); got != tc.want {
			t.Fatalf("HTTPStatus(%v)=%d, want %d", tc.bt, got, tc.want)
		}
	}
}

func TestRateLimitIdleEvictKeepsActive(t *testing.T) {
	c := NewRateLimitController(rate.Limit(1), 1,
		WithIdleEviction(50*time.Millisecond, 20*time.Millisecond),
	)
	defer c.Close()

	pass, block := c.TryPass("active")
	if block != nil {
		t.Fatalf("first pass blocked: %v", block)
	}
	pass.Done()

	// Exhaust burst so a fresh limiter would allow again immediately.
	_, block = c.TryPass("active")
	if block == nil {
		t.Fatal("expected block after burst exhausted")
	}

	// Touch active periodically while idle key ages out.
	idlePass, _ := c.TryPass("idle")
	if idlePass != nil {
		idlePass.Done()
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, _ = c.TryPass("active")
		time.Sleep(10 * time.Millisecond)
	}

	c.mu.Lock()
	_, activeOK := c.limiters["active"]
	_, idleOK := c.limiters["idle"]
	c.mu.Unlock()

	if !activeOK {
		t.Fatal("active limiter was evicted")
	}
	if idleOK {
		t.Fatal("idle limiter was not evicted")
	}

	// Active bucket should still be exhausted (not reset by eviction).
	_, block = c.TryPass("active")
	if block == nil {
		t.Fatal("active limiter burst was reset; eviction must not recreate hot keys")
	}
}

func TestCloseIdempotent(t *testing.T) {
	c := NewRateLimitController(10, 1, WithIdleEviction(time.Hour, time.Hour))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Close()
		}()
	}
	wg.Wait()

	cb := NewCircuitBreakerController(CircuitBreakerConfig{
		IdleEvictAfter:    time.Hour,
		IdleEvictInterval: time.Hour,
	})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Close()
		}()
	}
	wg.Wait()
}

func TestSetControllerClosesPrevious(t *testing.T) {
	t.Cleanup(Close)

	first := NewRateLimitController(10, 1, WithIdleEviction(time.Hour, 5*time.Millisecond))
	SetController(first)

	second := NewRateLimitController(10, 1, WithIdleEviction(time.Hour, time.Hour))
	SetController(second)

	select {
	case <-first.stopEvict:
		// closed
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first controller eviction loop was not stopped")
	}

	// Same instance reinstall is a no-op (must not close).
	SetController(second)
	select {
	case <-second.stopEvict:
		t.Fatal("reinstalling same controller must not Close it")
	case <-time.After(30 * time.Millisecond):
	}

	Close()
	select {
	case <-second.stopEvict:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("package Close did not stop current controller")
	}
}

func TestCompositeCloseForwards(t *testing.T) {
	rl := NewRateLimitController(10, 1, WithIdleEviction(time.Hour, time.Hour))
	cb := NewCircuitBreakerController(CircuitBreakerConfig{
		IdleEvictAfter:    time.Hour,
		IdleEvictInterval: time.Hour,
	})
	comp := NewCompositeController(rl, cb)
	comp.Close()
	comp.Close() // idempotent

	select {
	case <-rl.stopEvict:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("rate limit sub-controller not closed")
	}
	select {
	case <-cb.stopEvict:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("circuit breaker sub-controller not closed")
	}
}
