package alipay

import (
	"net/http"
	"time"
)

// Config 支付宝开放平台配置。
// PrivateKeyPEM：应用私钥（PKCS#1/PKCS#8 PEM）。
// AlipayPublicKeyPEM：支付宝公钥 PEM（用于验签异步通知与同步响应）。
//
// HTTPClient：可选。非 nil 时使用该客户端请求网关（HTTPTimeout 忽略）；nil 时内部创建独立 *http.Client 并使用 HTTPTimeout。
type Config struct {
	AppID              string
	PrivateKeyPEM      string
	AlipayPublicKeyPEM string
	IsProduction       bool
	HTTPTimeout        time.Duration
	Charset            string // 默认 utf-8
	HTTPClient         *http.Client
}

func (c Config) normalized() Config {
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 15 * time.Second
	}
	if c.Charset == "" {
		c.Charset = "utf-8"
	}
	return c
}

func (c Config) gatewayURL() string {
	if c.IsProduction {
		return "https://openapi.alipay.com/gateway.do"
	}
	return "https://openapi.alipaydev.com/gateway.do"
}
