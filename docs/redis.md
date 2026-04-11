# Redis - 缓存客户端

基于 go-redis/v8 的 Redis 客户端封装，支持单机和集群模式。

## 📖 功能特性

- ✅ 支持单机模式
- ✅ 支持集群模式
- ✅ 连接池管理
- ✅ 自动健康检查
- ✅ 并发安全初始化
- ✅ 统一的客户端接口

## 🚀 快速开始

### 单机模式

```go
package main

import (
    "context"
    "github.com/liukunxin/go-infra/pkg/redis/v8"
    "log"
)

func main() {
    // 1. 初始化 Redis（单机模式）
    v8.InitRedisCluster(&v8.Config{
        Mode:         "single",
        Addresses:    []string{"localhost:6379"},
        Password:     "",
        PoolSize:     100,
        MinIdleConns: 10,
        IdleTimeout:  300,  // 秒
    })

    // 2. 获取客户端
    client := v8.GetRedisClient()

    // 3. 基础操作
    ctx := context.Background()
    
    // Set
    client.Set(ctx, "key", "value", 0)
    
    // Get
    val, err := client.Get(ctx, "key").Result()
    if err != nil {
        log.Fatal(err)
    }
    log.Println("value:", val)
}
```

### 集群模式

```go
func main() {
    // 初始化 Redis（集群模式）
    v8.InitRedisCluster(&v8.Config{
        Mode:         "cluster",
        Addresses:    []string{
            "node1:6379",
            "node2:6379",
            "node3:6379",
        },
        Password:     "your_password",
        PoolSize:     200,
        MinIdleConns: 20,
        IdleTimeout:  300,
    })

    client := v8.GetRedisClient()
    // 使用方式与单机模式相同
}
```

## 📋 配置选项

### Config 结构

```go
type Config struct {
    Mode         string   // 模式: "single" 或 "cluster"
    Addresses    []string // Redis地址列表
    Password     string   // 密码
    PoolSize     int      // 连接池大小
    MinIdleConns int      // 最小空闲连接数
    IdleTimeout  int      // 空闲连接超时（秒）
}
```

### 推荐配置

```go
// 开发环境
&v8.Config{
    Mode:         "single",
    Addresses:    []string{"localhost:6379"},
    Password:     "",
    PoolSize:     50,
    MinIdleConns: 5,
    IdleTimeout:  300,
}

// 生产环境（单机）
&v8.Config{
    Mode:         "single",
    Addresses:    []string{"redis.prod.com:6379"},
    Password:     "strong_password",
    PoolSize:     200,
    MinIdleConns: 20,
    IdleTimeout:  300,
}

// 生产环境（集群）
&v8.Config{
    Mode:         "cluster",
    Addresses:    []string{"node1:6379", "node2:6379", "node3:6379"},
    Password:     "strong_password",
    PoolSize:     500,
    MinIdleConns: 50,
    IdleTimeout:  300,
}
```

## 💡 使用示例

### 示例1：字符串操作

```go
func stringOperations(ctx context.Context) {
    client := v8.GetRedisClient()

    // Set
    client.Set(ctx, "user:1:name", "张三", 0)

    // Get
    name, _ := client.Get(ctx, "user:1:name").Result()
    log.Println(name)  // "张三"

    // SetNX（不存在才设置）
    ok, _ := client.SetNX(ctx, "lock:order:123", "1", 10*time.Second).Result()
    if ok {
        log.Println("获取锁成功")
    }

    // SetEX（设置并指定过期时间）
    client.SetEX(ctx, "code:12345", "验证码", 5*time.Minute)

    // Incr
    client.Incr(ctx, "visit:count")

    // GetSet（设置新值并返回旧值）
    oldVal, _ := client.GetSet(ctx, "key", "new_value").Result()
}
```

### 示例2：Hash 操作

```go
func hashOperations(ctx context.Context) {
    client := v8.GetRedisClient()

    // HSet
    client.HSet(ctx, "user:1", map[string]interface{}{
        "name": "张三",
        "age":  25,
        "city": "北京",
    })

    // HGet
    name, _ := client.HGet(ctx, "user:1", "name").Result()

    // HGetAll
    userMap, _ := client.HGetAll(ctx, "user:1").Result()
    log.Printf("%+v", userMap)

    // HIncrBy
    client.HIncrBy(ctx, "user:1", "age", 1)

    // HDel
    client.HDel(ctx, "user:1", "city")
}
```

### 示例3：List 操作

