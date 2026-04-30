package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testAppConfig struct {
	AppName string `yaml:"app_name" validate:"required"`
	Log     struct {
		Level int `yaml:"level"`
	} `yaml:"log"`
}

func (c *testAppConfig) Validate() error {
	if c.Log.Level < 0 {
		return os.ErrInvalid
	}
	return nil
}

func TestLoadMergesBaseAndEnv(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	baseDir := t.TempDir()

	writeTestFile(t, filepath.Join(baseDir, "config.yml"), `
app_name: demo
log:
  level: 1
`)
	writeTestFile(t, filepath.Join(baseDir, "config.prod.yml"), `
log:
  level: 3
`)

	cfg, err := Load[testAppConfig](
		WithBaseDir(baseDir),
		WithEnvFrom("APP_ENV"),
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AppName != "demo" {
		t.Fatalf("AppName = %q, want %q", cfg.AppName, "demo")
	}
	if cfg.Log.Level != 3 {
		t.Fatalf("Log.Level = %d, want %d", cfg.Log.Level, 3)
	}
}

func TestLoadStrictYAML(t *testing.T) {
	baseDir := t.TempDir()

	writeTestFile(t, filepath.Join(baseDir, "config.yml"), `
app_name: demo
unknown_key: bad
`)

	_, err := Load[testAppConfig](
		WithBaseDir(baseDir),
		WithEnv(EnvLocal),
	)
	if err == nil {
		t.Fatal("Load() error = nil, want strict yaml parse error")
	}
	if !strings.Contains(err.Error(), "unknown_key") {
		t.Fatalf("Load() error = %v, want contains unknown_key", err)
	}
}

func TestLoadValidate(t *testing.T) {
	baseDir := t.TempDir()

	writeTestFile(t, filepath.Join(baseDir, "config.yml"), `
log:
  level: 1
`)

	_, err := Load[testAppConfig](
		WithBaseDir(baseDir),
		WithEnv(EnvLocal),
		WithValidate(true),
		WithTagValidation(true),
	)
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "tag validation failed") {
		t.Fatalf("Load() error = %v, want tag validation failed", err)
	}
}

func TestLoadUnsupportedEnv(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "config.yml"), `
app_name: demo
`)

	_, err := Load[testAppConfig](
		WithBaseDir(baseDir),
		WithEnv("uat"),
	)
	if err == nil {
		t.Fatal("Load() error = nil, want unsupported env error")
	}
	if !strings.Contains(err.Error(), "unsupported env") {
		t.Fatalf("Load() error = %v, want contains unsupported env", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write file %q failed: %v", path, err)
	}
}
