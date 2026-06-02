package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// Formatter serializes a log entry into a byte slice ready for output.
// ts is the time the Log() call was made (not the time of formatting).
type Formatter interface {
	Format(level int, ts time.Time, msg string, fields map[string]interface{}, traceId, spanId string) []byte
}

// TxtLineFormatter produces human-readable single-line text output.
type TxtLineFormatter struct{}

func (f *TxtLineFormatter) Format(level int, ts time.Time, msg string, fields map[string]interface{}, traceId, spanId string) []byte {
	var b bytes.Buffer
	b.WriteString(ts.Format(time.RFC3339Nano))
	b.WriteString(" [")
	b.WriteString(LevelToString(level))
	b.WriteString("] ")
	b.WriteString(msg)
	if traceId != "" {
		b.WriteString(" traceId=")
		b.WriteString(traceId)
	}
	if spanId != "" {
		b.WriteString(" spanId=")
		b.WriteString(spanId)
	}
	for k, v := range fields {
		fmt.Fprintf(&b, " %s=%v", k, v)
	}
	b.WriteByte('\n')
	return b.Bytes()
}

// JSONFormatter produces structured JSON output, one object per line.
type JSONFormatter struct{}

func (f *JSONFormatter) Format(level int, ts time.Time, msg string, fields map[string]interface{}, traceId, spanId string) []byte {
	data := make(map[string]interface{}, 4+len(fields))
	data["ts"] = ts.Format(time.RFC3339Nano)
	data["level"] = LevelToString(level)
	data["msg"] = msg
	if traceId != "" {
		data["traceId"] = traceId
	}
	if spanId != "" {
		data["spanId"] = spanId
	}
	for k, v := range fields {
		data[k] = v
	}
	b, _ := json.Marshal(data)
	return append(b, '\n')
}