```go
func listOperations(ctx context.Context) {
    client := v8.GetRedisClient()

    // LPush（左侧插入）
    client.LPush(ctx, "queue:tasks", "task1", "task2", "task3")

    // RPush（右侧插入）
    client.RPush(ctx, "queue:tasks", "task4")

    // LPop（左侧弹出）
    task, _ := client.LPop(ctx, "queue:tasks").Result()
    log.Println(task)  // "task3"

    // LRange（范围查询）
    tasks, _ := client.LRange(ctx, "queue:tasks", 0, -1).Result()
    log.Printf("%+v", tasks)

    // LLen（获取长度）
    length, _ := client.LLen(ctx, "queue:tasks").Result()
}
```

### 示例4：Set 操作

```go
func setOperations(ctx context.Context) {
    client := v8.GetRedisClient()

    // SAdd
    client.SAdd(ctx, "tags:article:1", "go", "redis", "cache")

    // SMembers（获取所有成员）
    tags, _ := client.SMembers(ctx, "tags:article:1").Result()
    log.Printf("%+v", tags)

    // SIsMember（判断是否存在）
    exists, _ := client.SIsMember(ctx, "tags:article:1", "go").Result()

    // SRem（删除成员）
    client.SRem(ctx, "tags:article:1", "cache")

    // SCard（获取成员数量）
    count, _ := client.SCard(ctx, "tags:article:1").Result()
}
```

### 示例5：ZSet 操作（排行榜）

```go
func zsetOperations(ctx context.Context) {
    client := v8.GetRedisClient()

    // ZAdd（添加成员和分数）
    client.ZAdd(ctx, "rank:score", &redis.Z{Score: 100, Member: "user1"})
    client.ZAdd(ctx, "rank:score", &redis.Z{Score: 90, Member: "user2"})
    client.ZAdd(ctx, "rank:score", &redis.Z{Score: 95, Member: "user3"})

    // ZRange（按分数升序）
    members, _ := client.ZRange(ctx, "rank:score", 0, -1).Result()

    // ZRevRange（按分数降序）
    topUsers, _ := client.ZRevRange(ctx, "rank:score", 0, 9).Result()  // 前10名

    // ZRank（获取排名）
    rank, _ := client.ZRank(ctx, "rank:score", "user1").Result()

    // ZScore（获取分数）
    score, _ := client.ZScore(ctx, "rank:score", "user1").Result()

    // ZIncrBy（增加分数）
    client.ZIncrBy(ctx, "rank:score", 10, "user1")
}
```

### 示例6：分布式锁

```go
func acquireLock(ctx context.Context, lockKey string, timeout time.Duration) (bool, error) {
    client := v8.GetRedisClient()
    
    // 使用 SetNX 实现分布式锁
    return client.SetNX(ctx, lockKey, "1", timeout).Result()
}

func releaseLock(ctx context.Context, lockKey string) error {
    client := v8.GetRedisClient()
    return client.Del(ctx, lockKey).Err()
}

// 使用示例
func processOrder(ctx context.Context, orderID string) error {
    lockKey := fmt.Sprintf("lock:order:%s", orderID)
    
    // 获取锁
    acquired, err := acquireLock(ctx, lockKey, 10*time.Second)
    if err != nil {
        return err
    }
    if !acquired {
        return errors.New("获取锁失败，订单正在处理中")
    }
    defer releaseLock(ctx, lockKey)
    
    // 处理订单...
    return nil
}
```

### 示例7：缓存模式

```go
func getUser(ctx context.Context, userID int64) (*User, error) {
    client := v8.GetRedisClient()
    cacheKey := fmt.Sprintf("user:%d", userID)
    
    // 1. 尝试从缓存获取
    cached, err := client.Get(ctx, cacheKey).Result()
    if err == nil {
        var user User
        json.Unmarshal([]byte(cached), &user)
        return &user, nil
    }
    
    // 2. 缓存未命中，从数据库查询
    user, err := db.GetUser(userID)
    if err != nil {
        return nil, err
    }
    
    // 3. 写入缓存
    data, _ := json.Marshal(user)
    client.SetEX(ctx, cacheKey, data, 10*time.Minute)
    
    return user, nil
}
```

### 示例8：Pipeline（批量操作）

```go
func batchOperations(ctx context.Context) error {
    client := v8.GetRedisClient()
    
    // 创建 Pipeline
    pipe := client.Pipeline()
    
    // 批量操作
    for i := 0; i < 100; i++ {
        key := fmt.Sprintf("key:%d", i)
        pipe.Set(ctx, key, i, 0)
    }
    
    // 执行所有命令
    _, err := pipe.Exec(ctx)
    return err
}
```

