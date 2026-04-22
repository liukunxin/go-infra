package wechat

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// TransactionNotifyPayload 支付成功等回调解密后的订单信息（常用字段）。
type TransactionNotifyPayload struct {
	TransactionID string `json:"transaction_id"`
	OutTradeNo    string `json:"out_trade_no"`
	TradeState    string `json:"trade_state"`
	TradeType     string `json:"trade_type"`
	BankType      string `json:"bank_type"`
	SuccessTime   string `json:"success_time"`
	Amount        struct {
		Total         int64  `json:"total"`
		PayerTotal    int64  `json:"payer_total"`
		Currency      string `json:"currency"`
		PayerCurrency string `json:"payer_currency"`
	} `json:"amount"`
	Payer struct {
		OpenID string `json:"openid"`
	} `json:"payer"`
}

type notifyEnvelope struct {
	EventType string `json:"event_type"`
	Resource  struct {
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
}

// VerifyNotifyHTTP 校验回调请求签名（需配置 PlatformCertPEM）。
func (c *Client) VerifyNotifyHTTP(headers http.Header, body []byte) error {
	if c.platPub == nil {
		return fmt.Errorf("%w: PlatformCertPEM required for notify verification", ErrInvalidConfig)
	}
	ts := headers.Get("Wechatpay-Timestamp")
	nonce := headers.Get("Wechatpay-Nonce")
	sig := headers.Get("Wechatpay-Signature")
	if ts == "" || nonce == "" || sig == "" {
		return fmt.Errorf("%w: missing wechatpay signature headers", ErrVerifySign)
	}
	tsVal, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: invalid Wechatpay-Timestamp", ErrVerifySign)
	}
	// 微信建议校验时间戳，降低重放风险（允许 ±5 分钟时钟偏差）
	const maxSkew = 300
	now := time.Now().Unix()
	if tsVal > now+maxSkew || tsVal < now-maxSkew {
		return fmt.Errorf("%w: Wechatpay-Timestamp out of range", ErrVerifySign)
	}
	msg := ts + "\n" + nonce + "\n" + string(body) + "\n"
	if err := verifySHA256WithRSA(c.platPub, msg, sig); err != nil {
		return fmt.Errorf("%w: %v", ErrVerifySign, err)
	}
	return nil
}

// DecryptNotifyBody 解密 resource.ciphertext（不验签；通常先调用 VerifyNotifyHTTP）。
func (c *Client) DecryptNotifyBody(body []byte) (*TransactionNotifyPayload, error) {
	var env notifyEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Resource.Algorithm != "AEAD_AES_256_GCM" {
		return nil, fmt.Errorf("unsupported resource algorithm: %s", env.Resource.Algorithm)
	}
	plain, err := decryptAES256GCM(c.cfg.APIv3Key, env.Resource.AssociatedData, env.Resource.Nonce, env.Resource.Ciphertext)
	if err != nil {
		return nil, err
	}
	var payload TransactionNotifyPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// ParseTransactionNotify 验签 + 解密 + 解析为 TransactionNotifyPayload。
func (c *Client) ParseTransactionNotify(headers http.Header, body []byte) (*TransactionNotifyPayload, error) {
	if err := c.VerifyNotifyHTTP(headers, body); err != nil {
		return nil, err
	}
	return c.DecryptNotifyBody(body)
}

func decryptAES256GCM(apiV3Key, associatedData, nonce, ciphertextB64 string) ([]byte, error) {
	key := []byte(apiV3Key)
	if len(key) != 32 {
		return nil, fmt.Errorf("APIv3Key 须为 32 个字符（UTF-8 下即 32 字节），当前长度 %d", len(key))
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceBytes := []byte(nonce)
	plain, err := gcm.Open(nil, nonceBytes, cipherBytes, []byte(associatedData))
	if err != nil {
		return nil, fmt.Errorf("gcm open: %w", err)
	}
	return plain, nil
}

// WriteNotifyAck 向微信返回成功 ACK（HTTP 204）。
func WriteNotifyAck(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
