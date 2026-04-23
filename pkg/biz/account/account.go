// Package account 提供通用账号管理能力，包括账号实体、登录绑定关系、状态管理与常用服务操作。
//
// 与 pkg/login 的边界：
//
//   - pkg/login  ——认证层：验证凭证是否合法（密码比对、验证码校验、微信 code 换 openid）、签发 JWT。
//   - pkg/account——账号层：管理账号数据，维护"登录凭证 → 账号 ID"的绑定关系，支持多登录方式绑定同一账号。
//
// 本包只定义接口与业务逻辑，不绑定任何 ORM 或数据库。业务侧按需实现 AccountStore 与 BindingStore 接口。
package account

import (
	"context"
	"errors"
	"time"
)

// Status 账号状态。
type Status int

const (
	StatusActive  Status = 1 // 正常
	StatusFrozen  Status = 2 // 冻结（禁止登录，数据保留）
	StatusDeleted Status = 3 // 已注销（软删除）
)

// LoginType 登录方式常量，与 pkg/login/jwt 的 Subject* 常量对齐，可直接用于 LoginBinding.LoginType。
const (
	LoginTypePassword      = "password"
	LoginTypePhone         = "phone"
	LoginTypeEmail         = "email"
	LoginTypeWechatMiniApp = "wechat_miniprogram"
	LoginTypeWechatWeb     = "wechat_web"
	LoginTypeWechatApp     = "wechat_app"
)

var (
	// ErrAccountNotFound 账号不存在。
	ErrAccountNotFound = errors.New("account: not found")
	// ErrAccountFrozen 账号已被冻结，禁止登录。
	ErrAccountFrozen = errors.New("account: frozen")
	// ErrAccountDeleted 账号已注销。
	ErrAccountDeleted = errors.New("account: deleted")
)

// Account 账号基础实体。
//
// 本包刻意不包含昵称、头像等业务字段，避免侵入具体业务 schema。
// 业务侧可在自己的 User 表中以 AccountID 关联扩展字段。
type Account struct {
	ID        string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AccountStore 账号持久化接口，由业务侧实现（GORM、sqlx 等均可）。
type AccountStore interface {
	// Create 创建账号。业务侧应保证 ID 唯一。
	Create(ctx context.Context, account *Account) error
	// GetByID 按 ID 查询账号。不存在时返回 ErrAccountNotFound。
	GetByID(ctx context.Context, id string) (*Account, error)
	// UpdateStatus 更新账号状态。
	UpdateStatus(ctx context.Context, id string, status Status) error
}

// IDGenerator 账号 ID 生成接口。
type IDGenerator interface {
	Generate() string
}

// IDGeneratorFunc 函数类型实现 IDGenerator，便于内联定义。
//
//	idGen := account.IDGeneratorFunc(func() string { return myUUID() })
type IDGeneratorFunc func() string

func (f IDGeneratorFunc) Generate() string { return f() }
