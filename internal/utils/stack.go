package utils

type Stack[T any] struct {
	// dummy  stackNode[T]
	top    *stackNode[T]
	length int
}

type stackNode[T any] struct {
	prev  *stackNode[T]
	value T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

func (s *Stack[T]) Len() int {
	return s.length
}

func (s *Stack[T]) Empty() bool {
	return s.length == 0
}

func (s *Stack[T]) Top() T {
	if s.top == nil {
		var zero T
		return zero
	}
	return s.top.value
}

func (s *Stack[T]) Pop() T {
	n := s.top
	s.top = n.prev
	s.length--
	return n.value
}

func (s *Stack[T]) Push(value T) {
	n := &stackNode[T]{
		prev:  s.top,
		value: value,
	}
	s.top = n
	s.length++
}

func (s *Stack[T]) Clear() {
	s.top = nil
	s.length = 0
}
