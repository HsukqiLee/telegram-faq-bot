package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tg_markdown "github.com/zavitkov/tg-markdown"
)

const (
	EditDebounceInterval = 1 * time.Second // 每1秒最多编辑一次（更快响应）
	MinEditThreshold     = 20              // 最少积累20个字符才编辑（更少字符触发）
)

// ChatStats 聊天统计信息
type ChatStats struct {
	InputTokens     int           // 输入tokens
	OutputTokens    int           // 输出tokens
	RemainingRounds int           // 剩余对话轮数
	Duration        time.Duration // 请求耗时
	Provider        string        // 使用的提供商
	Model           string        // 使用的模型
	TTL             time.Duration // 对话历史TTL
	IsCachedReply   bool          // 是否为缓存回复
}

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
func (m *StreamingManager) UpdateStream(bot *tgbotapi.BotAPI, streamKey string, newContent string, isComplete bool, stats *ChatStats) {
	m.mutex.RLock()
	streaming, exists := m.messages[streamKey]
	m.mutex.RUnlock()

	if !exists {
		log.Printf("DEBUG: Stream key %s not found", streamKey)
		return
	}

	streaming.Mutex.Lock()
	defer streaming.Mutex.Unlock()

	// 检查是否应该编辑消息（智能防抖）
	contentDiff := len(newContent) - len(streaming.Content)
	timeSinceLastEdit := time.Since(streaming.LastEdit)

	log.Printf("DEBUG: UpdateStream - streamKey: %s, contentDiff: %d, timeSinceLastEdit: %v, isComplete: %v",
		streamKey, contentDiff, timeSinceLastEdit, isComplete)

	// 智能防抖算法
	shouldEdit := isComplete ||
		(contentDiff >= MinEditThreshold && timeSinceLastEdit >= EditDebounceInterval) ||
		timeSinceLastEdit >= EditDebounceInterval*3 // 最长等待时间

	// 如果内容变化很大，立即更新
	if contentDiff >= MinEditThreshold*2 {
		shouldEdit = true
	}

	log.Printf("DEBUG: UpdateStream - shouldEdit: %v (contentDiff: %d, threshold: %d, timeSince: %v, interval: %v)",
		shouldEdit, contentDiff, MinEditThreshold, timeSinceLastEdit, EditDebounceInterval)

	if shouldEdit {
		displayContent := newContent
		if !isComplete {
			displayContent += " ⌨️" // 添加打字指示器
		}

		// Convert standard Markdown to Telegram MarkdownV2
		convertedContent := tg_markdown.ConvertMarkdownToTelegramMarkdownV2(displayContent)

		editMsg := tgbotapi.NewEditMessageText(streaming.ChatID, streaming.MessageID, convertedContent)

		// 首先尝试MarkdownV2格式
		editMsg.ParseMode = "MarkdownV2"
		if _, err := bot.Send(editMsg); err != nil {
			// 如果MarkdownV2格式失败，尝试标准Markdown
			editMsg.ParseMode = "Markdown"
			editMsg.Text = displayContent
			if _, err2 := bot.Send(editMsg); err2 != nil {
				// 如果Markdown格式也失败，使用普通文本
				cleanText := cleanTextForPlain(displayContent)
				editMsg.Text = cleanText
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

// AppendStats 在流式响应完成后追加统计信息
func (m *StreamingManager) AppendStats(bot *tgbotapi.BotAPI, streamKey string, stats *ChatStats) {
	m.mutex.RLock()
	streaming, exists := m.messages[streamKey]
	m.mutex.RUnlock()

	if !exists {
		log.Printf("DEBUG: Stream key %s not found for stats append", streamKey)
		return
	}

	streaming.Mutex.Lock()
	defer streaming.Mutex.Unlock()

	// 如果是缓存回复，不添加统计信息，因为缓存回复已经包含了"缓存回复"标识
	if stats.IsCachedReply {
		log.Printf("DEBUG: Skipping stats for cached reply")
		return
	}

	// 构建统计信息文本
	statsText := fmt.Sprintf("\n\n---\n📊 **统计信息**\n• 输入 tokens: %d\n• 输出 tokens: %d\n• 剩余对话轮数: %d\n• 响应耗时: %v",
		stats.InputTokens, stats.OutputTokens, stats.RemainingRounds, stats.Duration)

	if stats.Model != "" {
		statsText += fmt.Sprintf("\n• 使用模型: %s", stats.Model)
	}

	if stats.TTL > 0 {
		// 计算TTL的小时和分钟
		hours := int(stats.TTL.Hours())
		minutes := int(stats.TTL.Minutes()) % 60
		if hours > 0 {
			statsText += fmt.Sprintf("\n• 对话超时: %d小时%d分钟后清除", hours, minutes)
		} else {
			statsText += fmt.Sprintf("\n• 对话超时: %d分钟后清除", minutes)
		}
	}

	// 追加统计信息到现有内容
	finalContent := streaming.Content + statsText

	// Convert standard Markdown to Telegram MarkdownV2
	convertedContent := tg_markdown.ConvertMarkdownToTelegramMarkdownV2(finalContent)

	editMsg := tgbotapi.NewEditMessageText(streaming.ChatID, streaming.MessageID, convertedContent)

	// 首先尝试MarkdownV2格式
	editMsg.ParseMode = "MarkdownV2"
	if _, err := bot.Send(editMsg); err != nil {
		// 如果MarkdownV2格式失败，尝试标准Markdown
		editMsg.ParseMode = "Markdown"
		editMsg.Text = finalContent
		if _, err2 := bot.Send(editMsg); err2 != nil {
			// 如果Markdown格式也失败，使用普通文本
			cleanText := cleanTextForPlain(finalContent)
			editMsg.Text = cleanText
			editMsg.ParseMode = ""
			if _, err3 := bot.Send(editMsg); err3 != nil {
				log.Printf("Failed to append stats to message: %v", err3)
			}
		}
	}

	// 更新存储的内容
	streaming.Content = finalContent
	streaming.LastEdit = time.Now()

	log.Printf("DEBUG: Appended stats to stream %s", streamKey)
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

// cleanTextForPlain 清理文本用于纯文本显示
func cleanTextForPlain(text string) string {
	// 移除控制字符和特殊字符
	reg := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	text = reg.ReplaceAllString(text, "")

	// 确保文本不超过Telegram限制
	if len(text) > 4096 {
		text = text[:4093] + "..."
	}

	return text
}

// CleanText 清理文本，去除多余空格和换行
func CleanText(text string) string {
	// 替换连续的空格和换行符
	re := regexp.MustCompile(`[\s]+`)
	cleaned := re.ReplaceAllString(text, " ")

	// 去除首尾空格
	return strings.TrimSpace(cleaned)
}
