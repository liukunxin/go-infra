# MySQL 客户端

基于 [GORM](https://gorm.io) 的 MySQL 封装，内置连接池、重试初始化、事务助手和健康检查，开箱即用。

---

## 快速上手

### 1. 初始化（程序启动时调用一次）

```go
import "github.com/yourorg/go-infra/pkg/infra/mysql"

func main() {
    cfg := mysql.Config{
        DSN:             "root:password@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local",
        MaxOpenConns:    50,   // 最大连接数，默认 20
        MaxIdleConns:    10,   // 最大空闲连接，默认 5
        ConnMaxLifetime: time.Hour,
        ConnMaxIdleTime: 10 * time.Minute,
    }

    if err := mysql.Init(cfg); err != nil {
        log.Fatalf("mysql init: %v", err)
    }
}
```

### 2. 获取客户端

```go
db := mysql.GetClient()   // 如果未初始化会 panic
```

---

## 常用操作

### 查询单条记录

```go
type User struct {
    ID   uint   `gorm:"primaryKey"`
    Name string
    Age  int
}

var user User
err := mysql.GetClient().DB.Where("id = ?", 1).First(&user).Error
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        // 记录不存在
    }
    return err
}
fmt.Println(user.Name)
```

### 查询多条记录

```go
var users []User
err := mysql.GetClient().DB.Where("age > ?", 18).Find(&users).Error
```

### 插入

```go
user := User{Name: "Alice", Age: 20}
err := mysql.GetClient().DB.Create(&user).Error
```

### 更新

```go
// 更新指定字段（避免零值覆盖）
err := mysql.GetClient().DB.Model(&User{}).Where("id = ?", 1).
    Updates(map[string]interface{}{"name": "Bob", "age": 21}).Error
```

### 删除

```go
err := mysql.GetClient().DB.Where("id = ?", 1).Delete(&User{}).Error
```

---

## 事务

```go
err := mysql.GetClient().Transaction(ctx, func(tx *gorm.DB) error {
    if err := tx.Create(&Order{UserID: 1, Amount: 100}).Error; err != nil {
        return err // 返回非 nil 会自动回滚
    }
    if err := tx.Model(&Account{}).Where("user_id = ?", 1).
        UpdateColumn("balance", gorm.Expr("balance - ?", 100)).Error; err != nil {
        return err
    }
    return nil // 返回 nil 自动提交
})
```

---

## 执行原生 SQL

```go
// Exec（INSERT / UPDATE / DELETE）
result, err := mysql.GetClient().ExecContext(ctx,
    "UPDATE users SET status = ? WHERE id = ?", 1, 42)
if err != nil {
    return err
}
fmt.Println("受影响行数:", result.RowsAffected)

// RawQuery（SELECT）
type Row struct {
    Name  string
    Count int
}
var rows []Row
err = mysql.GetClient().RawQuery(ctx, &rows,
    "SELECT name, COUNT(*) AS count FROM users GROUP BY name")
```

---

## 健康检查

```go
if err := mysql.GetClient().HealthCheck(ctx); err != nil {
    // 数据库不可达
}
```

---

## 配置说明

| 字段               | 说明                     | 默认值       |
|--------------------|--------------------------|--------------|
| `DSN`              | 连接字符串（必填）        | —            |
| `MaxOpenConns`     | 最大连接数               | 20           |
| `MaxIdleConns`     | 最大空闲连接             | 5            |
| `ConnMaxLifetime`  | 连接最长存活时间         | 1h           |
| `ConnMaxIdleTime`  | 空闲连接最长存活时间     | 10m          |
| `MaxRetries`       | 初始化重试次数           | 3            |
| `RetryInterval`    | 重试间隔                 | 2s           |
| `SlowThreshold`    | 慢查询告警阈值           | 200ms        |
| `LogLevel`         | GORM 日志级别            | Warn         |

---

## 获取原始 `*gorm.DB`

需要使用 GORM 高级功能时，直接访问 `.DB` 字段：

```go
db := mysql.GetClient().DB
db.Session(&gorm.Session{PrepareStmt: true}).Find(&users)
```
