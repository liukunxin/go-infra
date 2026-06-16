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
// 设置了 region 时，配置目录切换为 {baseDir}/{region}/，各 region 完全隔离。
func Load[T any](opts ...Option) (*T, error) {
	c := defaultOptionConfig()
	for _, opt := range opts {
		if err := opt.Apply(c); err != nil {
			return nil, err
		}
	}
	baseDir := c.resolveBaseDir()

	if region := c.resolveRegion(); region != "" {
		baseDir = filepath.Join(baseDir, region)
	}

	resolvedEnv, err := normalizeEnv(c.resolveEnv())
	if err != nil {
		return nil, err
	}

	var out T
	basePath := filepath.Join(baseDir, c.fileName+c.fileExt)
	if err := decodeYAMLFile(basePath, &out, true, c.decryptor); err != nil {
		return nil, err
	}

	envPath := filepath.Join(baseDir, fmt.Sprintf("%s.%s%s", c.fileName, resolvedEnv, c.fileExt))
	if err := decodeYAMLFile(envPath, &out, c.requireEnvFile, c.decryptor); err != nil {
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

func decodeYAMLFile(path string, out any, required bool, decryptor Decryptor) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !required {
			return nil
		}
		return fmt.Errorf("config: read %q failed: %w", path, err)
	}

	if decryptor != nil {
		data, err = decryptor.DecryptBytes(data)
		if err != nil {
			return fmt.Errorf("config: decrypt %q failed: %w", path, err)
		}
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

func (c *optionConfig) resolveRegion() string {
	if c.region != "" {
		return c.region
	}
	if c.regionKey != "" {
		if val := os.Getenv(c.regionKey); val != "" {
			return strings.ToLower(strings.TrimSpace(val))
		}
	}
	return env.GetRegion()
}

func (c *optionConfig) resolveBaseDir() string {
	// 显式传入 WithBaseDir 时，严格使用调用方目录，不做回退。
	if c.baseDirSet {
		return c.baseDir
	}
	if isExistingDir(c.baseDir) {
		return c.baseDir
	}
	// 兼容历史目录结构，避免老项目升级时直接启动失败。
	legacy := "infra/config"
	if isExistingDir(legacy) {
		return legacy
	}
	return c.baseDir
}

func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
