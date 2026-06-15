package traffic

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerConfig configures per-resource circuit breaker behavior.
// Zero-value fields are replaced with production-safe defaults by NewCircuitBreakerController.
type CircuitBreakerConfig struct {
	// ErrorRateThreshold is the failure fraction [0,1] that trips the circuit.
	// Default: 0.5 (50 %)
	ErrorRateThreshold float64

	// MinRequests is the minimum number of requests that must be recorded in the
	// sliding window before the circuit can trip. Prevents false-positives on startup.
	// Default: 10
	MinRequests int

	// WindowSize is the number of recent requests tracked by the sliding window.
	// Default: 20
	WindowSize int

	// CooldownPeriod is how long the circuit stays Open before allowing probe
	// requests (transition to Half-Open). Default: 5s
	CooldownPeriod time.Duration

	// HalfOpenMaxRequests is the maximum number of concurrent probe requests
	// allowed while the circuit is Half-Open. Default: 1
	HalfOpenMaxRequests int

	// HalfOpenSuccessThreshold is the number of consecutive probe successes
	// required to close the circuit again. Default: 1
	HalfOpenSuccessThreshold int

	// IdleEvictAfter is how long a breaker in Closed state must be idle (no
	// TryPass calls) before it becomes eligible for eviction. Default: 10m
	IdleEvictAfter time.Duration

	// IdleEvictInterval is how often the background goroutine checks for idle
	// breakers to evict. Default: 1m
	IdleEvictInterval time.Duration
}

func (c CircuitBreakerConfig) withDefaults() CircuitBreakerConfig {
	if c.ErrorRateThreshold <= 0 {
		c.ErrorRateThreshold = 0.5
	}
	if c.MinRequests <= 0 {
		c.MinRequests = 10
	}
	if c.WindowSize <= 0 {
		c.WindowSize = 20
	}
	if c.CooldownPeriod <= 0 {
		c.CooldownPeriod = 5 * time.Second
	}
	if c.HalfOpenMaxRequests <= 0 {
		c.HalfOpenMaxRequests = 1
	}
	if c.HalfOpenSuccessThreshold <= 0 {
		c.HalfOpenSuccessThreshold = 1
	}
	if c.IdleEvictAfter <= 0 {
		c.IdleEvictAfter = 10 * time.Minute
	}
	if c.IdleEvictInterval <= 0 {
		c.IdleEvictInterval = 1 * time.Minute
	}
	return c
}

// ── circuit state ─────────────────────────────────────────────────────────────

type cbState int8

const (
	cbClosed   cbState = iota // normal operation
	cbOpen                    // all requests rejected
	cbHalfOpen                // limited probe requests allowed
)

