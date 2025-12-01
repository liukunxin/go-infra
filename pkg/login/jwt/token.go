package jwt

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Claims struct {
	UserID    string `json:"user_id"`
	OpenID    string `json:"open_id"`
	LoginTime int64  `json:"login_time"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

type Client struct {
	jwtSecret []byte
}

func NewClient(jwtSecret []byte) *Client {
	return &Client{
		jwtSecret: jwtSecret,
	}
}

func (c *Client) GenerateToken(uid string, openID string, sessionID string, expireTime time.Duration) (string, error) {
	claims := Claims{
		UserID:    uid,
		OpenID:    openID,
		SessionID: sessionID,
		LoginTime: time.Now().Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "wechat_app_user",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(c.jwtSecret)
}

func (c *Client) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return c.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
