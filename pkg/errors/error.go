package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ---------------------------
// 特定错误类型
// ---------------------------

type EventNotDefined string

func (e EventNotDefined) Error() string {
	return fmt.Sprintf("event not found, name: %s", string(e))
}

type ActionNotDefined string

func (e ActionNotDefined) Error() string {
	return fmt.Sprintf("action not found, name: %s", string(e))
}

// ---------------------------
// 自定义错误类型
// ---------------------------

type Error struct {
	error
	Code   Code
	Status Status
}

// newError returns an error object for the code, message.
func newError(status Status, code Code, err error) *Error {
	return &Error{
		Status: status,
		Code:   code,
		error:  err,
	}
}

func (e *Error) Unwrap() error {
	return e.error
}

// WarpError 对外包装
func WarpError(status Status, code Code, err error) error {
	if err == nil {
		return nil
	}
	return newError(status, code, err)
}

// FromError try to convert an error to *Error.
// It supports wrapped errors.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if se := new(Error); errors.As(err, &se) {
		return se
	}
	return newError(StatusInternalServerError, CodeInternalCallFailed, err)
}

// AsCode  解析error code
func AsCode(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeUnknown
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Status  Status `json:"-"`
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// UnWarpErrorResponse 从 error 构造响应
func UnWarpErrorResponse(err error) *ErrorResponse {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	er := &ErrorResponse{Status: StatusInternalServerError, Code: int(CodeInternalCallFailed), Message: err.Error()}
	var e *Error
	if errors.As(err, &e) {
		er = &ErrorResponse{
			Status:  e.Status,
			Code:    int(e.Code),
			Message: e.Error(),
		}
		// 优先返回错误信息，没有才返回code的信息
		if len(err.Error()) != 0 {
			er.Message = err.Error()
		}
	}
	// 如果是内部逻辑错误50005且为用户主动取消产生的，归为预期内错误
	if er.Code == int(CodeInternalCallFailed) && strings.Contains(er.Message, "context canceled") {
		er.Code = int(CodeUserContextCanceled)
	}
	return er
}
