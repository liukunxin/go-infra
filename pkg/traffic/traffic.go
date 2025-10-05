package traffic

import (
	"context"
	"sync/atomic"
)

const (
	TrafficTypeInbound TrafficType = iota + 1
	TrafficTypeOutbound
)

const (
	BlockTypeUnknown BlockType = iota
	BlockTypeLimit
	BlockTypeCircuitBreaking
	BlockTypeInternal
)

type TrafficType int32

func (t TrafficType) String() string {
	switch t {
	case TrafficTypeInbound:
		return "inbound"
	case TrafficTypeOutbound:
		return "outbound"
	default:
		return "unknown"
	}
}

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

var globalController atomic.Pointer[Controller]

func init() {
	var controller Controller = &DummyController{}
	globalController.Store(&controller)
}

func Init(ctx context.Context, opts ...Option) error {
	c := &optionConfig{}

	for _, opt := range opts {
		if err := opt.apply(c); err != nil {
			return err
		}
	}

	if c.controller != nil {
		SetController(c.controller)
	}

	return nil
}

func SetController(controller Controller) {
	if controller == nil {
		controller = &DummyController{}
	}

	globalController.Store(&controller)
}

func GetController() Controller {
	return *globalController.Load()
}

// Controller 流量控制器
type Controller interface {
	TryPass(resource string, opts ...TryPassOption) (Pass, BlockError)
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

func (c *DummyController) TryPass(resource string, opts ...TryPassOption) (Pass, BlockError) {
	return &dummyPass{}, nil
}

type dummyPass struct {
}

func (r *dummyPass) Error(err error) {
}

func (r *dummyPass) Done() {
}
