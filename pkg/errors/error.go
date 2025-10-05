package errors

import (
	"backend/go-infra/pkg/log"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type EventNotDefined string

func (e EventNotDefined) Error() string {
	return fmt.Sprintf("event not found, name: %s", string(e))
}

type ActionNotDefined string

func (e ActionNotDefined) Error() string {
	return fmt.Sprintf("action not found, name: %s", string(e))
}

const (
	StatusOK                  = Status(http.StatusOK)
	StatusBadRequest          = Status(http.StatusBadRequest)
	StatusUnauthorized        = Status(http.StatusUnauthorized)
	StatusForbidden           = Status(http.StatusForbidden)
	StatusNotFound            = Status(http.StatusNotFound)
	StatusTooManyRequests     = Status(http.StatusTooManyRequests)
	StatusInternalServerError = Status(http.StatusInternalServerError)
)

type Status int

func (s Status) Error() string {
	return http.StatusText(int(s))
}
func (s Status) HTTPCode() int {
	return int(s)
}

type Error struct {
	error
	Code   Code
	Status Status
}

func WarpError(status Status, code Code, err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		error:  err,
		Code:   code,
		Status: status,
	}
}

type ErrorResponse struct {
	Status  Status `json:"-"`
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func UnWarpErrorResponse(err error) *ErrorResponse {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	er := &ErrorResponse{Status: StatusInternalServerError, Code: int(CodeInternalCallFailed), Message: err.Error()}
	if e, ok := err.(*Error); ok {
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

// AsCode  解析error code
func AsCode(err error) Code {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return 0
}

// New returns an error object for the code, message.
func New(status Status, code Code, err error) *Error {
	return &Error{
		Status: status,
		Code:   code,
		error:  err,
	}
}

func (x *Error) Fields() log.Fields {
	f1 := log.Fields{
		"err_code":   x.Code,
		"err_status": x.Status,
		"err_msg":    x.Error(),
	}
	// 优先返回错误信息，没有才返回code的信息
	if len(x.Error()) != 0 {
		f1["err_msg"] = x.Error()
	}
	return f1
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
	return New(StatusInternalServerError, CodeInternalCallFailed, err)
}

func LogErrorRetrieve(err error) log.Fields {
	chatErr := FromError(err)
	if chatErr != nil {
		return chatErr.Fields()
	}
	return log.Fields{}
}
