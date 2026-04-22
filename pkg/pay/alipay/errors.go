package alipay

import "errors"

var (
	ErrInvalidConfig = errors.New("alipay: invalid config")
	ErrAPI           = errors.New("alipay: api error")
	ErrVerifySign    = errors.New("alipay: signature verify failed")
)
