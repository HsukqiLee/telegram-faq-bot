package utils

import (
	"context"
	"sync"
	"time"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	rates map[int64]*userRate
	mu    sync.RWMutex
}

type userRate struct {
	requests  int
	lastReset time.Time
	blocked   bool
}

// NewRateLimiter 创建新的速率限制器
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		rates: make(map[int64]*userRate),
	}
}

// Allow 检查用户是否允许发送请求
func (rl *RateLimiter) Allow(userID int64, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rate, exists := rl.rates[userID]

	if !exists {
		rl.rates[userID] = &userRate{
			requests:  1,
			lastReset: now,
			blocked:   false,
		}
		return true
	}

	// 重置窗口
	if now.Sub(rate.lastReset) >= window {
		rate.requests = 1
		rate.lastReset = now
		rate.blocked = false
		return true
	}

	// 检查是否超过限制
	if rate.requests >= limit {
		rate.blocked = true
		return false
	}

	rate.requests++
	return true
}

// IsBlocked 检查用户是否被阻止
func (rl *RateLimiter) IsBlocked(userID int64) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rate, exists := rl.rates[userID]; exists {
		return rate.blocked
	}
	return false
}

// Cleanup 清理过期的记录
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for userID, rate := range rl.rates {
		if now.Sub(rate.lastReset) > maxAge {
			delete(rl.rates, userID)
		}
	}
}

// Cache 简单的内存缓存
type Cache struct {
	items map[string]*cacheItem
	mu    sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expires    time.Time
	hits       int64
	lastAccess time.Time
}

// NewCache 创建新的缓存
func NewCache() *Cache {
	cache := &Cache{
		items: make(map[string]*cacheItem),
	}

	// 启动清理协程
	go cache.cleanup()

	return cache
}

// Set 设置缓存项
func (c *Cache) Set(key string, value interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:      value,
		expires:    time.Now().Add(duration),
		hits:       0,
		lastAccess: time.Now(),
	}
}

// Get 获取缓存项
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.expires) {
		return nil, false
	}

	item.hits++
	item.lastAccess = time.Now()

	return item.value, true
}

// Delete 删除缓存项
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// cleanup 清理过期项
func (c *Cache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expires) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// ContextManager 上下文管理器
type ContextManager struct {
	contexts map[string]context.CancelFunc
	mu       sync.RWMutex
}

// NewContextManager 创建新的上下文管理器
func NewContextManager() *ContextManager {
	return &ContextManager{
		contexts: make(map[string]context.CancelFunc),
	}
}

// Create 创建新的上下文
func (cm *ContextManager) Create(key string, timeout time.Duration) context.Context {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 取消已存在的上下文
	if cancel, exists := cm.contexts[key]; exists {
		cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	cm.contexts[key] = cancel

	return ctx
}

// Cancel 取消指定的上下文
func (cm *ContextManager) Cancel(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cancel, exists := cm.contexts[key]; exists {
		cancel()
		delete(cm.contexts, key)
	}
}

// CancelAll 取消所有上下文
func (cm *ContextManager) CancelAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for key, cancel := range cm.contexts {
		cancel()
		delete(cm.contexts, key)
	}
}
