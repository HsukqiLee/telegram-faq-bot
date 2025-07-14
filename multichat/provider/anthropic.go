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

// AnthropicProvider Anthropic API提供商
type AnthropicProvider struct {
	APIKey  string
	APIURL  string
	Timeout time.Duration
	client  *http.Client
}

// AnthropicMessage Anthropic消息格式
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest Anthropic请求格式
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
}

// AnthropicResponse Anthropic响应格式
type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// NewAnthropicProvider 创建Anthropic提供商
func NewAnthropicProvider(apiKey, apiURL string, timeout time.Duration) *AnthropicProvider {
	return &AnthropicProvider{
		APIKey:  apiKey,
		APIURL:  apiURL,
		Timeout: timeout,
		client:  utils.GetEnhancedClientWithTimeout(timeout),
	}
}

// GetName 返回提供商名称
func (p *AnthropicProvider) GetName() string {
	return "Anthropic"
}

// GetModels 获取可用模型列表
func (p *AnthropicProvider) GetModels() ([]Model, error) {
	// Anthropic没有公开的模型列表API，返回预定义的模型
	models := []Model{
		{
			ID:          "claude-3-5-sonnet-20241022",
			Name:        "Claude 3.5 Sonnet",
			Provider:    "Anthropic",
			Description: "Anthropic's most intelligent model",
		},
		{
			ID:          "claude-3-5-haiku-20241022",
			Name:        "Claude 3.5 Haiku",
			Provider:    "Anthropic",
			Description: "Fast and affordable model",
		},
		{
			ID:          "claude-3-opus-20240229",
			Name:        "Claude 3 Opus",
			Provider:    "Anthropic",
			Description: "Anthropic's most powerful model",
		},
		{
			ID:          "claude-3-sonnet-20240229",
			Name:        "Claude 3 Sonnet",
			Provider:    "Anthropic",
			Description: "Balanced performance and speed",
		},
		{
			ID:          "claude-3-haiku-20240307",
			Name:        "Claude 3 Haiku",
			Provider:    "Anthropic",
			Description: "Fast and cost-effective model",
		},
	}
	return models, nil
}

// Chat 进行对话
func (p *AnthropicProvider) Chat(messages []Message, model string) (*ChatResponse, error) {
	// 转换消息格式
	anthropicMessages := make([]AnthropicMessage, 0, len(messages))
	for _, msg := range messages {
		// Anthropic不支持system role，需要转换
		if msg.Role == "system" {
			// 将system消息转换为user消息的前缀
			if len(anthropicMessages) > 0 && anthropicMessages[len(anthropicMessages)-1].Role == "user" {
				anthropicMessages[len(anthropicMessages)-1].Content = msg.Content + "\n\n" + anthropicMessages[len(anthropicMessages)-1].Content
			} else {
				anthropicMessages = append(anthropicMessages, AnthropicMessage{
					Role:    "user",
					Content: msg.Content,
				})
			}
		} else {
			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	requestBody, err := json.Marshal(AnthropicRequest{
		Model:     model,
		MaxTokens: 4096,
		Messages:  anthropicMessages,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", p.APIURL+"/messages", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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

	var chatResp AnthropicResponse
	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(chatResp.Content) == 0 {
		return nil, fmt.Errorf("no response from Anthropic")
	}

	// 提取文本内容
	var content strings.Builder
	for _, c := range chatResp.Content {
		if c.Type == "text" {
			content.WriteString(c.Text)
		}
	}

	return &ChatResponse{
		Content:      content.String(),
		InputTokens:  chatResp.Usage.InputTokens,
		OutputTokens: chatResp.Usage.OutputTokens,
		Model:        model,
		Provider:     "Anthropic",
	}, nil
}
