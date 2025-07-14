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

// GeminiProvider Google Gemini API提供商
type GeminiProvider struct {
	APIKey  string
	APIURL  string
	Timeout time.Duration
	client  *http.Client
}

// GeminiContent Gemini内容格式
type GeminiContent struct {
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
	Role string `json:"role"`
}

// GeminiRequest Gemini请求格式
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

// GeminiResponse Gemini响应格式
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// NewGeminiProvider 创建Gemini提供商
func NewGeminiProvider(apiKey, apiURL string, timeout time.Duration) *GeminiProvider {
	return &GeminiProvider{
		APIKey:  apiKey,
		APIURL:  apiURL,
		Timeout: timeout,
		client:  utils.GetEnhancedClientWithTimeout(timeout),
	}
}

// GetName 返回提供商名称
func (p *GeminiProvider) GetName() string {
	return "Gemini"
}

// GetModels 获取可用模型列表
func (p *GeminiProvider) GetModels() ([]Model, error) {
	// Gemini API模型列表
	models := []Model{
		{
			ID:          "gemini-1.5-pro",
			Name:        "Gemini 1.5 Pro",
			Provider:    "Gemini",
			Description: "Most capable model with multimodal input",
		},
		{
			ID:          "gemini-1.5-flash",
			Name:        "Gemini 1.5 Flash",
			Provider:    "Gemini",
			Description: "Fast and versatile model for diverse tasks",
		},
		{
			ID:          "gemini-pro",
			Name:        "Gemini Pro",
			Provider:    "Gemini",
			Description: "Best model for text-only use cases",
		},
		{
			ID:          "gemini-pro-vision",
			Name:        "Gemini Pro Vision",
			Provider:    "Gemini",
			Description: "Best model for text and image understanding",
		},
	}
	return models, nil
}

// Chat 进行对话
func (p *GeminiProvider) Chat(messages []Message, model string) (*ChatResponse, error) {
	// 转换消息格式
	contents := make([]GeminiContent, 0, len(messages))

	for _, msg := range messages {
		role := msg.Role
		// Gemini使用不同的角色名称
		if role == "assistant" {
			role = "model"
		} else if role == "system" {
			// 将system消息合并到第一个user消息中
			if len(contents) == 0 {
				contents = append(contents, GeminiContent{
					Parts: []struct {
						Text string `json:"text"`
					}{{Text: msg.Content}},
					Role: "user",
				})
				continue
			} else {
				// 合并到最后一个user消息中
				for i := len(contents) - 1; i >= 0; i-- {
					if contents[i].Role == "user" {
						contents[i].Parts[0].Text = msg.Content + "\n\n" + contents[i].Parts[0].Text
						break
					}
				}
				continue
			}
		}

		contents = append(contents, GeminiContent{
			Parts: []struct {
				Text string `json:"text"`
			}{{Text: msg.Content}},
			Role: role,
		})
	}

	requestBody, err := json.Marshal(GeminiRequest{
		Contents: contents,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.APIURL, model, p.APIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
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

	var chatResp GeminiResponse
	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(chatResp.Candidates) == 0 || len(chatResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	// 提取文本内容
	var content strings.Builder
	for _, part := range chatResp.Candidates[0].Content.Parts {
		content.WriteString(part.Text)
	}

	return &ChatResponse{
		Content:      content.String(),
		InputTokens:  chatResp.UsageMetadata.PromptTokenCount,
		OutputTokens: chatResp.UsageMetadata.CandidatesTokenCount,
		Model:        model,
		Provider:     "Gemini",
	}, nil
}
