package code

import (
	"fmt"
	"math/rand"
)

// Generate 生成指定位数的纯数字随机验证码（如 6 位 → "038471"）。
// length 小于等于 0 时默认使用 6 位。
func Generate(length int) string {
	if length <= 0 {
		length = 6
	}
	max := 1
	for i := 0; i < length; i++ {
		max *= 10
	}
	n := rand.Intn(max) //nolint:gosec
	return fmt.Sprintf("%0*d", length, n)
}
