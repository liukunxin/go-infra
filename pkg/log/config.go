package log

// Config 配置
type Config struct {
	Level      int
	Formatter  string // "text" 或 "json"
	Output     string // "stdout"（可扩展 file/kafka）
	BufferSize int    // 异步缓冲区大小，默认 1000
}
