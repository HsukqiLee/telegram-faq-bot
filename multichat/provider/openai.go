package provider

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
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
	// 构建请求体 - 启用流式响应
	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true, // 启用流式响应
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
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// 发送请求
	resp, err := utils.DoRequest(p.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 对于错误响应，读取完整内容
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read error response: %v", err)
		}

		cleanBody := cleanResponseBody(body)

		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}

		if err := json.Unmarshal(cleanBody, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(cleanBody))
	}

	// 处理流式响应
	return p.handleStreamResponse(resp)
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

	// 清理响应体中的ANSI转义字符和控制字符
	cleanBody := cleanResponseBody(body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(cleanBody))
	}

	var modelsResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int    `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.Unmarshal(cleanBody, &modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v. Response body (first 500 chars): %s", err, truncateString(string(cleanBody), 500))
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

// cleanResponseBody 清理响应体中的ANSI转义字符和控制字符
func cleanResponseBody(body []byte) []byte {
	// 移除ANSI转义序列
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	cleaned := ansiRegex.ReplaceAll(body, []byte{})

	// 移除其他控制字符（保留换行符和制表符）
	controlRegex := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	cleaned = controlRegex.ReplaceAll(cleaned, []byte{})

	return cleaned
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// handleStreamResponse 处理流式响应
func (p *OpenAICompatibleProvider) handleStreamResponse(resp *http.Response) (*ChatResponse, error) {
	var fullContent strings.Builder
	var inputTokens, outputTokens int

	// 检查是否是gzip压缩的响应
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和非data行
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		// 移除"data: "前缀
		data := strings.TrimPrefix(line, "data: ")

		// 检查是否是结束标记
		if data == "[DONE]" {
			break
		}

		// 解析JSON数据
		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			// 忽略解析错误，继续处理下一行
			continue
		}

		// 累积内容
		if len(streamResp.Choices) > 0 {
			fullContent.WriteString(streamResp.Choices[0].Delta.Content)
		}

		// 获取token使用情况
		if streamResp.Usage != nil {
			inputTokens = streamResp.Usage.PromptTokens
			outputTokens = streamResp.Usage.CompletionTokens
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %v", err)
	}

	content := fullContent.String()
	if content == "" {
		return nil, fmt.Errorf("no content received from stream")
	}

	return &ChatResponse{
		Content:      content,
		Provider:     p.name,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}

// ChatWithCallback 发送聊天请求，支持流式回调
func (p *OpenAICompatibleProvider) ChatWithCallback(messages []Message, model string, callback StreamingCallback) (*ChatResponse, error) {
	// 构建请求体 - 启用流式响应
	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true, // 启用流式响应
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
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// 发送请求
	resp, err := utils.DoRequest(p.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 对于错误响应，读取完整内容
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read error response: %v", err)
		}

		cleanBody := cleanResponseBody(body)

		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}

		if err := json.Unmarshal(cleanBody, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(cleanBody))
	}

	// 处理流式响应
	result, err := p.handleStreamResponseWithCallback(resp, messages, callback)
	if err != nil {
		return nil, err
	}

	// 如果没有收到token使用信息，使用估算
	if result.InputTokens == 0 && result.OutputTokens == 0 {
		result.InputTokens = EstimateTokens(messages)
		result.OutputTokens = EstimateTokensFromText(result.Content)
		log.Printf("DEBUG: Using estimated tokens for ChatWithCallback - Input: %d, Output: %d", result.InputTokens, result.OutputTokens)
	}

	return result, nil
}

// handleStreamResponseWithCallback 处理流式响应并调用回调
func (p *OpenAICompatibleProvider) handleStreamResponseWithCallback(resp *http.Response, messages []Message, callback StreamingCallback) (*ChatResponse, error) {
	var fullContent strings.Builder
	var inputTokens, outputTokens int
	var chunkCount int

	log.Printf("DEBUG: Starting stream response processing for %s", p.name)

	// 检查是否是gzip压缩的响应
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和非data行
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		// 移除"data: "前缀
		data := strings.TrimPrefix(line, "data: ")

		// 检查是否是结束标记
		if data == "[DONE]" {
			log.Printf("DEBUG: Stream ended with [DONE] for %s, total chunks: %d", p.name, chunkCount)
			break
		}

		// 解析JSON数据
		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			// 忽略解析错误，继续处理下一行
			log.Printf("DEBUG: Failed to parse stream chunk for %s: %v, data: %s", p.name, err, data)
			continue
		}

		// 累积内容
		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			chunkContent := streamResp.Choices[0].Delta.Content
			fullContent.WriteString(chunkContent)
			chunkCount++

			// 调用回调函数，传递当前累积的内容
			if callback != nil {
				currentContent := fullContent.String()
				log.Printf("DEBUG: Streaming chunk %d for %s, chunk: %q, total length: %d", chunkCount, p.name, chunkContent, len(currentContent))
				if !callback(currentContent, false) {
					// 回调返回false，停止流式传输
					log.Printf("DEBUG: Callback returned false, stopping stream for %s", p.name)
					break
				}
			}
		}

		// 获取token使用情况
		if streamResp.Usage != nil {
			inputTokens = streamResp.Usage.PromptTokens
			outputTokens = streamResp.Usage.CompletionTokens
			log.Printf("DEBUG: Got token usage for %s - Input: %d, Output: %d", p.name, inputTokens, outputTokens)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %v", err)
	}

	content := fullContent.String()
	if content == "" {
		return nil, fmt.Errorf("no content received from stream")
	}

	log.Printf("DEBUG: Stream completed for %s - Total chunks: %d, Content length: %d", p.name, chunkCount, len(content))

	// 最终回调，标记完成
	if callback != nil {
		log.Printf("DEBUG: Making final callback for %s", p.name)
		callback(content, true)
	}

	// 如果没有收到token使用信息，使用估算
	if inputTokens == 0 && outputTokens == 0 {
		inputTokens = EstimateTokens(messages)
		outputTokens = EstimateTokensFromText(content)
		log.Printf("DEBUG: Using estimated tokens in handleStreamResponseWithCallback - Input: %d, Output: %d", inputTokens, outputTokens)
	}

	return &ChatResponse{
		Content:      content,
		Provider:     p.name,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}
