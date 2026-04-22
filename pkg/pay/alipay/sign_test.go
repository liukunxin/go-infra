package alipay

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
)

func TestBuildSignContent(t *testing.T) {
	v := url.Values{}
	v.Set("b", "2")
	v.Set("a", "1")
	v.Set("sign", "should_skip")
	v.Set("sign_type", "RSA2")
	v.Set("empty", "")
	got := buildSignContent(v)
	want := "a=1&b=2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// TestVerifyAndParseJSONResponse_ReturnsRootMap 验证验签通过时，返回的 map 包含正确 key。
func TestVerifyAndParseJSONResponse_ReturnsRootMap(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub := &priv.PublicKey

	responseValue := `{"code":"10000","msg":"Success","trade_no":"2024123456"}`
	sigB64, err := rsaSignPKCS1v15SHA256Base64(priv, responseValue)
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{
		"alipay_trade_query_response": json.RawMessage(responseValue),
		"sign":                        sigB64,
	})

	root, err := verifyAndParseJSONResponse(pub, body)
	if err != nil {
		t.Fatalf("正确签名应验签通过，got error: %v", err)
	}
	if _, ok := root["alipay_trade_query_response"]; !ok {
		t.Fatal("返回的 root map 应包含 alipay_trade_query_response 键")
	}
}

// TestVerifyAndParseJSONResponse_WrongContent_Fails 验证错误签名内容时验签必定失败。
func TestVerifyAndParseJSONResponse_WrongContent_Fails(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey

	responseValue := `{"code":"10000","msg":"Success"}`
	wrongContent := "alipay_trade_query_response=" + responseValue
	sigB64, _ := rsaSignPKCS1v15SHA256Base64(priv, wrongContent)

	body, _ := json.Marshal(map[string]any{
		"alipay_trade_query_response": json.RawMessage(responseValue),
		"sign":                        sigB64,
	})

	_, err := verifyAndParseJSONResponse(pub, body)
	if !errors.Is(err, ErrVerifySign) {
		t.Fatalf("期望 ErrVerifySign，got: %v", err)
	}
}

// TestVerifyAndParseJSONResponse_MissingSign 验证缺 sign 字段返回 ErrVerifySign。
func TestVerifyAndParseJSONResponse_MissingSign(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey

	body, _ := json.Marshal(map[string]any{
		"alipay_trade_query_response": json.RawMessage(`{"code":"10000"}`),
	})
	_, err := verifyAndParseJSONResponse(pub, body)
	if !errors.Is(err, ErrVerifySign) {
		t.Fatalf("期望 ErrVerifySign，got: %v", err)
	}
}
