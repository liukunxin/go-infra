package config

import (
	"fmt"
	"strings"

	"github.com/liukunxin/go-infra/internal/consts"
	"github.com/liukunxin/go-infra/internal/option"
)

type optionConfig struct {
	env            string
	envKey         string
	baseDir        string
	baseDirSet     bool
	fileName       string
	fileExt        string
	requireEnvFile bool
	validate       bool
	validateByTags bool
}

func defaultOptionConfig() *optionConfig {
	return &optionConfig{
		envKey:         consts.Env,
		baseDir:        "configs",
		fileName:       "config",
		fileExt:        ".yml",
		requireEnvFile: false,
		validate:       false,
		validateByTags: false,
	}
}

// Option 配置加载函数式选项。
type Option = option.Option[optionConfig]

// WithEnv 显式指定环境（local/test/gray/prod），优先级高于环境变量。
func WithEnv(environment string) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.env = environment
		return nil
	})
}

// WithEnvFrom 从指定环境变量读取环境名（为空则跳过）。
func WithEnvFrom(envKey string) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.envKey = strings.TrimSpace(envKey)
		return nil
	})
}

// WithBaseDir 设置配置目录，默认 configs。
func WithBaseDir(dir string) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return fmt.Errorf("config: base dir cannot be empty")
		}
		c.baseDir = dir
		c.baseDirSet = true
		return nil
	})
}

// WithFileName 设置配置文件名（不含扩展名），默认 config。
func WithFileName(name string) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("config: file name cannot be empty")
		}
		c.fileName = name
		return nil
	})
}

// WithFileExt 设置配置文件扩展名，默认 .yml。
func WithFileExt(ext string) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			return fmt.Errorf("config: file extension cannot be empty")
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		c.fileExt = ext
		return nil
	})
}

// WithRequireEnvFile 指定环境文件是否必须存在（默认 false）。
func WithRequireEnvFile(required bool) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.requireEnvFile = required
		return nil
	})
}

// WithValidate 开启配置校验（Validate 方法 + tag 校验，默认关闭）。
func WithValidate(enabled bool) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.validate = enabled
		return nil
	})
}

// WithTagValidation 启用 go-playground/validator 的 tag 校验（默认关闭）。
func WithTagValidation(enabled bool) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.validateByTags = enabled
		return nil
	})
}
