# Account - 通用账号管理

提供账号实体模型、登录方式绑定关系与常用账号管理操作，不依赖任何 ORM 或数据库，所有存储操作通过接口注入。

## 与 Login 模块的边界

```
用户输入凭证
      │
      ▼
┌─────────────────────────────────────────┐
│  pkg/login  （认证层）                  │
│  • 验证凭证是否合法                      │
│    ├─ password.Verify(明文, 哈希)        │
│    ├─ phone.Client.VerifyCode(手机, 码)  │
│    └─ wx.Client.GetMiniProgramSession   │
│  • 不关心"这是哪个账号"                  │
└──────────────────┬──────────────────────┘
                   │ 凭证合法
                   ▼
┌─────────────────────────────────────────┐
│  pkg/account （账号层）                 │
│  • FindOrCreate: 凭证 → 账号 ID         │
│  • Bind / Unbind: 管理多登录方式         │
│  • Freeze / Delete: 账号状态管理         │
└──────────────────┬──────────────────────┘
                   │ 得到 AccountID
                   ▼
            pkg/login/jwt 签发 Token
```

| | pkg/login | pkg/account |
|---|---|---|
| 核心问题 | 凭证合法吗？ | 这是哪个账号？怎么管理账号？ |
| 依赖存储 | 可选（验证码用 Redis） | 必须（账号与绑定关系需持久化） |
| 业务耦合 | 无（纯工具包） | 接口化（Store 由业务实现） |

---

## 功能概览

| 文件 | 能力 |
|------|------|
| `account.go` | `Account` 实体、`Status` 枚举、`LoginType` 常量、`AccountStore` 接口、`IDGenerator` 接口 |
| `binding.go` | `LoginBinding` 实体、`BindingStore` 接口 |
| `service.go` | `Service`：`FindOrCreate`、`GetByID`、`Bind`、`Unbind`、`ListBindings`、`Freeze`、`Unfreeze`、`Delete` |

---

## 快速上手

### 第一步：实现两个存储接口

业务侧用自己的 ORM 实现 `AccountStore` 和 `BindingStore`，以 GORM 为例：

```go
// accounts_store.go
type GormAccountStore struct{ db *gorm.DB }

func (s *GormAccountStore) Create(ctx context.Context, a *account.Account) error {
    return s.db.WithContext(ctx).Create(&AccountDO{
        ID: a.ID, Status: int(a.Status),
        CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
    }).Error
}

func (s *GormAccountStore) GetByID(ctx context.Context, id string) (*account.Account, error) {
    var row AccountDO
    if err := s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, account.ErrAccountNotFound
        }
        return nil, err
    }
    return &account.Account{ID: row.ID, Status: account.Status(row.Status),
        CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (s *GormAccountStore) UpdateStatus(ctx context.Context, id string, status account.Status) error {
    return s.db.WithContext(ctx).Model(&AccountDO{}).
        Where("id = ?", id).Update("status", status).Error
}
```

```go
// bindings_store.go
type GormBindingStore struct{ db *gorm.DB }

func (s *GormBindingStore) Bind(ctx context.Context, b *account.LoginBinding) error {
    err := s.db.WithContext(ctx).Create(&BindingDO{
        AccountID: b.AccountID, LoginType: b.LoginType,
        Credential: b.Credential, CreatedAt: b.CreatedAt,
    }).Error
    if isDuplicateKeyError(err) { // 依赖 DB 唯一索引
        return account.ErrBindingConflict
    }
    return err
}

func (s *GormBindingStore) Unbind(ctx context.Context, accountID, loginType string) error {
    return s.db.WithContext(ctx).
        Where("account_id = ? AND login_type = ?", accountID, loginType).
        Delete(&BindingDO{}).Error
}

func (s *GormBindingStore) FindByCredential(ctx context.Context, loginType, credential string) (*account.LoginBinding, error) {
    var row BindingDO
    if err := s.db.WithContext(ctx).
        Where("login_type = ? AND credential = ?", loginType, credential).
        First(&row).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, account.ErrBindingNotFound
        }
        return nil, err
    }
    return &account.LoginBinding{AccountID: row.AccountID, LoginType: row.LoginType,
        Credential: row.Credential, CreatedAt: row.CreatedAt}, nil
}

func (s *GormBindingStore) ListByAccountID(ctx context.Context, accountID string) ([]*account.LoginBinding, error) {
    var rows []BindingDO
    if err := s.db.WithContext(ctx).Where("account_id = ?", accountID).Find(&rows).Error; err != nil {
        return nil, err
    }
    out := make([]*account.LoginBinding, len(rows))
    for i, r := range rows {
        out[i] = &account.LoginBinding{AccountID: r.AccountID, LoginType: r.LoginType,
            Credential: r.Credential, CreatedAt: r.CreatedAt}
    }
    return out, nil
}
```

**推荐建表索引：**

```sql
-- accounts 表
CREATE UNIQUE INDEX idx_accounts_id ON accounts(id);

-- login_bindings 表（凭证全局唯一）
CREATE UNIQUE INDEX idx_bindings_credential ON login_bindings(login_type, credential);
CREATE INDEX idx_bindings_account_id ON login_bindings(account_id);
```

### 第二步：提供 ID 生成器

可使用项目已有的 `pkg/uuid`（Snowflake），也可自定义：

