package core

import (
	"encoding/json"
	"fmt"
	"time"
)

// Formatter 接口
type Formatter interface {
	Format(level int, msg string, fields map[string]interface{}, traceId, spanId string) []byte
}

// TxtLineFormatter 文本格式
type TxtLineFormatter struct{}

func (f *TxtLineFormatter) Format(level int, msg string, fields map[string]interface{}, traceId, spanId string) []byte {
	line := fmt.Sprintf("%s [%s] %s", time.Now().Format(time.RFC3339Nano), LevelToString(level), msg)
	if traceId != "" && spanId != "" {
		line += fmt.Sprintf(" traceId=%s spanId=%s", traceId, spanId)
	}
	for k, v := range fields {
		line += fmt.Sprintf(" %s=%v", k, v)
	}
	line += "\n"
	return []byte(line)
}

// JSONFormatter JSON 格式
type JSONFormatter struct{}

func (f *JSONFormatter) Format(level int, msg string, fields map[string]interface{}, traceId, spanId string) []byte {
	data := map[string]interface{}{
		"ts":    time.Now().Format(time.RFC3339Nano),
		"level": LevelToString(level),
		"msg":   msg,
	}
	if traceId != "" && spanId != "" {
		data["traceId"] = traceId
		data["spanId"] = spanId
	}
	for k, v := range fields {
		data[k] = v
	}
	b, _ := json.Marshal(data)
	b = append(b, '\n')
	return b
}
