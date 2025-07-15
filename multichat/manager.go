package multichat

import (
	"fmt"
	"log"
	"sync"
	"time"

	"TGFaqBot/config"
	"TGFaqBot/database"
	"TGFaqBot/multichat/provider"
	"TGFaqBot/utils"
)

// Manager 多渠道聊天管理器
type Manager struct {
	config       *config.Config
	providers    map[string]provider.Provider
	mu           sync.RWMutex
	configFile   string
	db           database.Database
	service      *MultiChatService
	conversation *ConversationManager
}

// NewManager 创建新的多渠道管理器
func NewManager(cfg *config.Config, configFile string, db database.Database) *Manager {
	manager := &Manager{
		config:     cfg,
		providers:  make(map[string]provider.Provider),
		configFile: configFile,
		db:         db,
	}

	manager.initializeProviders()
	manager.updateCachedModels()

	// 初始化Redis客户端（如果启用）
	var redisClient *database.RedisClient
	if cfg.Redis.Enabled {
		var err error
		redisClient, err = database.NewRedisClient(&cfg.Redis)
		if err != nil {
			log.Printf("Warning: Failed to initialize Redis client: %v", err)
		}
	}

	// 初始化对话系统
	manager.service = NewMultiChatService(&cfg.Chat, db)
	manager.conversation = NewConversationManager(manager.service, &cfg.Chat, redisClient)

	return manager
}

// initializeProviders 初始化所有启用的提供商
func (m *Manager) initializeProviders() {
	timeout := time.Duration(m.config.Chat.Timeout) * time.Second

	// OpenAI
	if m.config.Chat.OpenAI != nil && m.config.Chat.OpenAI.Enabled {
		providerTimeout := timeout
		if m.config.Chat.OpenAI.Timeout > 0 {
			providerTimeout = time.Duration(m.config.Chat.OpenAI.Timeout) * time.Second
		}
		m.providers["openai"] = provider.NewOpenAICompatibleProvider(
			"OpenAI",
			m.config.Chat.OpenAI.APIKey,
			m.config.Chat.OpenAI.APIURL,
			providerTimeout,
		)
		log.Printf("Initialized OpenAI provider")
	}

	// Anthropic
	if m.config.Chat.Anthropic != nil && m.config.Chat.Anthropic.Enabled {
		providerTimeout := timeout
		if m.config.Chat.Anthropic.Timeout > 0 {
			providerTimeout = time.Duration(m.config.Chat.Anthropic.Timeout) * time.Second
		}
		m.providers["anthropic"] = provider.NewAnthropicProvider(
			m.config.Chat.Anthropic.APIKey,
			m.config.Chat.Anthropic.APIURL,
			providerTimeout,
		)
		log.Printf("Initialized Anthropic provider")
	}

	// Gemini
	if m.config.Chat.Gemini != nil && m.config.Chat.Gemini.Enabled {
		providerTimeout := timeout
		if m.config.Chat.Gemini.Timeout > 0 {
			providerTimeout = time.Duration(m.config.Chat.Gemini.Timeout) * time.Second
		}
		m.providers["gemini"] = provider.NewGeminiProvider(
			m.config.Chat.Gemini.APIKey,
			m.config.Chat.Gemini.APIURL,
			providerTimeout,
		)
		log.Printf("Initialized Gemini provider")
	}

	// Ollama
	if m.config.Chat.Ollama != nil && m.config.Chat.Ollama.Enabled {
		providerTimeout := timeout
		if m.config.Chat.Ollama.Timeout > 0 {
			providerTimeout = time.Duration(m.config.Chat.Ollama.Timeout) * time.Second
		}
		m.providers["ollama"] = provider.NewOllamaProvider(
			m.config.Chat.Ollama.APIKey,
			m.config.Chat.Ollama.APIURL,
			providerTimeout,
		)
		log.Printf("Initialized Ollama provider")
	}
}

// HasEnabledProviders 检查是否有任何启用的AI提供商
func (m *Manager) HasEnabledProviders() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.providers) > 0
}

