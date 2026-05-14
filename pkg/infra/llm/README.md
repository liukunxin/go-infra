# LLM - 大模型 SDK

`pkg/infra/llm` 面向业务侧的核心目标只有一个：  
**用最少代码稳定调用 LLM。**

当前推荐你默认只用“最佳使用方式”，其余能力按需开启。

---

## 最佳使用方式（默认就用这个）

### 1) 初始化客户端

```go
client, err := llm.NewOpenAICompatibleClient(llm.OpenAICompatibleClientConfig{
	Provider:  "deepseek",
	Model:     "deepseek-chat",
	BaseURL:   "https://api.deepseek.com/v1",
	APIKeyEnv: "DEEPSEEK_API_KEY",
})
if err != nil {
	panic(err)
}
```

### 2) 文本调用

```go
text, err := client.AskText(context.Background(), "用一句话介绍 go-infra")
if err != nil {
	panic(err)
}
fmt.Println(text)
```

> 这就是首选路径：`NewOpenAICompatibleClient + AskText`。  
> 能覆盖大多数业务场景，不需要先理解复杂抽象。

---

## 常用能力（按需使用）

### 流式输出

```go
stream, err := client.AskStream(context.Background(), "写一个 5 行以内的 Go 示例")
if err != nil {
	panic(err)
}
for ev := range stream {
	if ev.Err != nil {
		panic(ev.Err)
	}
	if ev.Done {
		break
	}
	fmt.Print(ev.Delta)
}
```

### 全局单例（与其他 infra 模块风格一致）

```go
_ = llm.InitOpenAICompatibleClient(llm.OpenAICompatibleClientConfig{
	Provider:  "deepseek",
	Model:     "deepseek-chat",
	BaseURL:   "https://api.deepseek.com/v1",
	APIKeyEnv: "DEEPSEEK_API_KEY",
})
text, _ := llm.GetClient().AskText(context.Background(), "hello")
```

### 配置文件初始化（适合服务项目）

```go
type AppConfig struct {
	LLM llm.Config `yaml:"llm"`
}

cfg := config.MustLoad[AppConfig](config.WithBaseDir("config"))
if err := llm.InitFromConfig(cfg.LLM); err != nil {
	panic(err)
}
```

```yaml
llm:
  default_provider: deepseek
  default_model: deepseek-chat
  providers:
    deepseek:
      type: openai_compatible
      base_url: https://api.deepseek.com/v1
      api_key_env: DEEPSEEK_API_KEY
```

---

## 能力清单（当前已实现）

- `OpenAI 兼容协议` provider 接入
- 统一调用接口：`Ask` / `AskText` / `AskStream`
- 标准接口：`Generate` / `GenerateStream`（多消息与细粒度参数控制）
- 初始化方式：简单初始化、全局单例、配置文件初始化
- 重试能力：指数退避 + 抖动（默认关闭，按需开启）
- fallback：主 provider/model 失败切备 provider/model
- 可观测性：日志、Trace、Metrics 自动埋点

---

## 推荐级别（避免过度设计）

- **默认推荐**：`NewOpenAICompatibleClient + AskText/AskStream`
- **按需开启**：`RetryConfig`（网络抖动或上游限流明显时）
- **进阶场景再用**：fallback、多 provider 路由、`Generate/GenerateStream`

> 原则：先用简单路径跑通，只有遇到明确问题再开启高级能力。

---

## 可用性确认（对外暴露能力）

以下对外 API 已实现并可直接使用：

- 客户端初始化：
  - `NewOpenAICompatibleClient`
  - `InitOpenAICompatibleClient`
  - `NewFromConfig`
  - `InitFromConfig`
- 调用：
  - `Ask`
  - `AskText`
  - `AskStream`
  - `Generate`
  - `GenerateStream`
- 可靠性：
  - `RetryConfig`
  - `WithFallback`
  - `WithFallbackAnyModel`

---

## 错误与观测

- 常见错误：`ErrInvalidConfig`、`ErrProviderNotFound`、`ErrModelRequired`
- 上游错误：`*ProviderError`（含 `HTTPStatus`、`RequestID`、`Code`）
- 自动埋点：
  - 日志：provider/model/attempt
  - Trace：`llm.generate`、`llm.stream`
  - Metrics：`llm_requests_total`、`llm_latency_ms`
# LLM - 大模型 SDK

`pkg/infra/llm` 的目标很简单：  
让业务用最少代码调用模型，不强迫你一上来就接入复杂能力。

