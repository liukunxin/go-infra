package alipay

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// buildSignContent 构建待签名字符串。
// 规则：参数名 ASCII 升序，排除 sign、sign_type 及空值，格式 key=value&...
// 依据：支付宝开放平台传统网关（v1.0）签名规范。
func buildSignContent(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if v.Get(k) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+v.Get(k))
	}
	return strings.Join(parts, "&")
}

// verifyAndParseJSONResponse 校验支付宝网关同步响应签名，并返回解析后的 root map。
//
// 支付宝响应验签规范：签名内容是响应报文中 *_response 键对应的原始 JSON 值
// （json.RawMessage 保留了原始字节序，不经过再序列化，与官方签名内容一致）。
//
// 返回的 map 供调用方直接使用，避免上层再次 json.Unmarshal 造成二次解析。
func verifyAndParseJSONResponse(pub *rsa.PublicKey, body []byte) (map[string]json.RawMessage, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	sigRaw, ok := root["sign"]
	if !ok {
		return nil, ErrVerifySign
	}
	var sig string
	if err := json.Unmarshal(sigRaw, &sig); err != nil {
		return nil, err
	}
	// 找到 *_response 键（响应体中除 sign、sign_type 外的那个键）。
	var content string
	for k, v := range root {
		if k == "sign" || k == "sign_type" {
			continue
		}
		content = string(v)
		break
	}
	if content == "" {
		return nil, fmt.Errorf("%w: response key (*_response) not found in body", ErrVerifySign)
	}
	if err := rsaVerifyPKCS1v15SHA256Base64(pub, content, sig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVerifySign, err)
	}
	return root, nil
}
