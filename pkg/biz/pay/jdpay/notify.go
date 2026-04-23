package jdpay

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// NotifyPayload 京东支付异步回调通知内容。
// 京东支付以 POST JSON 方式推送到 callbackUrl。
type NotifyPayload struct {
	// MerchantNo 商户编号。
	MerchantNo string `json:"merchantNo"`
	// AppKey 应用密钥。
	AppKey string `json:"appkey"`
	// TradeNum 商户侧订单号。
	TradeNum string `json:"tradeNum"`
	// OrderNum 京东支付平台订单号。
	OrderNum string `json:"orderNum"`
	// TradeState 交易状态：TRADE_SUCCESS 表示支付成功。
	TradeState string `json:"tradeState"`
	// Amount 订单金额（分）。
	Amount string `json:"amount"`
	// Currency 货币类型。
	Currency string `json:"currency"`
	// PayTime 支付完成时间（格式 yyyyMMddHHmmss）。
	PayTime string `json:"payTime"`
	// Sign 签名字段（验签后不需要直接使用）。
	Sign string `json:"sign"`
}

// ParseAndVerifyNotify 解析并验签京东支付异步回调通知。
// 使用方在收到 POST 请求后调用此方法，验签通过后再更新订单状态。
//
// 验签通过后应：
//  1. 检查 payload.TradeState == "TRADE_SUCCESS"。
//  2. 用 payload.TradeNum 查找本地订单，核验金额一致。
//  3. 幂等更新订单状态（同一 TradeNum 可能收到多次回调）。
//  4. 返回 WriteNotifyAck。
func (c *Client) ParseAndVerifyNotify(r *http.Request) (*NotifyPayload, error) {
	var payload NotifyPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("jdpay: parse notify body: %w", err)
	}
	// 将 payload 转为 map 用于验签。
	params, err := structToStringMap(payload)
	if err != nil {
		return nil, err
	}
	if err := c.verifyNotifySign(params); err != nil {
		return nil, err
	}
	return &payload, nil
}

// WriteNotifyAck 向京东支付返回成功 ACK（HTTP 200，body 为 "success"）。
func WriteNotifyAck(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")
	_, _ = w.Write([]byte("success"))
}

// structToStringMap 将结构体转为 map[string]string，用于构建签名内容。
// 通过 JSON 中转处理，忽略空值字段。
func structToStringMap(v any) (map[string]string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var intermediate map[string]any
	if err := json.Unmarshal(raw, &intermediate); err != nil {
		return nil, err
	}
	result := make(map[string]string, len(intermediate))
	for k, val := range intermediate {
		if val == nil {
			continue
		}
		switch t := val.(type) {
		case string:
			if t != "" {
				result[k] = t
			}
		default:
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result, nil
}