```go
import pkguuid "github.com/liukunxin/go-infra/pkg/uuid"

// 方式一：使用 pkg/uuid Snowflake（需先调用 GetIDService）
idGen := account.IDGeneratorFunc(func() string {
    return fmt.Sprintf("%d", pkguuid.GetIDService().GenerateUserID())
})

// 方式二：UUID v4
import "github.com/gofrs/uuid"
idGen := account.IDGeneratorFunc(func() string {
    return uuid.Must(uuid.NewV4()).String()
})
```

### 第三步：初始化 Service

```go
svc := account.NewService(
    &GormAccountStore{db: db},
    &GormBindingStore{db: db},
    idGen,
)
```

---

## 登录全流程示例

### 手机号验证码登录

```go
// 1. 验证手机验证码（pkg/login/phone）
if err := phoneCli.VerifyCode(ctx, "13800138000", inputCode); err != nil {
    return err
}

// 2. 查找或创建账号（pkg/account）
acc, created, err := svc.FindOrCreate(ctx, account.LoginTypePhone, "13800138000")
if err != nil {
    return err
}
_ = created // true 表示本次新注册

// 3. 签发 JWT（pkg/login/jwt）
token, err := jwtCli.GenerateToken(acc.ID, "", sessionID, jwt.SubjectPhone, 7*24*time.Hour)
```

### 微信小程序登录

```go
// 1. 换取 openid（pkg/login/wx）
session, err := wxCli.GetMiniProgramSession(ctx, jsCode)
if err != nil {
    return err
}

// 2. 查找或创建账号
acc, _, err := svc.FindOrCreate(ctx, account.LoginTypeWechatMiniApp, session.OpenID)
if err != nil {
    return err
}

// 3. 签发 JWT
token, err := jwtCli.GenerateToken(acc.ID, session.OpenID, sessionID, jwt.SubjectWechatMiniApp, 7*24*time.Hour)
```

### 账号密码登录

```go
// 1. 从 DB 查出哈希密码（业务侧自行查询）
hashedPwd, err := myRepo.GetPasswordByAccountID(ctx, accountID)

// 2. 验证密码（pkg/login/password）
if !password.Verify(inputPassword, hashedPwd) {
    return errors.New("密码错误")
}

// 3. 查账号状态（pkg/account）
acc, err := svc.GetByID(ctx, accountID)
if err != nil {
    return err // ErrAccountFrozen / ErrAccountDeleted
}

// 4. 签发 JWT
token, err := jwtCli.GenerateToken(acc.ID, "", sessionID, jwt.SubjectPassword, 7*24*time.Hour)
```

---

## 多登录方式绑定

同一账号可绑定多种登录方式，例如先用手机注册，再绑定微信：

```go
// 绑定微信小程序
err := svc.Bind(ctx, acc.ID, account.LoginTypeWechatMiniApp, session.OpenID)
if errors.Is(err, account.ErrBindingConflict) {
    // openid 已绑定其他账号
}
if errors.Is(err, account.ErrBindingExists) {
    // 当前账号已绑定微信小程序
}

// 解绑手机号
err = svc.Unbind(ctx, acc.ID, account.LoginTypePhone)

// 列出已绑定的登录方式
bindings, err := svc.ListBindings(ctx, acc.ID)
for _, b := range bindings {
    fmt.Println(b.LoginType, b.Credential)
}
```

---

## 账号状态管理

```go
// 冻结（禁止登录，保留数据）
err := svc.Freeze(ctx, accountID)

// 解冻
err = svc.Unfreeze(ctx, accountID)

// 软删除（注销）
err = svc.Delete(ctx, accountID)

// 调用 GetByID / FindOrCreate 时，冻结或注销账号会自动返回错误
acc, err := svc.GetByID(ctx, accountID)
switch {
case errors.Is(err, account.ErrAccountFrozen):  // 已冻结
case errors.Is(err, account.ErrAccountDeleted): // 已注销
}
```

---

## 并发注册说明

`FindOrCreate` 不是原子操作。在高并发场景（如同一手机号同时触发两次注册请求），
可能出现两个 goroutine 都判断"绑定不存在"并分别创建账号。

**推荐应对方案**（任选其一）：

1. **数据库唯一索引（推荐）**：在 `(login_type, credential)` 上建唯一索引，`Bind` 冲突时返回 `ErrBindingConflict`，业务侧捕获后重新查询已有绑定。
2. **分布式锁**：在调用 `FindOrCreate` 前以 `loginType:credential` 为 key 加 Redis 锁。

---

## 错误一览

| 错误 | 含义 |
|------|------|
| `ErrAccountNotFound` | 账号 ID 不存在 |
| `ErrAccountFrozen` | 账号已冻结 |
| `ErrAccountDeleted` | 账号已注销 |
| `ErrBindingNotFound` | 登录绑定不存在（未注册或无此登录方式） |
| `ErrBindingConflict` | 凭证已绑定其他账号 |
| `ErrBindingExists` | 当前账号已绑定此类登录方式 |

---

## 相关代码路径

- `pkg/account/account.go` — Account 实体、Status、LoginType、AccountStore 接口
- `pkg/account/binding.go` — LoginBinding 实体、BindingStore 接口
- `pkg/account/service.go` — Service 业务编排
- `pkg/login/` — 认证层（凭证验证 + JWT 签发）
