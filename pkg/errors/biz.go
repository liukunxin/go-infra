package errors

import (
	"errors"
)

// BizError 业务错误：描述用户侧可读的操作失败原因。
// Controller 层识别该类型后以 HTTP 200 + 非零业务码返回，
// 而非 HTTP 500，确保前端能展示具体提示而非"系统异常"。
type BizError struct {
	msg string
}

func (e *BizError) Error() string { return e.msg }

// NewBiz 创建业务错误
func NewBiz(msg string) error {
	return &BizError{msg: msg}
}

// IsBiz 判断 err 是否为业务错误
func IsBiz(err error) bool {
	if err == nil {
		return false
	}
	var bizError *BizError
	ok := errors.As(err, &bizError)
	return ok
}
