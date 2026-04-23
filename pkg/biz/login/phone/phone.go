package phone

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/liukunxin/go-infra/pkg/biz/login/code"
)

var (
	// ErrTooFrequent 发送过于频繁，请稍后再试。
	ErrTooFrequent = errors.New("phone: send too frequent, please wait")
	// ErrCodeNotFound 验证码不存在或已过期。
	ErrCodeNotFound = errors.New("phone: code not found or expired")
	// ErrCodeMismatch 验证码不正确。
	ErrCodeMismatch = errors.New("phone: code mismatch")
)

// SmsSender 短信发送接口，由业务侧实现具体短信服务商。
type SmsSender interface {
	Send(ctx context.Context, phone, verifyCode string) error
}

// Config 手机验证码客户端配置，所有字段均有内置默认值。
type Config struct {
	CodeLength   int           // 验证码位数，默认 6
	CodeTTL      time.Duration // 验证码有效期，默认 5 分钟
	RateLimitTTL time.Duration // 同一手机号发送间隔，默认 60 秒
	KeyPrefix    string        // Redis key 前缀，默认 "login:phone"
}

func (c *Config) normalize() {
	if c.CodeLength <= 0 {
		c.CodeLength = 6
	}
	if c.CodeTTL <= 0 {
		c.CodeTTL = 5 * time.Minute
	}
	if c.RateLimitTTL <= 0 {
		c.RateLimitTTL = 60 * time.Second
	}
	if c.KeyPrefix == "" {
		c.KeyPrefix = "login:phone"
	}
}

// Client 手机号验证码登录客户端。
type Client struct {
	sender SmsSender
	store  code.CodeStore
	cfg    Config
}

// NewClient 创建手机验证码客户端。
// sender 由业务侧注入（实现具体短信发送逻辑）；store 推荐使用 code.NewRedisStore。
func NewClient(sender SmsSender, store code.CodeStore, cfg Config) *Client {
	cfg.normalize()
	return &Client{sender: sender, store: store, cfg: cfg}
}

// SendCode 向指定手机号发送验证码。
// 同一手机号在 RateLimitTTL 内重复调用会返回 ErrTooFrequent。
func (c *Client) SendCode(ctx context.Context, phone string) error {
	rateLimitKey := fmt.Sprintf("%s:limit:%s", c.cfg.KeyPrefix, phone)
	exists, err := c.store.Exists(ctx, rateLimitKey)
	if err != nil {
		return err
	}
	if exists {
		return ErrTooFrequent
	}

	verifyCode := code.Generate(c.cfg.CodeLength)
	if err := c.sender.Send(ctx, phone, verifyCode); err != nil {
		return err
	}

	codeKey := fmt.Sprintf("%s:code:%s", c.cfg.KeyPrefix, phone)
	if err := c.store.Save(ctx, codeKey, verifyCode, c.cfg.CodeTTL); err != nil {
		return err
	}
	return c.store.Save(ctx, rateLimitKey, "1", c.cfg.RateLimitTTL)
}

// VerifyCode 校验手机号验证码。
// 校验成功后验证码立即失效（一次性）。
func (c *Client) VerifyCode(ctx context.Context, phone, inputCode string) error {
	codeKey := fmt.Sprintf("%s:code:%s", c.cfg.KeyPrefix, phone)
	err := c.store.Verify(ctx, codeKey, inputCode)
	if errors.Is(err, code.ErrNotFound) {
		return ErrCodeNotFound
	}
	if errors.Is(err, code.ErrMismatch) {
		return ErrCodeMismatch
	}
	return err
}
