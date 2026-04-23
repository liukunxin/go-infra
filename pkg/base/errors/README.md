# Errors - 错误处理模块

统一的错误处理和HTTP状态码封装，提供标准化的错误响应格式。

## 📖 功能特性

- ✅ 统一的错误包装和解析
- ✅ HTTP状态码映射
- ✅ 业务错误码定义
- ✅ 支持错误链（error wrapping）
- ✅ 自动转换为标准响应格式

## 🚀 快速开始

### 基础用法

```go
package main

import (
    "fmt"
    "github.com/liukunxin/go-infra/pkg/errors"
)

func main() {
    // 1. 包装错误
    err := doSomething()
    if err != nil {
        wrappedErr := errors.WarpError(
            errors.StatusBadRequest,  // HTTP状态码
            40001,                     // 业务错误码
            err,                       // 原始错误
        )
        fmt.Println(wrappedErr)
    }

    // 2. 解析错误码
    code := errors.AsCode(wrappedErr)
    fmt.Printf("错误码: %d\n", code)

    // 3. 构造错误响应
    resp := errors.UnWarpErrorResponse(wrappedErr)
    fmt.Printf("响应: %+v\n", resp)
}

func doSomething() error {
    return fmt.Errorf("数据库连接失败")
}
```

## 📋 预定义状态码

### HTTP状态码

```go
const (
    StatusOK                  = 200  // 成功
    StatusBadRequest          = 400  // 请求错误
    StatusUnauthorized        = 401  // 未认证
    StatusForbidden           = 403  // 无权限
    StatusNotFound            = 404  // 资源不存在
    StatusTooManyRequests     = 429  // 请求过多
    StatusInternalServerError = 500  // 内部错误
)
```

### 业务错误码

```go
const (
    CodeInternalCallFailed  = 50005  // 内部调用失败
    CodeUserContextCanceled = 50006  // 用户取消请求
    CodeUnknown            = 0       // 未知错误
)
```

## 💡 使用示例

### 示例1：基础错误处理

```go
package service

import (
    "database/sql"
    kerr "github.com/liukunxin/go-infra/pkg/errors"
)

func GetUser(id int64) (*User, error) {
    user, err := db.FindUser(id)
    if err != nil {
        if err == sql.ErrNoRows {
            // 资源不存在
            return nil, kerr.WarpError(
                kerr.StatusNotFound,
                40401,
                fmt.Errorf("用户不存在: %d", id),
            )
        }
        // 内部错误
        return nil, kerr.WarpError(
            kerr.StatusInternalServerError,
            kerr.CodeInternalCallFailed,
            err,
        )
    }
    return user, nil
}
```

### 示例2：在Controller中使用

```go
package controller

import (
    "github.com/gin-gonic/gin"
    "github.com/liukunxin/go-infra/pkg/controller"
    kerr "github.com/liukunxin/go-infra/pkg/errors"
)

type UserController struct {
    controller.GinBase
}

func (uc *UserController) GetUser(c *gin.Context) {
    // 业务逻辑
    user, err := service.GetUser(123)
    if err != nil {
        // 自动处理错误响应（包含TraceID）
        uc.ErrorResponse(c, err)
        return
    }

    // 成功响应
    uc.SuccessResponse(c, user)
}
```

### 示例3：自定义业务错误码

```go
package myerrors

import (
    kerr "github.com/liukunxin/go-infra/pkg/errors"
)

// 定义业务错误码
const (
    CodeUserNotFound      kerr.Code = 40401
    CodePasswordWrong     kerr.Code = 40101
    CodeTokenExpired      kerr.Code = 40102
    CodeInsufficientFunds kerr.Code = 40301
)

// 业务错误构造函数
func ErrUserNotFound(userID int64) error {
    return kerr.WarpError(
        kerr.StatusNotFound,
        CodeUserNotFound,
        fmt.Errorf("用户不存在: %d", userID),
    )
}

func ErrPasswordWrong() error {
    return kerr.WarpError(
        kerr.StatusUnauthorized,
        CodePasswordWrong,
        fmt.Errorf("密码错误"),
    )
}

func ErrTokenExpired() error {
    return kerr.WarpError(
        kerr.StatusUnauthorized,
        CodeTokenExpired,
        fmt.Errorf("token已过期"),
    )
}
```

### 示例4：错误响应格式

```go
// 成功响应
{
    "code": 0,
    "msg": "success",
    "data": {...},
    "trace_id": "abc123..."
}

// 错误响应
{
    "code": 40401,
    "msg": "用户不存在: 123",
    "data": null,
    "trace_id": "abc123..."
}
```

## 🔧 高级用法