// updateCachedModels 更新缓存的模型列表
func (m *Manager) updateCachedModels() {
	m.mu.Lock()
	defer m.mu.Unlock()

	allSuccessful := true
	var allModels []config.Model

	for name, provider := range m.providers {
		log.Printf("Fetching models for provider: %s", name)
		models, err := provider.GetModels()
		if err != nil {
			log.Printf("Failed to fetch models for %s: %v", name, err)
			allSuccessful = false

			// 检查主模型表中是否有有效的缓存模型（1天内）作为备用
			cachedModels, dbErr := m.db.GetModels(name)
			if dbErr == nil && isCachedModelValid(cachedModels) {
				log.Printf("Using models from main table for %s (updated within 24 hours, %d models)", name, len(cachedModels))

				// 将缓存的模型添加到全局模型列表中
				for _, model := range cachedModels {
					allModels = append(allModels, config.Model{
						ID:       model.ID,
						Name:     model.Name,
						Provider: model.Provider,
					})
				}
				continue
			}

			// 如果没有有效缓存，使用默认模型作为fallback
			log.Printf("No valid cached models found, using default models for %s", name)
			defaultModels := getDefaultModels(name)
			if len(defaultModels) > 0 {
				if err := m.db.SaveModels(name, defaultModels); err != nil {
					log.Printf("Failed to save default models for %s: %v", name, err)
				} else {
					log.Printf("Cached %d default models for %s", len(defaultModels), name)
				}

				// 将默认模型添加到全局模型列表中
				for _, model := range defaultModels {
					allModels = append(allModels, config.Model{
						ID:       model.ID,
						Name:     model.Name,
						Provider: model.Provider,
					})
				}
			}
			continue
		}

		// 成功获取到模型，转换为配置格式
		for _, model := range models {
			allModels = append(allModels, config.Model{
				ID:       model.ID,
				Name:     model.Name,
				Provider: model.Provider,
			})
		}

		// 转换为数据库格式并保存到主模型表
		dbModels := make([]database.ModelInfo, len(models))
		for i, model := range models {
			dbModels[i] = database.ModelInfo{
				ID:          model.ID,
				Name:        model.Name,
				Provider:    model.Provider,
				Description: "",
				UpdatedAt:   time.Now().Format("2006-01-02 15:04:05"),
			}
		}

		// 保存到数据库主模型表
		if err := m.db.SaveModels(name, dbModels); err != nil {
			log.Printf("Failed to save models for %s: %v", name, err)
		} else {
			log.Printf("Successfully fetched and cached %d models for %s", len(dbModels), name)
		}
	}

	// 如果有部分provider失败，尝试使用全局缓存作为fallback
	if !allSuccessful {
		log.Printf("Some providers failed, checking global model cache for fallback")
		cachedModels, cacheTime, cacheErr := m.db.GetModelCache()
		if cacheErr == nil && len(cachedModels) > 0 && cacheTime != "" {
			// 检查缓存时间是否在24小时内
			cacheTimeParsed, parseErr := time.Parse("2006-01-02 15:04:05", cacheTime)
			if parseErr == nil && time.Since(cacheTimeParsed) < 24*time.Hour {
				log.Printf("Found valid global model cache (cached %v ago, %d models), using as fallback", time.Since(cacheTimeParsed), len(cachedModels))

				// 将缓存的模型按provider保存到主模型表
				providerModels := make(map[string][]database.ModelInfo)
				for _, model := range cachedModels {
					if _, exists := providerModels[model.Provider]; !exists {
						providerModels[model.Provider] = []database.ModelInfo{}
					}
					providerModels[model.Provider] = append(providerModels[model.Provider], database.ModelInfo{
						ID:          model.ID,
						Name:        model.Name,
						Provider:    model.Provider,
						Description: "",
						UpdatedAt:   cacheTime, // 保持原始缓存时间
					})
				}

				// 保存缓存的模型到各个provider的主表
				for provider, models := range providerModels {
					if saveErr := m.db.SaveModels(provider, models); saveErr != nil {
						log.Printf("Failed to save cached models to main table for %s: %v", provider, saveErr)
					} else {
						log.Printf("Restored %d cached models for %s from global cache", len(models), provider)
					}
				}
				return // 使用缓存成功，不再更新全局缓存
			} else {
				log.Printf("Global model cache is outdated (>24h), will not use as fallback")
			}
		} else {
			log.Printf("No valid global model cache found: %v", cacheErr)
		}
	}

	// 更新全局模型缓存（仅在成功获取新模型时）
	if allSuccessful && len(allModels) > 0 {
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		if err := m.db.SetModelCache(allModels, currentTime); err != nil {
			log.Printf("Failed to update global model cache: %v", err)
		} else {
			log.Printf("Updated global model cache with %d models", len(allModels))
		}
	}
}

