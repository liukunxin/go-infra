package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ---- JSAPI / 小程序 -------------------------------------------------------

// JSAPIOrderRequest JSAPI/小程序下单请求。
type JSAPIOrderRequest struct {
	Description string
	OutTradeNo  string
	NotifyURL   string
	AmountFen   int64 // 单位：分
	OpenID      string
}

// JSAPIOrderResponse 下单成功返回 prepay_id。
type JSAPIOrderResponse struct {
	PrepayID string `json:"prepay_id"`
}

// CreateJSAPIOrder 调用 /v3/pay/transactions/jsapi。
func (c *Client) CreateJSAPIOrder(ctx context.Context, in JSAPIOrderRequest) (*JSAPIOrderResponse, error) {
	if in.OutTradeNo == "" || in.OpenID == "" || in.NotifyURL == "" || in.AmountFen <= 0 {
		return nil, fmt.Errorf("%w: OutTradeNo/OpenID/NotifyURL required, AmountFen must be > 0", ErrInvalidConfig)
	}
	path := "/v3/pay/transactions/jsapi"
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
		"payer": map[string]any{
			"openid": in.OpenID,
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
	var out JSAPIOrderResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	if out.PrepayID == "" {
		return nil, fmt.Errorf("%w: empty prepay_id body=%s", ErrAPI, string(respBody))
	}
	return &out, nil
}

// ClientPayParams 前端 wx.requestPayment / 小程序调起支付参数（APIv3 RSA 签名）。
type ClientPayParams struct {
	AppID     string `json:"appId"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

// BuildClientPayParams 根据 prepay_id 生成小程序 / 公众号调起支付参数。
func (c *Client) BuildClientPayParams(prepayID string) (*ClientPayParams, error) {
	if prepayID == "" {
		return nil, fmt.Errorf("%w: prepayID required", ErrInvalidConfig)
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randomNonce()
	pkg := "prepay_id=" + prepayID
	msg := c.cfg.AppID + "\n" + ts + "\n" + nonce + "\n" + pkg + "\n"
	paySign, err := signSHA256WithRSA(c.priv, msg)
	if err != nil {
		return nil, err
	}
	return &ClientPayParams{
		AppID:     c.cfg.AppID,
		TimeStamp: ts,
		NonceStr:  nonce,
		Package:   pkg,
		SignType:  "RSA",
		PaySign:   paySign,
	}, nil
}

// ---- Native 扫码 ----------------------------------------------------------

// NativeOrderRequest Native 支付（扫码）下单。
type NativeOrderRequest struct {
	Description string
	OutTradeNo  string
	NotifyURL   string
	AmountFen   int64 // 单位：分
}

// NativeOrderResponse 返回 code_url 供生成二维码。
type NativeOrderResponse struct {
	CodeURL string `json:"code_url"`
}

// CreateNativeOrder 调用 /v3/pay/transactions/native。
func (c *Client) CreateNativeOrder(ctx context.Context, in NativeOrderRequest) (*NativeOrderResponse, error) {
	if in.OutTradeNo == "" || in.NotifyURL == "" || in.AmountFen <= 0 {
		return nil, fmt.Errorf("%w: OutTradeNo/NotifyURL required, AmountFen must be > 0", ErrInvalidConfig)
	}
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
