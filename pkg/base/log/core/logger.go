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

// mapPool 用于复用 map[string]interface{} 对象，减少内存分配
var mapPool = sync.Pool{
	New: func() interface{} {
		// 预分配一定容量，减少扩容
		return make(map[string]interface{}, 8)
	},
}

// ringSlot 插槽
type ringSlot struct {
	seq   uint64         // sequence number (for future扩展，not used heavily)
	entry unsafe.Pointer // *logEntry
	ready uint32         // 0/1 ready flag
}

// logEntry 表示一条日志
type logEntry struct {
	level   int
	msg     string
	traceId string // TraceID: 整个请求链路的唯一标识，用于关联日志
	spanId  string // SpanID: 可选，用于详细的分布式追踪（如Jaeger）
	ts      time.Time
}

// Logger 是核心 logger（MPSC ring buffer + single consumer）
type Logger struct {
	// 配置（可原子读取/写入，不加锁）
	level     int32
	formatter atomic.Pointer[Formatter]
	provider  atomic.Pointer[Provider]

	// 环形队列
	capMask uint64
	slots   []ringSlot
	tail    uint64 // 索引（生产者写入位置；原子增加）
	head    uint64 // 索引（消费位置；仅消费 goroutine 访问）

	// control
	done   chan struct{}
	closed uint32 // 0/1

	// 默认字段（只在构建 WihFields 的时候读写，用 mutex 不需要高频修改，这里简单用原子替换）
	// 使用指针指向 map，替换时写入新的指针（无锁读）
	defaultFields atomic.Pointer[map[string]interface{}]
}

// NewLogger 创建 logger。
// bufferSize 必须为 2 的幂；如果不是，内部会向上取到最近的幂。
func NewLogger(level int, provider Provider, formatter Formatter, bufferSize int) *Logger {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	// 向上取幂
	capacity := nextPowOfTwo(uint64(bufferSize))
	slots := make([]ringSlot, capacity)
	for i := range slots {
		slots[i].seq = uint64(i)
		slots[i].entry = nil
		slots[i].ready = 0
	}
	l := &Logger{
		level:   int32(level),
		capMask: capacity - 1,
		slots:   slots,
		tail:    0,
		head:    0,
		done:    make(chan struct{}),
	}
	// set provider/formatter atomically
	l.provider.Store(&provider)
	l.formatter.Store(&formatter)
	// default fields
	df := make(map[string]interface{})
	l.defaultFields.Store(&df)

	// start consumer goroutine
	go l.consumerLoop()
	return l
}

// nextPowOfTwo
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

// SetLevel 动态设置级别（原子）
func (l *Logger) SetLevel(level int) {
	atomic.StoreInt32(&l.level, int32(level))
}

// GetLevel
func (l *Logger) GetLevel() int {
	return int(atomic.LoadInt32(&l.level))
}

// SetFormatter
func (l *Logger) SetFormatter(f Formatter) {
	l.formatter.Store(&f)
}

// SetProvider
func (l *Logger) SetProvider(p Provider) {
	l.provider.Store(&p)
}

// WithFields 返回一个"子 logger"——实际上是共享同一 Logger，但将 defaultFields 原子替换为合并后的 map。
// 该操作使用原子交换指针方式，避免锁。返回的 *Logger 仍然是同一对象（fields pointer changed globally）。
// 注意：为了简单性，这里返回同一指针（不会复制底层队列）；如果你需要独立 defaultFields，请 copy 一个新的 Logger（可按需扩展）。
// 注意：此 map 会被长期持有，不能使用对象池（会导致内存泄漏）
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	// 读取当前 map
	ptr := l.defaultFields.Load()
	cur := *ptr
	
	// 创建新的 map（不使用对象池，因为这个map需要长期保留）
	merged := make(map[string]interface{}, len(cur)+len(fields))
	
	// 合并字段
	for k, v := range cur {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	l.defaultFields.Store(&merged)
	return l
}

// enqueue 尝试把 entry 放入环形队列，非阻塞。
// 返回 true 表示成功入队，false 表示队列已满（调用方应回退至同步写）。
func (l *Logger) enqueue(e *logEntry) bool {
	// fast check capacity
	for {
		t := atomic.AddUint64(&l.tail, 1) - 1 // claim a slot index (monotonic)
		head := atomic.LoadUint64(&l.head)
		if t-head >= uint64(len(l.slots)) {
			// full
			// rollback tail decrement (best effort)
			atomic.AddUint64(&l.tail, ^uint64(0)) // -1
			return false
		}
		idx := t & l.capMask
		slot := &l.slots[idx]

		// put entry ptr then set ready
		atomic.StorePointer(&slot.entry, unsafe.Pointer(e))
		atomic.StoreUint32(&slot.ready, 1)
		return true
	}
}

