package wechat

import (
	"context"
	"encoding/json"
	"net/http"
)

// RefundRequest 退款请求（金额单位：分）。
type RefundRequest struct {
	OutTradeNo  string
	OutRefundNo string
	Reason      string
	RefundFen   int64
	TotalFen    int64 // 原订单总金额
}

// RefundResult 退款受理结果（常用字段）。
type RefundResult struct {
	RefundID      string `json:"refund_id"`
	OutRefundNo   string `json:"out_refund_no"`
	TransactionID string `json:"transaction_id"`
	OutTradeNo    string `json:"out_trade_no"`
	Status        string `json:"status"`
}

// Refund 调用 /v3/refund/domestic/refunds。
func (c *Client) Refund(ctx context.Context, in RefundRequest) (*RefundResult, error) {
	path := "/v3/refund/domestic/refunds"
	body := map[string]any{
		"out_trade_no":  in.OutTradeNo,
		"out_refund_no": in.OutRefundNo,
		"amount": map[string]any{
			"refund":   in.RefundFen,
			"total":    in.TotalFen,
			"currency": "CNY",
		},
	}
	if in.Reason != "" {
		body["reason"] = in.Reason
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	respBody, code, err := c.do(ctx, http.MethodPost, path, raw)
	if err != nil {
		return nil, err
	}
	if !isHTTP2xx(code) {
		return nil, parseAPIError(respBody, code)
	}
	var out RefundResult
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
