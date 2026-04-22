package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// JSAPIOrderRequest JSAPI/小程序下单请求。
type JSAPIOrderRequest struct {
	Description string
	OutTradeNo  string
	NotifyURL   string
	AmountFen   int64
	OpenID      string
}

// JSAPIOrderResponse 下单成功返回 prepay_id。
type JSAPIOrderResponse struct {
	PrepayID string `json:"prepay_id"`
}

// CreateJSAPIOrder 调用 /v3/pay/transactions/jsapi。
func (c *Client) CreateJSAPIOrder(ctx context.Context, in JSAPIOrderRequest) (*JSAPIOrderResponse, error) {
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

// ClientPayParams 前端 wx.requestPayment / 小程序调起支付参数（APIv3 调起签名）。
type ClientPayParams struct {
	AppID     string `json:"appId"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

// BuildClientPayParams 根据 prepay_id 生成调起支付参数（SignType 固定 RSA）。
func (c *Client) BuildClientPayParams(prepayID string) (*ClientPayParams, error) {
	ts := fmt.Sprintf("%d", time.Now().Unix())
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
