# MySQL - 数据库客户端

基于 GORM 的 MySQL 客户端封装，提供连接池管理、事务支持和合理的默认配置。

## 📖 功能特性

- ✅ 基于 GORM v2
- ✅ 连接池自动管理
- ✅ 合理的默认配置
- ✅ 自动重连机制
- ✅ 事务封装
- ✅ PreparedStatement 缓存
- ✅ 健康检查

## 🚀 快速开始

### 基础用法

```go
package main

import (
    "context"
    "log"
    "github.com/liukunxin/go-infra/pkg/mysql"
    "gorm.io/gorm/logger"
)

func main() {
    // 1. 初始化（使用默认配置）
    mysql.Init(mysql.Config{
        DSN: "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
    })

    // 2. 获取客户端
    client := mysql.GetClient()

    // 3. 基础查询
    var users []User
    ctx := context.Background()
    err := client.GetGormDB(ctx).Find(&users).Error
    if err != nil {
        log.Fatal(err)
    }
}

type User struct {
    ID   uint   `gorm:"primarykey"`
    Name string `gorm:"column:name"`
    Age  int    `gorm:"column:age"`
}
```

## 📋 配置选项

### Config 结构

```go
type Config struct {
    DSN               string          // 数据库连接字符串
    MaxOpenConns      int             // 最大打开连接数（默认100）
    MaxIdleConns      int             // 最大空闲连接数（默认10）
    ConnMaxLifetime   time.Duration   // 连接最大存活时间（默认1小时）
    ConnMaxIdleTime   time.Duration   // 连接最大空闲时间（默认10分钟）
    EnablePrepareStmt bool            // 启用预编译语句缓存（默认false）
    SkipDefaultTx     bool            // 跳过默认事务（默认false）
    ConnRetryTimes    int             // 连接重试次数（默认3）
    ConnRetryInterval time.Duration   // 重试间隔（默认2秒）
    GormLogLevel      logger.LogLevel // GORM日志级别
}
```

### 默认配置说明

```go
// 使用默认配置初始化
mysql.Init(mysql.Config{
    DSN: "...",
    // 以下为自动应用的默认值：
    // MaxOpenConns: 100      - 最大连接数
    // MaxIdleConns: 10       - 空闲连接数
    // ConnMaxLifetime: 1h    - 连接最大存活1小时
    // ConnMaxIdleTime: 10m   - 空闲连接10分钟回收
    // ConnRetryTimes: 3      - 连接失败重试3次
    // ConnRetryInterval: 2s  - 重试间隔2秒
})
```

## 💡 使用示例

### 示例1：完整配置

```go
func initDB() {
    mysql.Init(mysql.Config{
        DSN:               "user:pass@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local",
        MaxOpenConns:      200,             // 高并发场景
        MaxIdleConns:      20,              // 保持更多空闲连接
        ConnMaxLifetime:   2 * time.Hour,   // 2小时回收
        ConnMaxIdleTime:   15 * time.Minute, // 15分钟回收空闲
        EnablePrepareStmt: true,            // 启用预编译（提升性能）
        SkipDefaultTx:     true,            // 跳过默认事务（提升写入性能）
        GormLogLevel:      logger.Info,     // 开发环境显示SQL
    })
}
```

### 示例2：基础 CRUD

```go
func userService() {
    client := mysql.GetClient()
    ctx := context.Background()
    db := client.GetGormDB(ctx)

    // 创建
    user := &User{Name: "张三", Age: 25}
    db.Create(user)

    // 查询单个
    var foundUser User
    db.Where("name = ?", "张三").First(&foundUser)

    // 查询列表
    var users []User
    db.Where("age > ?", 18).Find(&users)

    // 更新
    db.Model(&foundUser).Update("age", 26)

    // 删除
    db.Delete(&foundUser)
}
```

### 示例3：事务处理

```go
func transferMoney(ctx context.Context, fromUserID, toUserID int64, amount float64) error {
    client := mysql.GetClient()

    // 使用事务封装
    return client.WithTransaction(ctx, func(tx *gorm.DB) error {
        // 扣款
        if err := tx.Model(&Account{}).
            Where("user_id = ?", fromUserID).
            Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
            return err  // 自动回滚
        }

        // 加款
        if err := tx.Model(&Account{}).
            Where("user_id = ?", toUserID).
            Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
            return err  // 自动回滚
        }

        return nil  // 自动提交
    })
}
```

