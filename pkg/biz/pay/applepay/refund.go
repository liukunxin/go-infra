package applepay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// RefundHistoryResponse GET /inApps/v2/refund/lookup/{transactionId} 响应。
type RefundHistoryResponse struct {
	// HasMore 是否有更多退款记录。
	HasMore bool `json:"hasMore"`
	// Revision 下次请求游标。
	Revision string `json:"revision"`
	// SignedTransactions 退款交易的 JWS 字符串列表，使用 DecodeJWSTransaction 解码。
	// 解码后 RevocationReason != 0 表示退款原因：1=用户主动申请，其他值见 Apple 文档。
	SignedTransactions []string `json:"signedTransactions"`
}

// GetRefundHistory 获取用户退款历史（分页）。
// transactionID：用户名下任意一条交易 ID，用于定位用户。
// cursor：上次响应 RefundHistoryResponse.Revision；首次查询传空字符串。
func (c *Client) GetRefundHistory(ctx context.Context, transactionID, cursor string) (*RefundHistoryResponse, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("%w: transactionID required", ErrInvalidConfig)
	}
	path := "/inApps/v2/refund/lookup/" + url.PathEscape(transactionID)
	if cursor != "" {
		path += "?revision=" + url.QueryEscape(cursor)
	}
	body, code, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if !isHTTP2xx(code) {
		return nil, parseAPIError(body, code)
	}
	var out RefundHistoryResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAllRefundHistory 获取用户全量退款历史（自动翻页）。
func (c *Client) GetAllRefundHistory(ctx context.Context, transactionID string) ([]string, error) {
	var all []string
	cursor := ""
	for {
		resp, err := c.GetRefundHistory(ctx, transactionID, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.SignedTransactions...)
		if !resp.HasMore {
			break
		}
		cursor = resp.Revision
	}
	return all, nil
}
