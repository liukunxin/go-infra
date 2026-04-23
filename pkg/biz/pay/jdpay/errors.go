package jdpay

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidConfig 配置缺少必要字段。
	ErrInvalidConfig = errors.New("jdpay: invalid config")
	// ErrAPI 京东支付 API 返回业务级错误。
	ErrAPI = errors.New("jdpay: api error")
	// ErrVerifySign 响应或回调签名验证失败。
	ErrVerifySign = errors.New("jdpay: signature verify failed")
)

// APIError 京东支付 API 返回的错误结构。
type APIError struct {
	ResultCode string
	ResultMsg  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("jdpay api code=%s msg=%s", e.ResultCode, e.ResultMsg)
}

func (e *APIError) Unwrap() error { return ErrAPI }

// resultCodeOK 京东支付成功响应码。
const resultCodeOK = "000000"
