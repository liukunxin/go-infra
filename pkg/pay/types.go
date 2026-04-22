package pay

// Provider 支付渠道（业务层路由或埋点可用）
type Provider string

const (
	ProviderWechat Provider = "wechat"
	ProviderAlipay Provider = "alipay"
)
