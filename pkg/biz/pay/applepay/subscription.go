package applepay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// SubscriptionStatus 单条自动续订订阅的当前状态。
type SubscriptionStatus struct {
	// Status 状态码：1=活跃，2=过期，3=账单宽限期，4=账单重试，5=撤销，6=暂停。
	Status int `json:"status"`
	// SignedRenewalInfo 续订信息 JWS 字符串，调用 DecodeJWSRenewalInfo 解码。
	SignedRenewalInfo string `json:"signedRenewalInfo"`
	// SignedTransactionInfo 最近一条交易 JWS 字符串，调用 DecodeJWSTransaction 解码。
	SignedTransactionInfo string `json:"signedTransactionInfo"`
}

// SubscriptionGroup 订阅组及其包含的各订阅状态列表。
type SubscriptionGroup struct {
	SubscriptionGroupIdentifier string               `json:"subscriptionGroupIdentifier"`
	LastTransactions            []SubscriptionStatus `json:"lastTransactions"`
}

// StatusResponse GET /inApps/v1/subscriptions/{transactionId} 响应。
type StatusResponse struct {
	AppAppleID  int64               `json:"appAppleId"`
	BundleID    string              `json:"bundleId"`
	Environment string              `json:"environment"`
	Data        []SubscriptionGroup `json:"data"`
}

// ExtendRenewalDateRequest PUT /inApps/v1/subscriptions/extend/{originalTransactionId} 请求体。
type ExtendRenewalDateRequest struct {
	// ExtendByDays 延长天数，最大 90 天。
	ExtendByDays int `json:"extendByDays"`
	// ExtendReasonCode 延长原因：1=客户满意度，2=其他原因，3=服务问题，4=应用崩溃。
	ExtendReasonCode int `json:"extendReasonCode"`
	// RequestIdentifier 请求幂等标识（UUID），同一 identifier 重复提交只执行一次。
	RequestIdentifier string `json:"requestIdentifier"`
}

// ExtendRenewalDateResponse 延长续订日期响应。
type ExtendRenewalDateResponse struct {
	// EffectiveDate 新的续订日期（Unix 毫秒时间戳）。
	EffectiveDate int64 `json:"effectiveDate"`
	// OriginalTransactionID 操作的原始交易 ID。
	OriginalTransactionID string `json:"originalTransactionId"`
	// Success 是否成功延长。
	Success bool `json:"success"`
	// WebOrderLineItemID 相关的 Web 订单行项目 ID。
	WebOrderLineItemID string `json:"webOrderLineItemId"`
}

// GetAllSubscriptionStatuses 查询用户所有自动续订订阅的当前状态。
// transactionID：用户名下任意一条订阅交易的 ID。
func (c *Client) GetAllSubscriptionStatuses(ctx context.Context, transactionID string) (*StatusResponse, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("%w: transactionID required", ErrInvalidConfig)
	}
	path := "/inApps/v1/subscriptions/" + url.PathEscape(transactionID)
	body, code, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if !isHTTP2xx(code) {
		return nil, parseAPIError(body, code)
	}
	var out StatusResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ExtendSubscriptionRenewalDate 延长用户订阅的续订日期（补偿服务中断等场景）。
// originalTransactionID：订阅的原始交易 ID（originalTransactionIdentifier）。
func (c *Client) ExtendSubscriptionRenewalDate(ctx context.Context, originalTransactionID string, req ExtendRenewalDateRequest) (*ExtendRenewalDateResponse, error) {
	if originalTransactionID == "" {
		return nil, fmt.Errorf("%w: originalTransactionID required", ErrInvalidConfig)
	}
	if req.RequestIdentifier == "" {
		return nil, fmt.Errorf("%w: RequestIdentifier required for idempotency", ErrInvalidConfig)
	}
	if req.ExtendByDays <= 0 || req.ExtendByDays > 90 {
		return nil, fmt.Errorf("%w: ExtendByDays must be between 1 and 90", ErrInvalidConfig)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	path := "/inApps/v1/subscriptions/extend/" + url.PathEscape(originalTransactionID)
	body, code, err := c.do(ctx, http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if !isHTTP2xx(code) {
		return nil, parseAPIError(body, code)
	}
	var out ExtendRenewalDateResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
