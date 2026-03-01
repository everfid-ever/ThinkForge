package agent

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

const (
	defaultCacheTTL        = 5 * time.Minute // 缓存 TTL
	defaultMaxCacheSize    = 1000            // 最大缓存条目数
	MinCacheConfidence     = 0.5            // 最低缓存置信度阈值
)

// intentCacheEntry 缓存条目
type intentCacheEntry struct {
	intent    *RAGIntent
	expiresAt time.Time
}

// IntentCache 意图分类结果缓存（线程安全）
type IntentCache struct {
	mu      sync.RWMutex
	entries map[string]*intentCacheEntry
	ttl     time.Duration
	maxSize int
}

// NewIntentCache 创建缓存实例
func NewIntentCache(ttl time.Duration, maxSize int) *IntentCache {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	if maxSize <= 0 {
		maxSize = defaultMaxCacheSize
	}
	c := &IntentCache{
		entries: make(map[string]*intentCacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
	go c.startGC() // 后台定期清理过期条目
	return c
}

// cacheKey 生成缓存 key：MD5(convID + ":" + question)
// convID 为空时 key 中只用 question（跨会话共享，适用于无状态问题）
func (c *IntentCache) cacheKey(convID, question string) string {
	raw := convID + ":" + question
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

// Get 从缓存获取意图，未命中或已过期返回 nil, false
func (c *IntentCache) Get(convID, question string) (*RAGIntent, bool) {
	key := c.cacheKey(convID, question)
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}
	return entry.intent, true
}

// Set 写入缓存
func (c *IntentCache) Set(convID, question string, intent *RAGIntent) {
	key := c.cacheKey(convID, question)
	c.mu.Lock()
	defer c.mu.Unlock()
	// 超出 maxSize 时，简单丢弃（不写入），避免无限增长
	if len(c.entries) >= c.maxSize {
		return
	}
	c.entries[key] = &intentCacheEntry{
		intent:    intent,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate 使指定会话的所有缓存失效（会话结束时调用）
func (c *IntentCache) Invalidate(convID string) {
	// 由于 key 是 MD5，无法直接按前缀查找，此处保留接口（后续可扩展）
}

// Size 返回当前缓存条目数
func (c *IntentCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// startGC 后台定期清理过期条目
func (c *IntentCache) startGC() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		c.gc()
	}
}

func (c *IntentCache) gc() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.entries {
		if now.After(v.expiresAt) {
			delete(c.entries, k)
		}
	}
}
