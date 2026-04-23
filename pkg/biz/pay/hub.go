package pay

import (
	"github.com/liukunxin/go-infra/pkg/biz/pay/alipay"
	"github.com/liukunxin/go-infra/pkg/biz/pay/applepay"
	"github.com/liukunxin/go-infra/pkg/biz/pay/jdpay"
	"github.com/liukunxin/go-infra/pkg/biz/pay/wechat"
)

// Hub 聚合全部支付渠道客户端。字段不可导出，防止外部随意置 nil 引发 panic。
// 通过对应方法安全取用，渠道未配置时返回 nil，调用方应在使用前判 nil。
type Hub struct {
	wx    *wechat.Client
	ali   *alipay.Client
	apple *applepay.Client
	jd    *jdpay.Client
}

// HubOption Hub 构造选项，按需传入各渠道客户端。
type HubOption func(*Hub)

// WithWechat 注入微信支付客户端。
func WithWechat(c *wechat.Client) HubOption { return func(h *Hub) { h.wx = c } }

// WithAlipay 注入支付宝客户端。
func WithAlipay(c *alipay.Client) HubOption { return func(h *Hub) { h.ali = c } }

// WithApplePay 注入 Apple 内购（App Store Server API）客户端。
func WithApplePay(c *applepay.Client) HubOption { return func(h *Hub) { h.apple = c } }

// WithJDPay 注入京东支付客户端。
func WithJDPay(c *jdpay.Client) HubOption { return func(h *Hub) { h.jd = c } }

// NewHub 创建聚合入口，按需传入渠道客户端。
//
//	hub := pay.NewHub(
//	    pay.WithWechat(wxCli),
//	    pay.WithAlipay(aliCli),
//	    pay.WithApplePay(appleCli),
//	    pay.WithJDPay(jdCli),
//	)
func NewHub(opts ...HubOption) *Hub {
	h := &Hub{}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Wechat 返回微信支付客户端，未配置时为 nil。
func (h *Hub) Wechat() *wechat.Client { return h.wx }

// Alipay 返回支付宝客户端，未配置时为 nil。
func (h *Hub) Alipay() *alipay.Client { return h.ali }

// ApplePay 返回 Apple 内购客户端，未配置时为 nil。
func (h *Hub) ApplePay() *applepay.Client { return h.apple }

// JDPay 返回京东支付客户端，未配置时为 nil。
func (h *Hub) JDPay() *jdpay.Client { return h.jd }
