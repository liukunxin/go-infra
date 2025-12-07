package log

import "github.com/liukunxin/go-infra/pkg/log/core"

var logger *core.Logger

// Init 初始化日志
func Init(cfg Config) {
	var formatter core.Formatter
	if cfg.Formatter == "json" {
		formatter = &core.JSONFormatter{}
	} else {
		formatter = &core.TxtLineFormatter{}
	}
	provider := core.NewStdProvider()
	logger = core.NewLogger(cfg.Level, provider, formatter, cfg.BufferSize)
}

// Close 关闭日志（确保异步队列写完）
func Close() {
	if logger != nil {
		logger.Close()
	}
}
