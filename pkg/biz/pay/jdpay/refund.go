package jdpay

import (
	"context"
	"fmt"
)

// ApplyRefundRequest 申请退款请求。
type ApplyRefundRequest struct {
	// TradeNum 商户侧原订单号。
	TradeNum string
	// OutRefundNo 商户侧退款单号（全局唯一，用于幂等）。
	OutRefundNo string
	// RefundAmount 退款金额（单位：分，不超过原订单金额）。
	RefundAmount int64
	// RefundReason 退款原因（可选）。
	RefundReason string
}

// ApplyRefundResponse 申请退款响应。
type ApplyRefundResponse struct {
	// RefundNum 京东支付退款单号。
	RefundNum string
	// OutRefundNo 商户侧退款单号（与请求一致）。
	OutRefundNo string
	// TradeNum 商户侧原订单号。
	TradeNum string
	// RefundState 退款状态（初始状态通常为 REFUNDING）。
	RefundState string
}

// ApplyRefund 申请退款。
// 退款为异步处理，提交成功后需通过 QueryRefund 或等待回调确认最终结果。
// POST /unified/pay/refund/apply
func (c *Client) ApplyRefund(ctx context.Context, req ApplyRefundRequest) (*ApplyRefundResponse, error) {
	if req.TradeNum == "" || req.OutRefundNo == "" || req.RefundAmount <= 0 {
		return nil, fmt.Errorf("%w: TradeNum/OutRefundNo required, RefundAmount must be > 0", ErrInvalidConfig)
	}
	params := map[string]string{
		"tradeNum":     req.TradeNum,
		"outRefundNo":  req.OutRefundNo,
		"refundAmount": fmt.Sprintf("%d", req.RefundAmount),
	}
	if req.RefundReason != "" {
		params["refundReason"] = req.RefundReason
	}
	result, err := c.post(ctx, "/unified/pay/refund/apply", params)
	if err != nil {
		return nil, err
	}
	return &ApplyRefundResponse{
		RefundNum:   result["refundNum"],
		OutRefundNo: result["outRefundNo"],
		TradeNum:    result["tradeNum"],
		RefundState: result["refundState"],
	}, nil
}
