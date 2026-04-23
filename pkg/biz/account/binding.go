package account

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrBindingNotFound 登录绑定关系不存在。
	ErrBindingNotFound = errors.New("account: binding not found")
	// ErrBindingConflict 该凭证已绑定其他账号。
	ErrBindingConflict = errors.New("account: binding credential already bound to another account")
	// ErrBindingExists 该账号已绑定同类型登录方式。
	ErrBindingExists = errors.New("account: login type already bound to this account")
)

// LoginBinding 登录绑定关系，将一种登录凭证映射到一个账号 ID。
//
// 每条记录对应一种登录方式与账号的绑定，例如：
//
//	LoginType="phone"   Credential="13800138000" → AccountID="acc_001"
//	LoginType="wechat_miniprogram" Credential="oXxx_openid" → AccountID="acc_001"
//
// 同一账号可绑定多种登录方式；同一凭证只能绑定一个账号（由存储层保证唯一）。
type LoginBinding struct {
	AccountID  string
	LoginType  string // 建议使用本包 LoginType* 常量
	Credential string // 手机号 / 邮箱 / openid 等
	CreatedAt  time.Time
}

// BindingStore 登录绑定关系持久化接口，由业务侧实现。
//
// 数据库建议在 (login_type, credential) 上建立唯一索引，以防并发注册时绑定冲突。
type BindingStore interface {
	// Bind 创建绑定关系。(login_type, credential) 已存在时应返回 ErrBindingConflict。
	Bind(ctx context.Context, binding *LoginBinding) error
	// Unbind 删除指定账号的某类登录绑定。
	Unbind(ctx context.Context, accountID, loginType string) error
	// FindByCredential 按登录方式与凭证查找绑定关系。
	// 不存在时返回 ErrBindingNotFound。
	FindByCredential(ctx context.Context, loginType, credential string) (*LoginBinding, error)
	// ListByAccountID 列出账号的所有登录绑定（用于展示已绑定的登录方式）。
	ListByAccountID(ctx context.Context, accountID string) ([]*LoginBinding, error)
}
