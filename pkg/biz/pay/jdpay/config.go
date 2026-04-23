package jdpay

import (
	"net/http"
	"time"
)

// Config 京东支付配置。
//
// MerchantNo: 京东支付商户编号，在京东商家平台申请获取。
// AppKey: 应用密钥（appkey），与商户编号对应，用于标识接入方。
// PrivateKeyPEM: 商户 RSA 私钥 PEM（PKCS#8 或 PKCS#1，用于签名请求）。
// JDPublicKeyPEM: 京东支付平台 RSA 公钥 PEM（用于验签响应和回调通知）。
//
// IsSandbox: true 时连接沙箱环境（UAT），false 时连接生产环境。
//
// HTTPClient: 可选。非 nil 时使用该客户端发请求（HTTPTimeout 被忽略）；
// nil 时内部创建独立 *http.Client 并使用 HTTPTimeout（默认 15s）。
type Config struct {
	MerchantNo    string
	AppKey        string
	PrivateKeyPEM string
	JDPublicKeyPEM string
	IsSandbox     bool
	HTTPTimeout   time.Duration
	HTTPClient    *http.Client
}

func (c Config) normalized() Config {
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 15 * time.Second
	}
	return c
}

func (c Config) baseURL() string {
	if c.IsSandbox {
		return "https://payuat.jd.com"
	}
	return "https://payapp.jd.com"
}
