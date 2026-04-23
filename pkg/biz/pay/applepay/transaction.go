package applepay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// TransactionInfoResponse GET /inApps/v1/transactions/{transactionId} 响应。
type TransactionInfoResponse struct {
	// SignedTransactionInfo Apple 签名的 JWS 交易字符串，调用 DecodeJWSTransaction 解码。
	SignedTransactionInfo string `json:"signedTransactionInfo"`
}

// HistoryResponse GET /inApps/v2/history/{transactionId} 响应。
type HistoryResponse struct {
	// AppAppleID App 在 App Store 的数字 ID。
	AppAppleID int64 `json:"appAppleId"`
	// BundleID App Bundle Identifier。
	BundleID string `json:"bundleId"`
	// Environment 环境标识：Production 或 Sandbox。
	Environment string `json:"environment"`
	// HasMore 是否有更多记录（分页）。
	HasMore bool `json:"hasMore"`
	// Revision 下次请求的游标，传入 GetTransactionHistory 的 cursor 参数。
	Revision string `json:"revision"`
	// SignedTransactions JWS 交易字符串列表，使用 DecodeJWSTransaction 逐条解码。
	SignedTransactions []string `json:"signedTransactions"`
}

// GetTransactionInfo 查询单条交易信息。
// transactionID：客户端通过 StoreKit 获取的 transactionIdentifier 或 originalTransactionIdentifier。
func (c *Client) GetTransactionInfo(ctx context.Context, transactionID string) (*TransactionInfoResponse, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("%w: transactionID required", ErrInvalidConfig)
	}
	path := "/inApps/v1/transactions/" + url.PathEscape(transactionID)
	body, code, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if !isHTTP2xx(code) {
		return nil, parseAPIError(body, code)
	}
	var out TransactionInfoResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTransactionHistory 获取用户 App 内购交易历史（分页）。
// transactionID：用户名下任意一条交易的 ID，用于定位用户。
// cursor：上次响应 HistoryResponse.Revision 的值；首次查询传空字符串。
//
// 注意：单次最多返回 20 条，通过 HasMore + Revision 迭代翻页。
func (c *Client) GetTransactionHistory(ctx context.Context, transactionID, cursor string) (*HistoryResponse, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("%w: transactionID required", ErrInvalidConfig)
	}
	path := "/inApps/v2/history/" + url.PathEscape(transactionID)
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
	var out HistoryResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAllTransactionHistory 获取用户全量交易历史（自动翻页，返回所有 SignedTransaction）。
// 适用于需要一次性拿到全量数据的场景；记录量大时请注意内存和超时。
func (c *Client) GetAllTransactionHistory(ctx context.Context, transactionID string) ([]string, error) {
	var all []string
	cursor := ""
	for {
		resp, err := c.GetTransactionHistory(ctx, transactionID, cursor)
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
