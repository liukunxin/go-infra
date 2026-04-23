package core

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// mapPool reuses map[string]interface{} objects to reduce allocations in the hot write path.
var mapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]interface{}, 8)
	},
}

// entryPool reuses logEntry objects so that high-frequency logging doesn't pressure the GC.
var entryPool = sync.Pool{
	New: func() interface{} { return new(logEntry) },
}

// ringSlot is one cell in the ring buffer.
type ringSlot struct {
	entry unsafe.Pointer // *logEntry
	ready uint32         // 0 = empty, 1 = ready to consume
}

// logEntry represents a single log record.
type logEntry struct {
	level   int
	msg     string
	traceId string
	spanId  string
	ts      time.Time
	// fields carries per-call structured key-value pairs (e.g. from ContextLogger.WithFields).
	// The map must be a copy owned by the entry; callers must not share maps.
	fields map[string]interface{}
}

// Logger is the core logger implementation: MPSC lock-free ring buffer + single consumer.
type Logger struct {
	level     int32
	formatter atomic.Pointer[Formatter]
	provider  atomic.Pointer[Provider]

	capMask uint64
	slots   []ringSlot
	tail    uint64 // monotonically increasing claim counter (producers)
	head    uint64 // next slot to consume (consumer only)

	done   chan struct{}
	closed uint32 // CAS: 0 → 1 on Close

	// defaultFields are global fields set once (e.g. service name, version).
	// Stored as pointer-to-map for lock-free atomic swap.
	defaultFields atomic.Pointer[map[string]interface{}]
}

// NewLogger creates a Logger. bufferSize is rounded up to the next power of two.
func NewLogger(level int, provider Provider, formatter Formatter, bufferSize int) *Logger {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	capacity := nextPowOfTwo(uint64(bufferSize))
	l := &Logger{
		level:   int32(level),
		capMask: capacity - 1,
		slots:   make([]ringSlot, capacity),
		done:    make(chan struct{}),
	}
	l.provider.Store(&provider)
	l.formatter.Store(&formatter)
	df := make(map[string]interface{})
	l.defaultFields.Store(&df)
	go l.consumerLoop()
	return l
}

func nextPowOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	n--
	for i := uint(1); i < 64; i <<= 1 {
		n |= n >> i
	}
	return n + 1
}

// SetLevel dynamically changes the minimum log level.
func (l *Logger) SetLevel(level int) { atomic.StoreInt32(&l.level, int32(level)) }

// GetLevel returns the current minimum log level.
func (l *Logger) GetLevel() int { return int(atomic.LoadInt32(&l.level)) }

// SetFormatter atomically replaces the formatter.
func (l *Logger) SetFormatter(f Formatter) { l.formatter.Store(&f) }

// SetProvider atomically replaces the output provider.
func (l *Logger) SetProvider(p Provider) { l.provider.Store(&p) }

// WithFields merges fields into this logger's defaultFields and returns the SAME logger.
//
// WARNING: this mutates the shared logger state and affects all goroutines using it.
// It is intended only for one-time global enrichment (e.g. service name, pod name) during
// startup, NOT for per-request fields. For per-request fields use ContextLogger.WithFields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ptr := l.defaultFields.Load()
	cur := *ptr
	merged := make(map[string]interface{}, len(cur)+len(fields))
	for k, v := range cur {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	l.defaultFields.Store(&merged)
	return l
}

// Log enqueues a log record. If the ring buffer is full it falls back to a synchronous write.
// fields may be nil. The map must be a copy owned by the caller; it must not be modified after
// this call returns.
func (l *Logger) Log(level int, msg string, traceId, spanId string, fields map[string]interface{}) {
	if level < int(atomic.LoadInt32(&l.level)) {
		return
	}
	e := entryPool.Get().(*logEntry)
	e.level = level
	e.msg = msg
	e.traceId = traceId
	e.spanId = spanId
	e.ts = time.Now()
	e.fields = fields

	if !l.enqueue(e) {
		// Ring buffer full: write synchronously to avoid dropping logs.
		l.writeEntry(e)
	}
}

// Output formats msg with optional args and includes caller file/line information.
func (l *Logger) Output(callDepth int, level int, format string, args ...interface{}) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	if callDepth > 0 {
		if pc, file, line, ok := runtime.Caller(callDepth); ok {
			fn := runtime.FuncForPC(pc)
			caller := shortFile(file) + fmt.Sprintf(":%d", line)
			if fn != nil {
				caller = shortFunc(fn.Name()) + " " + caller
			}
			msg = caller + " " + msg
		}
	}
	l.Log(level, msg, "", "", nil)
}

