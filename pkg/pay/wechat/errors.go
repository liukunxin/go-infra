package wechat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// 哨兵错误，可用 errors.Is 判断类型。
var (
	ErrInvalidConfig        = errors.New("wechat pay: invalid config")
	ErrAPI                  = errors.New("wechat pay: api error")
	ErrVerifySign           = errors.New("wechat pay: notify signature verify failed")
	ErrUnsupportedAlgorithm = errors.New("wechat pay: unsupported encryption algorithm")
)

// APIError 是微信 APIv3 返回的业务错误，实现了 error 与 Unwrap（返回 ErrAPI）。
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Body    string `json:"-"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("wechat pay api status=%d code=%s msg=%s body=%s", e.Status, e.Code, e.Message, e.Body)
}

func (e *APIError) Unwrap() error { return ErrAPI }

func parseAPIError(body []byte, status int) error {
	var e APIError
	e.Status = status
	e.Body = string(body)
	_ = json.Unmarshal(body, &e)
	if e.Code == "" && e.Message == "" {
		e.Message = "unknown error"
	}
	return &e
}

func isHTTP2xx(code int) bool { return code >= http.StatusOK && code < http.StatusMultipleChoices }
