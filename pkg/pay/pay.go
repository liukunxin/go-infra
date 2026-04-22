// Package pay 提供微信支付（APIv3）与支付宝（RSA2）的轻量封装，便于业务侧快速接入。
//
// 子包：
//   - wechat：JSAPI/小程序、Native 扫码、订单查询/关单、退款、回调验签与解密
//   - alipay：APP 调起串、当面付预下单、订单查询、退款、异步通知验签
//
// 聚合入口可使用 Hub 一次注入两侧客户端。
package pay
