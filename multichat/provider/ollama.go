package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"TGFaqBot/utils"
)

// OllamaProvider Ollama API提供商（使用Ollama原生格式）
type OllamaProvider struct {
	APIKey  string
	APIURL  string
	Timeout time.Duration
	client  *http.Client
}

// OllamaMessage Ollama原生消息格式
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaRequest Ollama原生请求格式
type OllamaRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// OllamaResponse Ollama原生响应格式
type OllamaResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

// OllamaModelsResponse Ollama模型列表响应
type OllamaModelsResponse struct {
	Models []struct {
		Name       string `json:"name"`
		ModifiedAt string `json:"modified_at"`
		Size       int64  `json:"size"`
	} `json:"models"`
}

// NewOllamaProvider 创建Ollama提供商
func NewOllamaProvider(apiKey, apiURL string, timeout time.Duration) *OllamaProvider {
	return &OllamaProvider{
		APIKey:  apiKey,
		APIURL:  apiURL,
		Timeout: timeout,
		client:  utils.GetEnhancedClientWithTimeout(timeout),
	}
}

// GetName 返回提供商名称
func (p *OllamaProvider) GetName() string {
	return "Ollama"
}

// GetModels 获取可用模型列表
func (p *OllamaProvider) GetModels() ([]Model, error) {
	req, err := http.NewRequest("GET", p.APIURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := utils.DoRequest(p.client, req)
	if err != nil {
		return nil, fmt.Errorf("error fetching models: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var modelsResp OllamaModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	models := make([]Model, 0, len(modelsResp.Models))
	for _, model := range modelsResp.Models {
		models = append(models, Model{
			ID:          model.Name,
			Name:        model.Name,
			Provider:    "Ollama",
			Description: fmt.Sprintf("Ollama model %s", model.Name),
		})
	}

	return models, nil
}

// Chat 进行对话
func (p *OllamaProvider) Chat(messages []Message, model string) (*ChatResponse, error) {
	// 转换消息格式为Ollama原生格式
	ollamaMessages := make([]OllamaMessage, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = OllamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	requestBody, err := json.Marshal(OllamaRequest{
		Model:    model,
		Messages: ollamaMessages,
		Stream:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", p.APIURL+"/api/chat", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := utils.DoRequest(p.client, req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var chatResp OllamaResponse
	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	if !chatResp.Done {
		return nil, fmt.Errorf("incomplete response from Ollama")
	}

	// Ollama原生API不提供token使用信息，使用估算
	inputTokens := EstimateTokens(messages)
	outputTokens := EstimateTokensFromText(chatResp.Message.Content)

	return &ChatResponse{
		Content:      chatResp.Message.Content,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        model,
		Provider:     "Ollama",
	}, nil
}

// ChatWithCallback 进行对话，支持流式回调
func (p *OllamaProvider) ChatWithCallback(messages []Message, model string, callback func(string, bool) bool) (*ChatResponse, error) {
	// Ollama目前不支持真正的流式响应，回退到普通方式
	response, err := p.Chat(messages, model)
	if err != nil {
		return nil, err
	}

	// 模拟流式回调
	if callback != nil {
		callback(response.Content, true)
	}

	return response, nil
}
