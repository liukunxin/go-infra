package wechat

import (
	"net/http"
	"time"
)

// Config 微信支付 APIv3（直连商户）配置。
// PrivateKeyPEM：商户 API 证书私钥（PKCS#8 或 PKCS#1 PEM）。
// CertificateSerialNo：商户证书序列号（与私钥匹配）。
// APIv3Key：商户平台设置的 APIv3 密钥（32 个字符，用于回调解密等）。
// PlatformCertPEM：微信支付平台证书 PEM（-----BEGIN CERTIFICATE-----，商户平台下载），用于验签回调 HTTP 头。
// 平台证书会轮换：当前实现按「单份平台证书」验签；轮换后请及时更新 PEM，或后续扩展为按 Wechatpay-Serial 多证书映射。
//
// HTTPClient：可选。非 nil 时使用该客户端发请求（此时 HTTPTimeout 被忽略，以注入实例的 Timeout/Transport 为准）；nil 时内部创建独立 *http.Client 并使用 HTTPTimeout。
type Config struct {
	AppID               string
	MchID               string
	CertificateSerialNo string
	PrivateKeyPEM       string
	APIv3Key            string
	PlatformCertPEM     string
	HTTPTimeout         time.Duration
	HTTPClient          *http.Client
}

func (c Config) normalized() Config {
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 15 * time.Second
	}
	return c
}
