package applepay

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"math/big"
	"strings"
)

// jwsHeader JWS Compact Serialization header（关注字段）。
type jwsHeader struct {
	Alg string   `json:"alg"`
	X5c []string `json:"x5c"` // base64 标准编码（非 URL 编码）的 DER 证书链；index 0 为叶证书。
}

// JWSTransactionDecodedPayload Apple 签名的交易 payload（JWSTransaction 解码后）。
// 字段参考：https://developer.apple.com/documentation/appstoreserverapi/jwstransactiondecodedpayload
type JWSTransactionDecodedPayload struct {
	AppAccountToken             string `json:"appAccountToken"`
	BundleID                    string `json:"bundleId"`
	Currency                    string `json:"currency"`
	Environment                 string `json:"environment"`
	ExpiresDate                 int64  `json:"expiresDate"`
	InAppOwnershipType          string `json:"inAppOwnershipType"`
	IsUpgraded                  bool   `json:"isUpgraded"`
	OfferDiscountType           string `json:"offerDiscountType"`
	OfferIdentifier             string `json:"offerIdentifier"`
	OfferType                   int    `json:"offerType"`
	OriginalPurchaseDate        int64  `json:"originalPurchaseDate"`
	OriginalTransactionID       string `json:"originalTransactionId"`
	Price                       int64  `json:"price"`
	ProductID                   string `json:"productId"`
	PurchaseDate                int64  `json:"purchaseDate"`
	Quantity                    int    `json:"quantity"`
	RevocationDate              int64  `json:"revocationDate"`
	RevocationReason            int    `json:"revocationReason"`
	SignedDate                  int64  `json:"signedDate"`
	Storefront                  string `json:"storefront"`
	StorefrontID                string `json:"storefrontId"`
	SubscriptionGroupIdentifier string `json:"subscriptionGroupIdentifier"`
	TransactionID               string `json:"transactionId"`
	TransactionReason           string `json:"transactionReason"`
	Type                        string `json:"type"`
	WebOrderLineItemID          string `json:"webOrderLineItemId"`
}

// JWSRenewalInfoDecodedPayload Apple 签名的订阅续订信息（JWSRenewalInfo 解码后）。
// 字段参考：https://developer.apple.com/documentation/appstoreserverapi/jwsrenewalinfodecodedpayload
type JWSRenewalInfoDecodedPayload struct {
	AutoRenewProductID          string `json:"autoRenewProductId"`
	AutoRenewStatus             int    `json:"autoRenewStatus"`
	Environment                 string `json:"environment"`
	ExpirationIntent            int    `json:"expirationIntent"`
	GracePeriodExpiresDate      int64  `json:"gracePeriodExpiresDate"`
	IsInBillingRetryPeriod      bool   `json:"isInBillingRetryPeriod"`
	OfferIdentifier             string `json:"offerIdentifier"`
	OfferType                   int    `json:"offerType"`
	OriginalTransactionID       string `json:"originalTransactionId"`
	PriceIncreaseStatus         int    `json:"priceIncreaseStatus"`
	ProductID                   string `json:"productId"`
	RecentSubscriptionStartDate int64  `json:"recentSubscriptionStartDate"`
	RenewalDate                 int64  `json:"renewalDate"`
	SignedDate                  int64  `json:"signedDate"`
}