### 示例4：原始 SQL

```go
// 执行原始 SQL 查询
type Result struct {
    UserID int64
    Total  float64
}

func queryUserTotal(ctx context.Context, userID int64) (*Result, error) {
    client := mysql.GetClient()
    var result Result

    sql := "SELECT user_id, SUM(amount) as total FROM orders WHERE user_id = ?"
    err := client.RawQuery(ctx, &result, sql, userID)
    return &result, err
}

// 执行原始 SQL 更新
func batchUpdate(ctx context.Context) error {
    client := mysql.GetClient()
    sql := "UPDATE users SET status = ? WHERE created_at < ?"
    rows, err := client.ExecContext(ctx, sql, "inactive", time.Now().AddDate(0, -6, 0))
    log.Printf("更新了 %d 行", rows)
    return err
}
```

### 示例5：复杂查询

```go
func complexQuery(ctx context.Context) ([]User, error) {
    client := mysql.GetClient()
    db := client.GetGormDB(ctx)

    var users []User
    err := db.
        Select("users.*, COUNT(orders.id) as order_count").
        Joins("LEFT JOIN orders ON orders.user_id = users.id").
        Where("users.status = ?", "active").
        Group("users.id").
        Having("order_count > ?", 5).
        Order("order_count DESC").
        Limit(10).
        Find(&users).Error

    return users, err
}
```

### 示例6：批量操作

```go
func batchInsert(ctx context.Context, users []User) error {
    client := mysql.GetClient()
    db := client.GetGormDB(ctx)

    // 批量插入（自动分批）
    return db.CreateInBatches(users, 100).Error  // 每批100条
}

func batchUpdate(ctx context.Context) error {
    client := mysql.GetClient()
    db := client.GetGormDB(ctx)

    // 批量更新
    return db.Model(&User{}).
        Where("status = ?", "pending").
        Updates(map[string]interface{}{
            "status":     "active",
            "updated_at": time.Now(),
        }).Error
}
```

## 🔧 高级用法

### 分页查询

```go
func getPaginatedUsers(ctx context.Context, page, pageSize int) ([]User, int64, error) {
    client := mysql.GetClient()
    db := client.GetGormDB(ctx)

    var users []User
    var total int64

    // 查询总数
    db.Model(&User{}).Count(&total)

    // 分页查询
    offset := (page - 1) * pageSize
    err := db.Offset(offset).Limit(pageSize).Find(&users).Error

    return users, total, err
}
```

### 软删除

```go
type User struct {
    ID        uint   `gorm:"primarykey"`
    Name      string
    DeletedAt gorm.DeletedAt `gorm:"index"`  // 软删除字段
}

func softDelete(ctx context.Context, userID uint) error {
    client := mysql.GetClient()
    db := client.GetGormDB(ctx)

    // 软删除（只更新 deleted_at 字段）
    return db.Delete(&User{}, userID).Error
}

func findWithDeleted(ctx context.Context) ([]User, error) {
    client := mysql.GetClient()
    db := client.GetGormDB(ctx)

    var users []User
    // 包含软删除的记录
    err := db.Unscoped().Find(&users).Error
    return users, err
}
```

### 悲观锁

```go
func updateWithLock(ctx context.Context, userID int64) error {
    client := mysql.GetClient()

    return client.WithTransaction(ctx, func(tx *gorm.DB) error {
        var user User
        // 加锁查询
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            Where("id = ?", userID).
            First(&user).Error; err != nil {
            return err
        }

        // 更新
        user.Balance += 100
        return tx.Save(&user).Error
    })
}
```

### 关联查询

```go
type User struct {
    ID     uint
    Name   string
    Orders []Order `gorm:"foreignKey:UserID"`
}

type Order struct {
    ID     uint
    UserID uint
    Amount float64
}

func getUserWithOrders(ctx context.Context, userID uint) (*User, error) {
    client := mysql.GetClient()
    db := client.GetGormDB(ctx)

    var user User
    // 预加载关联数据
    err := db.Preload("Orders").First(&user, userID).Error
    return &user, err
}
```