### 示例9：发布订阅

```go
// 发布消息
func publishMessage(ctx context.Context, channel, message string) error {
    client := v8.GetRedisClient()
    return client.Publish(ctx, channel, message).Err()
}

// 订阅消息
func subscribeChannel(ctx context.Context, channel string) {
    client := v8.GetRedisClient()
    pubsub := client.Subscribe(ctx, channel)
    defer pubsub.Close()
    
    ch := pubsub.Channel()
    for msg := range ch {
        log.Printf("收到消息: %s", msg.Payload)
    }
}
```

## 🎯 最佳实践

### 1. 合理设置过期时间

```go
// ✅ 好的做法 - 设置过期时间
client.SetEX(ctx, "session:"+sessionID, data, 30*time.Minute)

// ❌ 不好的做法 - 不设置过期时间
client.Set(ctx, "session:"+sessionID, data, 0)  // 永不过期！
```

### 2. 使用 Pipeline 提升性能

```go
// ✅ 好的做法 - 使用 Pipeline
pipe := client.Pipeline()
for _, key := range keys {
    pipe.Get(ctx, key)
}
results, _ := pipe.Exec(ctx)

// ❌ 不好的做法 - 逐个查询
for _, key := range keys {
    client.Get(ctx, key)  // N次网络往返
}
```

### 3. 键名规范

```go
// ✅ 好的命名规范
"user:1:profile"           // 用户1的资料
"article:123:view_count"   // 文章123的浏览数
"cache:product:456"        // 商品456的缓存
"lock:order:789"           // 订单789的锁

// ❌ 不好的命名
"u1"                       // 不清晰
"data_123_abc"             // 混乱
```

### 4. 错误处理

```go
// ✅ 好的做法 - 检查特定错误
val, err := client.Get(ctx, key).Result()
if err == redis.Nil {
    log.Println("key 不存在")
} else if err != nil {
    log.Printf("Redis 错误: %v", err)
}

// ❌ 不好的做法 - 忽略错误
val, _ := client.Get(ctx, key).Result()  // 可能是错误导致的空值
```

### 5. 避免大 Key

```go
// ❌ 不好的做法 - 单个 Key 存储大量数据
client.HSet(ctx, "user:all", allUsersMap)  // 可能几百万条记录！

// ✅ 好的做法 - 分片存储
for userID, userData := range users {
    client.HSet(ctx, fmt.Sprintf("user:%d", userID), userData)
}
```

## 📊 性能优化

### 连接池配置

```go
// 根据并发量调整
&v8.Config{
    PoolSize:     maxConcurrent * 2,  // 一般为最大并发的2倍
    MinIdleConns: maxConcurrent / 2,  // 保持一定的空闲连接
    IdleTimeout:  300,
}
```

### 使用 Pipeline

```go
// Pipeline 可以将多个命令打包发送，减少网络往返
pipe := client.Pipeline()
pipe.Set(ctx, "k1", "v1", 0)
pipe.Set(ctx, "k2", "v2", 0)
pipe.Set(ctx, "k3", "v3", 0)
pipe.Exec(ctx)  // 一次网络往返
```

### 避免慢查询

```go
// ✅ 快速操作
client.Get(ctx, key)
client.Set(ctx, key, val, 0)

// ⚠️ 慢操作（谨慎使用）
client.Keys(ctx, "*")          // 全表扫描
client.HGetAll(ctx, largeHash) // 大Hash
client.ZRange(ctx, key, 0, -1) // 大ZSet
```

## ⚠️ 注意事项

1. **初始化只能调用一次** - 使用 sync.Once 保护
2. **集群模式需要多节点** - 至少3个节点
3. **密码要加密存储** - 不要硬编码在代码中
4. **生产环境监控连接池** - 关注连接数和超时
5. **避免阻塞操作** - 如 KEYS、SCAN 等

## 🔗 相关模块

- [Log](log.md) - Redis 操作日志
- [Trace](trace.md) - Redis 操作追踪
- [Metrics](metrics.md) - Redis 性能监控

## 📖 推荐阅读

- [Redis 官方文档](https://redis.io/docs/)
- [go-redis 文档](https://redis.uptrace.dev/)
- [Redis 最佳实践](https://redis.io/docs/manual/patterns/)
