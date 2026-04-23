package code

import (
	"context"
	"sync"
	"time"
)

type memEntry struct {
	code      string
	expiresAt time.Time
}

type memoryStore struct {
	mu      sync.RWMutex
	entries map[string]memEntry
}

// NewMemoryStore 创建基于本地内存的 CodeStore。
//
// 适用场景：单进程部署、验证码量级较小（日活万级以内）、不希望引入 Redis 依赖的情形。
//
// 使用限制：
//   - 进程重启后所有验证码立即失效。
//   - 多实例（水平扩展）部署时各节点内存相互独立：用户收到验证码的请求打到 A 节点，
//     验证请求若打到 B 节点则无法命中——此场景请改用 NewRedisStore。
//
// 过期策略：惰性过期（访问时检查 expiresAt）+ 后台定时 GC（默认每 2 分钟清理一次），
// 双重保障不持续泄露内存。
//
// 如需自定义 GC 间隔，使用 NewMemoryStoreWithCleanup。
func NewMemoryStore() CodeStore {
	return NewMemoryStoreWithCleanup(2 * time.Minute)
}

// NewMemoryStoreWithCleanup 同 NewMemoryStore，但可自定义后台 GC 间隔。
// cleanupInterval <= 0 时回退为 2 分钟。
// 注意：后台 goroutine 与 store 生命周期绑定，通常在应用全生命周期内存活，无需手动停止。
func NewMemoryStoreWithCleanup(cleanupInterval time.Duration) CodeStore {
	if cleanupInterval <= 0 {
		cleanupInterval = 2 * time.Minute
	}
	s := &memoryStore{
		entries: make(map[string]memEntry),
	}
	go s.cleanupLoop(cleanupInterval)
	return s
}

// cleanupLoop 后台定期扫描并删除已过期的条目，防止长期无访问时内存持续增长。
func (s *memoryStore) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.deleteExpired()
	}
}

func (s *memoryStore) deleteExpired() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, e := range s.entries {
		if now.After(e.expiresAt) {
			delete(s.entries, k)
		}
	}
}

func (s *memoryStore) Save(_ context.Context, key, code string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = memEntry{
		code:      code,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (s *memoryStore) Verify(_ context.Context, key, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		// 顺手删除过期条目，惰性 GC。
		delete(s.entries, key)
		return ErrNotFound
	}
	if e.code != code {
		return ErrMismatch
	}
	// 校验成功立即删除，验证码一次性使用。
	delete(s.entries, key)
	return nil
}

func (s *memoryStore) Exists(_ context.Context, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return false, nil
	}
	return true, nil
}
