package log

import (
	"fmt"
	"sync/atomic"

	"github.com/liukunxin/go-infra/pkg/base/log/core"
)

// loggerPtr stores the active logger. atomic.Pointer provides type-safe, race-free
// access without needing unsafe.Pointer (available since Go 1.19).
var loggerPtr atomic.Pointer[core.Logger]

func loadLogger() *core.Logger {
	return loggerPtr.Load()
}

func storeLogger(l *core.Logger) {
	loggerPtr.Store(l)
}

// newFormatter selects a Formatter based on cfg.Formatter ("json" → JSONFormatter, else TxtLineFormatter).
func newFormatter(cfg Config) core.Formatter {
	if cfg.Formatter == "json" {
		return &core.JSONFormatter{}
	}
	return &core.TxtLineFormatter{}
}

// Init initializes the global logger writing to stdout. Returns an error so the
// caller can decide whether to fatal, panic, or fall back — rather than silently
// dropping all logs.
func Init(cfg Config) error {
	if cfg.Output == "file" {
		return fmt.Errorf("log: file output requires a path; use InitWithFile")
	}
	l := core.NewLogger(cfg.Level, core.NewStdProvider(), newFormatter(cfg), cfg.BufferSize)
	storeLogger(l)
	return nil
}

// InitWithFile initializes the global logger writing to a rotating file at path.
// Rotation defaults: 100 MB per file, 30 backups, 30-day retention.
// For custom rotation settings construct core.NewFileProviderWithOptions directly.
func InitWithFile(cfg Config, path string) error {
	provider, err := core.NewFileProvider(path)
	if err != nil {
		return fmt.Errorf("log: open file %s: %w", path, err)
	}
	l := core.NewLogger(cfg.Level, provider, newFormatter(cfg), cfg.BufferSize)
	storeLogger(l)
	return nil
}

// Close flushes and stops the logger. Safe to call even if Init was never called.
func Close() {
	l := loadLogger()
	if l != nil {
		l.Close()
	}
}