// Chat 使用指定提供商进行对话
func (m *Manager) Chat(providerName string, messages []provider.Message, model string) (*provider.ChatResponse, error) {
	m.mu.RLock()
	p, exists := m.providers[providerName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not found or not enabled", providerName)
	}

	return p.Chat(messages, model)
}

// GetAvailableProviders 获取当前可用的提供商列表
func (m *Manager) GetAvailableProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var providers []string
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}

// IsProviderAvailable 检查指定提供商是否可用
func (m *Manager) IsProviderAvailable(provider string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.providers[provider]
	return exists
}

// GetCachedModels 获取缓存的模型列表（优先使用有效缓存，然后默认）
func (m *Manager) GetCachedModels(providerName string) []config.Model {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 首先尝试从数据库获取缓存的模型
	dbModels, err := m.db.GetModels(providerName)
	if err == nil && len(dbModels) > 0 {
		// 检查缓存是否仍然有效
		if isCachedModelValid(dbModels) {
			log.Printf("Using cached models for %s (%d models)", providerName, len(dbModels))
			// 转换为配置格式
			configModels := make([]config.Model, len(dbModels))
			for i, model := range dbModels {
				configModels[i] = config.Model{
					ID:       model.ID,
					Name:     model.Name,
					Provider: model.Provider,
				}
			}
			return configModels
		}
		log.Printf("Cached models for %s are outdated (>24h)", providerName)
	}

	// 如果没有有效缓存，使用默认模型
	defaultModels := getDefaultModels(providerName)
	if len(defaultModels) > 0 {
		log.Printf("Using default models for %s (%d models)", providerName, len(defaultModels))
		// 转换为配置格式
		configModels := make([]config.Model, len(defaultModels))
		for i, model := range defaultModels {
			configModels[i] = config.Model{
				ID:       model.ID,
				Name:     model.Name,
				Provider: model.Provider,
			}
		}
		return configModels
	}

	log.Printf("No models available for provider %s", providerName)
	return nil
}

// GetAllCachedModels 获取所有缓存的模型（优先使用有效缓存，然后默认）
func (m *Manager) GetAllCachedModels() map[string][]config.Model {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]config.Model)

	// 为每个提供商获取有效的模型列表
	for providerName := range m.providers {
		// 首先尝试从数据库获取缓存的模型
		dbModels, err := m.db.GetModels(providerName)
		if err == nil && len(dbModels) > 0 {
			// 检查缓存是否仍然有效
			if isCachedModelValid(dbModels) {
				log.Printf("Using cached models for %s (%d models)", providerName, len(dbModels))
				configModels := make([]config.Model, len(dbModels))
				for i, model := range dbModels {
					configModels[i] = config.Model{
						ID:       model.ID,
						Name:     model.Name,
						Provider: model.Provider,
					}
				}
				result[providerName] = configModels
				continue
			}
			log.Printf("Cached models for %s are outdated (>24h)", providerName)
		}

		// 如果没有有效缓存，使用默认模型
		defaultModels := getDefaultModels(providerName)
		if len(defaultModels) > 0 {
			log.Printf("Using default models for %s (%d models)", providerName, len(defaultModels))
			configModels := make([]config.Model, len(defaultModels))
			for i, model := range defaultModels {
				configModels[i] = config.Model{
					ID:       model.ID,
					Name:     model.Name,
					Provider: model.Provider,
				}
			}
			result[providerName] = configModels
		}
	}

	return result
}

