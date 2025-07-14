package multichat

import (
	"fmt"
	"log"
	"time"

	"TGFaqBot/config"
	"TGFaqBot/multichat/provider"
)

// MultiChatService 多渠道聊天服务
type MultiChatService struct {
	config    *config.ChatConfig
	providers map[string]provider.Provider
}

// Provider 重新导出provider.Provider以保持兼容性
type Provider = provider.Provider

// Message 重新导出provider.Message以保持兼容性
type Message = provider.Message

// NewMultiChatService 创建新的多渠道聊天服务
func NewMultiChatService(chatConfig *config.ChatConfig) *MultiChatService {
	service := &MultiChatService{
		config:    chatConfig,
		providers: make(map[string]Provider),
	}

	// 初始化启用的提供商
	enabledProviders := chatConfig.GetEnabledProviders()

	for name, providerConfig := range enabledProviders {
		provider, err := service.createProvider(name, providerConfig)
		if err != nil {
			log.Printf("Failed to create provider %s: %v", name, err)
			continue
		}
		service.providers[name] = provider
		log.Printf("Initialized provider: %s", name)
	}

	return service
}

// createProvider 创建具体的提供商实例
func (s *MultiChatService) createProvider(name string, config *config.ProviderConfig) (Provider, error) {
	timeout := time.Duration(s.config.Timeout) * time.Second

	switch name {
	case "openai":
		return provider.NewOpenAICompatibleProvider("OpenAI", config.APIKey, config.APIURL, timeout), nil
	case "anthropic":
		return provider.NewAnthropicProvider(config.APIKey, config.APIURL, timeout), nil
	case "gemini":
		return provider.NewGeminiProvider(config.APIKey, config.APIURL, timeout), nil
	case "ollama":
		return provider.NewOllamaProvider(config.APIKey, config.APIURL, timeout), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// GetCompletion 获取聊天完成，自动选择第一个可用的提供商
func (s *MultiChatService) GetCompletion(messages []Message, preferredProvider string) (string, int, int, string, error) {
	// 如果指定了首选提供商，先尝试使用它
	if preferredProvider != "" {
		if provider, exists := s.providers[preferredProvider]; exists {
			// 获取该提供商的默认模型
			model := s.getDefaultModel(preferredProvider)

			response, err := provider.Chat(messages, model)
			if err == nil {
				return response.Content, response.InputTokens, response.OutputTokens, preferredProvider, nil
			}
			log.Printf("Preferred provider %s failed: %v", preferredProvider, err)
		}
	}

	// 遍历所有可用提供商
	for name, provider := range s.providers {
		if name == preferredProvider {
			continue // 已经尝试过了
		}

		// 获取该提供商的默认模型
		model := s.getDefaultModel(name)

		response, err := provider.Chat(messages, model)
		if err == nil {
			return response.Content, response.InputTokens, response.OutputTokens, name, nil
		}
		log.Printf("Provider %s failed: %v", name, err)
	}

	return "", 0, 0, "", fmt.Errorf("all providers failed")
}

// RefreshModels 刷新所有提供商的模型列表
func (s *MultiChatService) RefreshModels() error {
	var lastError error

	for name, provider := range s.providers {
		models, err := provider.GetModels()
		if err != nil {
			log.Printf("Failed to refresh models for %s: %v", name, err)
			lastError = err
			continue
		}

		// 更新缓存
		// TODO: Fix model type conversion
		// s.config.UpdateCachedModels(name, models)
		log.Printf("Refreshed %d models for %s", len(models), name)
	}

	return lastError
}

// GetCachedModels 获取所有缓存的模型
func (s *MultiChatService) GetCachedModels() map[string][]config.Model {
	if s.config.CachedModels == nil {
		return make(map[string][]config.Model)
	}
	return s.config.CachedModels
}

// GetProviders 获取所有提供商名称
func (s *MultiChatService) GetProviders() []string {
	var providers []string
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

// HasPrefix 检查消息是否匹配聊天前缀
func (s *MultiChatService) HasPrefix(message string) bool {
	if s.config.Prefix == "" {
		return true // 如果没有设置前缀，总是匹配
	}
	return len(message) > len(s.config.Prefix) && message[:len(s.config.Prefix)] == s.config.Prefix
}

// StripPrefix 移除消息前缀
func (s *MultiChatService) StripPrefix(message string) string {
	if s.config.Prefix == "" {
		return message
	}
	if len(message) > len(s.config.Prefix) && message[:len(s.config.Prefix)] == s.config.Prefix {
		return message[len(s.config.Prefix):]
	}
	return message
}

// getDefaultModel 获取指定提供商的默认模型
func (s *MultiChatService) getDefaultModel(providerName string) string {
	enabledProviders := s.config.GetEnabledProviders()
	if providerConfig, exists := enabledProviders[providerName]; exists {
		if providerConfig.DefaultModel != "" {
			return providerConfig.DefaultModel
		}
	}

	// 根据提供商提供默认值
	switch providerName {
	case "openai":
		return "gpt-3.5-turbo"
	case "anthropic":
		return "claude-3-haiku-20240307"
	case "gemini":
		return "gemini-pro"
	case "ollama":
		return "llama2"
	default:
		return "default"
	}
}
