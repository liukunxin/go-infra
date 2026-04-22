package pay

import (
	"github.com/liukunxin/go-infra/pkg/pay/alipay"
	"github.com/liukunxin/go-infra/pkg/pay/wechat"
)

// Hub 聚合微信与支付宝客户端。字段不可导出，防止外部随意置 nil 引发 panic。
// 通过 Wechat() / Alipay() 安全取用，渠道未配置时返回 nil，调用方应在使用前判 nil。
type Hub struct {
	wx  *wechat.Client
	ali *alipay.Client
}

// NewHub 创建聚合入口；wx、ali 均可为 nil（仅使用一侧时传另一侧 nil）。
func NewHub(wx *wechat.Client, ali *alipay.Client) *Hub {
	return &Hub{wx: wx, ali: ali}
}

// Wechat 返回微信支付客户端，未配置时为 nil。
func (h *Hub) Wechat() *wechat.Client { return h.wx }

// Alipay 返回支付宝客户端，未配置时为 nil。
func (h *Hub) Alipay() *alipay.Client { return h.ali }
