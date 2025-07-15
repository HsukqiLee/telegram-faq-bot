package provider

import "time"

// Provider AI提供商接口
type Provider interface {
	GetName() string
	GetModels() ([]Model, error)
	Chat(messages []Message, model string) (*ChatResponse, error)
}

// StreamingProvider 支持流式回调的AI提供商接口
type StreamingProvider interface {
	Provider
	ChatWithCallback(messages []Message, model string, callback StreamingCallback) (*ChatResponse, error)
}

// Message 消息结构
type Message struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Time    time.Time `json:"time,omitempty"`
}

// Model 模型信息
type Model struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Content      string `json:"content"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
}

// StreamingCallback 流式响应回调函数类型
type StreamingCallback func(partialResponse string, isComplete bool) bool

// estimateTokens 估算消息的token数量
func EstimateTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		// 粗略估算：1个token约等于4个字符
		total += len(msg.Content) / 4
	}
	return total
}

// estimateTokensFromText 从文本估算token数量
func EstimateTokensFromText(text string) int {
	// 粗略估算：1个token约等于4个字符
	return len(text) / 4
}
