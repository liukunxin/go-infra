package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 常用登录类型常量，可直接传入 GenerateToken 的 subject 参数。
const (
	SubjectPassword        = "password"
	SubjectPhone           = "phone"
	SubjectEmail           = "email"
	SubjectWechatMiniApp   = "wechat_miniprogram"
	SubjectWechatWeb       = "wechat_web"
	SubjectWechatApp       = "wechat_app"
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
	return &Client{jwtSecret: jwtSecret}
}

// GenerateToken 签发 JWT。
// subject 标识登录类型，建议使用本包预定义的 Subject* 常量，例如 SubjectPhone、SubjectWechatMiniApp。
func (c *Client) GenerateToken(uid, openID, sessionID, subject string, expireTime time.Duration) (string, error) {
	claims := Claims{
		UserID:    uid,
		OpenID:    openID,
		SessionID: sessionID,
		LoginTime: time.Now().Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   subject,
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