// GetDefaultProvider 获取默认提供商（第一个启用的）
func (m *Manager) GetDefaultProvider() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 按优先级顺序返回第一个可用的提供商
	preferredOrder := []string{"openai", "anthropic", "gemini", "groq", "azure", "openrouter", "ollama"}

	for _, name := range preferredOrder {
		if _, exists := m.providers[name]; exists {
			return name
		}
	}

	// 如果没有找到优先的，返回第一个可用的
	for name := range m.providers {
		return name
	}

	return ""
}

// GetProviderConfig 获取提供商配置
func (m *Manager) GetProviderConfig(providerName string) *config.ProviderConfig {
	switch providerName {
	case "openai":
		return m.config.Chat.OpenAI
	case "anthropic":
		return m.config.Chat.Anthropic
	case "gemini":
		return m.config.Chat.Gemini
	case "ollama":
		return m.config.Chat.Ollama
	default:
		return nil
	}
}

// GetSystemPrompt 获取系统提示词（提供商特定或全局）
func (m *Manager) GetSystemPrompt(providerName string) string {
	providerConfig := m.GetProviderConfig(providerName)
	if providerConfig != nil && providerConfig.SystemPrompt != "" {
		return providerConfig.SystemPrompt
	}
	return m.config.Chat.SystemPrompt
}

// ShouldTriggerChat 检查是否应该触发聊天
func (m *Manager) ShouldTriggerChat(text string) bool {
	if m.config.Chat.Prefix == "" {
		return true // 没有前缀时默认触发
	}
	return len(text) >= len(m.config.Chat.Prefix) && text[:len(m.config.Chat.Prefix)] == m.config.Chat.Prefix
}

// RemovePrefix 移除聊天前缀
func (m *Manager) RemovePrefix(text string) string {
	if m.config.Chat.Prefix == "" {
		return text
	}
	if len(text) >= len(m.config.Chat.Prefix) && text[:len(m.config.Chat.Prefix)] == m.config.Chat.Prefix {
		return text[len(m.config.Chat.Prefix):]
	}
	return text
}

// FormatResponse 格式化响应
func (m *Manager) FormatResponse(response *provider.ChatResponse, duration time.Duration, totalInput, totalOutput int, remainingRounds, remainingMinutes, remainingSeconds int) string {
	return utils.FormatResponse(response.Content, response.InputTokens, response.OutputTokens, totalInput, totalOutput,
		duration, remainingRounds, remainingMinutes, remainingSeconds, fmt.Sprintf("%s/%s", response.Provider, response.Model))
}

// GetResponse 获取AI响应（通过ConversationManager）
func (m *Manager) GetResponse(chatID int64, userMessage string, preferredProvider string, preferredModel string) (string, int, int, time.Duration, int, bool, string, error) {
	return m.conversation.GetResponse(chatID, userMessage, preferredProvider, preferredModel)
}

// GetResponseWithCallback 获取AI响应，支持流式回调（通过ConversationManager）
func (m *Manager) GetResponseWithCallback(chatID int64, userMessage string, preferredProvider string, preferredModel string, callback func(string, bool) bool) (string, int, int, time.Duration, int, bool, string, error) {
	return m.conversation.GetResponseWithCallback(chatID, userMessage, preferredProvider, preferredModel, callback)
}

// ClearConversation 清除对话历史
func (m *Manager) ClearConversation(chatID int64) {
	systemPrompt := m.GetSystemPrompt("")
	m.conversation.ClearConversation(chatID, systemPrompt)
}

// GetTokenCounts 获取token计数
func (m *Manager) GetTokenCounts(chatID int64) (int, int) {
	return m.conversation.GetTokenCounts(chatID)
}

// GetRemainingTime 获取剩余时间
func (m *Manager) GetRemainingTime(chatID int64) (int, int) {
	return m.conversation.GetRemainingTime(chatID)
}

