# Kafka 客户端

基于 [segmentio/kafka-go](https://github.com/segmentio/kafka-go) 的 Kafka 封装，提供**生产者**和**消费者组**，支持 SASL/TLS 认证、消息重试和死信队列回调。

---

## 快速上手

### 1. 生产者

```go
import "github.com/yourorg/go-infra/pkg/infra/kafka"

func main() {
    // 初始化全局生产者
    err := kafka.InitProducer(
        kafka.Config{
            Brokers:  []string{"127.0.0.1:9092"},
            ClientID: "my-service",
        },
        kafka.ProducerConfig{
            RequiredAcks: kafka.RequireAll,   // 等待所有副本确认（最安全）
            MaxAttempts:  3,
        },
    )
    if err != nil {
        log.Fatalf("kafka producer init: %v", err)
    }
}

// 发送消息
func sendOrder(ctx context.Context, order Order) error {
    data, _ := json.Marshal(order)
    return kafka.GetProducer().Send(ctx, kafka.Message{
        Topic: "order.created",
        Key:   []byte(order.ID),    // 相同 Key 发到同一分区，保证顺序
        Value: data,
    })
}

// 批量发送（一次网络请求）
func sendBatch(ctx context.Context, orders []Order) error {
    msgs := make([]kafka.Message, 0, len(orders))
    for _, o := range orders {
        data, _ := json.Marshal(o)
        msgs = append(msgs, kafka.Message{Topic: "order.created", Value: data})
    }
    return kafka.GetProducer().Send(ctx, msgs...)
}
```

### 2. 消费者组

```go
func main() {
    cg, err := kafka.InitConsumer(
        kafka.Config{Brokers: []string{"127.0.0.1:9092"}},
        kafka.ConsumerConfig{
            GroupID:    "order-service",
            MaxRetries: 3,                    // 失败重试 3 次
            RetryBackoff: time.Second,        // 重试间隔（指数增长）
        },
    )
    if err != nil {
        log.Fatalf("kafka consumer init: %v", err)
    }

    // 注册消息处理函数
    cg.Subscribe("order.created", handleOrderCreated)
    cg.Subscribe("order.paid",    handleOrderPaid)

    // 设置死信队列回调（所有重试耗尽后调用）
    cg.OnError = func(ctx context.Context, msg kafka.Message, err error) {
        log.Printf("消息处理失败，写入死信队列: topic=%s offset=%d err=%v",
            msg.Topic, msg.Offset, err)
        // 可以把 msg 写入 Redis/DB/另一个 Kafka topic
    }

    // Start 会阻塞，建议在独立 goroutine 中运行
    go func() {
        if err := cg.Start(context.Background()); err != nil {
            log.Printf("consumer stopped: %v", err)
        }
    }()
}

func handleOrderCreated(ctx context.Context, msg kafka.Message) error {
    var order Order
    if err := json.Unmarshal(msg.Value, &order); err != nil {
        return err  // 返回错误会触发重试
    }
    // ... 处理业务逻辑
    return nil  // 返回 nil 提交 offset
}
```

---

## 消息可靠性说明

| 机制              | 说明                                                            |
|------------------|----------------------------------------------------------------|
| **手动提交**      | Offset 在 Handler 返回 nil **之后**才提交，宕机不丢消息          |
| **at-least-once** | 极端情况下（提交前崩溃）消息会重复投递，Handler 应实现幂等      |
| **自动重试**      | Handler 失败后按 `RetryBackoff` 指数退避重试，最多 `MaxRetries` 次 |
| **死信队列**      | 超出重试次数后触发 `OnError`，消息跳过（不阻塞后续消息）        |

---

## 认证配置

### SASL/PLAIN

```go
cfg := kafka.Config{
    Brokers: []string{"kafka:9093"},
    SASL: &kafka.SASLConfig{
        Mechanism: "PLAIN",
        Username:  "kafka-user",
        Password:  "kafka-pass",
    },
}
```

### SASL/SCRAM-SHA-256 或 SCRAM-SHA-512

```go
cfg := kafka.Config{
    Brokers: []string{"kafka:9094"},
    SASL: &kafka.SASLConfig{
        Mechanism: "SCRAM-SHA-256",  // 或 "SCRAM-SHA-512"
        Username:  "kafka-user",
        Password:  "kafka-pass",
    },
}
```

### TLS + CA 证书

```go
caCert, _ := os.ReadFile("/path/to/ca.crt")
cfg := kafka.Config{
    Brokers: []string{"kafka:9095"},
    TLS: &kafka.TLSConfig{
        CACert: caCert,
    },
}
```

---

## 优雅关闭

```go
// 生产者：等待缓冲消息全部发送完毕再退出
defer kafka.GetProducer().Close()

// 消费者：停止拉取，等待当前正在处理的消息完成
defer kafka.GetConsumer().Close()
```

---

## 监控生产者

```go
stats := kafka.GetProducer().Stats()
fmt.Printf("发送成功: %d  失败: %d  重试: %d\n",
    stats.Writes, stats.Errors, stats.Retries)
```

---

## 配置说明

### Config（公共）

| 字段       | 说明                                    |
|------------|----------------------------------------|
| `Brokers`  | Broker 地址列表（必填）                  |
| `ClientID` | 客户端标识，便于在 Kafka 侧追踪           |
| `SASL`     | SASL 认证配置（可选）                    |
| `TLS`      | TLS 配置（可选）                         |

### ProducerConfig

| 字段                    | 说明                               | 默认值    |
|------------------------|-----------------------------------|-----------|
| `BatchSize`            | 每批最多消息数                      | 100       |
| `BatchTimeout`         | 批次等待时间                        | 10ms      |
| `WriteTimeout`         | 写超时                              | 10s       |
| `MaxAttempts`          | 最大重试次数                        | 3         |
| `RequiredAcks`         | 确认级别（None/Leader/All）         | All       |
| `Compression`          | 压缩算法（None/Gzip/Snappy/Lz4）   | None      |
| `AllowAutoTopicCreation` | 自动创建 topic                    | false     |

### ConsumerConfig

| 字段           | 说明                               | 默认值    |
|---------------|-----------------------------------|-----------|
| `GroupID`     | 消费组 ID（必填）                   | —         |
| `MinBytes`    | 一次 Fetch 最少字节数               | 1 B       |
| `MaxBytes`    | 一次 Fetch 最多字节数               | 10 MB     |
| `MaxWait`     | Fetch 最长等待时间                  | 1s        |
| `StartOffset` | 新消费组从何处开始（-1=最新/-2=最早）| -2（最早）|
| `MaxRetries`  | Handler 失败后重试次数              | 3         |
| `RetryBackoff`| 首次重试等待（指数增长）            | 500ms     |
