package errors

import (
	"context"
	"errors"
	"fmt"
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

// WrapError wraps an error with an HTTP status and business code.
// Returns nil if err is nil.
func WrapError(status Status, code Code, err error) error {
	if err == nil {
		return nil
	}
	return newError(status, code, err)
}

// FromError tries to convert an error to *Error.
// It supports wrapped errors. Unknown errors are mapped to 500 / CodeInternalCallFailed.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if se := new(Error); errors.As(err, &se) {
		return se
	}
	return newError(StatusInternalServerError, CodeInternalCallFailed, err)
}

// AsCode extracts the business Code from an error chain.
// Returns CodeUnknown if no *Error is found.
func AsCode(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeUnknown
}

// ErrorResponse is the JSON payload returned to callers on error.
type ErrorResponse struct {
	Status  Status `json:"-"`
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// UnwrapErrorResponse builds an ErrorResponse from an error.
// It maps context.Canceled to CodeUserContextCanceled regardless of wrapping depth.
func UnwrapErrorResponse(err error) *ErrorResponse {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}

	er := &ErrorResponse{
		Status:  StatusInternalServerError,
		Code:    int(CodeInternalCallFailed),
		Message: err.Error(),
	}

	var e *Error
	if errors.As(err, &e) {
		er.Status = e.Status
		er.Code = int(e.Code)
		er.Message = err.Error() // use full chain so outer context is preserved
	}

	// Reclassify context cancellation regardless of how deeply it is wrapped.
	if er.Code == int(CodeInternalCallFailed) && errors.Is(err, context.Canceled) {
		er.Code = int(CodeUserContextCanceled)
	}

	return er
}
