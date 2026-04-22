package alipay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// AppPayRequest APP 支付下单（生成 orderStr 给客户端 SDK，本地签名不落库网关）。
type AppPayRequest struct {
	Subject     string
	OutTradeNo  string
	TotalAmount string // 元，如 "0.01"
	NotifyURL   string
	Body        string
}

// TradeAppPay 生成 APP 调起支付串（alipay.trade.app.pay 参数 + RSA2 签名）。
func (c *Client) TradeAppPay(in AppPayRequest) (orderStr string, err error) {
	if in.OutTradeNo == "" || in.Subject == "" || in.TotalAmount == "" {
		return "", fmt.Errorf("%w: OutTradeNo/Subject/TotalAmount required", ErrInvalidConfig)
	}
	biz := map[string]any{
		"subject":      in.Subject,
		"out_trade_no": in.OutTradeNo,
		"total_amount": in.TotalAmount,
		"product_code": "QUICK_MSECURITY_PAY",
	}
	if in.Body != "" {
		biz["body"] = in.Body
	}
	extra := url.Values{}
	if in.NotifyURL != "" {
		extra.Set("notify_url", in.NotifyURL)
	}
	v, err := c.signedForm("alipay.trade.app.pay", biz, extra)
	if err != nil {
		return "", err
	}
	return v.Encode(), nil
}

// PrecreateRequest 当面付预创建（扫码）。
type PrecreateRequest struct {
	Subject     string
	OutTradeNo  string
	TotalAmount string
	NotifyURL   string
	StoreID     string
}

// PrecreateResponse 返回二维码内容。
type PrecreateResponse struct {
	QRCode string `json:"qr_code"`
}

// TradePrecreate 调用 alipay.trade.precreate。
func (c *Client) TradePrecreate(ctx context.Context, in PrecreateRequest) (*PrecreateResponse, error) {
	if in.OutTradeNo == "" || in.Subject == "" || in.TotalAmount == "" {
		return nil, fmt.Errorf("%w: OutTradeNo/Subject/TotalAmount required", ErrInvalidConfig)
	}
	biz := map[string]any{
		"subject":      in.Subject,
		"out_trade_no": in.OutTradeNo,
		"total_amount": in.TotalAmount,
	}
	if in.StoreID != "" {
		biz["store_id"] = in.StoreID
	}
	extra := url.Values{}
	if in.NotifyURL != "" {
		extra.Set("notify_url", in.NotifyURL)
	}
	root, err := c.postGateway(ctx, "alipay.trade.precreate", biz, extra)
	if err != nil {
		return nil, err
	}
	raw, ok := root["alipay_trade_precreate_response"]
	if !ok {
		return nil, fmt.Errorf("%w: missing alipay_trade_precreate_response", ErrAPI)
	}
	var resp struct {
		Code   string `json:"code"`
		Msg    string `json:"msg"`
		QRCode string `json:"qr_code"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "10000" {
		return nil, fmt.Errorf("%w: code=%s msg=%s", ErrAPI, resp.Code, resp.Msg)
	}
	if resp.QRCode == "" {
		return nil, fmt.Errorf("%w: empty qr_code", ErrAPI)
	}
	return &PrecreateResponse{QRCode: resp.QRCode}, nil
}

// TradeQueryRequest 订单查询。
type TradeQueryRequest struct {
	OutTradeNo string
}

// TradeQueryResponse 常用查询字段。
type TradeQueryResponse struct {
	TradeStatus string `json:"trade_status"`
	OutTradeNo  string `json:"out_trade_no"`
	TradeNo     string `json:"trade_no"`
	BuyerUserID string `json:"buyer_user_id"`
	TotalAmount string `json:"total_amount"`
}

// TradeQuery 调用 alipay.trade.query。
func (c *Client) TradeQuery(ctx context.Context, in TradeQueryRequest) (*TradeQueryResponse, error) {
	if in.OutTradeNo == "" {
		return nil, fmt.Errorf("%w: OutTradeNo required", ErrInvalidConfig)
	}
	biz := map[string]any{
		"out_trade_no": in.OutTradeNo,
	}
	root, err := c.postGateway(ctx, "alipay.trade.query", biz, nil)
	if err != nil {
		return nil, err
	}
	raw, ok := root["alipay_trade_query_response"]
	if !ok {
		return nil, fmt.Errorf("%w: missing alipay_trade_query_response", ErrAPI)
	}
	var resp struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		TradeQueryResponse
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "10000" {
		return nil, fmt.Errorf("%w: code=%s msg=%s", ErrAPI, resp.Code, resp.Msg)
	}
	if resp.OutTradeNo == "" {
		resp.OutTradeNo = in.OutTradeNo
	}
	out := resp.TradeQueryResponse
	return &out, nil
}

// RefundRequest 退款。
type RefundRequest struct {
	OutTradeNo   string
	RefundAmount string // 元
	RefundReason string
	OutRequestNo string // 部分退款幂等号
}

// RefundResponse 退款结果摘要。
type RefundResponse struct {
	OutTradeNo   string `json:"out_trade_no"`
	TradeNo      string `json:"trade_no"`
	RefundFee    string `json:"refund_fee"`
	OutRequestNo string `json:"out_request_no"`
}

// TradeRefund 调用 alipay.trade.refund。
func (c *Client) TradeRefund(ctx context.Context, in RefundRequest) (*RefundResponse, error) {
	if in.OutTradeNo == "" || in.RefundAmount == "" || in.OutRequestNo == "" {
		return nil, fmt.Errorf("%w: OutTradeNo/RefundAmount/OutRequestNo required", ErrInvalidConfig)
	}
	biz := map[string]any{
		"out_trade_no":   in.OutTradeNo,
		"refund_amount":  in.RefundAmount,
		"out_request_no": in.OutRequestNo,
	}
	if in.RefundReason != "" {
		biz["refund_reason"] = in.RefundReason
	}
	root, err := c.postGateway(ctx, "alipay.trade.refund", biz, nil)
	if err != nil {
		return nil, err
	}
	raw, ok := root["alipay_trade_refund_response"]
	if !ok {
		return nil, fmt.Errorf("%w: missing alipay_trade_refund_response", ErrAPI)
	}
	var resp struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		RefundResponse
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "10000" {
		return nil, fmt.Errorf("%w: code=%s msg=%s", ErrAPI, resp.Code, resp.Msg)
	}
	out := resp.RefundResponse
	return &out, nil
}
