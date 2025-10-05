package utils

import (
	"context"
	"log"
	"runtime"
	"sync"
)

var _ sync.Locker = (*LockGuard)(nil)

type LockGuard struct {
	locker   sync.Locker
	ownsLock bool
}

func NewLockGuard(locker sync.Locker) *LockGuard {
	return &LockGuard{
		locker:   locker,
		ownsLock: false,
	}
}

func Lock(locker sync.Locker) *LockGuard {
	g := NewLockGuard(locker)
	g.Lock()
	return g
}

func (g *LockGuard) Lock() {
	g.locker.Lock()
	g.ownsLock = true
}

func (g *LockGuard) Unlock() {
	if g.ownsLock {
		g.locker.Unlock()
		g.ownsLock = false
		return
	}
}

type DeferStack struct {
	stack *Stack[func(context.Context)]
}

func NewDeferStack() *DeferStack {
	return &DeferStack{
		stack: NewStack[func(context.Context)](),
	}
}

func (s *DeferStack) Defer(f func(context.Context)) {
	s.stack.Push(f)
}

func (s *DeferStack) Run(ctx context.Context) {
	for !s.stack.Empty() {
		f := s.stack.Pop()
		f(ctx)
	}
}

func (s *DeferStack) Cancel() {
	s.stack.Clear()
}

func Recovery() {
	if e := recover(); e != nil {
		buf := make([]byte, 2048)
		buf = buf[:runtime.Stack(buf, true)]
		log.Printf("panic fail with error=[%v] stack==%s\n", e, buf)
	}
}
