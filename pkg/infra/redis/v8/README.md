# Redis 客户端

基于 [go-redis/v8](https://github.com/go-redis/redis) 的 Redis 封装，同时支持**单节点**和**集群**模式，内置连接池管理，开箱即用。

---

## 快速上手

### 1. 初始化（程序启动时调用一次）

**单节点模式**

```go
import "github.com/yourorg/go-infra/pkg/infra/redis/v8"

func main() {
    cfg := redis.Config{
        Addrs:       []string{"127.0.0.1:6379"},
        Password:    "",      // 无密码则留空
        DB:          0,
        PoolSize:    20,
        MinIdleConns: 5,
        DialTimeout: 5 * time.Second,
        ReadTimeout: 3 * time.Second,
        WriteTimeout: 3 * time.Second,
        IdleTimeout:  5 * time.Minute,
    }

    if err := redis.Init(cfg); err != nil {
        log.Fatalf("redis init: %v", err)
    }
}
```

**集群模式**（配置多个地址即自动切换为集群）

```go
cfg := redis.Config{
    Addrs: []string{
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

## 常用操作

### 字符串

```go
ctx := context.Background()
rdb := redis.GetClient()

// 设置 key（带过期时间）
err := rdb.Set(ctx, "user:name", "Alice", 10*time.Minute).Err()

// 获取 key
val, err := rdb.Get(ctx, "user:name").Result()
if errors.Is(err, goredis.Nil) {
    // key 不存在
}

// 原子自增
newVal, err := rdb.Incr(ctx, "counter").Result()
```

### 哈希表

```go
// 设置哈希字段
err := rdb.HSet(ctx, "user:1", "name", "Alice", "age", "20").Err()

// 读取单个字段
name, err := rdb.HGet(ctx, "user:1", "name").Result()

// 读取所有字段
fields, err := rdb.HGetAll(ctx, "user:1").Result()
```

### 列表

```go
// 从左侧推入
err := rdb.LPush(ctx, "task_queue", "task1", "task2").Err()

// 阻塞弹出（适合队列消费，超时 5s）
val, err := rdb.BLPop(ctx, 5*time.Second, "task_queue").Result()
```

### 集合

```go
// 添加成员
err := rdb.SAdd(ctx, "online_users", "uid_1", "uid_2").Err()

// 判断是否在集合中
exists, err := rdb.SIsMember(ctx, "online_users", "uid_1").Result()
```

### 有序集合

```go
// 添加带分数的成员
err := rdb.ZAdd(ctx, "leaderboard",
    &goredis.Z{Score: 100, Member: "player1"},
    &goredis.Z{Score: 200, Member: "player2"},
).Err()

// 查询排行榜（分数从高到低，取前 10 名）
members, err := rdb.ZRevRangeWithScores(ctx, "leaderboard", 0, 9).Result()
```

---

## 分布式锁

```go
// 获取锁（NX = 不存在才设置，EX = 过期时间）
ok, err := rdb.SetNX(ctx, "lock:order:42", "locked", 30*time.Second).Result()
if !ok {
    // 锁已被其他实例持有
    return errors.New("请勿重复提交")
}
defer rdb.Del(ctx, "lock:order:42")

// ... 执行业务逻辑
```

---

## Pipeline（批量命令，减少网络往返）

```go
pipe := rdb.Pipeline()
pipe.Set(ctx, "k1", "v1", time.Minute)
pipe.Set(ctx, "k2", "v2", time.Minute)
pipe.Incr(ctx, "counter")
_, err := pipe.Exec(ctx)
```

---

## 配置说明

| 字段            | 说明                               | 默认值  |
|-----------------|-----------------------------------|---------|
| `Addrs`         | 节点地址列表（必填）               | —       |
| `Password`      | 密码                               | ""      |
| `DB`            | 数据库编号（集群模式忽略）         | 0       |
| `PoolSize`      | 连接池大小                         | 10      |
| `MinIdleConns`  | 最小空闲连接数                     | 2       |
| `DialTimeout`   | 连接超时                           | 5s      |
| `ReadTimeout`   | 读超时                             | 3s      |
| `WriteTimeout`  | 写超时                             | 3s      |
| `IdleTimeout`   | 空闲连接关闭时间（`time.Duration`）| 5min    |