// DecodeJWSTransaction 解码 Apple 签名的 JWS 交易字符串，仅解码不验签。
// 适用于已通过 App Store Server Notifications 接收到并信任的场景，
// 或用于调试。生产环境建议使用 DecodeJWSTransactionVerified。
func DecodeJWSTransaction(jwsString string) (*JWSTransactionDecodedPayload, error) {
	var out JWSTransactionDecodedPayload
	if err := decodeJWSPayload(jwsString, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DecodeJWSTransactionVerified 解码并验证 Apple 签名的 JWS 交易字符串。
// appleRootCertPEM: Apple Root CA PEM，从 https://www.apple.com/certificateauthority/ 下载
// AppleRootCA-G3.cer 后使用 openssl x509 -inform DER -in AppleRootCA-G3.cer 转换。
func DecodeJWSTransactionVerified(jwsString, appleRootCertPEM string) (*JWSTransactionDecodedPayload, error) {
	var out JWSTransactionDecodedPayload
	if err := verifyAndDecodeJWS(jwsString, &out, []string{appleRootCertPEM}); err != nil {
		return nil, err
	}
	return &out, nil
}

// DecodeJWSRenewalInfo 解码 Apple 签名的 JWS 续订信息，仅解码不验签。
func DecodeJWSRenewalInfo(jwsString string) (*JWSRenewalInfoDecodedPayload, error) {
	var out JWSRenewalInfoDecodedPayload
	if err := decodeJWSPayload(jwsString, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DecodeJWSRenewalInfoVerified 解码并验证 Apple 签名的 JWS 续订信息。
func DecodeJWSRenewalInfoVerified(jwsString, appleRootCertPEM string) (*JWSRenewalInfoDecodedPayload, error) {
	var out JWSRenewalInfoDecodedPayload
	if err := verifyAndDecodeJWS(jwsString, &out, []string{appleRootCertPEM}); err != nil {
		return nil, err
	}
	return &out, nil
}

// decodeJWSPayload 仅对 JWS payload 做 base64url 解码，不验签。
func decodeJWSPayload(jwsString string, out any) error {
	parts := strings.Split(jwsString, ".")
	if len(parts) != 3 {
		return fmt.Errorf("applepay: invalid JWS format, expected 3 parts got %d", len(parts))
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("applepay: decode JWS payload: %w", err)
	}
	return json.Unmarshal(payloadJSON, out)
}

// verifyAndDecodeJWS 验证 JWS 签名（x5c 证书链 + ECDSA）并解码 payload。
// rootCertPEMs 为空时跳过根证书链验证（不推荐生产使用）。
func verifyAndDecodeJWS(jwsString string, out any, rootCertPEMs []string) error {
	parts := strings.Split(jwsString, ".")
	if len(parts) != 3 {
		return fmt.Errorf("%w: invalid JWS format, expected 3 parts got %d", ErrVerifySign, len(parts))
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("%w: decode header: %v", ErrVerifySign, err)
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("%w: decode payload: %v", ErrVerifySign, err)
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("%w: decode signature: %v", ErrVerifySign, err)
	}

	var header jwsHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return fmt.Errorf("%w: parse JWS header: %v", ErrVerifySign, err)
	}
	if len(header.X5c) == 0 {
		return fmt.Errorf("%w: x5c is empty in JWS header", ErrVerifySign)
	}

	// x5c 中每个元素是 base64 标准编码（非 URL 编码）的 DER 格式证书。
	certs := make([]*x509.Certificate, 0, len(header.X5c))
	for i, certB64 := range header.X5c {
		derBytes, err := base64.StdEncoding.DecodeString(certB64)
		if err != nil {
			return fmt.Errorf("%w: decode x5c[%d]: %v", ErrVerifySign, i, err)
		}
		cert, err := x509.ParseCertificate(derBytes)
		if err != nil {
			return fmt.Errorf("%w: parse x5c[%d]: %v", ErrVerifySign, i, err)
		}
		certs = append(certs, cert)
	}

	// 当提供根证书时，验证完整的证书链（叶 → 中间 → 根）。
	if len(rootCertPEMs) > 0 {
		rootPool := x509.NewCertPool()
		for _, pemStr := range rootCertPEMs {
			if !rootPool.AppendCertsFromPEM([]byte(pemStr)) {
				return fmt.Errorf("%w: failed to add Apple root cert to pool", ErrVerifySign)
			}
		}
		interPool := x509.NewCertPool()
		for _, cert := range certs[1:] {
			interPool.AddCert(cert)
		}
		_, err := certs[0].Verify(x509.VerifyOptions{
			Roots:         rootPool,
			Intermediates: interPool,
		})
		if err != nil {
			return fmt.Errorf("%w: certificate chain verify failed: %v", ErrVerifySign, err)
		}
	}

	// 用叶证书的 EC 公钥验证 JWS 签名。
	// JWS 签名内容：base64url(header) + "." + base64url(payload)
	// JWS ECDSA 签名编码（RFC 7518 §3.4）：r || s 原始字节拼接，各填充至曲线阶字节长度。
	leafPub, ok := certs[0].PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: leaf cert public key is not ECDSA", ErrVerifySign)
	}
	if err := verifyECDSAJWSSignature(leafPub, header.Alg, parts[0]+"."+parts[1], sigBytes); err != nil {
		return fmt.Errorf("%w: %v", ErrVerifySign, err)
	}

	return json.Unmarshal(payloadJSON, out)
}

// verifyECDSAJWSSignature 验证 JWS ECDSA 签名。
// alg 决定哈希算法：ES256→SHA-256，ES384→SHA-384，ES512→SHA-512。
// sigBytes 是 r || s 原始字节拼接格式（RFC 7518 §3.4）。
func verifyECDSAJWSSignature(pub *ecdsa.PublicKey, alg, signingInput string, sigBytes []byte) error {
	keySize := (pub.Curve.Params().BitSize + 7) / 8
	if len(sigBytes) != 2*keySize {
		return fmt.Errorf("signature length %d, expected %d for curve %s", len(sigBytes), 2*keySize, pub.Curve.Params().Name)
	}

	var h hash.Hash
	switch alg {
	case "ES256":
		h = sha256.New()
	case "ES384":
		h = sha512.New384()
	case "ES512":
		h = sha512.New()
	default:
		return fmt.Errorf("unsupported JWS alg %q (expected ES256/ES384/ES512)", alg)
	}
	_, _ = h.Write([]byte(signingInput))
	digest := h.Sum(nil)

	r := new(big.Int).SetBytes(sigBytes[:keySize])
	s := new(big.Int).SetBytes(sigBytes[keySize:])
	if !ecdsa.Verify(pub, digest, r, s) {
		return fmt.Errorf("ECDSA signature verification failed")
	}
	return nil
}