// consumerLoop 单消费者循环
func (l *Logger) consumerLoop() {
	for {
		h := l.head
		idx := h & l.capMask
		slot := &l.slots[idx]

		ready := atomic.LoadUint32(&slot.ready)
		if ready == 0 {
			// no data, check closed
			if atomic.LoadUint32(&l.closed) == 1 {
				// drain remaining items if any then exit
				// check tail
				t := atomic.LoadUint64(&l.tail)
				for l.head < t {
					// read again
					idx2 := l.head & l.capMask
					slot2 := &l.slots[idx2]
					for atomic.LoadUint32(&slot2.ready) == 0 {
						// spin a bit
						runtime.Gosched()
					}
					p := (*logEntry)(atomic.LoadPointer(&slot2.entry))
					l.writeOne(p)
					atomic.StorePointer(&slot2.entry, nil)
					atomic.StoreUint32(&slot2.ready, 0)
					l.head++
				}
				close(l.done)
				return
			}
			// yield CPU
			runtime.Gosched()
			continue
		}

		p := (*logEntry)(atomic.LoadPointer(&slot.entry))
		if p != nil {
			l.writeOne(p)
		}
		// clear slot
		atomic.StorePointer(&slot.entry, nil)
		atomic.StoreUint32(&slot.ready, 0)
		l.head++
	}
}

// writeOne 格式化并写一行（直接调用 provider）
func (l *Logger) writeOne(e *logEntry) {
	formatterPtr := l.formatter.Load()
	providerPtr := l.provider.Load()
	var f Formatter
	var p Provider
	if formatterPtr != nil {
		f = *formatterPtr
	}
	if providerPtr != nil {
		p = *providerPtr
	}
	if f == nil || p == nil {
		// fallback to stdout simple formatting
		line := fmt.Sprintf("%s %s %s\n", e.ts.Format(time.RFC3339Nano), LevelToString(e.level), e.msg)
		fmt.Print(line)
		return
	}
	// combine default fields + entry fields (使用对象池优化)
	dfPtr := l.defaultFields.Load()
	var combined map[string]interface{}
	if dfPtr != nil && len(*dfPtr) > 0 {
		combined = mapPool.Get().(map[string]interface{})
		// 清空 map
		for k := range combined {
			delete(combined, k)
		}
		for k, v := range *dfPtr {
			combined[k] = v
		}
	}
	b := f.Format(e.level, e.msg, combined, e.traceId, e.spanId)
	p.WriteLine(b)
	
	// 归还 map 到对象池
	if combined != nil {
		mapPool.Put(combined)
	}
}

// Log 将一条日志放入队列（若队列满则退化为同步写）
// traceId/spanId 可为空
func (l *Logger) Log(level int, msg string, traceId, spanId string) {
	curLevel := int(atomic.LoadInt32(&l.level))
	if level < curLevel {
		return
	}
	e := &logEntry{
		level:   level,
		msg:     msg,
		traceId: traceId,
		spanId:  spanId,
		ts:      time.Now(),
	}
	ok := l.enqueue(e)
	if !ok {
		// 回退到同步写（直接调用 formatter+provider）
		formatterPtr := l.formatter.Load()
		providerPtr := l.provider.Load()
		var f Formatter
		var p Provider
		if formatterPtr != nil {
			f = *formatterPtr
		}
		if providerPtr != nil {
			p = *providerPtr
		}
		if f == nil || p == nil {
			// fallback stdout
			fmt.Printf("%s %s %s\n", e.ts.Format(time.RFC3339Nano), LevelToString(e.level), e.msg)
			return
		}
		// merge default fields (使用对象池优化)
		dfPtr := l.defaultFields.Load()
		var combined map[string]interface{}
		if dfPtr != nil && len(*dfPtr) > 0 {
			combined = mapPool.Get().(map[string]interface{})
			// 清空 map
			for k := range combined {
				delete(combined, k)
			}
			for k, v := range *dfPtr {
				combined[k] = v
			}
		}
		b := f.Format(e.level, e.msg, combined, e.traceId, e.spanId)
		p.WriteLine(b)
		
		// 归还 map 到对象池
		if combined != nil {
			mapPool.Put(combined)
		}
	}
}

// Output 用于兼容 Output(callDepth, level, format, args...)
func (l *Logger) Output(callDepth int, level int, format string, args ...interface{}) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	// 获取 caller 信息
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
	l.Log(level, msg, "", "")
}

// OutputMsg 兼容无格式化参数
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
	l.Log(level, msg, "", "")
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

// Close 关闭 logger 并等待消费者处理完
func (l *Logger) Close() {
	if !atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		return
	}
	// spin until consumer closes done
	// consumer checks closed flag and will close l.done when finished
	for {
		// set closed flag (already set)
		// wait for done to be closed
		select {
		case <-l.done:
			return
		default:
			runtime.Gosched()
		}
	}
}