## 🎯 最佳实践

### 1. 使用合理的连接池配置

```go
// 开发环境
mysql.Init(mysql.Config{
    DSN:          "...",
    MaxOpenConns: 50,   // 较小的连接池
    MaxIdleConns: 5,
})

// 生产环境（高并发）
mysql.Init(mysql.Config{
    DSN:          "...",
    MaxOpenConns: 200,  // 根据数据库能力调整
    MaxIdleConns: 20,
})

// 生产环境（低并发）
mysql.Init(mysql.Config{
    DSN:          "...",
    MaxOpenConns: 100,  // 使用默认值
    MaxIdleConns: 10,
})
```

### 2. 启用 PreparedStatement 缓存

```go
// 适合读多写少的场景
mysql.Init(mysql.Config{
    DSN:               "...",
    EnablePrepareStmt: true,  // 减少 SQL 解析开销
})
```

### 3. 合理使用事务

```go
// ✅ 好的做法 - 小事务
func updateUser(ctx context.Context, user *User) error {
    return client.WithTransaction(ctx, func(tx *gorm.DB) error {
        return tx.Save(user).Error
    })
}

// ❌ 不好的做法 - 大事务（长时间持有锁）
func bigTransaction(ctx context.Context) error {
    return client.WithTransaction(ctx, func(tx *gorm.DB) error {
        // 大量操作...
        time.Sleep(10 * time.Second)  // 危险！
        return nil
    })
}
```

### 4. 使用上下文传递

```go
// ✅ 好的做法 - 传递 context
func queryUser(ctx context.Context, id int64) (*User, error) {
    db := client.GetGormDB(ctx)  // 带上下文
    var user User
    err := db.First(&user, id).Error
    return &user, err
}

// ❌ 不好的做法 - 不传递 context
func queryUser(id int64) (*User, error) {
    db := client.GetGormDB(context.Background())
    // ...
}
```

### 5. 避免 N+1 查询

```go
// ❌ 不好的做法 - N+1 查询
func getUsers() []User {
    var users []User
    db.Find(&users)
    for i := range users {
        db.Model(&users[i]).Association("Orders").Find(&users[i].Orders)  // N次查询！
    }
    return users
}

// ✅ 好的做法 - 预加载
func getUsers() []User {
    var users []User
    db.Preload("Orders").Find(&users)  // 只需2次查询
    return users
}
```

## 📊 性能优化

### 索引优化

```go
type User struct {
    ID    uint   `gorm:"primarykey"`
    Email string `gorm:"uniqueIndex"`        // 唯一索引
    Name  string `gorm:"index"`              // 普通索引
    Age   int    `gorm:"index:idx_age_city"` // 联合索引
    City  string `gorm:"index:idx_age_city"` // 联合索引
}
```

### 批量操作

```go
// 批量插入比单条插入快10-100倍
func batchInsert(users []User) {
    db.CreateInBatches(users, 1000)  // 每批1000条
}
```

### 查询优化

```go
// 只查询需要的字段
db.Select("id", "name").Find(&users)

// 使用索引
db.Where("email = ?", email).First(&user)  // email 有索引

// 避免全表扫描
db.Where("name LIKE ?", name+"%").Find(&users)  // 前缀匹配可用索引
```

## ⚠️ 注意事项

1. **DSN 格式要正确** - 包含 `charset=utf8mb4&parseTime=True&loc=Local`
2. **连接池配置要合理** - 根据数据库承受能力配置
3. **事务要尽快提交** - 避免长事务
4. **使用 context 传递** - 支持超时和取消
5. **生产环境关闭 SQL 日志** - `GormLogLevel: logger.Silent`

## 🔗 相关模块

- [Trace](trace.md) - 数据库操作追踪
- [Log](log.md) - 数据库日志记录
- [Errors](errors.md) - 数据库错误处理

## 📖 推荐阅读

- [GORM 官方文档](https://gorm.io/docs/)
- [MySQL 连接池最佳实践](https://github.com/go-sql-driver/mysql#important-settings)
