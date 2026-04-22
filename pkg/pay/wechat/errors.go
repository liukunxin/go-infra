package wechat

import "errors"

var (
	ErrInvalidConfig = errors.New("wechat pay: invalid config")
	ErrAPI           = errors.New("wechat pay: api error")
	ErrVerifySign    = errors.New("wechat pay: notify signature verify failed")
)