// OutputMsg is like Output but skips format-string expansion.
func (l *Logger) OutputMsg(callDepth int, level int, msg string) {
	if callDepth > 0 {
		if pc, file, line, ok := runtime.Caller(callDepth); ok {
			fn := runtime.FuncForPC(pc)
			caller := shortFile(file) + fmt.Sprintf(":%d", line)
			if fn != nil {
				caller = shortFunc(fn.Name()) + " " + caller
			}
			msg = caller + " " + msg
		}
	}
	l.Log(level, msg, "", "", nil)
}

// Close signals the consumer to stop after draining all buffered entries and waits for it.
func (l *Logger) Close() {
	if !atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		return
	}
	<-l.done // park until consumer goroutine exits cleanly
}

// enqueue tries to place e into the ring buffer without blocking.
// Returns false when the buffer is full; the caller should fall back to a sync write.
func (l *Logger) enqueue(e *logEntry) bool {
	t := atomic.AddUint64(&l.tail, 1) - 1
	if t-atomic.LoadUint64(&l.head) >= uint64(len(l.slots)) {
		// Slot is beyond current capacity; undo the claim and report full.
		atomic.AddUint64(&l.tail, ^uint64(0))
		return false
	}
	slot := &l.slots[t&l.capMask]
	atomic.StorePointer(&slot.entry, unsafe.Pointer(e))
	atomic.StoreUint32(&slot.ready, 1)
	return true
}

// consumerLoop is the single consumer goroutine. It drains the ring buffer and
// writes entries to the configured provider.
func (l *Logger) consumerLoop() {
	for {
		slot := &l.slots[l.head&l.capMask]

		if atomic.LoadUint32(&slot.ready) == 0 {
			if atomic.LoadUint32(&l.closed) == 1 {
				l.drainRemaining()
				close(l.done)
				return
			}
			// Sleep briefly instead of busy-spinning to avoid wasting a CPU core
			// when the queue is idle. 1 µs adds negligible latency for async logs.
			time.Sleep(time.Microsecond)
			continue
		}

		p := (*logEntry)(atomic.LoadPointer(&slot.entry))
		atomic.StorePointer(&slot.entry, nil)
		atomic.StoreUint32(&slot.ready, 0)
		l.head++

		if p != nil {
			l.writeEntry(p)
		}
	}
}

// drainRemaining consumes all entries that were enqueued before Close was called.
func (l *Logger) drainRemaining() {
	t := atomic.LoadUint64(&l.tail)
	for l.head < t {
		slot := &l.slots[l.head&l.capMask]
		// Spin until the producer finishes writing this slot.
		for atomic.LoadUint32(&slot.ready) == 0 {
			runtime.Gosched()
		}
		p := (*logEntry)(atomic.LoadPointer(&slot.entry))
		atomic.StorePointer(&slot.entry, nil)
		atomic.StoreUint32(&slot.ready, 0)
		l.head++
		if p != nil {
			l.writeEntry(p)
		}
	}
}

// writeEntry formats and outputs a single log entry, then returns the entry to the pool.
func (l *Logger) writeEntry(e *logEntry) {
	fmtPtr := l.formatter.Load()
	pvdPtr := l.provider.Load()

	var f Formatter
	var p Provider
	if fmtPtr != nil {
		f = *fmtPtr
	}
	if pvdPtr != nil {
		p = *pvdPtr
	}

	if f == nil || p == nil {
		fmt.Printf("%s %s %s\n", e.ts.Format(time.RFC3339Nano), LevelToString(e.level), e.msg)
		releaseEntry(e)
		return
	}

	// Merge defaultFields and per-entry fields into a single map for the formatter.
	// Uses the pool to avoid an allocation on every log call.
	dfPtr := l.defaultFields.Load()
	hasDefault := dfPtr != nil && len(*dfPtr) > 0
	hasEntry := len(e.fields) > 0

	var combined map[string]interface{}
	if hasDefault || hasEntry {
		combined = mapPool.Get().(map[string]interface{})
		for k := range combined {
			delete(combined, k)
		}
		if hasDefault {
			for k, v := range *dfPtr {
				combined[k] = v
			}
		}
		if hasEntry {
			for k, v := range e.fields {
				combined[k] = v
			}
		}
	}

	b := f.Format(e.level, e.ts, e.msg, combined, e.traceId, e.spanId)
	p.WriteLine(b)

	if combined != nil {
		mapPool.Put(combined)
	}

	releaseEntry(e)
}

// releaseEntry clears and returns a logEntry to the pool.
func releaseEntry(e *logEntry) {
	e.msg = ""
	e.traceId = ""
	e.spanId = ""
	e.fields = nil
	entryPool.Put(e)
}

func shortFile(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}

func shortFunc(name string) string {
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		return name[idx+1:]
	}
	return name
}
