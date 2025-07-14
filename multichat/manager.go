package multichat

import (
	"fmt"
	"log"
	"sync"
	"time"

	"TGFaqBot/config"
	"TGFaqBot/multichat/provider"
	"TGFaqBot/utils"
)

// Manager 多渠道聊天管理器
type Manager struct {
	config     *config.Config
	providers  map[string]provider.Provider
	mu         sync.RWMutex
	configFile string
}

// NewManager 创建新的多渠道管理器
func NewManager(cfg *config.Config, configFile string) *Manager {
	manager := &Manager{
		config:     cfg,
		providers:  make(map[string]provider.Provider),
		configFile: configFile,
	}

	manager.initializeProviders()
	manager.updateCachedModels()

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

	if m.config.Chat.CachedModels == nil {
		m.config.Chat.CachedModels = make(map[string][]config.Model)
	}

	for name, provider := range m.providers {
		log.Printf("Fetching models for provider: %s", name)
		models, err := provider.GetModels()
		if err != nil {
			log.Printf("Failed to fetch models for %s: %v", name, err)
			continue
		}

		// 转换为配置格式
		configModels := make([]config.Model, len(models))
		for i, model := range models {
			configModels[i] = config.Model{
				ID:       model.ID,
				Name:     model.Name,
				Provider: model.Provider,
			}
		}

		m.config.Chat.CachedModels[name] = configModels
		log.Printf("Cached %d models for %s", len(configModels), name)
	}

	// 保存更新后的配置
	if err := config.SaveConfig(m.configFile, m.config); err != nil {
		log.Printf("Failed to save updated config: %v", err)
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

// GetAvailableProviders 获取所有可用的提供商
func (m *Manager) GetAvailableProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}

// GetCachedModels 获取缓存的模型列表
func (m *Manager) GetCachedModels(providerName string) []config.Model {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config.Chat.CachedModels == nil {
		return nil
	}
	return m.config.Chat.CachedModels[providerName]
}

// GetAllCachedModels 获取所有缓存的模型
func (m *Manager) GetAllCachedModels() map[string][]config.Model {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config.Chat.CachedModels == nil {
		return make(map[string][]config.Model)
	}

	// 返回副本
	result := make(map[string][]config.Model)
	for k, v := range m.config.Chat.CachedModels {
		result[k] = v
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
