# Redis 客户端

基于 [redis/go-redis/v9](https://github.com/redis/go-redis) 的 Redis 封装，同时支持**单节点**和**集群**模式，内置连接池管理，开箱即用。

---

## 快速上手

### 1. 初始化（程序启动时调用一次）

**单节点模式**

```go
import "github.com/liukunxin/go-infra/pkg/infra/redis"

func main() {
    cfg := &redis.Config{
        Mode:         "single",
        Addresses:    []string{"127.0.0.1:6379"},
        Password:     "",
        PoolSize:     20,
        MinIdleConns: 5,
        IdleTimeout:  5 * time.Minute,
    }

    if err := redis.Init(cfg); err != nil {
        log.Fatalf("redis init: %v", err)
    }
}
```

**集群模式**

```go
cfg := &redis.Config{
    Mode: "cluster",
    Addresses: []string{
        "127.0.0.1:7000",
        "127.0.0.1:7001",
        "127.0.0.1:7002",
    },
    Password: "cluster_password",
}
```

### 2. 获取客户端

```go
rdb := redis.GetClient()   // 未初始化时 panic
```

---

## 配置说明

| 字段            | 说明                               | 默认值  |
|-----------------|-----------------------------------|---------|
| `Mode`          | `"single"` 或 `"cluster"`        | —       |
| `Addresses`     | 节点地址列表（必填）               | —       |
| `Password`      | 密码                               | ""      |
| `PoolSize`      | 连接池大小                         | 10      |
| `MinIdleConns`  | 最小空闲连接数                     | 0       |
| `IdleTimeout`   | 空闲连接关闭时间（`time.Duration`）| 0       |
