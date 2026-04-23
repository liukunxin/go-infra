package traffic

import "sync"

// CompositeController chains multiple Controllers so that a request must pass
// ALL of them. Controllers are evaluated in order; the first one to block
// causes any already-acquired passes to be released via Done().
//
// Typical usage — rate limit + circuit break together:
//
//	ctrl := traffic.NewCompositeController(
//	    traffic.NewRateLimitController(500, 50),
//	    traffic.NewCircuitBreakerController(traffic.CircuitBreakerConfig{}),
//	)
//	traffic.Init(traffic.WithController(ctrl))
type CompositeController struct {
	controllers []Controller
}

// NewCompositeController creates a controller that applies each sub-controller in order.
func NewCompositeController(controllers ...Controller) *CompositeController {
	return &CompositeController{controllers: controllers}
}

// TryPass implements Controller.
// If any sub-controller blocks, all passes already acquired are released via Done()
// and the BlockError from the blocking controller is returned.
func (c *CompositeController) TryPass(resource string, opts ...TryPassOption) (Pass, BlockError) {
	acquired := make([]Pass, 0, len(c.controllers))
	for _, ctrl := range c.controllers {
		pass, blockErr := ctrl.TryPass(resource, opts...)
		if blockErr != nil {
			// Release passes from controllers that already allowed this request.
			for _, p := range acquired {
				p.Done()
			}
			return nil, blockErr
		}
		acquired = append(acquired, pass)
	}
	return &compositePass{passes: acquired}, nil
}

// compositePass fans out Error/Done to all sub-passes.
// Only the first call is forwarded; subsequent calls (e.g. a deferred Done after
// an explicit Error) are no-ops.
type compositePass struct {
	mu     sync.Mutex
	done   bool
	passes []Pass
}

// Error records a failure on all sub-passes. Subsequent calls are no-ops.
func (p *compositePass) Error(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.done {
		p.done = true
		for _, sp := range p.passes {
			sp.Error(err)
		}
	}
}

// Done records success on all sub-passes. Subsequent calls are no-ops.
func (p *compositePass) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.done {
		p.done = true
		for _, sp := range p.passes {
			sp.Done()
		}
	}
}