---

## 最常用（默认就用这个）

```go
client, err := llm.NewOpenAICompatibleClient(llm.OpenAICompatibleClientConfig{
	Provider:  "deepseek",
	Model:     "deepseek-chat",
	BaseURL:   "https://api.deepseek.com/v1",
	APIKeyEnv: "DEEPSEEK_API_KEY",
})
if err != nil {
	panic(err)
}

text, err := client.AskText(context.Background(), "用一句话介绍 go-infra")
if err != nil {
	panic(err)
}
fmt.Println(text)
```

这套用法覆盖绝大多数场景：  
- 初始化：`NewOpenAICompatibleClient`  
- 文本问答：`AskText`  

---

## 常用补充

### 流式输出

```go
stream, err := client.AskStream(context.Background(), "写一个 5 行以内的 Go 示例")
if err != nil {
	panic(err)
}
for ev := range stream {
	if ev.Err != nil {
		panic(ev.Err)
	}
	if ev.Done {
		break
	}
	fmt.Print(ev.Delta)
}
```

### 全局单例（和仓库其他 infra 模块风格一致）

```go
_ = llm.InitOpenAICompatibleClient(llm.OpenAICompatibleClientConfig{
	Provider:  "deepseek",
	Model:     "deepseek-chat",
	BaseURL:   "https://api.deepseek.com/v1",
	APIKeyEnv: "DEEPSEEK_API_KEY",
})

text, _ := llm.GetClient().AskText(context.Background(), "hello")
```

### 配置文件初始化（适合服务化项目）

```go
type AppConfig struct {
	LLM llm.Config `yaml:"llm"`
}

cfg := config.MustLoad[AppConfig](config.WithBaseDir("config"))
if err := llm.InitFromConfig(cfg.LLM); err != nil {
	panic(err)
}
```

`yaml` 示例：

```yaml
llm:
  default_provider: deepseek
  default_model: deepseek-chat
  providers:
    deepseek:
      type: openai_compatible
      base_url: https://api.deepseek.com/v1
      api_key_env: DEEPSEEK_API_KEY
```

---

## 重试（按需开启）

默认不重试；只有你需要时再开：

```go
client, _ := llm.NewOpenAICompatibleClient(llm.OpenAICompatibleClientConfig{
	Provider:  "deepseek",
	Model:     "deepseek-chat",
	BaseURL:   "https://api.deepseek.com/v1",
	APIKeyEnv: "DEEPSEEK_API_KEY",
	Retry: llm.RetryConfig{
		Enabled:     true,
		MaxAttempts: 3,
		BaseBackoff: 200 * time.Millisecond,
		MaxBackoff:  2 * time.Second,
		RetryOn429:  true,
		RetryOn5xx:  true,
	},
})
```

---

## 高级能力（确实需要再用）

### Fallback（主模型失败切备用）

```go
client, _ := llm.New(
	llm.WithProvider("deepseek", deepseekProvider),
	llm.WithProvider("openai", openaiProvider),
	llm.WithDefaultProvider("deepseek"),
	llm.WithDefaultModel("deepseek-chat"),
	llm.WithFallbackAnyModel("deepseek",
		llm.FallbackTarget{Provider: "openai", Model: "gpt-4.1-mini"},
	),
)
```

---

## 什么时候用标准 API

只有下面情况再用 `Generate/GenerateStream`：
- 你要自己构造多轮 `messages`
- 你要精细控制 `temperature/top_p/max_tokens`
- 你要做多 provider 路由策略

否则，优先使用 `AskText/AskStream` 即可。

---

## 错误与可观测

- 常见错误：`ErrInvalidConfig`、`ErrProviderNotFound`、`ErrModelRequired`
- 上游错误：`*ProviderError`（含 `HTTPStatus`、`RequestID`、`Code`）
- 自动埋点：
  - 日志：provider/model/attempt
  - Trace：`llm.generate`、`llm.stream`
  - Metrics：`llm_requests_total`、`llm_latency_ms`
# LLM - 大模型统一调用 SDK

`pkg/infra/llm` 提供统一的大模型调用接口，屏蔽不同厂商 API 差异，支持：

- 非流式文本生成（`Generate`）
- 流式文本生成（`GenerateStream`）
- 默认 provider/model 与单次调用覆盖
- 失败自动 fallback（主模型失败切换备模型）
- 可配置请求重试（指数退避 + 抖动）
- 内置基础可观测性（日志 + Trace + Metrics）
- OpenAI 兼容协议接入（可覆盖大多数主流模型平台）

