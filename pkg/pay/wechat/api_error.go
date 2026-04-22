package wechat

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIError 微信返回的业务错误结构。
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
