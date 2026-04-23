package jdpay

import (
	"context"
	"fmt"
)

// OrderQueryResponse 订单查询响应。
type OrderQueryResponse struct {
	// OrderNum 京东支付平台订单号。
	OrderNum string
	// TradeNum 商户侧订单号。
	TradeNum string
	// TradeState 交易状态：
	//   WAIT_BUYER_PAY - 等待支付
	//   TRADE_SUCCESS  - 支付成功
	//   TRADE_CLOSED   - 交易关闭
	//   TRADE_REFUND   - 已退款
	TradeState string
	// Amount 订单金额（分）。
	Amount string
	// Currency 货币类型。
	Currency string
	// PayTime 支付完成时间（格式 yyyyMMddHHmmss）。
	PayTime string
	// BuyerID 买家在京东支付的用户 ID（可选，视商户协议）。
	BuyerID string
}

// QueryOrder 查询订单状态。
// POST /unified/pay/query
func (c *Client) QueryOrder(ctx context.Context, tradeNum string) (*OrderQueryResponse, error) {
	if tradeNum == "" {
		return nil, fmt.Errorf("%w: tradeNum required", ErrInvalidConfig)
	}
	params := map[string]string{
		"tradeNum": tradeNum,
	}
	result, err := c.post(ctx, "/unified/pay/query", params)
	if err != nil {
		return nil, err
	}
	return &OrderQueryResponse{
		OrderNum:   result["orderNum"],
		TradeNum:   result["tradeNum"],
		TradeState: result["tradeState"],
		Amount:     result["amount"],
		Currency:   result["currency"],
		PayTime:    result["payTime"],
		BuyerID:    result["buyerId"],
	}, nil
}

// RefundQueryResponse 退款查询响应。
type RefundQueryResponse struct {
	// RefundNum 京东支付退款单号。
	RefundNum string
	// OutRefundNo 商户侧退款单号。
	OutRefundNo string
	// RefundState 退款状态：
	//   REFUND_SUCCESS - 退款成功
	//   REFUND_FAIL    - 退款失败
	//   REFUNDING      - 退款处理中
	RefundState string
	// RefundAmount 退款金额（分）。
	RefundAmount string
	// RefundTime 退款完成时间（格式 yyyyMMddHHmmss）。
	RefundTime string
}

// QueryRefund 查询退款状态。
// POST /unified/pay/refund/query
func (c *Client) QueryRefund(ctx context.Context, outRefundNo string) (*RefundQueryResponse, error) {
	if outRefundNo == "" {
		return nil, fmt.Errorf("%w: outRefundNo required", ErrInvalidConfig)
	}
	params := map[string]string{
		"outRefundNo": outRefundNo,
	}
	result, err := c.post(ctx, "/unified/pay/refund/query", params)
	if err != nil {
		return nil, err
	}
	return &RefundQueryResponse{
		RefundNum:    result["refundNum"],
		OutRefundNo:  result["outRefundNo"],
		RefundState:  result["refundState"],
		RefundAmount: result["refundAmount"],
		RefundTime:   result["refundTime"],
	}, nil
}
