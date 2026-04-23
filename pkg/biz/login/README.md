# Login - 多方式通用登录

基于**标准库**封装的通用登录能力，支持账号密码、手机号验证码、邮箱验证码、微信（小程序/网页/APP）多种登录方式，以及 JWT 令牌签发与解析。无第三方业务 SDK 依赖，各子包可按需独立引用。

## 功能概览

| 子包 | 能力 |
|------|------|
| `pkg/login/jwt` | HS256 签发/解析 JWT，Subject 标识登录类型 |
| `pkg/login/password` | bcrypt 密码哈希生成与验证 |
| `pkg/login/phone` | 手机号验证码发送/校验，可插拔短信接口，内置发送频率限制 |
| `pkg/login/email` | 邮箱验证码发送/校验，可插拔邮件接口，内置发送频率限制 |
| `pkg/login/wx` | 微信小程序 `jscode2session`、网页 OAuth、APP OAuth，含用户信息获取 |
| `pkg/login/code` | CodeStore 接口 + Redis 默认实现 + 随机码生成器（phone/email 共用） |

## 依赖与导入

```go
import (
    "github.com/liukunxin/go-infra/pkg/login/jwt"
    "github.com/liukunxin/go-infra/pkg/login/password"
    "github.com/liukunxin/go-infra/pkg/login/phone"
    "github.com/liukunxin/go-infra/pkg/login/email"
    "github.com/liukunxin/go-infra/pkg/login/wx"
    "github.com/liukunxin/go-infra/pkg/login/code"
)
```

---

## JWT 令牌

### 初始化

```go
jwtCli := jwt.NewClient([]byte("your-secret-key"))
```

### 签发令牌

`subject` 参数标识登录类型，本包提供预定义常量，也可传自定义字符串：

```go
// 预定义常量：jwt.SubjectPassword / SubjectPhone / SubjectEmail
//             jwt.SubjectWechatMiniApp / SubjectWechatWeb / SubjectWechatApp
token, err := jwtCli.GenerateToken(
    "user_id_123",          // uid
    "openid_abc",           // openID（非微信场景传空字符串）
    "session_id_xyz",       // sessionID
    jwt.SubjectPhone,       // 登录类型
    7*24*time.Hour,         // 有效期
)
```

### 解析令牌

```go
claims, err := jwtCli.ParseToken(token)
if err != nil {
    // 令牌过期或签名错误
}
fmt.Println(claims.UserID, claims.Subject) // user_id_123  phone
```

---

## 账号密码登录

`password` 包仅提供哈希工具，**不涉及存储**，业务侧自行操作数据库。

### 注册时哈希密码

```go
hashed, err := password.Hash("user_plain_password")
// 将 hashed 存入数据库
```

### 登录时验证密码

```go
// 从数据库取出 hashed，与用户输入的明文比对
ok := password.Verify(inputPassword, hashedFromDB)
if !ok {
    // 密码错误
}
```

---

## 手机号验证码登录

### 实现短信发送接口

业务侧实现 `phone.SmsSender`，对接具体短信服务商（阿里云、腾讯云等）：

```go
type MySmsService struct{}

func (s *MySmsService) Send(ctx context.Context, phoneNum, verifyCode string) error {
    // 调用短信服务商 SDK 发送
    return nil
}
```

### 初始化客户端

```go
import (
    redisv8 "github.com/liukunxin/go-infra/pkg/redis/v8"
    "github.com/liukunxin/go-infra/pkg/login/code"
    "github.com/liukunxin/go-infra/pkg/login/phone"
)

store := code.NewRedisStore(redisv8.GetRedisClient())
phoneCli := phone.NewClient(&MySmsService{}, store, phone.Config{
    CodeLength:   6,              // 验证码位数，默认 6
    CodeTTL:      5 * time.Minute,  // 验证码有效期
    RateLimitTTL: 60 * time.Second, // 发送频率限制
    // KeyPrefix 默认 "login:phone"
})
```

### 发送验证码

```go
err := phoneCli.SendCode(ctx, "13800138000")
if errors.Is(err, phone.ErrTooFrequent) {
    // 60 秒内已发送过，提示用户稍后再试
}
```

### 校验验证码

```go
err := phoneCli.VerifyCode(ctx, "13800138000", inputCode)
switch {
case errors.Is(err, phone.ErrCodeNotFound):
    // 验证码不存在或已过期
case errors.Is(err, phone.ErrCodeMismatch):
    // 验证码错误
case err == nil:
    // 校验成功，生成 JWT
    token, _ := jwtCli.GenerateToken(uid, "", sessionID, jwt.SubjectPhone, 7*24*time.Hour)
}
```

---

## 邮箱验证码登录

