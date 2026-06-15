# Config - 通用配置加载

`pkg/base/config` 提供统一的配置加载能力，支持：

- 基础配置 + 环境配置覆盖（`config.yml` + `config.<env>.yml`）
- 严格 YAML 解析（未知字段直接报错）
- 可选校验（`Validate()` + struct tag）

## 快速使用

```go
package main

import (
	"fmt"

	kconfig "github.com/liukunxin/go-infra/pkg/base/config"
	klog "github.com/liukunxin/go-infra/pkg/base/log"
	"github.com/liukunxin/go-infra/pkg/infra/mysql"
)

type App struct {
	AppName string       `yaml:"app_name" validate:"required"`
	Log     klog.Config  `yaml:"log"`
	Mysql   mysql.Config `yaml:"mysql"`
}

func main() {
	cfg, err := kconfig.Load[App](
		kconfig.WithEnvFrom("env"),
		kconfig.WithBaseDir("configs"),
		kconfig.WithValidate(true),
		kconfig.WithTagValidation(true),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.AppName)
}
```

## 环境映射

- `local` / `dev` / `develop` / `development` -> `config.local.yml`
- `test` / `testing` -> `config.test.yml`
- `gray` / `staging` -> `config.gray.yml`
- `prod` / `production` / `release` -> `config.prod.yml`

## 默认目录与兼容策略

- 默认读取目录：`configs`
- 如未显式设置 `WithBaseDir(...)` 且 `configs` 不存在，会自动回退到历史目录 `infra/config`

## 多区域部署（Region）

对于需要多区域部署的服务（如新加坡、美西），可以通过 region 将配置按区域完全隔离。

**核心规则：设置了 region 后，配置目录切换为 `configs/{region}/`，不再读取默认 `configs/` 下的文件。各 region 之间完全独立。**

### 目录结构

```
configs/                    ← 默认（不设 region 时使用）
├── config.yml
└── config.prod.yml
configs/sg/                 ← region=sg 时使用（完全独立）
├── config.yml
└── config.prod.yml
configs/us/                 ← region=us 时使用（完全独立）
├── config.yml
└── config.prod.yml
```

### 使用方式

```go
// 方式1：通过环境变量（生产部署推荐）
// 部署时设置环境变量 region=sg，SDK 自动切到 configs/sg/
cfg := kconfig.MustLoad[App]()

// 方式2：代码显式指定（本地调试）
cfg := kconfig.MustLoad[App](kconfig.WithRegion("sg"))

// 方式3：自定义环境变量名
cfg := kconfig.MustLoad[App](kconfig.WithRegionFrom("APP_REGION"))
```

### Region 来源优先级

1. `WithRegion("sg")` — 代码显式指定（最高）
2. 环境变量（默认读 `region`，可通过 `WithRegionFrom` 自定义）
3. `env.GetRegion()` — 全局状态兜底

### 注意事项

- 不设 region 时行为与之前完全一致，零影响
- 设了 region 后只读对应子目录，不继承默认 `configs/` 的任何配置
- 各 region 目录内的文件结构与默认目录一致（`config.yml` + `config.<env>.yml`）
