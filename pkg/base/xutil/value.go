// Package xutil 提供项目通用工具函数，供 SDK 各包与业务方直接使用。
package xutil

import (
	"context"
	"reflect"
	"time"
)

// IsNil 判断任意值是否为 nil（安全处理接口、指针、map、slice、chan、func 等引用类型）。
// 直接用 v == nil 对含 nil 指针的接口会返回 false，本函数避免该陷阱。
func IsNil(v any) bool {
	vi := reflect.ValueOf(v)
	switch vi.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return vi.IsNil()
	default:
		return false
	}
}

// Ptr 将值包装为指针，方便内联构造含指针字段的结构体。
//
//	s := &MyStruct{Name: xutil.Ptr("Alice")}
func Ptr[T any](v T) *T {
	return &v
}

// PtrCopy 浅拷贝指针指向的值，返回指向新副本的指针。
func PtrCopy[T any](ptr *T) *T {
	cp := *ptr
	return &cp
}

// Default 若指针为 nil 则返回默认值，否则返回解引用值。
func Default[T any](v *T, def T) T {
	if v == nil {
		return def
	}
	return *v
}

// As 将 any 类型断言为 T，断言失败时返回 T 的零值。
func As[T any](v any) T {
	t, ok := v.(T)
	if ok {
		return t
	}
	var zero T
	return zero
}

// AsDefault 将 any 类型断言为 T，断言失败时返回 def。
func AsDefault[T any](v any, def T) T {
	t, ok := v.(T)
	if ok {
		return t
	}
	return def
}

// ValueFromContext 从 context 中取出 key 对应的值并断言为 V，取不到时返回零值。
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

// ElapsedMs 计算两个时间点之间的毫秒数（float64 精度）。
func ElapsedMs(begin, end time.Time) float64 {
	return float64(end.Sub(begin)) / float64(time.Millisecond)
}
