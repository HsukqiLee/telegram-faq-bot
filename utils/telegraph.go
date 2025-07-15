package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	TelegraphAPIURL      = "https://api.telegra.ph"
	TelegraphUploadURL   = "https://telegra.ph/upload"
	TelegraphAccountName = "TelegramFAQBot"
)

// TelegraphClient Telegraph API 客户端
type TelegraphClient struct {
	AccessToken string
	AuthorName  string
	AuthorURL   string
	HTTPClient  *http.Client
}

// TelegraphAccount Telegraph 账号信息
type TelegraphAccount struct {
	ShortName   string `json:"short_name"`
	AuthorName  string `json:"author_name"`
	AuthorURL   string `json:"author_url"`
	AccessToken string `json:"access_token"`
	AuthURL     string `json:"auth_url"`
	PageCount   int    `json:"page_count"`
}

// TelegraphPage Telegraph 页面
type TelegraphPage struct {
	Path        string      `json:"path"`
	URL         string      `json:"url"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	AuthorName  string      `json:"author_name,omitempty"`
	AuthorURL   string      `json:"author_url,omitempty"`
	ImageURL    string      `json:"image_url,omitempty"`
	Content     interface{} `json:"content"`
	Views       int         `json:"views"`
	CanEdit     bool        `json:"can_edit,omitempty"`
}

// TelegraphNode Telegraph 内容节点
type TelegraphNode struct {
	Tag      string            `json:"tag,omitempty"`
	Attrs    map[string]string `json:"attrs,omitempty"`
	Children []interface{}     `json:"children,omitempty"`
	Text     string            `json:"-"` // 用于文本节点
}

// TelegraphResponse API 响应
type TelegraphResponse struct {
	OK     bool        `json:"ok"`
	Error  string      `json:"error,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

// TelegraphUploadResponse 文件上传响应
type TelegraphUploadResponse struct {
	Src string `json:"src"`
}

// NewTelegraphClient 创建新的 Telegraph 客户端
func NewTelegraphClient() *TelegraphClient {
	return &TelegraphClient{
		AuthorName: TelegraphAccountName,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateAccount 创建 Telegraph 账号
func (tc *TelegraphClient) CreateAccount(shortName, authorName string) (*TelegraphAccount, error) {
	data := map[string]string{
		"short_name":  shortName,
		"author_name": authorName,
	}

	resp, err := tc.makeRequest("POST", "/createAccount", data)
	if err != nil {
		return nil, err
	}

	var account TelegraphAccount
	if err := json.Unmarshal(resp, &account); err != nil {
		return nil, fmt.Errorf("failed to parse account response: %w", err)
	}

	tc.AccessToken = account.AccessToken
	tc.AuthorName = account.AuthorName
	return &account, nil
}

// UploadFile 上传文件到 Telegraph
func (tc *TelegraphClient) UploadFile(fileData []byte, filename string) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 创建文件字段
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileData); err != nil {
		return "", fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// 发送请求
	req, err := http.NewRequest("POST", TelegraphUploadURL, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := tc.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read upload response: %w", err)
	}

	var uploadResp []TelegraphUploadResponse
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return "", fmt.Errorf("failed to parse upload response: %w", err)
	}

	if len(uploadResp) == 0 {
		return "", fmt.Errorf("no file uploaded")
	}

	return "https://telegra.ph" + uploadResp[0].Src, nil
}

// CreatePage 创建 Telegraph 页面
func (tc *TelegraphClient) CreatePage(title, content string, images []string) (*TelegraphPage, error) {
	if tc.AccessToken == "" {
		// 如果没有访问令牌，创建临时账号
		account, err := tc.CreateAccount(TelegraphAccountName, tc.AuthorName)
		if err != nil {
			return nil, fmt.Errorf("failed to create account: %w", err)
		}
		tc.AccessToken = account.AccessToken
	}

	// 构建内容节点
	contentNodes := tc.buildContentNodes(content, images)

	data := map[string]interface{}{
		"access_token":   tc.AccessToken,
		"title":          title,
		"content":        contentNodes,
		"return_content": true,
	}

	if tc.AuthorName != "" {
		data["author_name"] = tc.AuthorName
	}

	resp, err := tc.makeRequest("POST", "/createPage", data)
	if err != nil {
		return nil, err
	}

	var page TelegraphPage
	if err := json.Unmarshal(resp, &page); err != nil {
		return nil, fmt.Errorf("failed to parse page response: %w", err)
	}

	return &page, nil
}

// buildContentNodes 构建内容节点
func (tc *TelegraphClient) buildContentNodes(content string, images []string) []interface{} {
	var nodes []interface{}

	// 添加图片
	for _, imageURL := range images {
		nodes = append(nodes, map[string]interface{}{
			"tag": "img",
			"attrs": map[string]string{
				"src": imageURL,
			},
		})
	}

	// 添加文本内容
	if content != "" {
		// 将文本按段落分割
		paragraphs := strings.Split(content, "\n\n")
		for _, paragraph := range paragraphs {
			if strings.TrimSpace(paragraph) != "" {
				nodes = append(nodes, map[string]interface{}{
					"tag":      "p",
					"children": []string{strings.TrimSpace(paragraph)},
				})
			}
		}
	}

	return nodes
}

// CreateTextPage 创建纯文本页面
func (tc *TelegraphClient) CreateTextPage(title, content string) (*TelegraphPage, error) {
	return tc.CreatePage(title, content, nil)
}

// CreateImagePage 创建图文页面
func (tc *TelegraphClient) CreateImagePage(title, content string, images []string) (*TelegraphPage, error) {
	return tc.CreatePage(title, content, images)
}

// makeRequest 发送 HTTP 请求
func (tc *TelegraphClient) makeRequest(method, endpoint string, data interface{}) ([]byte, error) {
	var body io.Reader
	var contentType string

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
		contentType = "application/json"
	}

	req, err := http.NewRequest(method, TelegraphAPIURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := tc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp TelegraphResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("API error: %s", apiResp.Error)
	}

	resultBytes, err := json.Marshal(apiResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultBytes, nil
}

// GetFileExtension 根据内容类型获取文件扩展名
func GetFileExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg" // 默认扩展名
	}
}

// IsImageContentType 检查是否为图片类型
func IsImageContentType(contentType string) bool {
	imageTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, imageType := range imageTypes {
		if strings.Contains(contentType, imageType) {
			return true
		}
	}
	return false
}
