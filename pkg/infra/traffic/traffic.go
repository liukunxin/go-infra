package traffic

import (
	"net/http"
	"sync/atomic"
)

const (
	BlockTypeUnknown BlockType = iota
	BlockTypeLimit
	BlockTypeCircuitBreaking
	BlockTypeInternal
)

type BlockType int32

func (t BlockType) String() string {
	switch t {
	case BlockTypeLimit:
		return "limit"
	case BlockTypeCircuitBreaking:
		return "circuit_breaking"
	case BlockTypeInternal:
		return "internal"
	default:
		return "unknown"
	}
}

// HTTPStatus maps a BlockType to the recommended HTTP status code.
// Limit → 429, CircuitBreaking → 503, Internal/unknown → 500.
func HTTPStatus(t BlockType) int {
	switch t {
	case BlockTypeLimit:
		return http.StatusTooManyRequests
	case BlockTypeCircuitBreaking:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

type closer interface {
	Close()
}

var globalController atomic.Pointer[Controller]

func init() {
	var controller Controller = &DummyController{}
	globalController.Store(&controller)
}

func Init(opts ...Option) error {
	c := &optionConfig{}

	for _, opt := range opts {
		if err := opt.Apply(c); err != nil {
			return err
		}
	}

	if c.controller != nil {
		SetController(c.controller)
	}

	return nil
}

// SetController replaces the global controller. If the previous instance
// implements Close(), it is closed. Installing the same instance is a no-op.
func SetController(controller Controller) {
	if controller == nil {
		controller = &DummyController{}
	}

	old := globalController.Load()
	if old != nil && *old == controller {
		return
	}

	globalController.Store(&controller)

	if old != nil {
		closeIfNeeded(*old)
	}
}

// Close shuts down the current global controller (if it implements Close)
// and restores DummyController. Safe to call multiple times.
func Close() {
	var dummy Controller = &DummyController{}
	old := globalController.Swap(&dummy)
	if old != nil {
		closeIfNeeded(*old)
	}
}

func closeIfNeeded(c Controller) {
	if cl, ok := c.(closer); ok {
		cl.Close()
	}
}

func GetController() Controller {
	return *globalController.Load()
}

// Controller 流量控制器
type Controller interface {
	TryPass(resource string) (Pass, BlockError)
}

// Pass 允许通过
type Pass interface {
	Error(err error)
	Done()
}

// BlockError 阻止通过
type BlockError interface {
	error
	BlockType() BlockType
	BlockMsg() string
}

type InternalError struct {
	error
}

var _ BlockError = (*InternalError)(nil)

func NewInternalError(err error) *InternalError {
	return &InternalError{
		error: err,
	}
}

func (e *InternalError) BlockType() BlockType {
	return BlockTypeInternal
}

func (e *InternalError) BlockMsg() string {
	return e.Error()
}

type DummyController struct {
}

func (c *DummyController) TryPass(resource string) (Pass, BlockError) {
	return &dummyPass{}, nil
}

type dummyPass struct {
}

func (r *dummyPass) Error(err error) {
}

func (r *dummyPass) Done() {
}
