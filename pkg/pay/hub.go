package pay

import (
	"github.com/liukunxin/go-infra/pkg/pay/alipay"
	"github.com/liukunxin/go-infra/pkg/pay/wechat"
)

// Hub 聚合微信与支付宝客户端，业务可一次注入、按需调用。
// 未启用的渠道对应字段可为 nil。
type Hub struct {
	Wechat *wechat.Client
	Alipay *alipay.Client
}

// NewHub 创建聚合入口；wx、ali 均可为 nil（仅使用一侧时传另一侧 nil）。
func NewHub(wx *wechat.Client, ali *alipay.Client) *Hub {
	return &Hub{Wechat: wx, Alipay: ali}
}
