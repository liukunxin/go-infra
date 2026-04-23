package jdpay

import (
	"context"
	"fmt"
	"time"
)

// TradeType 交易类型（支付场景）。
type TradeType string

const (
	// TradeTypeH5 H5 网页支付（移动端浏览器，拉起京东/微信/支付宝等收银台）。
	TradeTypeH5 TradeType = "H5"
	// TradeTypePC PC 网关支付（电脑浏览器跳转收银台）。
	TradeTypePC TradeType = "PC"
	// TradeTypeAPP APP 内支付（拉起京东钱包 App 或收银台页面）。
	TradeTypeAPP TradeType = "APP"
	// TradeTypeQRCode 扫码支付（商家展示二维码，用户扫码）。
	TradeTypeQRCode TradeType = "QRCODE"
)

// UnifiedOrderRequest 统一下单请求。
type UnifiedOrderRequest struct {
	// TradeNum 商户侧订单号（全局唯一，最长 32 位）。
	TradeNum string
	// TradeName 商品名称（最长 128 字符）。
	TradeName string
	// TradeDesc 商品描述（可选，最长 256 字符）。
	TradeDesc string
	// Amount 订单金额（单位：分）。
	Amount int64
	// Currency 货币类型，默认 CNY。
	Currency string
	// TradeType 支付场景。
	TradeType TradeType
	// CallbackURL 异步通知回调地址（必须 HTTPS）。
	CallbackURL string
	// ReturnURL 同步跳转地址（H5/PC 场景支付完成后跳转，可选）。
	ReturnURL string
	// UserIP 用户 IP（H5/PC 场景必填）。
	UserIP string
	// ExpireMinutes 订单有效时长（分钟），0 使用默认值（通常 30 分钟）。
	ExpireMinutes int
}

// UnifiedOrderResponse 统一下单响应。
type UnifiedOrderResponse struct {
	// OrderNum 京东支付平台订单号。
	OrderNum string
	// TradeNum 商户侧订单号（与请求一致）。
	TradeNum string
	// PayURL 支付跳转 URL（H5/PC 场景：引导用户访问此地址完成支付）。
	PayURL string
	// QRCode 二维码内容（扫码场景）。
	QRCode string
	// Token APP 支付令牌（APP 场景：传入京东钱包 SDK）。
	Token string
}

// UnifiedOrder 统一下单接口，根据 TradeType 返回相应的支付参数。
// POST /unified/pay/order
func (c *Client) UnifiedOrder(ctx context.Context, req UnifiedOrderRequest) (*UnifiedOrderResponse, error) {
	if req.TradeNum == "" || req.TradeName == "" || req.Amount <= 0 || req.CallbackURL == "" {
		return nil, fmt.Errorf("%w: TradeNum/TradeName/Amount/CallbackURL required", ErrInvalidConfig)
	}
	if req.TradeType == "" {
		return nil, fmt.Errorf("%w: TradeType required", ErrInvalidConfig)
	}

	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	tradeTime := time.Now().Format("20060102150405")

	params := map[string]string{
		"tradeNum":    req.TradeNum,
		"tradeName":   req.TradeName,
		"tradeTime":   tradeTime,
		"amount":      fmt.Sprintf("%d", req.Amount),
		"currency":    currency,
		"tradeType":   string(req.TradeType),
		"callbackUrl": req.CallbackURL,
	}
	if req.TradeDesc != "" {
		params["tradeDesc"] = req.TradeDesc
	}
	if req.ReturnURL != "" {
		params["returnUrl"] = req.ReturnURL
	}
	if req.UserIP != "" {
		params["userIp"] = req.UserIP
	}
	if req.ExpireMinutes > 0 {
		expireTime := time.Now().Add(time.Duration(req.ExpireMinutes) * time.Minute).Format("20060102150405")
		params["expireTime"] = expireTime
	}

	result, err := c.post(ctx, "/unified/pay/order", params)
	if err != nil {
		return nil, err
	}

	return &UnifiedOrderResponse{
		OrderNum: result["orderNum"],
		TradeNum: result["tradeNum"],
		PayURL:   result["payUrl"],
		QRCode:   result["qrCode"],
		Token:    result["token"],
	}, nil
}

// CloseOrder 关闭订单（订单创建后未支付，需主动关闭时调用）。
// POST /unified/pay/close
func (c *Client) CloseOrder(ctx context.Context, tradeNum string) error {
	if tradeNum == "" {
		return fmt.Errorf("%w: tradeNum required", ErrInvalidConfig)
	}
	params := map[string]string{
		"tradeNum": tradeNum,
	}
	_, err := c.post(ctx, "/unified/pay/close", params)
	return err
}
