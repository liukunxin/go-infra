package utils

import (
	"context"
	"reflect"
	"time"
)

func IsNil(v any) bool {
	vi := reflect.ValueOf(v)

	switch vi.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return vi.IsNil()
	default:
		return false
	}
}

func Ptr[T any](v T) *T {
	return &v
}

func PtrCopy[T any](ptr *T) *T {
	copy := *ptr
	return &copy
}

func As[T any](v any) T {
	t, ok := v.(T)
	if ok {
		return t
	}
	var zero T
	return zero
}

func Default[T any](v *T, def T) T {
	if v == nil {
		return def
	}
	return *v
}

func AsDefault[T any](v any, def T) T {
	t, ok := v.(T)
	if ok {
		return t
	}
	return def
}

func ValueFromContext[V any](ctx context.Context, key any) V {
	if ctx == nil {
		var zero V
		return zero
	}
	if v, ok := ctx.Value(key).(V); ok {
		return v
	}

	var zero V
	return zero
}

func Float64ElapsedTime(beginTime time.Time, endTime time.Time) float64 {
	return float64(endTime.Sub(beginTime)) / float64(time.Millisecond)
}
