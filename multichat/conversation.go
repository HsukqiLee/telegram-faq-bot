package multichat

import (
	"log"
	"sync"
	"time"

	"TGFaqBot/config"
)

// ConversationManager 管理多渠道对话
type ConversationManager struct {
	conversations      map[int64]*Conversation
	conversationsMutex sync.RWMutex
	totalInputTokens   map[int64]int
	totalOutputTokens  map[int64]int
	multiChatService   *MultiChatService
	config             *config.ChatConfig
}

// Conversation 对话结构
type Conversation struct {
	History     []Message
	LastUpdated time.Time
	Provider    string // 当前使用的提供商
}

// NewConversationManager 创建对话管理器
func NewConversationManager(multiChatService *MultiChatService, chatConfig *config.ChatConfig) *ConversationManager {
	return &ConversationManager{
		conversations:     make(map[int64]*Conversation),
		totalInputTokens:  make(map[int64]int),
		totalOutputTokens: make(map[int64]int),
		multiChatService:  multiChatService,
		config:            chatConfig,
	}
}

// Init 初始化对话
func (cm *ConversationManager) Init(chatID int64, systemPrompt string) {
	cm.conversationsMutex.Lock()
	defer cm.conversationsMutex.Unlock()

	if _, exists := cm.conversations[chatID]; !exists {
		cm.conversations[chatID] = &Conversation{
			History:     []Message{},
			LastUpdated: time.Now(),
			Provider:    "", // 将自动选择
		}

		// 添加系统提示词
		if systemPrompt != "" {
			cm.conversations[chatID].History = append(
				cm.conversations[chatID].History,
				Message{Role: "system", Content: systemPrompt, Time: time.Now()},
			)
		}
	}
}

// GetResponse 获取AI响应
func (cm *ConversationManager) GetResponse(chatID int64, userMessage string, preferredProvider string) (string, int, int, time.Duration, int, bool, string, error) {
	start := time.Now()

	// 初始化对话
	systemPrompt := cm.config.SystemPrompt
	cm.Init(chatID, systemPrompt)

	cm.conversationsMutex.Lock()
	convo := cm.conversations[chatID]

	// 检查对话是否过期
	if time.Since(convo.LastUpdated).Minutes() > float64(cm.config.HistoryTimeoutMinutes) {
		// 重置对话
		convo.History = []Message{}
		convo.LastUpdated = time.Now()
		convo.Provider = ""

		// 恢复系统提示词
		if systemPrompt != "" {
			convo.History = append(convo.History, Message{
				Role:    "system",
				Content: systemPrompt,
				Time:    time.Now(),
			})
		}
	}

	// 添加用户消息
	convo.History = append(convo.History, Message{
		Role:    "user",
		Content: userMessage,
		Time:    time.Now(),
	})

	// 计算历史轮数
	historyCount := 0
	for _, msg := range convo.History {
		if msg.Role != "system" {
			historyCount++
		}
	}

	currentRound := (historyCount + 1) / 2
	remainingRounds := cm.config.HistoryLength - currentRound
	shouldReset := false

	if remainingRounds <= 0 {
		remainingRounds = 0
		shouldReset = true
	}

	// 管理历史记录长度
	if currentRound > cm.config.HistoryLength {
		var newHistory []Message

		// 保留系统提示词
		for _, msg := range convo.History {
			if msg.Role == "system" {
				newHistory = append(newHistory, msg)
			}
		}

		// 保留最近的消息
		messagesToKeep := cm.config.HistoryLength * 2
		if len(convo.History) > messagesToKeep {
			for i := len(convo.History) - messagesToKeep; i < len(convo.History); i++ {
				if i >= 0 && convo.History[i].Role != "system" {
					newHistory = append(newHistory, convo.History[i])
				}
			}
			convo.History = newHistory
		}
	}

	// 准备API请求的消息
	apiMessages := make([]Message, len(convo.History))
	copy(apiMessages, convo.History)
	cm.conversationsMutex.Unlock()

	// 使用首选提供商或之前使用的提供商
	if preferredProvider == "" {
		preferredProvider = convo.Provider
	}

	// 调用多渠道服务
	response, inputTokens, outputTokens, usedProvider, err := cm.multiChatService.GetCompletion(apiMessages, preferredProvider)
	if err != nil {
		return "", 0, 0, time.Since(start), remainingRounds, shouldReset, "", err
	}

	// 更新对话
	cm.conversationsMutex.Lock()
	convo.History = append(convo.History, Message{
		Role:    "assistant",
		Content: response,
		Time:    time.Now(),
	})
	convo.LastUpdated = time.Now()
	convo.Provider = usedProvider

	// 更新token计数
	if _, exists := cm.totalInputTokens[chatID]; !exists {
		cm.totalInputTokens[chatID] = 0
	}
	if _, exists := cm.totalOutputTokens[chatID]; !exists {
		cm.totalOutputTokens[chatID] = 0
	}
	cm.totalInputTokens[chatID] += inputTokens
	cm.totalOutputTokens[chatID] += outputTokens

	totalInputCount := cm.totalInputTokens[chatID]
	totalOutputCount := cm.totalOutputTokens[chatID]
	log.Printf("Chat ID %d: Request tokens: %d, Response tokens: %d, Total input: %d, Total output: %d, Rounds: %d/%d, Provider: %s",
		chatID, inputTokens, outputTokens, totalInputCount, totalOutputCount, currentRound, cm.config.HistoryLength, usedProvider)

	cm.conversationsMutex.Unlock()

	return response, inputTokens, outputTokens, time.Since(start), remainingRounds, shouldReset, usedProvider, nil
}

// ClearConversation 清除对话历史
func (cm *ConversationManager) ClearConversation(chatID int64, systemPrompt string) {
	cm.conversationsMutex.Lock()
	defer cm.conversationsMutex.Unlock()

	if convo, exists := cm.conversations[chatID]; exists {
		convo.History = []Message{}
		convo.LastUpdated = time.Now()
		convo.Provider = ""

		// 恢复系统提示词
		if systemPrompt != "" {
			convo.History = append(convo.History, Message{
				Role:    "system",
				Content: systemPrompt,
				Time:    time.Now(),
			})
		}

		// 重置token计数
		cm.totalInputTokens[chatID] = 0
		cm.totalOutputTokens[chatID] = 0
	}
}

// GetTokenCounts 获取token计数
func (cm *ConversationManager) GetTokenCounts(chatID int64) (int, int) {
	cm.conversationsMutex.RLock()
	defer cm.conversationsMutex.RUnlock()

	inputCount := cm.totalInputTokens[chatID]
	outputCount := cm.totalOutputTokens[chatID]
	return inputCount, outputCount
}

// GetRemainingTime 获取剩余时间
func (cm *ConversationManager) GetRemainingTime(chatID int64) (int, int) {
	cm.conversationsMutex.RLock()
	defer cm.conversationsMutex.RUnlock()

	if convo, exists := cm.conversations[chatID]; exists {
		elapsedSeconds := int(time.Since(convo.LastUpdated).Seconds())
		remainingSeconds := (cm.config.HistoryTimeoutMinutes * 60) - elapsedSeconds
		if remainingSeconds < 0 {
			remainingSeconds = 0
		}

		minutes := remainingSeconds / 60
		seconds := remainingSeconds % 60
		return minutes, seconds
	}

	return cm.config.HistoryTimeoutMinutes, 0
}
