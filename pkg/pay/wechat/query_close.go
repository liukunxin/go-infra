package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// TransactionQueryResult 订单查询结果（常用字段）。
type TransactionQueryResult struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
	TradeType     string `json:"trade_type"`
	BankType      string `json:"bank_type"`
	SuccessTime   string `json:"success_time"`
	Amount        struct {
		Total         int64  `json:"total"`
		PayerTotal    int64  `json:"payer_total"`
		Currency      string `json:"currency"`
		PayerCurrency string `json:"payer_currency"`
	} `json:"amount"`
}

// QueryOrderByOutTradeNo GET /v3/pay/transactions/out-trade-no/{out_trade_no}。
func (c *Client) QueryOrderByOutTradeNo(ctx context.Context, outTradeNo string) (*TransactionQueryResult, error) {
	path := fmt.Sprintf("/v3/pay/transactions/out-trade-no/%s?mchid=%s",
		url.PathEscape(outTradeNo), url.QueryEscape(c.cfg.MchID))
	respBody, code, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if !isHTTP2xx(code) {
		return nil, parseAPIError(respBody, code)
	}
	var out TransactionQueryResult
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CloseOrder POST /v3/pay/transactions/out-trade-no/{out_trade_no}/close。
func (c *Client) CloseOrder(ctx context.Context, outTradeNo string) error {
	path := fmt.Sprintf("/v3/pay/transactions/out-trade-no/%s/close", url.PathEscape(outTradeNo))
	body, err := json.Marshal(map[string]string{"mchid": c.cfg.MchID})
	if err != nil {
		return err
	}
	respBody, code, err := c.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	if !isHTTP2xx(code) {
		return parseAPIError(respBody, code)
	}
	return nil
}