---

## 设计目标

- **便捷好用**：业务只关注 `messages + model`，不关心厂商请求细节
- **低侵入**：独立模块，不影响现有 `base/infra/biz` 其他能力
- **可扩展**：新增厂商仅需实现 `Provider` 接口

---

## 快速开始

### 0) 最简方式（推荐）

首次接入可直接用“一步初始化 + 一句调用”：

```go
client, err := llm.NewOpenAICompatibleClient(llm.OpenAICompatibleClientConfig{
	Provider:  "deepseek",
	Model:     "deepseek-chat",
	BaseURL:   "https://api.deepseek.com/v1",
	APIKeyEnv: "DEEPSEEK_API_KEY",
})
if err != nil {
	panic(err)
}

text, err := client.AskText(context.Background(), "用一句话介绍 go-infra")
if err != nil {
	panic(err)
}
fmt.Println(text)
```

如使用全局单例：

```go
_ = llm.InitOpenAICompatibleClient(llm.OpenAICompatibleClientConfig{
	Provider:  "deepseek",
	Model:     "deepseek-chat",
	BaseURL:   "https://api.deepseek.com/v1",
	APIKeyEnv: "DEEPSEEK_API_KEY",
})
text, _ := llm.GetClient().AskText(context.Background(), "hello")
```

### 1) 创建 Provider 与 Client（进阶可控）

```go
package main

import (
	"context"
	"fmt"

	"github.com/liukunxin/go-infra/pkg/infra/llm"
)

func main() {
	deepseekProvider, err := llm.NewOpenAICompatibleProvider("deepseek", llm.OpenAICompatibleConfig{
		BaseURL: "https://api.deepseek.com/v1",
		APIKey:  "sk-xxxx",
	})
	if err != nil {
		panic(err)
	}

	client, err := llm.New(
		llm.WithProvider("deepseek", deepseekProvider),
		llm.WithDefaultProvider("deepseek"),
		llm.WithDefaultModel("deepseek-chat"),
	)
	if err != nil {
		panic(err)
	}

	resp, err := client.Generate(context.Background(), llm.GenerateRequest{
		Messages: []llm.Message{
			llm.SystemMessage("你是一个简洁专业的助手"),
			llm.UserMessage("用一句话介绍 go-infra"),
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Content)
}
```

### 2) 流式调用

```go
stream, err := client.GenerateStream(context.Background(), llm.GenerateRequest{
	Messages: []llm.Message{
		llm.UserMessage("写一个 5 行以内的 Go 示例"),
	},
})
if err != nil {
	panic(err)
}

for event := range stream {
	if event.Err != nil {
		panic(event.Err)
	}
	if event.Done {
		break
	}
	fmt.Print(event.Delta)
}
```

### 3) 单次调用覆盖 provider/model

```go
resp, err := client.Generate(ctx, llm.GenerateRequest{
	Messages: []llm.Message{llm.UserMessage("hello")},
}, llm.WithCallProvider("deepseek"), llm.WithCallModel("deepseek-reasoner"))
```

---

## 全局单例模式（与其他 infra 模块一致）

```go
_ = llm.Init(
	llm.WithProvider("openai", provider),
	llm.WithDefaultProvider("openai"),
	llm.WithDefaultModel("gpt-4.1-mini"),
)

cli := llm.GetClient()
```

---

## 配置文件方式初始化（推荐）

可结合 `pkg/base/config` 使用：

```go
type AppConfig struct {
	LLM llm.Config `yaml:"llm"`
}

cfg := config.MustLoad[AppConfig](config.WithBaseDir("config"))
if err := llm.InitFromConfig(cfg.LLM); err != nil {
	panic(err)
}
client := llm.GetClient()
```

示例 `llm` 配置：

```yaml
llm:
  default_provider: deepseek
  default_model: deepseek-chat
  providers:
    deepseek:
      type: openai_compatible
      base_url: https://api.deepseek.com/v1
      api_key_env: DEEPSEEK_API_KEY
      http_timeout: 30s
      retry:
        enabled: true
        max_attempts: 3
        base_backoff: 200ms
        max_backoff: 2s
        jitter_ratio: 0.2
        retry_on_429: true
        retry_on_5xx: true
    openai:
      type: openai_compatible
      base_url: https://api.openai.com/v1
      api_key_env: OPENAI_API_KEY
  fallbacks:
    - primary_provider: deepseek
      primary_model: deepseek-chat
      targets:
        - provider: openai
          model: gpt-4.1-mini
```

