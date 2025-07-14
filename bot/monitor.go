package bot

import (
	"log"
	"sync/atomic"
	"time"

	"TGFaqBot/handlers"
)

// Metrics 性能指标
type Metrics struct {
	ActiveRequests  int64
	TotalRequests   int64
	AverageResponse time.Duration
	LastCleanup     time.Time
}

// Monitor 性能监控器
type Monitor struct {
	metrics  *Metrics
	streamer *handlers.StreamingManager
}

// NewMonitor 创建新的性能监控器
func NewMonitor(streamer *handlers.StreamingManager) *Monitor {
	return &Monitor{
		metrics: &Metrics{
			LastCleanup: time.Now(),
		},
		streamer: streamer,
	}
}

// IncrementActiveRequests 增加活跃请求数
func (m *Monitor) IncrementActiveRequests() {
	atomic.AddInt64(&m.metrics.ActiveRequests, 1)
	atomic.AddInt64(&m.metrics.TotalRequests, 1)
}

// DecrementActiveRequests 减少活跃请求数
func (m *Monitor) DecrementActiveRequests() {
	atomic.AddInt64(&m.metrics.ActiveRequests, -1)
}

// GetActiveRequests 获取当前活跃请求数
func (m *Monitor) GetActiveRequests() int64 {
	return atomic.LoadInt64(&m.metrics.ActiveRequests)
}

// GetTotalRequests 获取总请求数
func (m *Monitor) GetTotalRequests() int64 {
	return atomic.LoadInt64(&m.metrics.TotalRequests)
}

// StartOptimizations 启动性能优化服务
func (m *Monitor) StartOptimizations() {
	// 每5分钟清理一次过期的流式消息
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			m.streamer.CleanupOldStreams()
		}
	}()

	// 记录性能指标
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			active := m.GetActiveRequests()
			total := m.GetTotalRequests()
			if active > 0 {
				log.Printf("Performance: Active requests: %d, Total requests: %d", active, total)
			}
		}
	}()
}