func (s cbState) String() string {
	switch s {
	case cbClosed:
		return "closed"
	case cbOpen:
		return "open"
	case cbHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ── sliding window ────────────────────────────────────────────────────────────

// slidingWindow is a fixed-capacity circular buffer that tracks the failure
// rate over the last 'size' requests. Must be used under an external lock.
//
//	slot values: 1 = success, 0 = failure, -1 = empty (never written)
type slidingWindow struct {
	buf  []int8
	pos  int
	full bool
	fail int
	size int
}

func newSlidingWindow(size int) *slidingWindow {
	buf := make([]int8, size)
	for i := range buf {
		buf[i] = -1
	}
	return &slidingWindow{buf: buf, size: size}
}

func (w *slidingWindow) record(success bool) {
	old := w.buf[w.pos]
	if old == 0 {
		w.fail-- // evict the old failure that's being overwritten
	}
	if success {
		w.buf[w.pos] = 1
	} else {
		w.buf[w.pos] = 0
		w.fail++
	}
	w.pos = (w.pos + 1) % w.size
	if !w.full && w.pos == 0 {
		w.full = true
	}
}

func (w *slidingWindow) count() int {
	if w.full {
		return w.size
	}
	return w.pos
}

func (w *slidingWindow) failRate() float64 {
	n := w.count()
	if n == 0 {
		return 0
	}
	return float64(w.fail) / float64(n)
}

func (w *slidingWindow) reset() {
	for i := range w.buf {
		w.buf[i] = -1
	}
	w.pos, w.full, w.fail = 0, false, 0
}

// ── per-resource breaker ──────────────────────────────────────────────────────

// resourceBreaker is the state machine for a single resource.
type resourceBreaker struct {
	mu     sync.Mutex
	state  cbState
	window *slidingWindow
	cfg    CircuitBreakerConfig

	openedAt              time.Time
	halfOpenInFlight      int // probes currently executing
	halfOpenSuccessStreak int // consecutive successes in Half-Open
	lastAccess            time.Time // updated on every tryAcquire; used for eviction
}

// tryAcquire decides whether to allow a request.
// Returns (true, recordFn) when allowed; the caller MUST call recordFn exactly once.
// Returns (false, nil) when blocked.
func (rb *resourceBreaker) tryAcquire() (bool, func(success bool)) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.lastAccess = time.Now()

	switch rb.state {
	case cbClosed:
		return true, rb.makeRecordFn(false)
	case cbOpen:
		if time.Since(rb.openedAt) < rb.cfg.CooldownPeriod {
			return false, nil
		}
		rb.toHalfOpen()
		return rb.tryHalfOpen()
	case cbHalfOpen:
		return rb.tryHalfOpen()
	}
	return false, nil
}

func (rb *resourceBreaker) tryHalfOpen() (bool, func(bool)) {
	if rb.halfOpenInFlight >= rb.cfg.HalfOpenMaxRequests {
		return false, nil
	}
	rb.halfOpenInFlight++
	return true, rb.makeRecordFn(true)
}

// makeRecordFn returns a closure that records the outcome under rb.mu.
// isProbe indicates this request is a Half-Open probe.
func (rb *resourceBreaker) makeRecordFn(isProbe bool) func(bool) {
	return func(success bool) {
		rb.mu.Lock()
		defer rb.mu.Unlock()

		if isProbe {
			if rb.halfOpenInFlight > 0 {
				rb.halfOpenInFlight--
			}
			if !success {
				rb.toOpen()
				return
			}
			rb.halfOpenSuccessStreak++
			if rb.halfOpenSuccessStreak >= rb.cfg.HalfOpenSuccessThreshold {
				rb.toClosed()
			}
			return
		}

		rb.window.record(success)
		if rb.window.count() >= rb.cfg.MinRequests &&
			rb.window.failRate() >= rb.cfg.ErrorRateThreshold {
			rb.toOpen()
		}
	}
}

func (rb *resourceBreaker) toOpen() {
	rb.state = cbOpen
	rb.openedAt = time.Now()
	rb.window.reset()
}

func (rb *resourceBreaker) toHalfOpen() {
	rb.state = cbHalfOpen
	rb.halfOpenInFlight = 0
	rb.halfOpenSuccessStreak = 0
}

func (rb *resourceBreaker) toClosed() {
	rb.state = cbClosed
	rb.window.reset()
}

// ── CircuitBreakerController ──────────────────────────────────────────────────

// CircuitBreakerController implements Controller with per-resource circuit breaking.
//
// State transitions:
//
//	Closed  ──(error rate ≥ threshold)──▶  Open
//	Open    ──(cooldown expires)────────▶  Half-Open
//	Half-Open ──(probe succeeds × N)───▶  Closed
//	Half-Open ──(probe fails)──────────▶  Open
type CircuitBreakerController struct {
	mu       sync.Mutex
	breakers map[string]*resourceBreaker
	cfg      CircuitBreakerConfig

	stopEvict chan struct{}
}

// NewCircuitBreakerController creates a controller. Zero-value config fields use defaults.
// A background goroutine periodically evicts breakers that have been idle (in Closed state)
// for longer than IdleEvictAfter. Call Close to stop the goroutine.
func NewCircuitBreakerController(cfg CircuitBreakerConfig) *CircuitBreakerController {
	cfg = cfg.withDefaults()
	c := &CircuitBreakerController{
		breakers:  make(map[string]*resourceBreaker),
		cfg:       cfg,
		stopEvict: make(chan struct{}),
	}
	go c.evictLoop()
	return c
}

// Close stops the background eviction goroutine.
func (c *CircuitBreakerController) Close() {
	select {
	case <-c.stopEvict:
	default:
		close(c.stopEvict)
	}
}

func (c *CircuitBreakerController) evictLoop() {
	ticker := time.NewTicker(c.cfg.IdleEvictInterval)
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

func (c *CircuitBreakerController) evictIdle() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for resource, rb := range c.breakers {
		rb.mu.Lock()
		idle := rb.state == cbClosed && now.Sub(rb.lastAccess) >= c.cfg.IdleEvictAfter
		rb.mu.Unlock()
		if idle {
			delete(c.breakers, resource)
		}
	}
}

func (c *CircuitBreakerController) breakerFor(resource string) *resourceBreaker {
	c.mu.Lock()
	defer c.mu.Unlock()
	if rb, ok := c.breakers[resource]; ok {
		return rb
	}
	rb := &resourceBreaker{
		state:  cbClosed,
		window: newSlidingWindow(c.cfg.WindowSize),
		cfg:    c.cfg,
	}
	c.breakers[resource] = rb
	return rb
}

// TryPass implements Controller.
func (c *CircuitBreakerController) TryPass(resource string, opts ...TryPassOption) (Pass, BlockError) {
	allowed, recordFn := c.breakerFor(resource).tryAcquire()
	if !allowed {
		return nil, &cbBlockError{resource: resource}
	}
	return &cbPass{recordFn: recordFn}, nil
}

// BreakerState is a snapshot of one resource's circuit breaker state for observability.
type BreakerState struct {
	Resource string
	State    string
	FailRate float64
	Total    int
}

// States returns a snapshot of every resource's circuit breaker state.
// Useful for health-check endpoints and dashboards.
func (c *CircuitBreakerController) States() []BreakerState {
	c.mu.Lock()
	keys := make([]string, 0, len(c.breakers))
	ptrs := make([]*resourceBreaker, 0, len(c.breakers))
	for k, v := range c.breakers {
		keys = append(keys, k)
		ptrs = append(ptrs, v)
	}
	c.mu.Unlock()

	states := make([]BreakerState, 0, len(keys))
	for i, rb := range ptrs {
		rb.mu.Lock()
		states = append(states, BreakerState{
			Resource: keys[i],
			State:    rb.state.String(),
			FailRate: rb.window.failRate(),
			Total:    rb.window.count(),
		})
		rb.mu.Unlock()
	}
	return states
}

// ── cbPass ────────────────────────────────────────────────────────────────────

// cbPass is the Pass token for a circuit-breaker–controlled request.
//
// Usage patterns both work correctly:
//
//	// Pattern A — explicit (preferred)
//	if err := doWork(); err != nil {
//	    pass.Error(err)
//	    return err
//	}
//	pass.Done()
//
//	// Pattern B — deferred Done + conditional Error
//	defer pass.Done()
//	if err := doWork(); err != nil {
//	    pass.Error(err) // Error wins; subsequent Done is a no-op
//	    return err
//	}
type cbPass struct {
	mu       sync.Mutex
	settled  bool
	recordFn func(bool)
}

// Error records the request as failed. Subsequent calls (including Done) are no-ops.
func (p *cbPass) Error(_ error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.settled {
		p.settled = true
		p.recordFn(false)
	}
}

// Done records the request as successful. Subsequent calls (including Error) are no-ops.
func (p *cbPass) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.settled {
		p.settled = true
		p.recordFn(true)
	}
}

// cbBlockError is returned when the circuit is open.
type cbBlockError struct{ resource string }

func (e *cbBlockError) Error() string {
	return fmt.Sprintf("circuit breaker open for resource %q", e.resource)
}
func (e *cbBlockError) BlockType() BlockType { return BlockTypeCircuitBreaking }
func (e *cbBlockError) BlockMsg() string     { return e.Error() }
