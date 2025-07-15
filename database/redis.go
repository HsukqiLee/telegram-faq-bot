package database

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"TGFaqBot/config"
)

// RedisClient Redis客户端包装器
type RedisClient struct {
	client         *redis.Client
	ttl            time.Duration
	aiCacheEnabled bool          // 是否启用AI缓存
	aiCacheTTL     time.Duration // AI缓存过期时间
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(conf *config.RedisConfig) (*RedisClient, error) {
	if !conf.Enabled {
		return nil, nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Password: conf.Password,
		DB:       conf.Database,
	})

	// 测试连接
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	ttl := time.Duration(conf.TTL) * time.Second
	if ttl == 0 {
		ttl = 30 * time.Minute // 默认30分钟
	}

	aiCacheTTL := time.Duration(conf.AICacheTTL) * time.Second
	if aiCacheTTL == 0 {
		aiCacheTTL = 1 * time.Hour // 默认1小时
	}

	log.Printf("Redis client connected successfully (TTL: %v, AI Cache: %v, AI Cache TTL: %v)",
		ttl, conf.AICacheEnabled, aiCacheTTL)

	return &RedisClient{
		client:         rdb,
		ttl:            ttl,
		aiCacheEnabled: conf.AICacheEnabled,
		aiCacheTTL:     aiCacheTTL,
	}, nil
}

// SetConversation 存储对话数据
func (r *RedisClient) SetConversation(chatID int64, data interface{}) error {
	if r == nil || r.client == nil {
		return nil // Redis未启用
	}

	ctx := context.Background()
	key := fmt.Sprintf("conversation:%d", chatID)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, jsonData, r.ttl).Err()
}

// GetConversation 获取对话数据
func (r *RedisClient) GetConversation(chatID int64, result interface{}) error {
	if r == nil || r.client == nil {
		return redis.Nil // Redis未启用
	}

	ctx := context.Background()
	key := fmt.Sprintf("conversation:%d", chatID)

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), result)
}

// DeleteConversation 删除对话数据
func (r *RedisClient) DeleteConversation(chatID int64) error {
	if r == nil || r.client == nil {
		return nil // Redis未启用
	}

	ctx := context.Background()
	key := fmt.Sprintf("conversation:%d", chatID)

	return r.client.Del(ctx, key).Err()
}

// SetLastMessage 存储最后一条用户消息
func (r *RedisClient) SetLastMessage(chatID int64, message string) error {
	if r == nil || r.client == nil {
		return nil // Redis未启用
	}

	ctx := context.Background()
	key := fmt.Sprintf("last_message:%d", chatID)

	return r.client.Set(ctx, key, message, r.ttl).Err()
}

// GetLastMessage 获取最后一条用户消息
func (r *RedisClient) GetLastMessage(chatID int64) (string, error) {
	if r == nil || r.client == nil {
		return "", redis.Nil // Redis未启用
	}

	ctx := context.Background()
	key := fmt.Sprintf("last_message:%d", chatID)

	return r.client.Get(ctx, key).Result()
}

// Close 关闭Redis连接
func (r *RedisClient) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

// IsEnabled 检查Redis是否启用
func (r *RedisClient) IsEnabled() bool {
	return r != nil && r.client != nil
}

// AI缓存相关方法

// IsAICacheEnabled 检查AI缓存是否启用
func (r *RedisClient) IsAICacheEnabled() bool {
	return r != nil && r.client != nil && r.aiCacheEnabled
}

// SetAICache 存储AI回复缓存
// 缓存键格式: ai_cache:{provider}:{model}:{question_hash}
func (r *RedisClient) SetAICache(provider, model, question, response string) error {
	if !r.IsAICacheEnabled() {
		return nil // AI缓存未启用
	}

	ctx := context.Background()
	key := r.generateAICacheKey(provider, model, question)

	return r.client.Set(ctx, key, response, r.aiCacheTTL).Err()
}

// GetAICache 获取AI回复缓存
func (r *RedisClient) GetAICache(provider, model, question string) (string, error) {
	if !r.IsAICacheEnabled() {
		return "", redis.Nil // AI缓存未启用
	}

	ctx := context.Background()
	key := r.generateAICacheKey(provider, model, question)

	return r.client.Get(ctx, key).Result()
}

// generateAICacheKey 生成AI缓存键
func (r *RedisClient) generateAICacheKey(provider, model, question string) string {
	// 使用SHA-256哈希来生成固定长度的问题标识
	hash := sha256.Sum256([]byte(question))
	questionHash := fmt.Sprintf("%x", hash)
	return fmt.Sprintf("ai_cache:%s:%s:%s", provider, model, questionHash)
}
