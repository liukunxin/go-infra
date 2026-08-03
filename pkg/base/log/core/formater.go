package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Formatter serializes a log entry into a byte slice ready for output.
// ts is the time the Log() call was made (not the time of formatting).
type Formatter interface {
	Format(level int, ts time.Time, msg string, fields map[string]interface{}, traceId, spanId string) []byte
}

var formatBufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 256))
	},
}

func getFormatBuf() *bytes.Buffer {
	buf := formatBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putFormatBuf(buf *bytes.Buffer) {
	// Cap retained buffer size to avoid retaining huge buffers in the pool.
	if buf.Cap() > 64*1024 {
		return
	}
	formatBufPool.Put(buf)
}

// TxtLineFormatter produces human-readable single-line text output.
type TxtLineFormatter struct{}

func (f *TxtLineFormatter) Format(level int, ts time.Time, msg string, fields map[string]interface{}, traceId, spanId string) []byte {
	buf := getFormatBuf()
	buf.WriteString(ts.Format(time.RFC3339Nano))
	buf.WriteString(" [")
	buf.WriteString(LevelToString(level))
	buf.WriteString("] ")
	buf.WriteString(msg)
	if traceId != "" {
		buf.WriteString(" traceId=")
		buf.WriteString(traceId)
	}
	if spanId != "" {
		buf.WriteString(" spanId=")
		buf.WriteString(spanId)
	}
	for k, v := range fields {
		fmt.Fprintf(buf, " %s=%v", k, v)
	}
	buf.WriteByte('\n')
	out := append([]byte(nil), buf.Bytes()...)
	putFormatBuf(buf)
	return out
}

// JSONFormatter produces structured JSON output, one object per line.
type JSONFormatter struct{}

func (f *JSONFormatter) Format(level int, ts time.Time, msg string, fields map[string]interface{}, traceId, spanId string) []byte {
	data := mapPool.Get().(map[string]interface{})
	for k := range data {
		delete(data, k)
	}
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

	buf := getFormatBuf()
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(data) // Encode appends '\n'

	out := append([]byte(nil), buf.Bytes()...)
	putFormatBuf(buf)
	mapPool.Put(data)
	return out
}
