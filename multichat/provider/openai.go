package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"TGFaqBot/utils"
)

// OpenAICompatibleProvider 兼容OpenAI API格式的提供商
// 支持：OpenAI、Groq、OpenRouter、Azure OpenAI等
type OpenAICompatibleProvider struct {
	name       string
	apiKey     string
	apiURL     string
	httpClient *http.Client
	models     []Model
}

// NewOpenAICompatibleProvider 创建OpenAI兼容格式的提供商
func NewOpenAICompatibleProvider(name, apiKey, apiURL string, timeout time.Duration) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		name:       name,
		apiKey:     apiKey,
		apiURL:     strings.TrimSuffix(apiURL, "/"),
		httpClient: utils.GetEnhancedClientWithTimeout(timeout),
	}
}

// GetName 返回提供商名称
func (p *OpenAICompatibleProvider) GetName() string {
	return p.name
}

// Chat 发送聊天请求
func (p *OpenAICompatibleProvider) Chat(messages []Message, model string) (*ChatResponse, error) {
	// 构建请求体
	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", p.apiURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// 发送请求
	resp, err := utils.DoRequest(p.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	response := &ChatResponse{
		Content:      chatResp.Choices[0].Message.Content,
		Provider:     p.name,
		Model:        model,
		InputTokens:  0,
		OutputTokens: 0,
	}

	if chatResp.Usage != nil {
		response.InputTokens = chatResp.Usage.PromptTokens
		response.OutputTokens = chatResp.Usage.CompletionTokens
	}

	return response, nil
}

// GetModels 获取可用模型列表
func (p *OpenAICompatibleProvider) GetModels() ([]Model, error) {
	req, err := http.NewRequest("GET", p.apiURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := utils.DoRequestWithCompression(p.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int    `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	var models []Model
	for _, m := range modelsResp.Data {
		models = append(models, Model{
			ID:       m.ID,
			Name:     m.ID,
			Provider: p.name,
		})
	}

	p.models = models
	return models, nil
}

// GetCachedModels 返回缓存的模型
func (p *OpenAICompatibleProvider) GetCachedModels() []Model {
	return p.models
}

// SetCachedModels 设置缓存的模型
func (p *OpenAICompatibleProvider) SetCachedModels(models []Model) {
	p.models = models
}
