package multichat

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"TGFaqBot/config"
	"TGFaqBot/database"
)

// ConversationManager 管理多渠道对话
type ConversationManager struct {
	conversations      map[int64]*Conversation
	conversationsMutex sync.RWMutex
	totalInputTokens   map[int64]int
	totalOutputTokens  map[int64]int
	multiChatService   *MultiChatService
	config             *config.ChatConfig
	redisClient        *database.RedisClient
}

// Conversation 对话结构
type Conversation struct {
	History       []Message `json:"history"`
	LastUpdated   time.Time `json:"last_updated"`
	Provider      string    `json:"provider"`        // 当前使用的提供商
	LastUserInput string    `json:"last_user_input"` // 最后一条用户输入，用于/retry
}

// NewConversationManager 创建对话管理器
func NewConversationManager(multiChatService *MultiChatService, chatConfig *config.ChatConfig, redisClient *database.RedisClient) *ConversationManager {
	return &ConversationManager{
		conversations:     make(map[int64]*Conversation),
		totalInputTokens:  make(map[int64]int),
		totalOutputTokens: make(map[int64]int),
		multiChatService:  multiChatService,
		config:            chatConfig,
		redisClient:       redisClient,
	}
}

// Init 初始化对话
func (cm *ConversationManager) Init(chatID int64, systemPrompt string) {
	cm.conversationsMutex.Lock()
	defer cm.conversationsMutex.Unlock()

	if _, exists := cm.conversations[chatID]; !exists {
		// 先尝试从Redis加载
		if redisConvo := cm.loadConversationFromRedis(chatID); redisConvo != nil {
			cm.conversations[chatID] = redisConvo
			return
		}

		// 创建新对话
		cm.conversations[chatID] = &Conversation{
			History:       []Message{},
			LastUpdated:   time.Now(),
			Provider:      "", // 将自动选择
			LastUserInput: "",
		}

		// 添加系统提示词
		if systemPrompt != "" {
			cm.conversations[chatID].History = append(
				cm.conversations[chatID].History,
				Message{Role: "system", Content: systemPrompt, Time: time.Now()},
			)
		}

		// 保存到Redis
		cm.saveConversationToRedis(chatID, cm.conversations[chatID])
	}
}

// loadConversationFromRedis 从Redis加载对话
func (cm *ConversationManager) loadConversationFromRedis(chatID int64) *Conversation {
	if cm.redisClient == nil || !cm.redisClient.IsEnabled() {
		return nil
	}

	var conversation Conversation
	err := cm.redisClient.GetConversation(chatID, &conversation)
	if err != nil {
		if err != redis.Nil {
			log.Printf("Error loading conversation from Redis: %v", err)
		}
		return nil
	}

	return &conversation
}

// saveConversationToRedis 保存对话到Redis
func (cm *ConversationManager) saveConversationToRedis(chatID int64, conversation *Conversation) {
	if cm.redisClient == nil || !cm.redisClient.IsEnabled() {
		return
	}

	err := cm.redisClient.SetConversation(chatID, conversation)
	if err != nil {
		log.Printf("Error saving conversation to Redis: %v", err)
	}
}

// GetLastUserInput 获取最后一条用户输入
func (cm *ConversationManager) GetLastUserInput(chatID int64) string {
	cm.conversationsMutex.RLock()
	defer cm.conversationsMutex.RUnlock()

	// 先尝试从内存获取
	if convo, exists := cm.conversations[chatID]; exists {
		return convo.LastUserInput
	}

	// 从Redis获取
	if redisConvo := cm.loadConversationFromRedis(chatID); redisConvo != nil {
		return redisConvo.LastUserInput
	}

	return ""
}

// GetResponse 获取AI响应
func (cm *ConversationManager) GetResponse(chatID int64, userMessage string, preferredProvider string, preferredModel string) (string, int, int, time.Duration, int, bool, string, error) {
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

	// 保存最后的用户输入，用于/retry功能
	convo.LastUserInput = userMessage

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
	response, inputTokens, outputTokens, usedProvider, err := cm.multiChatService.GetCompletion(apiMessages, preferredProvider, preferredModel)
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

	// 保存对话到Redis
	cm.saveConversationToRedis(chatID, convo)

	cm.conversationsMutex.Unlock()

	return response, inputTokens, outputTokens, time.Since(start), remainingRounds, shouldReset, usedProvider, nil
}

