package applepay

import (
	"net/http"
	"time"
)

// Config App Store Server API 配置。
//
// KeyID: App Store Connect API 密钥 ID（JWT Header kid），在 App Store Connect →
// 用户和访问 → 集成 → App Store Connect API 中创建，格式如 "XXXXXXXXXX"。
//
// IssuerID: App Store Connect 颁发者 ID（JWT Payload iss），在同一页面获取，为 UUID 格式。
//
// BundleID: 需要查询的 App Bundle Identifier，如 "com.example.myapp"。
//
// PrivateKey: 从 App Store Connect 下载的 .p8 文件内容，PKCS#8 格式 EC 私钥
// （-----BEGIN PRIVATE KEY----- 开头，对应 P-256 曲线）。
//
// AppleRootCertPEM: 用于验证 JWS 证书链的 Apple 根证书 PEM。
// 从 https://www.apple.com/certificateauthority/ 下载 AppleRootCA-G3.cer，
// 使用 openssl x509 -inform DER -in AppleRootCA-G3.cer -out root.pem 转换。
// 留空则跳过根证书链验证（不推荐用于生产环境）。
//
// IsSandbox: true 时连接沙箱环境（测试），false 时连接生产环境。
//
// HTTPClient: 可选。非 nil 时使用该客户端发请求（HTTPTimeout 被忽略）；
// nil 时内部创建独立 *http.Client 并使用 HTTPTimeout（默认 15s）。
type Config struct {
	KeyID            string
	IssuerID         string
	BundleID         string
	PrivateKey       string
	AppleRootCertPEM string
	IsSandbox        bool
	HTTPTimeout      time.Duration
	HTTPClient       *http.Client
}

func (c Config) normalized() Config {
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 15 * time.Second
	}
	return c
}

func (c Config) baseURL() string {
	if c.IsSandbox {
		return "https://api.storekit-sandbox.itunes.apple.com"
	}
	return "https://api.storekit.itunes.apple.com"
}
