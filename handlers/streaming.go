package handlers

import (
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	EditDebounceInterval = 2 * time.Second // 每2秒最多编辑一次
	MinEditThreshold     = 50              // 最少积累50个字符才编辑
)

// StreamingMessage 流式消息结构
type StreamingMessage struct {
	ChatID    int64
	MessageID int
	Content   string
	LastEdit  time.Time
	Mutex     sync.Mutex
}

// StreamingManager 流式输出管理器
type StreamingManager struct {
	messages map[string]*StreamingMessage
	mutex    sync.RWMutex
	throttle map[int64]time.Time
}

// NewStreamingManager 创建新的流式输出管理器
func NewStreamingManager() *StreamingManager {
	return &StreamingManager{
		messages: make(map[string]*StreamingMessage),
		throttle: make(map[int64]time.Time),
	}
}

// CreateStream 创建流式消息
func (m *StreamingManager) CreateStream(streamKey string, chatID int64, messageID int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.messages[streamKey] = &StreamingMessage{
		ChatID:    chatID,
		MessageID: messageID,
		Content:   "",
		LastEdit:  time.Now(),
	}
}

// UpdateStream 更新流式消息内容
func (m *StreamingManager) UpdateStream(bot *tgbotapi.BotAPI, streamKey string, newContent string, isComplete bool) {
	m.mutex.RLock()
	streaming, exists := m.messages[streamKey]
	m.mutex.RUnlock()

	if !exists {
		return
	}

	streaming.Mutex.Lock()
	defer streaming.Mutex.Unlock()

	// 检查是否应该编辑消息（智能防抖）
	contentDiff := len(newContent) - len(streaming.Content)
	timeSinceLastEdit := time.Since(streaming.LastEdit)

	// 智能防抖算法
	shouldEdit := isComplete ||
		(contentDiff >= MinEditThreshold && timeSinceLastEdit >= EditDebounceInterval) ||
		timeSinceLastEdit >= EditDebounceInterval*3 // 最长等待时间

	// 如果内容变化很大，立即更新
	if contentDiff >= MinEditThreshold*2 {
		shouldEdit = true
	}

	if shouldEdit {
		displayContent := newContent
		if !isComplete {
			displayContent += " ⌨️" // 添加打字指示器
		}

		editMsg := tgbotapi.NewEditMessageText(streaming.ChatID, streaming.MessageID, displayContent)

		// 尝试MarkdownV2格式
		editMsg.ParseMode = "MarkdownV2"
		if _, err := bot.Send(editMsg); err != nil {
			// 如果MarkdownV2格式失败，尝试Markdown格式
			editMsg.ParseMode = "Markdown"
			if _, err2 := bot.Send(editMsg); err2 != nil {
				// 如果Markdown也失败，使用普通文本
				editMsg.ParseMode = ""
				if _, err3 := bot.Send(editMsg); err3 != nil {
					// Log error but don't panic
				}
			}
		}

		streaming.Content = newContent
		streaming.LastEdit = time.Now()
	}
}

// DeleteStream 删除流式消息
func (m *StreamingManager) DeleteStream(streamKey string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.messages, streamKey)
}

// CanSendMessage 检查是否可以发送消息（防止spam）
func (m *StreamingManager) CanSendMessage(chatID int64) bool {
	m.mutex.RLock()
	lastTime, exists := m.throttle[chatID]
	m.mutex.RUnlock()

	if !exists || time.Since(lastTime) >= 1*time.Second {
		m.mutex.Lock()
		m.throttle[chatID] = time.Now()
		m.mutex.Unlock()
		return true
	}
	return false
}

// CleanupOldStreams 清理过期的流式消息
func (m *StreamingManager) CleanupOldStreams() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute) // 清理10分钟前的消息
	for key, streaming := range m.messages {
		if streaming.LastEdit.Before(cutoff) {
			delete(m.messages, key)
		}
	}
}