// GetResponseWithCallback 获取AI响应，支持流式回调
func (cm *ConversationManager) GetResponseWithCallback(chatID int64, userMessage string, preferredProvider string, preferredModel string, callback func(string, bool) bool) (string, int, int, time.Duration, int, bool, string, error) {
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

	// 保存最后的用户输入，用于/retry功能
	convo.LastUserInput = userMessage

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

	// 尝试从AI缓存获取回复
	if cm.redisClient != nil && cm.redisClient.IsAICacheEnabled() {
		// 确定实际使用的提供商和模型
		actualProvider := preferredProvider
		actualModel := preferredModel
		
		// 如果没有指定，从服务获取默认值
		if actualProvider == "" || actualModel == "" {
			defaultProvider, defaultModel := cm.multiChatService.GetDefaultProviderAndModel()
			if actualProvider == "" {
				actualProvider = defaultProvider
			}
			if actualModel == "" {
				actualModel = defaultModel
			}
		}

		// 尝试获取缓存
		cachedResponse, err := cm.redisClient.GetAICache(actualProvider, actualModel, userMessage)
		if err == nil && cachedResponse != "" {
			log.Printf("AI cache hit for chat %d: %s/%s", chatID, actualProvider, actualModel)
			
			// 发送缓存回复，添加缓存标识
			responseWithCacheNote := cachedResponse + "\n\n💾 缓存回复"
			
			// 调用回调函数发送完整响应
			if callback != nil {
				callback(responseWithCacheNote, true)
			}
			
			// 对于缓存回复，不添加到对话历史，不消耗轮数，返回特殊标识
			cm.conversationsMutex.Unlock()
			
			// 返回缓存标识，tokens为0，不影响轮数
			return responseWithCacheNote, 0, 0, time.Since(start), remainingRounds + 1, false, actualProvider, nil
		}
	}

	// 调用多渠道服务的流式方法
	response, inputTokens, outputTokens, usedProvider, err := cm.multiChatService.GetCompletionWithCallback(apiMessages, preferredProvider, preferredModel, callback)
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

	// 保存对话到Redis
	cm.saveConversationToRedis(chatID, convo)

	// 将AI响应存储到缓存
	if cm.redisClient != nil && cm.redisClient.IsAICacheEnabled() {
		// 获取实际使用的模型
		actualModel := preferredModel
		if actualModel == "" {
			actualModel = cm.multiChatService.GetDefaultModel(usedProvider)
		}
		
		// 存储到AI缓存
		if cacheErr := cm.redisClient.SetAICache(usedProvider, actualModel, userMessage, response); cacheErr != nil {
			log.Printf("Failed to cache AI response for chat %d: %v", chatID, cacheErr)
		} else {
			log.Printf("AI response cached for chat %d: %s/%s", chatID, usedProvider, actualModel)
		}
	}

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
		convo.LastUserInput = "" // 清除最后的用户输入

		// 恢复系统提示词
		if systemPrompt != "" {
			convo.History = append(convo.History, Message{
				Role:    "system",
				Content: systemPrompt,
				Time:    time.Now(),
			})
		}

		// 保存清空后的对话到Redis
		cm.saveConversationToRedis(chatID, convo)
	} else {
		// 如果内存中没有对话，也要清除Redis中的记录
		if cm.redisClient != nil && cm.redisClient.IsEnabled() {
			cm.redisClient.DeleteConversation(chatID)
		}
	}

	// 从Redis中删除对话数据
	if cm.redisClient != nil && cm.redisClient.IsEnabled() {
		cm.redisClient.DeleteConversation(chatID)
	}

	// 重置token计数
	delete(cm.totalInputTokens, chatID)
	delete(cm.totalOutputTokens, chatID)
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

// RetryLastMessage 重试最后一条用户消息
func (cm *ConversationManager) RetryLastMessage(chatID int64, preferredProvider string, preferredModel string) (string, int, int, time.Duration, int, bool, string, error) {
	lastInput := cm.GetLastUserInput(chatID)
	if lastInput == "" {
		return "", 0, 0, 0, 0, false, "", fmt.Errorf("没有找到可重试的消息")
	}

	// 移除最后的AI回复（如果存在）
	cm.conversationsMutex.Lock()
	if convo, exists := cm.conversations[chatID]; exists {
		// 从历史记录中移除最后的assistant回复
		if len(convo.History) > 0 && convo.History[len(convo.History)-1].Role == "assistant" {
			convo.History = convo.History[:len(convo.History)-1]
		}
		// 也移除最后的user消息，因为GetResponse会重新添加
		if len(convo.History) > 0 && convo.History[len(convo.History)-1].Role == "user" {
			convo.History = convo.History[:len(convo.History)-1]
		}
	}
	cm.conversationsMutex.Unlock()

	// 使用保存的用户输入重新获取响应
	return cm.GetResponse(chatID, lastInput, preferredProvider, preferredModel)
}

// RetryLastMessageWithCallback 重试最后一条用户消息（带回调）
func (cm *ConversationManager) RetryLastMessageWithCallback(chatID int64, preferredProvider string, preferredModel string, callback func(string, bool) bool) (string, int, int, time.Duration, int, bool, string, error) {
	lastInput := cm.GetLastUserInput(chatID)
	if lastInput == "" {
		return "", 0, 0, 0, 0, false, "", fmt.Errorf("没有找到可重试的消息")
	}

	// 移除最后的AI回复（如果存在）
	cm.conversationsMutex.Lock()
	if convo, exists := cm.conversations[chatID]; exists {
		// 从历史记录中移除最后的assistant回复
		if len(convo.History) > 0 && convo.History[len(convo.History)-1].Role == "assistant" {
			convo.History = convo.History[:len(convo.History)-1]
		}
		// 也移除最后的user消息，因为GetResponseWithCallback会重新添加
		if len(convo.History) > 0 && convo.History[len(convo.History)-1].Role == "user" {
			convo.History = convo.History[:len(convo.History)-1]
		}
	}
	cm.conversationsMutex.Unlock()

	// 使用保存的用户输入重新获取响应
	return cm.GetResponseWithCallback(chatID, lastInput, preferredProvider, preferredModel, callback)
}
