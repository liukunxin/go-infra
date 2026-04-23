package wechat

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("wechat pay: private key PEM decode failed")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if pk, ok := key.(*rsa.PrivateKey); ok {
			return pk, nil
		}
		return nil, fmt.Errorf("wechat pay: private key is not RSA")
	}
	pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("wechat pay: parse private key: %w", err)
	}
	return pk, nil
}

func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("wechat pay: public key PEM decode failed")
	}
	if pubAny, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		pub, ok := pubAny.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("wechat pay: public key is not RSA")
		}
		return pub, nil
	}
	// 支持微信支付平台证书 PEM（BEGIN CERTIFICATE）
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("wechat pay: parse public key or certificate: %w", err)
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("wechat pay: certificate public key is not RSA")
	}
	return pub, nil
}

func signSHA256WithRSA(priv *rsa.PrivateKey, message string) (string, error) {
	h := sha256.Sum256([]byte(message))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func verifySHA256WithRSA(pub *rsa.PublicKey, message string, sigB64 string) error {
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return err
	}
	h := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig)
}
