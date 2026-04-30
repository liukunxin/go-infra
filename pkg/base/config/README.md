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
