package jdpay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"sort"
	"strings"
)

// parseRSAPrivateKey 解析 PKCS#8 或 PKCS#1 PEM 格式的 RSA 私钥。
func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("jdpay: private key PEM decode failed")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if pk, ok := key.(*rsa.PrivateKey); ok {
			return pk, nil
		}
		return nil, fmt.Errorf("jdpay: private key is not RSA")
	}
	pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("jdpay: parse private key: %w", err)
	}
	return pk, nil
}

// parseRSAPublicKey 解析 PKIX PEM 格式的 RSA 公钥（京东支付平台公钥）。
func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("jdpay: public key PEM decode failed")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("jdpay: parse public key: %w", err)
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("jdpay: public key is not RSA")
	}
	return pub, nil
}

// buildSignContent 构建待签名字符串。
// 规则：参数名 ASCII 升序排列，排除 sign 及空值，格式 key=value&key=value...
func buildSignContent(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	return strings.Join(parts, "&")
}

// rsaSignSHA256Base64 使用 RSA-SHA256 对内容签名，返回 base64 标准编码字符串。
func rsaSignSHA256Base64(priv *rsa.PrivateKey, content string) (string, error) {
	h := sha256.Sum256([]byte(content))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// rsaVerifySHA256Base64 使用 RSA-SHA256 验证 base64 编码的签名。
func rsaVerifySHA256Base64(pub *rsa.PublicKey, content, sigB64 string) error {
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return err
	}
	h := sha256.Sum256([]byte(content))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig)
}