### 错误类型判断

```go
import (
    "errors"
    kerr "github.com/liukunxin/go-infra/pkg/errors"
)

func handleError(err error) {
    // 方法1：使用 FromError
    ke := kerr.FromError(err)
    if ke != nil {
        fmt.Printf("状态码: %d, 错误码: %d\n", ke.Status, ke.Code)
    }

    // 方法2：使用 errors.As
    var e *kerr.Error
    if errors.As(err, &e) {
        fmt.Printf("这是一个 kerr.Error: %+v\n", e)
    }

    // 方法3：只获取错误码
    code := kerr.AsCode(err)
    fmt.Printf("错误码: %d\n", code)
}
```

### 错误链追踪

```go
func complexOperation() error {
    err := step1()
    if err != nil {
        // 包装错误，保留错误链
        return kerr.WarpError(
            kerr.StatusInternalServerError,
            50001,
            fmt.Errorf("步骤1失败: %w", err),
        )
    }
    return nil
}

// 使用 errors.Unwrap 获取原始错误
originalErr := errors.Unwrap(wrappedErr)
```

### 特殊错误处理

```go
// 用户主动取消请求
if errors.Is(err, context.Canceled) {
    return kerr.WarpError(
        kerr.StatusBadRequest,
        kerr.CodeUserContextCanceled,
        err,
    )
}

// 超时错误
if errors.Is(err, context.DeadlineExceeded) {
    return kerr.WarpError(
        kerr.StatusRequestTimeout,
        50008,
        err,
    )
}
```

## 📊 错误码规范建议

### 错误码设计原则

```
错误码格式: XYZZ
X - 错误类型 (4=客户端错误, 5=服务端错误)
Y - 模块编号 (0=通用, 1=用户, 2=订单, 3=支付等)
ZZ - 具体错误序号

示例:
40001 - 通用客户端错误
40101 - 用户模块客户端错误
50001 - 通用服务端错误
50201 - 订单模块服务端错误
```

### 推荐的错误码分类

```go
// 通用错误 (400xx, 500xx)
const (
    CodeBadRequest       = 40000  // 请求参数错误
    CodeUnauthorized     = 40100  // 未认证
    CodeForbidden        = 40300  // 无权限
    CodeNotFound         = 40400  // 资源不存在
    CodeConflict         = 40900  // 资源冲突
    CodeInternalError    = 50000  // 内部错误
    CodeServiceUnavail   = 50300  // 服务不可用
)

// 用户模块 (401xx, 501xx)
const (
    CodeUserNotFound     = 40101
    CodePasswordWrong    = 40102
    CodeUserDisabled     = 40103
)

// 订单模块 (402xx, 502xx)
const (
    CodeOrderNotFound    = 40201
    CodeOrderCanceled    = 40202
    CodeOrderPaid        = 40203
)
```

## 🎯 最佳实践

### 1. 在Service层包装错误

```go
// ❌ 不好的做法
func GetUser(id int64) (*User, error) {
    return db.Find(id)  // 直接返回数据库错误
}

// ✅ 好的做法
func GetUser(id int64) (*User, error) {
    user, err := db.Find(id)
    if err != nil {
        return nil, kerr.WarpError(
            kerr.StatusInternalServerError,
            50001,
            fmt.Errorf("查询用户失败: %w", err),
        )
    }
    return user, nil
}
```

### 2. 在Controller层统一处理

```go
// ✅ 使用 GinBase 的统一处理
func (c *UserController) GetUser(ctx *gin.Context) {
    user, err := service.GetUser(123)
    if err != nil {
        c.ErrorResponse(ctx, err)  // 统一处理
        return
    }
    c.SuccessResponse(ctx, user)
}
```

### 3. 错误日志记录

```go
import "github.com/liukunxin/go-infra/pkg/log"

func handleError(ctx context.Context, err error) {
    ke := kerr.FromError(err)
    log.WithContext(ctx).WithFields(map[string]interface{}{
        "error_code":   ke.Code,
        "http_status":  ke.Status,
        "error_msg":    ke.Error(),
    }).Error("业务错误")
}
```

## ⚠️ 注意事项

1. **不要过度包装** - 一个错误只包装一次
2. **保留错误链** - 使用 `%w` 格式化，保留原始错误
3. **错误码要唯一** - 避免错误码冲突
4. **错误信息要明确** - 包含足够的上下文信息
5. **敏感信息脱敏** - 不要在错误信息中暴露密码、token等

## 🔗 相关模块

- [Controller](controller.md) - 统一的响应格式
- [Log](log.md) - 错误日志记录
- [Trace](trace.md) - 错误追踪