---

## Fallback（主备切换）

可通过代码注册 fallback：

```go
client, _ := llm.New(
	llm.WithProvider("deepseek", deepseekProvider),
	llm.WithProvider("openai", openaiProvider),
	llm.WithDefaultProvider("deepseek"),
	llm.WithDefaultModel("deepseek-chat"),
	llm.WithFallback("deepseek", "deepseek-chat",
		llm.FallbackTarget{Provider: "openai", Model: "gpt-4.1-mini"},
	),
)
```

当主路由请求失败（且非 `context canceled/deadline` 等不可重试错误）时，SDK 会自动尝试备路由。

也可以使用 `WithFallbackAnyModel`，省略主模型名：

```go
client, _ := llm.New(
	llm.WithProvider("deepseek", deepseekProvider),
	llm.WithProvider("openai", openaiProvider),
	llm.WithFallbackAnyModel("deepseek",
		llm.FallbackTarget{Provider: "openai", Model: "gpt-4.1-mini"},
	),
)
```

---

## 重试策略（指数退避 + 抖动）

`OpenAICompatibleProvider` 支持可配置重试（默认不开启，保持单次请求语义）：

```go
provider, _ := llm.NewOpenAICompatibleProvider("openai", llm.OpenAICompatibleConfig{
	BaseURL: "https://api.openai.com/v1",
	APIKey:  "sk-xxxx",
	Retry: llm.RetryConfig{
		Enabled:     true,
		MaxAttempts: 3,
		BaseBackoff: 200 * time.Millisecond,
		MaxBackoff:  2 * time.Second,
		JitterRatio: 0.2,
		RetryOn429:  true,
		RetryOn5xx:  true,
	},
})
```

重试规则：

- 连接错误、超时等传输层错误可重试（`context canceled/deadline exceeded` 不重试）
- 可按配置重试 `429` 与 `5xx`
- 流式场景仅对“建连阶段”重试，开始返回流后不再重放

---

## 当前使用方式是否最简单？

不是。当前仓库现在提供了三层用法，但**对外推荐只使用一种：最简层**。

- **最简层（推荐）**：`NewOpenAICompatibleClient + AskText`（最短路径，默认选它）
- **标准层（底层能力）**：`New(...) + Generate(...)`（当你需要手工拼消息、调温度等细节时再用）
- **进阶层（生产增强）**：多 provider + fallback + 配置化 + 重试

结论：

- 对业务团队：文档和示例只看最简层即可。
- 对 SDK 维护：标准层需要保留，它是最简层的底座，也用于复杂场景扩展。

---

## 可复用 HTTP 连接池

可与 `pkg/infra/http_client` 组合，复用连接池：

```go
shared := http_client.NewClient(http_client.Config{Timeout: 30 * time.Second})
provider, _ := llm.NewOpenAICompatibleProvider("openai", llm.OpenAICompatibleConfig{
	BaseURL:    "https://api.openai.com/v1",
	APIKey:     "sk-xxxx",
	HTTPClient: shared.HTTPClient(),
})
```

---

## 错误处理

- 配置错误：`ErrInvalidConfig`
- provider 缺失：`ErrNoProviders` / `ErrProviderNotFound`
- 请求未指定模型：`ErrModelRequired`
- 上游平台错误：`*ProviderError`（含 `HTTPStatus`、`RequestID`、`Code`、`Type`）

```go
resp, err := client.Generate(ctx, req)
if err != nil {
	var pe *llm.ProviderError
	if errors.As(err, &pe) {
		// pe.Provider / pe.HTTPStatus / pe.RequestID / pe.Code
	}
}
```

---

## 可观测性

`llm.Client` 内置以下埋点：

- **日志**：每次调用会记录 provider/model/attempt（复用 `pkg/base/log`）
- **Trace**：创建 `llm.generate` / `llm.stream` span（OTel tracer `go-infra/llm`）
- **Metrics**：
  - `llm_requests_total{provider,model,operation,status}`
  - `llm_latency_ms{provider,model,operation,status}`

只要你的服务已初始化本仓库的日志/trace/metrics 体系，LLM 模块会自动接入同一套观测链路。

---

## 接口扩展

若接入非 OpenAI 协议厂商，实现以下接口即可：

```go
type Provider interface {
	Name() string
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamEvent, error)
}
```
