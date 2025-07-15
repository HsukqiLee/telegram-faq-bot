package multichat

import (
	"fmt"
	"log"
	"strings"
	"time"

	"TGFaqBot/config"
	"TGFaqBot/database"
	"TGFaqBot/multichat/provider"
)

// MultiChatService 多渠道聊天服务
type MultiChatService struct {
	config    *config.ChatConfig
	providers map[string]provider.Provider
	db        database.Database
}

// Provider 重新导出provider.Provider以保持兼容性
type Provider = provider.Provider

// Message 重新导出provider.Message以保持兼容性
type Message = provider.Message

// ChatResponse 重新导出provider.ChatResponse以保持兼容性
type ChatResponse = provider.ChatResponse

// NewMultiChatService 创建新的多渠道聊天服务
func NewMultiChatService(chatConfig *config.ChatConfig, db database.Database) *MultiChatService {
	service := &MultiChatService{
		config:    chatConfig,
		providers: make(map[string]Provider),
		db:        db,
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
func (s *MultiChatService) GetCompletion(messages []Message, preferredProvider string, preferredModel string) (string, int, int, string, error) {
	var errors []string

	// 如果同时指定了首选提供商和模型，先尝试使用指定的组合
	if preferredProvider != "" && preferredModel != "" {
		if provider, exists := s.providers[preferredProvider]; exists {
			// 尝试使用指定的模型
			response, err := provider.Chat(messages, preferredModel)
			if err == nil {
				return response.Content, response.InputTokens, response.OutputTokens, preferredProvider, nil
			}

			// 分类错误类型
			errorType := "Unknown"
			if strings.Contains(err.Error(), "failed to send request") {
				errorType = "Network"
			} else if strings.Contains(err.Error(), "API returned status") {
				errorType = "API"
			} else if strings.Contains(err.Error(), "failed to parse response") {
				errorType = "Parse"
			} else if strings.Contains(err.Error(), "API error") {
				errorType = "Service"
			}

			errorMsg := fmt.Sprintf("Provider %s (preferred model %s): [%s] %v", preferredProvider, preferredModel, errorType, err)
			log.Printf("%s", errorMsg)
			errors = append(errors, errorMsg)
		} else {
			errorMsg := fmt.Sprintf("Preferred provider %s not found", preferredProvider)
			log.Printf("%s", errorMsg)
			errors = append(errors, errorMsg)
		}
	}

	// 如果只指定了首选提供商，尝试使用默认模型
	if preferredProvider != "" && preferredModel == "" {
		if provider, exists := s.providers[preferredProvider]; exists {
			// 获取该提供商的默认模型
			model := s.GetDefaultModel(preferredProvider)
			response, err := provider.Chat(messages, model)
			if err == nil {
				return response.Content, response.InputTokens, response.OutputTokens, preferredProvider, nil
			}

			// 分类错误类型
			errorType := "Unknown"
			if strings.Contains(err.Error(), "failed to send request") {
				errorType = "Network"
			} else if strings.Contains(err.Error(), "API returned status") {
				errorType = "API"
			} else if strings.Contains(err.Error(), "failed to parse response") {
				errorType = "Parse"
			} else if strings.Contains(err.Error(), "API error") {
				errorType = "Service"
			}

			errorMsg := fmt.Sprintf("Provider %s (%s): [%s] %v", preferredProvider, model, errorType, err)
			log.Printf("%s", errorMsg)
			errors = append(errors, errorMsg)
		} else {
			errorMsg := fmt.Sprintf("Preferred provider %s not found", preferredProvider)
			log.Printf("%s", errorMsg)
			errors = append(errors, errorMsg)
		}
	}

	// 遍历所有可用提供商
	for name, provider := range s.providers {
		if name == preferredProvider {
			continue // 已经尝试过了
		}

		// 获取该提供商的默认模型
		model := s.GetDefaultModel(name)

		response, err := provider.Chat(messages, model)
		if err == nil {
			return response.Content, response.InputTokens, response.OutputTokens, name, nil
		}

		// 分类错误类型
		errorType := "Unknown"
		if strings.Contains(err.Error(), "failed to send request") {
			errorType = "Network"
		} else if strings.Contains(err.Error(), "API returned status") {
			errorType = "API"
		} else if strings.Contains(err.Error(), "failed to parse response") {
			errorType = "Parse"
		} else if strings.Contains(err.Error(), "API error") {
			errorType = "Service"
		}

		errorMsg := fmt.Sprintf("Provider %s (%s): [%s] %v", name, model, errorType, err)
		log.Printf("%s", errorMsg)
		errors = append(errors, errorMsg)
	}

	// 如果没有提供商，添加相应错误信息
	if len(s.providers) == 0 {
		errors = append(errors, "No providers available")
	}

	// 返回包含所有错误详情的错误
	detailedError := fmt.Sprintf("all providers failed: %s", strings.Join(errors, "; "))
	return "", 0, 0, "", fmt.Errorf("%s", detailedError)
}

// GetCompletionWithCallback 获取聊天完成，支持流式回调
func (s *MultiChatService) GetCompletionWithCallback(messages []Message, preferredProvider string, preferredModel string, callback func(string, bool) bool) (string, int, int, string, error) {
	var errors []string

	log.Printf("DEBUG: GetCompletionWithCallback called - preferredProvider: %s, preferredModel: %s", preferredProvider, preferredModel)

	// 如果同时指定了首选提供商和模型，先尝试使用指定的组合
	if preferredProvider != "" && preferredModel != "" {
		if providerInstance, exists := s.providers[preferredProvider]; exists {
			// 检查provider是否支持流式回调
			if streamProvider, ok := providerInstance.(provider.StreamingProvider); ok {
				log.Printf("DEBUG: Using streaming callback for %s with model %s", preferredProvider, preferredModel)
				// 尝试使用流式回调
				response, err := streamProvider.ChatWithCallback(messages, preferredModel, callback)
				if err == nil {
					log.Printf("DEBUG: Streaming callback successful for %s", preferredProvider)
					return response.Content, response.InputTokens, response.OutputTokens, preferredProvider, nil
				}

				errorMsg := fmt.Sprintf("Provider %s (streaming, preferred model %s): %v", preferredProvider, preferredModel, err)
				log.Printf("%s", errorMsg)
				errors = append(errors, errorMsg)
			} else {
				log.Printf("DEBUG: Provider %s does not support streaming callback, falling back to normal mode", preferredProvider)
				// 回退到普通方式
				response, err := providerInstance.Chat(messages, preferredModel)
				if err == nil {
					// 模拟流式回调
					if callback != nil {
						log.Printf("DEBUG: Simulating streaming callback for %s (fallback mode)", preferredProvider)
						callback(response.Content, true)
					}
					return response.Content, response.InputTokens, response.OutputTokens, preferredProvider, nil
				}

				errorMsg := fmt.Sprintf("Provider %s (fallback, preferred model %s): %v", preferredProvider, preferredModel, err)
				log.Printf("%s", errorMsg)
				errors = append(errors, errorMsg)
			}
		} else {
			errorMsg := fmt.Sprintf("Preferred provider %s not found", preferredProvider)
			log.Printf("%s", errorMsg)
			errors = append(errors, errorMsg)
		}
	}

	// 如果只指定了首选提供商，尝试使用默认模型
	if preferredProvider != "" && preferredModel == "" {
		if providerInstance, exists := s.providers[preferredProvider]; exists {
			model := s.GetDefaultModel(preferredProvider)

			// 检查provider是否支持流式回调
			if streamProvider, ok := providerInstance.(provider.StreamingProvider); ok {
				// 尝试使用流式回调
				response, err := streamProvider.ChatWithCallback(messages, model, callback)
				if err == nil {
					return response.Content, response.InputTokens, response.OutputTokens, preferredProvider, nil
				}

				errorMsg := fmt.Sprintf("Provider %s (streaming, %s): %v", preferredProvider, model, err)
				log.Printf("%s", errorMsg)
				errors = append(errors, errorMsg)
			} else {
				// 回退到普通方式
				response, err := providerInstance.Chat(messages, model)
				if err == nil {
					// 模拟流式回调
					if callback != nil {
						callback(response.Content, true)
					}
					return response.Content, response.InputTokens, response.OutputTokens, preferredProvider, nil
				}

				errorMsg := fmt.Sprintf("Provider %s (fallback, %s): %v", preferredProvider, model, err)
				log.Printf("%s", errorMsg)
				errors = append(errors, errorMsg)
			}
		} else {
			errorMsg := fmt.Sprintf("Preferred provider %s not found", preferredProvider)
			log.Printf("%s", errorMsg)
			errors = append(errors, errorMsg)
		}
	}

	// 遍历所有提供商，寻找可用的模型
	for providerName, providerInstance := range s.providers {
		// 跳过首选提供商，因为已经尝试过了
		if providerName == preferredProvider {
			continue
		}

		// 检查provider是否支持流式回调
		if streamProvider, ok := providerInstance.(provider.StreamingProvider); ok {
			// 如果指定了首选模型，先尝试使用首选模型
			if preferredModel != "" {
				response, err := streamProvider.ChatWithCallback(messages, preferredModel, callback)
				if err == nil {
					return response.Content, response.InputTokens, response.OutputTokens, providerName, nil
				}

				errorMsg := fmt.Sprintf("Provider %s (streaming, preferred model %s): %v", providerName, preferredModel, err)
				log.Printf("%s", errorMsg)
				errors = append(errors, errorMsg)
			}

			// 尝试使用默认模型
			model := s.GetDefaultModel(providerName)
			response, err := streamProvider.ChatWithCallback(messages, model, callback)
			if err == nil {
				return response.Content, response.InputTokens, response.OutputTokens, providerName, nil
			}

			errorMsg := fmt.Sprintf("Provider %s (streaming, %s): %v", providerName, model, err)
			log.Printf("%s", errorMsg)
			errors = append(errors, errorMsg)
		} else {
			// 如果指定了首选模型，先尝试使用首选模型
			if preferredModel != "" {
				response, err := providerInstance.Chat(messages, preferredModel)
				if err == nil {
					// 模拟流式回调
					if callback != nil {
						callback(response.Content, true)
					}
					return response.Content, response.InputTokens, response.OutputTokens, providerName, nil
				}

				errorMsg := fmt.Sprintf("Provider %s (fallback, preferred model %s): %v", providerName, preferredModel, err)
				log.Printf("%s", errorMsg)
				errors = append(errors, errorMsg)
			}

			// 尝试使用默认模型
			model := s.GetDefaultModel(providerName)
			response, err := providerInstance.Chat(messages, model)
			if err == nil {
				// 模拟流式回调
				if callback != nil {
					callback(response.Content, true)
				}
				return response.Content, response.InputTokens, response.OutputTokens, providerName, nil
			}

			errorMsg := fmt.Sprintf("Provider %s (fallback, %s): %v", providerName, model, err)
			log.Printf("%s", errorMsg)
			errors = append(errors, errorMsg)
		}
	}

	return "", 0, 0, "", fmt.Errorf("failed to get AI response: all providers failed: %s", strings.Join(errors, "; "))
}

// GetDefaultProviderAndModel 获取默认的提供商和模型
func (s *MultiChatService) GetDefaultProviderAndModel() (string, string) {
	// 获取第一个启用的提供商
	enabledProviders := s.config.GetEnabledProviders()
	for providerName := range enabledProviders {
		defaultModel := s.GetDefaultModel(providerName)
		return providerName, defaultModel
	}

	// 如果没有启用的提供商，返回默认值
	return "openai", "gpt-3.5-turbo"
}

// GetDefaultModel 获取指定提供商的默认模型
func (s *MultiChatService) GetDefaultModel(providerName string) string {
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

// GetDiagnosticInfo 获取诊断信息（用于管理员错误显示）
func (s *MultiChatService) GetDiagnosticInfo() string {
	var info []string

	info = append(info, fmt.Sprintf("Total providers configured: %d", len(s.providers)))

	for name := range s.providers {
		info = append(info, fmt.Sprintf("Provider %s: available", name))
	}

	return strings.Join(info, "; ")
}