// GetProviderCount 获取可用提供商数量（用于诊断）
func (m *Manager) GetProviderCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, provider := range m.providers {
		if provider != nil {
			count++
		}
	}
	return count
}

// GetProviderNames 获取提供商名称列表（用于诊断）
func (m *Manager) GetProviderNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// GetDiagnosticInfo 获取详细诊断信息（用于管理员错误显示）
func (m *Manager) GetDiagnosticInfo() string {
	if m.service != nil {
		return m.service.GetDiagnosticInfo()
	}
	return "Service not initialized"
}

// GetConversationManager 获取对话管理器
func (m *Manager) GetConversationManager() *ConversationManager {
	return m.conversation
}

// isCachedModelValid 检查缓存的模型是否仍然有效（1天内）
func isCachedModelValid(models []database.ModelInfo) bool {
	if len(models) == 0 {
		return false
	}

	// 解析最新的模型更新时间
	latestUpdate, err := time.Parse("2006-01-02 15:04:05", models[0].UpdatedAt)
	if err != nil {
		log.Printf("Failed to parse model update time: %v", err)
		return false
	}

	// 检查所有模型的更新时间，使用最新的
	for _, model := range models {
		updateTime, err := time.Parse("2006-01-02 15:04:05", model.UpdatedAt)
		if err != nil {
			continue
		}
		if updateTime.After(latestUpdate) {
			latestUpdate = updateTime
		}
	}

	// 检查是否在1天内
	return time.Since(latestUpdate) < 24*time.Hour
}

// getDefaultModels 返回默认的模型列表（当API调用失败时使用）
func getDefaultModels(providerName string) []database.ModelInfo {
	now := time.Now().Format("2006-01-02 15:04:05")

	switch providerName {
	case "openai":
		return []database.ModelInfo{
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai", UpdatedAt: now},
			{ID: "gpt-4o", Name: "GPT-4o", Provider: "openai", UpdatedAt: now},
			{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "openai", UpdatedAt: now},
			{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: "openai", UpdatedAt: now},
		}
	case "anthropic":
		return []database.ModelInfo{
			{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "anthropic", UpdatedAt: now},
			{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", Provider: "anthropic", UpdatedAt: now},
			{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Provider: "anthropic", UpdatedAt: now},
		}
	case "gemini":
		return []database.ModelInfo{
			{ID: "gemini-pro", Name: "Gemini Pro", Provider: "gemini", UpdatedAt: now},
			{ID: "gemini-pro-vision", Name: "Gemini Pro Vision", Provider: "gemini", UpdatedAt: now},
		}
	case "ollama":
		return []database.ModelInfo{
			{ID: "llama2", Name: "Llama 2", Provider: "ollama", UpdatedAt: now},
		}
	default:
		return []database.ModelInfo{}
	}
}

// GetValidModels 获取有效的模型列表（优先使用缓存，然后默认）
// 注意：此方法不加锁，调用者需要自己处理锁
func (m *Manager) GetValidModels(providerName string) ([]database.ModelInfo, error) {
	// 首先尝试从数据库获取缓存的模型
	cachedModels, err := m.db.GetModels(providerName)
	if err == nil && len(cachedModels) > 0 {
		// 检查缓存是否仍然有效
		if isCachedModelValid(cachedModels) {
			log.Printf("Using cached models for %s (%d models)", providerName, len(cachedModels))
			return cachedModels, nil
		}
		log.Printf("Cached models for %s are outdated (>24h)", providerName)
	}

	// 如果没有有效缓存，返回默认模型
	defaultModels := getDefaultModels(providerName)
	if len(defaultModels) > 0 {
		log.Printf("Using default models for %s (%d models)", providerName, len(defaultModels))
		// 保存默认模型到数据库
		if saveErr := m.db.SaveModels(providerName, defaultModels); saveErr != nil {
			log.Printf("Failed to save default models for %s: %v", providerName, saveErr)
		}
		return defaultModels, nil
	}

	return nil, fmt.Errorf("no models available for provider %s", providerName)
}

// GetService 获取多渠道服务实例
func (m *Manager) GetService() *MultiChatService {
	return m.service
}
