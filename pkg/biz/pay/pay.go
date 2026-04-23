// Package pay 提供微信支付（APIv3）、支付宝（RSA2）、Apple 内购（App Store Server API）
// 与京东支付的轻量封装，像"聚合收银台"一样对外暴露统一接口，便于业务侧快速接入。
//
// 子包：
//   - wechat：JSAPI/小程序、Native 扫码、订单查询/关单、退款、回调验签与解密
//   - alipay：APP 调起串、当面付预下单、订单查询、退款、异步通知验签
//   - applepay：App Store Server API；交易查询、订阅状态、退款历史、JWS 解码验签
//   - jdpay：统一下单（H5/PC/APP/扫码）、订单查询、退款、异步回调验签
//
// 聚合入口可使用 Hub 一次注入所有渠道客户端。
package pay

// Provider 标识支付渠道，可用于业务层路由或埋点。
type Provider string

const (
	ProviderWechat   Provider = "wechat"
	ProviderAlipay   Provider = "alipay"
	ProviderApplePay Provider = "applepay"
	ProviderJDPay    Provider = "jdpay"
)
