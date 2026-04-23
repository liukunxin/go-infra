package image

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// tokenClaims is the JWT-like payload embedded in every access token.
type tokenClaims struct {
	ImageID string `json:"id"`
	// OwnerID binds the token to a specific viewer.
	// Empty means the image is publicly accessible to any token holder.
	OwnerID string `json:"uid,omitempty"`
	Exp     int64  `json:"exp"` // Unix timestamp (seconds)
}

// signer produces and verifies compact access tokens using HMAC-SHA256.
//
// Token format (URL-safe, no padding):
//
//	base64url(JSON payload) "." base64url(HMAC-SHA256 signature)
type signer struct {
	secret []byte
}

func newSigner(secret string) (*signer, error) {
	if len(secret) == 0 {
		return nil, ErrMissingSecret
	}
	return &signer{secret: []byte(secret)}, nil
}

func (s *signer) sign(imageID, ownerID string, ttl time.Duration) (string, error) {
	claims := tokenClaims{
		ImageID: imageID,
		OwnerID: ownerID,
		Exp:     time.Now().Add(ttl).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + s.mac(encoded), nil
}

func (s *signer) verify(token string) (*tokenClaims, error) {
	dot := strings.LastIndex(token, ".")
	if dot < 0 {
		return nil, ErrTokenInvalid
	}
	encoded, sig := token[:dot], token[dot+1:]

	// Constant-time comparison prevents timing attacks.
	if !hmac.Equal([]byte(sig), []byte(s.mac(encoded))) {
		return nil, ErrTokenInvalid
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	var claims tokenClaims
	if err = json.Unmarshal(raw, &claims); err != nil {
		return nil, ErrTokenInvalid
	}
	if time.Now().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}
	return &claims, nil
}

func (s *signer) mac(data string) string {
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
