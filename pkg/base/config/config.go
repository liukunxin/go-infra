package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/liukunxin/go-infra/pkg/base/env"
	"gopkg.in/yaml.v2"
)

const (
	EnvLocal = "local"
	EnvTest  = "test"
	EnvGray  = "gray"
	EnvProd  = "prod"
)

type validatable interface {
	Validate() error
}

// Load 按基础配置 + 环境配置（覆盖）顺序加载配置。
func Load[T any](opts ...Option) (*T, error) {
	c := defaultOptionConfig()
	for _, opt := range opts {
		if err := opt.Apply(c); err != nil {
			return nil, err
		}
	}

	resolvedEnv, err := normalizeEnv(c.resolveEnv())
	if err != nil {
		return nil, err
	}

	var out T
	basePath := filepath.Join(c.baseDir, c.fileName+c.fileExt)
	if err := decodeYAMLFile(basePath, &out, true); err != nil {
		return nil, err
	}

	envPath := filepath.Join(c.baseDir, fmt.Sprintf("%s.%s%s", c.fileName, resolvedEnv, c.fileExt))
	if err := decodeYAMLFile(envPath, &out, c.requireEnvFile); err != nil {
		return nil, err
	}

	if c.validate {
		if err := validateConfig(&out, c.validateByTags); err != nil {
			return nil, err
		}
	}

	return &out, nil
}

// MustLoad 在配置加载失败时 panic，适合在 main 初始化阶段使用。
func MustLoad[T any](opts ...Option) *T {
	cfg, err := Load[T](opts...)
	if err != nil {
		panic(err)
	}
	return cfg
}

func decodeYAMLFile(path string, out any, required bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !required {
			return nil
		}
		return fmt.Errorf("config: read %q failed: %w", path, err)
	}

	if err = yaml.UnmarshalStrict(data, out); err != nil {
		return fmt.Errorf("config: parse %q failed: %w", path, err)
	}
	return nil
}

func validateConfig[T any](cfg *T, byTags bool) error {
	if byTags {
		if err := validator.New().Struct(cfg); err != nil {
			return fmt.Errorf("config: tag validation failed: %w", err)
		}
	}

	if v, ok := any(cfg).(validatable); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("config: custom validation failed: %w", err)
		}
	}
	return nil
}

func normalizeEnv(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "", EnvLocal, "dev", "develop", "development":
		return EnvLocal, nil
	case EnvTest, "testing":
		return EnvTest, nil
	case EnvGray, "staging":
		return EnvGray, nil
	case EnvProd, "production", "release":
		return EnvProd, nil
	default:
		return "", fmt.Errorf("config: unsupported env %q", raw)
	}
}

func (c *optionConfig) resolveEnv() string {
	if c.env != "" {
		return c.env
	}
	if c.envKey != "" {
		if val := os.Getenv(c.envKey); val != "" {
			return val
		}
	}
	return env.GetEnv()
}
