package log

// Config 配置
type Config struct {
	Level      int    `json:"level" toml:"level" yaml:"level"`
	Formatter  string `json:"formatter" toml:"formatter" yaml:"formatter"`       // "text" 或 "json"
	Output     string `yaml:"output" json:"output" toml:"output"`                // "stdout"（可扩展 file/kafka）
	BufferSize int    `json:"buffer_size" yaml:"buffer_size" toml:"buffer_size"` // 异步缓冲区大小，默认 1000
}
