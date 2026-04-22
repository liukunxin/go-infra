package alipay

import (
	"crypto/rsa"
	"encoding/json"
	"net/url"
	"sort"
	"strings"
)

// 构建待签名字符串：参数名 ASCII 升序，排除 sign、sign_type、空值；格式 key=value&...
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

func verifyJSONResponse(pub *rsa.PublicKey, body []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return err
	}
	sigRaw, ok := root["sign"]
	if !ok {
		return ErrVerifySign
	}
	var sig string
	if err := json.Unmarshal(sigRaw, &sig); err != nil {
		return err
	}
	delete(root, "sign")
	keys := make([]string, 0, len(root))
	for k := range root {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+string(root[k]))
	}
	content := strings.Join(parts, "&")
	return rsaVerifyPKCS1v15SHA256Base64(pub, content, sig)
}