结构与手机号登录完全对称，差异仅在接口名称。

### 实现邮件发送接口

```go
type MyEmailService struct{}

func (s *MyEmailService) Send(ctx context.Context, emailAddr, verifyCode string) error {
    // 调用邮件服务发送验证码邮件
    return nil
}
```

### 初始化客户端

```go
emailCli := email.NewClient(&MyEmailService{}, store, email.Config{
    CodeTTL: 10 * time.Minute, // 邮件送达通常比短信慢，默认 10 分钟
    // 其余字段同 phone.Config
})
```

### 发送与校验

```go
// 发送
err := emailCli.SendCode(ctx, "user@example.com")

// 校验
err = emailCli.VerifyCode(ctx, "user@example.com", inputCode)
switch {
case errors.Is(err, email.ErrCodeNotFound): // 过期
case errors.Is(err, email.ErrCodeMismatch): // 错误
case err == nil:                             // 成功
}
```

---

## 微信登录

### 配置

```go
wxCli := wx.NewClient(wx.Config{
    AppID:       "wx...",
    AppSecret:   "your_app_secret",
    HTTPTimeout: 10 * time.Second,
    // HTTPClient: http_client.GetHTTPClient().HTTPClient(), // 可注入全局连接池
})
```

### 小程序登录（jscode2session）

```go
session, err := wxCli.GetMiniProgramSession(ctx, jsCode)
if err != nil {
    // *wx.WxError 表示微信业务错误（如 code 无效）
}
// session.OpenID, session.SessionKey, session.UnionID
token, _ := jwtCli.GenerateToken(uid, session.OpenID, sessionID, jwt.SubjectWechatMiniApp, 7*24*time.Hour)
```

### 网页 OAuth 登录（公众号/H5）

```go
// 第一步：获取 access_token
tokenResp, err := wxCli.GetWebAccessToken(ctx, oauthCode)
// tokenResp.AccessToken, tokenResp.OpenID, tokenResp.UnionID

// 第二步（可选）：获取完整用户信息（需用户授权 snsapi_userinfo scope）
userInfo, err := wxCli.GetUserInfo(ctx, tokenResp.AccessToken, tokenResp.OpenID)
// userInfo.Nickname, userInfo.HeadImgURL, userInfo.UnionID
```

### APP 登录（微信开放平台移动应用）

```go
tokenResp, err := wxCli.GetAppAccessToken(ctx, authCode)
userInfo, err := wxCli.GetUserInfo(ctx, tokenResp.AccessToken, tokenResp.OpenID)
```

### 跨端 UnionID 统一身份

UnionID 由同一微信开放平台账号下的所有应用共享，可用于打通小程序、H5、APP 的用户身份：

```go
// 小程序：session.UnionID（需绑定开放平台）
// 网页/APP：tokenResp.UnionID 或 userInfo.UnionID
```

### 错误处理

微信 API 业务错误（如 code 过期、appid 错误）会返回 `*wx.WxError`：

```go
var wxErr *wx.WxError
if errors.As(err, &wxErr) {
    fmt.Printf("微信错误码: %d, 描述: %s\n", wxErr.ErrCode, wxErr.ErrMsg)
}
```

---

## 可选：注入 `pkg/http_client`

微信客户端默认使用独立 `*http.Client`。若希望与全库连接池策略一致：

```go
import "github.com/liukunxin/go-infra/pkg/http_client"

wxCli := wx.NewClient(wx.Config{
    AppID:      "wx...",
    AppSecret:  "...",
    HTTPClient: http_client.GetHTTPClient().HTTPClient(),
})
```

注入 `HTTPClient` 后，`HTTPTimeout` 字段会被忽略，以注入实例的 `Timeout`、`Transport` 为准。

---

## 安全建议

- JWT 密钥建议 32 字节以上，来自配置中心或密钥管理服务，勿写入代码库。
- 验证码校验成功后立即生成 JWT，不应复用已使用的验证码（本包已自动删除）。
- 密码哈希使用 bcrypt，禁止明文或 MD5/SHA1 存储。
- 微信 access_token 有效期通常为 2 小时，敏感操作建议检查有效期或使用 refresh_token 刷新。
- 生产环境所有回调和跳转 URL 均应使用 HTTPS。

---

## 相关代码路径

- `pkg/login/jwt/token.go` — JWT 签发与解析
- `pkg/login/password/password.go` — bcrypt 哈希工具
- `pkg/login/phone/phone.go` — 手机号验证码
- `pkg/login/email/email.go` — 邮箱验证码
- `pkg/login/wx/session.go` — 微信三端登录
- `pkg/login/code/store.go` — CodeStore 接口与 Redis 实现
- `pkg/login/code/generator.go` — 随机验证码生成
