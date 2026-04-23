package applepay

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrInvalidConfig 配置缺少必要字段。
	ErrInvalidConfig = errors.New("applepay: invalid config")
	// ErrAPI App Store Server API 返回业务级错误。
	ErrAPI = errors.New("applepay: api error")
	// ErrVerifySign JWS 签名或证书链验证失败。
	ErrVerifySign = errors.New("applepay: jws signature verify failed")
)

// APIError App Store Server API 返回的错误结构（HTTP 4xx/5xx 时解析）。
// 实现了 error 接口与 Unwrap（返回 ErrAPI），支持 errors.Is(err, ErrAPI) 判断。
type APIError struct {
	HTTPStatus   int
	ErrorCode    int64  `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("applepay api status=%d code=%d msg=%s", e.HTTPStatus, e.ErrorCode, e.ErrorMessage)
}

func (e *APIError) Unwrap() error { return ErrAPI }

func parseAPIError(body []byte, status int) error {
	var e APIError
	e.HTTPStatus = status
	_ = json.Unmarshal(body, &e)
	if e.ErrorMessage == "" {
		e.ErrorMessage = "unknown error"
	}
	return &e
}

func isHTTP2xx(code int) bool { return code >= 200 && code < 300 }
