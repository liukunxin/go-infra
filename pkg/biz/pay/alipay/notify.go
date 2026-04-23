package alipay

import (
	"fmt"
	"net/http"
	"net/url"
)

// ParseNotifyForm 解析支付宝异步通知表单（application/x-www-form-urlencoded）。
func ParseNotifyForm(r *http.Request) (url.Values, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	return r.Form, nil
}

// VerifyNotify 校验异步通知签名（使用配置中的支付宝公钥）。
func (c *Client) VerifyNotify(v url.Values) error {
	sig := v.Get("sign")
	if sig == "" {
		return fmt.Errorf("%w: missing sign", ErrVerifySign)
	}
	cp := cloneURLValues(v)
	cp.Del("sign")
	content := buildSignContent(cp)
	if err := rsaVerifyPKCS1v15SHA256Base64(c.aliPub, content, sig); err != nil {
		return fmt.Errorf("%w: %v", ErrVerifySign, err)
	}
	return nil
}

// WriteNotifyAck 返回 success 字符串（支付宝要求纯文本 success）。
func WriteNotifyAck(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")
	_, _ = w.Write([]byte("success"))
}

func cloneURLValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}
	return out
}
