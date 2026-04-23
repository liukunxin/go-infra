package applepay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// NotificationDecodedPayload App Store Server Notification JWS payload 解码结果。
// 字段参考：https://developer.apple.com/documentation/appstoreservernotifications
type NotificationDecodedPayload struct {
	// NotificationType 通知类型，如 SUBSCRIBED、DID_RENEW、EXPIRED、REFUND 等。
	NotificationType string `json:"notificationType"`
	// Subtype 通知子类型，如 INITIAL_BUY、RESUBSCRIBE、UPGRADE 等。
	Subtype string `json:"subtype"`
	// NotificationUUID 通知唯一标识，用于幂等处理。
	NotificationUUID string `json:"notificationUUID"`
	// Version 通知版本，当前为 "2.0"。
	Version string `json:"version"`
	// SignedDate 通知签名时间（Unix 毫秒时间戳）。
	SignedDate int64 `json:"signedDate"`
	// Data 通知数据。
	Data NotificationData `json:"data"`
	// Summary 批量延长续订日期时的摘要信息（仅部分通知类型包含）。
	Summary *NotificationSummary `json:"summary,omitempty"`
}

// NotificationData 通知中的数据体。
type NotificationData struct {
	AppAppleID            int64  `json:"appAppleId"`
	BundleID              string `json:"bundleId"`
	BundleVersion         string `json:"bundleVersion"`
	Environment           string `json:"environment"`
	SignedRenewalInfo     string `json:"signedRenewalInfo"`
	SignedTransactionInfo string `json:"signedTransactionInfo"`
}

// NotificationSummary 批量延长续订日期操作的摘要。
type NotificationSummary struct {
	RequestIdentifier            string   `json:"requestIdentifier"`
	Environment                  string   `json:"environment"`
	AppAppleID                   int64    `json:"appAppleId"`
	ProductID                    string   `json:"productId"`
	StorefrontCountryCodes       []string `json:"storefrontCountryCodes"`
	FailedCount                  int64    `json:"failedCount"`
	SucceededCount               int64    `json:"succeededCount"`
}

// SendTestNotificationResponse POST /inApps/v1/notifications/test 响应。
type SendTestNotificationResponse struct {
	// TestNotificationToken 用于查询测试通知状态的令牌。
	TestNotificationToken string `json:"testNotificationToken"`
}

// CheckTestNotificationResponse GET /inApps/v1/notifications/test/{token} 响应。
type CheckTestNotificationResponse struct {
	// SignedPayload 测试通知的 JWS payload，使用 ParseServerNotification 解码。
	SignedPayload string `json:"signedPayload"`
	// SendAttempts 发送尝试记录。
	SendAttempts []NotificationSendAttempt `json:"sendAttempts"`
}

// NotificationSendAttempt 单次通知发送尝试的结果。
type NotificationSendAttempt struct {
	AttemptDate  int64  `json:"attemptDate"`
	SendAttemptResult string `json:"sendAttemptResult"`
}

// ParseServerNotification 解码 App Store Server Notification 中的 signedPayload（JWS 格式）。
// signedPayload 来自 Apple 推送到你服务器的 POST 请求体中的 "signedPayload" 字段。
//
// 若 appleRootCertPEM 不为空，则同时验证证书链；为空则仅解码（生产环境建议验证）。
func ParseServerNotification(signedPayload, appleRootCertPEM string) (*NotificationDecodedPayload, error) {
	var out NotificationDecodedPayload
	if appleRootCertPEM != "" {
		if err := verifyAndDecodeJWS(signedPayload, &out, []string{appleRootCertPEM}); err != nil {
			return nil, err
		}
	} else {
		if err := decodeJWSPayload(signedPayload, &out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

// RequestTestNotification 向 Apple 服务器请求发送一条测试通知。
// 返回 token，再用 GetTestNotificationStatus 查询发送结果。
func (c *Client) RequestTestNotification(ctx context.Context) (*SendTestNotificationResponse, error) {
	body, code, err := c.do(ctx, http.MethodPost, "/inApps/v1/notifications/test", nil)
	if err != nil {
		return nil, err
	}
	if !isHTTP2xx(code) {
		return nil, parseAPIError(body, code)
	}
	var out SendTestNotificationResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.TestNotificationToken == "" {
		return nil, fmt.Errorf("%w: empty testNotificationToken", ErrAPI)
	}
	return &out, nil
}

// GetTestNotificationStatus 查询测试通知的发送状态。
// testNotificationToken：RequestTestNotification 返回的令牌。
func (c *Client) GetTestNotificationStatus(ctx context.Context, testNotificationToken string) (*CheckTestNotificationResponse, error) {
	if testNotificationToken == "" {
		return nil, fmt.Errorf("%w: testNotificationToken required", ErrInvalidConfig)
	}
	path := "/inApps/v1/notifications/test/" + testNotificationToken
	body, code, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if !isHTTP2xx(code) {
		return nil, parseAPIError(body, code)
	}
	var out CheckTestNotificationResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
