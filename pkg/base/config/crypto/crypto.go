package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var encPattern = regexp.MustCompile(`ENC\(([A-Za-z0-9+/=]+)\)`)

// Encrypt 使用 AES-256-GCM 加密明文，返回 ENC(base64) 格式。
func Encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("crypto: generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return "ENC(" + encoded + ")", nil
}

// Decrypt 解密 base64 编码的密文（不含 ENC() 包裹），返回明文。
func Decrypt(key []byte, encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 decode: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("crypto: ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt failed: %w", err)
	}
	return string(plaintext), nil
}

// DecryptYAML 扫描原始 YAML 字节中的所有 ENC(...) 标记并解密替换。
// 无 ENC(...) 标记时原样返回，零开销。
func DecryptYAML(key []byte, data []byte) ([]byte, error) {
	if !encPattern.Match(data) {
		return data, nil
	}

	var decryptErr error
	result := encPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		if decryptErr != nil {
			return match
		}
		sub := encPattern.FindSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		plaintext, err := Decrypt(key, string(sub[1]))
		if err != nil {
			decryptErr = err
			return match
		}
		return []byte(plaintext)
	})
	if decryptErr != nil {
		return nil, decryptErr
	}
	return result, nil
}

// GenerateKey 生成 32 字节随机 AES-256 密钥，返回 hex 编码。
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("crypto: generate key: %w", err)
	}
	return hex.EncodeToString(key), nil
}

// ParseKey 解析密钥字符串（支持 hex 和 base64 格式），返回 32 字节密钥。
func ParseKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) == 64 {
		key, err := hex.DecodeString(raw)
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(key) == 32 {
		return key, nil
	}
	return nil, fmt.Errorf("crypto: invalid key format, expect 32-byte hex (64 chars) or base64")
}

// ParseKeyFromEnv 从环境变量读取并解析密钥。
func ParseKeyFromEnv(envKey string) ([]byte, error) {
	raw := os.Getenv(envKey)
	if raw == "" {
		return nil, fmt.Errorf("crypto: environment variable %q is not set", envKey)
	}
	return ParseKey(raw)
}
