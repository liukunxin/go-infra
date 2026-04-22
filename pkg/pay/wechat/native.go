package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// NativeOrderRequest Native 支付（扫码）下单。
type NativeOrderRequest struct {
	Description string
	OutTradeNo  string
	NotifyURL   string
	AmountFen   int64
}

// NativeOrderResponse 返回 code_url 供生成二维码。
type NativeOrderResponse struct {
	CodeURL string `json:"code_url"`
}

// CreateNativeOrder 调用 /v3/pay/transactions/native。
func (c *Client) CreateNativeOrder(ctx context.Context, in NativeOrderRequest) (*NativeOrderResponse, error) {
	path := "/v3/pay/transactions/native"
	body := map[string]any{
		"appid":        c.cfg.AppID,
		"mchid":        c.cfg.MchID,
		"description":  in.Description,
		"out_trade_no": in.OutTradeNo,
		"notify_url":   in.NotifyURL,
		"amount": map[string]any{
			"total":    in.AmountFen,
			"currency": "CNY",
		},
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
	var out NativeOrderResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	if out.CodeURL == "" {
		return nil, fmt.Errorf("%w: empty code_url body=%s", ErrAPI, string(respBody))
	}
	return &out, nil
}
