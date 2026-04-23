package errors

import (
	"errors"
)

// BizError describes a user-facing operation failure.
// Controller layer recognises this type and responds with HTTP 200 + non-zero business code,
// rather than HTTP 500, so the frontend can display a meaningful message.
//
// Use NewBiz for generic business errors, NewBizWithCode when the caller needs to
// distinguish error types programmatically (e.g. "insufficient balance" vs "user not found").
type BizError struct {
	code Code
	msg  string
}

func (e *BizError) Error() string { return e.msg }

// BizCode returns the business code embedded in the error, or CodeUnknown if none was set.
func (e *BizError) BizCode() Code { return e.code }

// NewBiz creates a business error without a specific code.
func NewBiz(msg string) error {
	return &BizError{msg: msg}
}

// NewBizWithCode creates a business error with a specific business code,
// allowing callers to differentiate error categories via AsCode / errors.As.
func NewBizWithCode(code Code, msg string) error {
	return &BizError{code: code, msg: msg}
}

// IsBiz reports whether err is (or wraps) a BizError.
func IsBiz(err error) bool {
	if err == nil {
		return false
	}
	var bizError *BizError
	return errors.As(err, &bizError)
}

// AsBizCode extracts the BizCode from a BizError in the chain.
// Returns CodeUnknown if err is not a BizError.
func AsBizCode(err error) Code {
	var e *BizError
	if errors.As(err, &e) {
		return e.BizCode()
	}
	return CodeUnknown
}
