package log

import "github.com/liukunxin/go-infra/pkg/base/log/core"

// 日志级别常量，与 core.Level* 对齐，方便直接在 Config 中使用。
const (
	LevelDebug = core.LevelDebug // 0
	LevelInfo  = core.LevelInfo  // 1
	LevelWarn  = core.LevelWarn  // 2
	LevelError = core.LevelError // 3
	LevelFatal = core.LevelFatal // 4
)

// Config 配置
type Config struct {
	Level      int    `json:"level" toml:"level" yaml:"level"`                   // 日志级别，使用 log.LevelInfo 等常量
	Formatter  string `json:"formatter" toml:"formatter" yaml:"formatter"`       // "text" 或 "json"
	Output     string `yaml:"output" json:"output" toml:"output"`                // "stdout"（可扩展 file/kafka）
	BufferSize int    `json:"buffer_size" yaml:"buffer_size" toml:"buffer_size"` // 异步缓冲区大小，默认 1000
}
