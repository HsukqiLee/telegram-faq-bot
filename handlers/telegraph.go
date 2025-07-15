package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"TGFaqBot/database"
	"TGFaqBot/utils"
)

// TelegraphHandler 处理 Telegraph 相关功能
type TelegraphHandler struct {
	db        database.Database
	telegraph *utils.TelegraphClient
}

// NewTelegraphHandler 创建新的 Telegraph 处理器
func NewTelegraphHandler(db database.Database) *TelegraphHandler {
	return &TelegraphHandler{
		db:        db,
		telegraph: utils.NewTelegraphClient(),
	}
}

// HandleImageUpload 处理图片上传到 Telegraph
func (th *TelegraphHandler) HandleImageUpload(bot *tgbotapi.BotAPI, message *tgbotapi.Message, key string, matchType database.MatchType, title string) error {
	if len(message.Photo) == 0 {
		return fmt.Errorf("no photo found in message")
	}

	// 获取最大尺寸的图片
	photo := message.Photo[len(message.Photo)-1]

	// 下载图片
	fileConfig := tgbotapi.FileConfig{FileID: photo.FileID}
	file, err := bot.GetFile(fileConfig)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// 下载文件内容
	resp, err := http.Get(file.Link(bot.Token))
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read file data: %w", err)
	}

	// 上传到 Telegraph
	filename := fmt.Sprintf("image_%s%s", photo.FileUniqueID, utils.GetFileExtension(resp.Header.Get("Content-Type")))
	imageURL, err := th.telegraph.UploadFile(fileData, filename)
	if err != nil {
		return fmt.Errorf("failed to upload to Telegraph: %w", err)
	}

	// 创建 Telegraph 页面
	content := ""
	if message.Caption != "" {
		content = message.Caption
	}

	page, err := th.telegraph.CreateImagePage(title, content, []string{imageURL})
	if err != nil {
		return fmt.Errorf("failed to create Telegraph page: %w", err)
	}

	// 保存到数据库
	return th.db.AddTelegraphEntry(key, matchType, content, "telegraph_image", page.URL, page.Path)
}

// HandleTextUpload 处理文本上传到 Telegraph
func (th *TelegraphHandler) HandleTextUpload(key string, matchType database.MatchType, title, content string) error {
	// 创建 Telegraph 页面
	page, err := th.telegraph.CreateTextPage(title, content)
	if err != nil {
		return fmt.Errorf("failed to create Telegraph page: %w", err)
	}

	// 保存到数据库
	return th.db.AddTelegraphEntry(key, matchType, content, "telegraph_text", page.URL, page.Path)
}

// SendTelegraphContent 发送 Telegraph 内容
func (th *TelegraphHandler) SendTelegraphContent(bot *tgbotapi.BotAPI, chatID int64, entry *database.Entry) error {
	switch entry.ContentType {
	case "telegraph_image", "telegraph_text":
		// 发送 Telegraph 链接，Telegram 会自动生成预览
		msg := tgbotapi.NewMessage(chatID, entry.TelegraphURL)
		msg.DisableWebPagePreview = false // 确保显示预览
		_, err := bot.Send(msg)
		return err
	default:
		// 发送普通文本
		return utils.SendTextMessage(bot, chatID, entry.Value)
	}
}

// ParseTelegraphCommand 解析 Telegraph 命令
func (th *TelegraphHandler) ParseTelegraphCommand(text string) (action, key, title, content string, matchType database.MatchType, err error) {
	parts := strings.Split(text, " ")
	if len(parts) < 3 {
		return "", "", "", "", "", fmt.Errorf("invalid command format")
	}

	action = parts[0] // "image" 或 "text"

	// 解析匹配类型
	matchTypeStr := parts[1]
	matchTypeInt, parseErr := strconv.Atoi(matchTypeStr)
	if parseErr != nil || (matchTypeInt != 1 && matchTypeInt != 2 && matchTypeInt != 3) {
		return "", "", "", "", "", fmt.Errorf("invalid match type: %s (use 1, 2, or 3)", matchTypeStr)
	}

	matchType, convertErr := database.MatchTypeFromInt(matchTypeInt)
	if convertErr != nil {
		return "", "", "", "", "", fmt.Errorf("invalid match type: %v", convertErr)
	}

	// 解析键名
	key = parts[2]

	// 解析标题（第4个参数）
	if len(parts) >= 4 {
		title = parts[3]
	} else {
		title = key // 默认使用键名作为标题
	}

	// 解析内容（剩余部分）
	if len(parts) > 4 {
		content = strings.Join(parts[4:], " ")
	}

	return action, key, title, content, matchType, nil
}

// HandleTelegraphTextCommand 处理 Telegraph 文本命令
func (th *TelegraphHandler) HandleTelegraphTextCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, commandText string) error {
	action, key, title, content, matchType, err := th.ParseTelegraphCommand(commandText)
	if err != nil {
		return err
	}

	if action != "text" {
		return fmt.Errorf("expected 'text' action, got '%s'", action)
	}

	if content == "" {
		return fmt.Errorf("content is required for text Telegraph pages")
	}

	err = th.HandleTextUpload(key, matchType, title, content)
	if err != nil {
		return fmt.Errorf("failed to create Telegraph text page: %w", err)
	}

	msg := fmt.Sprintf("✅ Telegraph 文本页面已创建：\n📝 键名：%s\n📄 标题：%s\n🔗 类型：%s", key, title, matchType.String())
	return utils.SendTextMessage(bot, message.Chat.ID, msg)
}

// HandleTelegraphImageCommand 处理 Telegraph 图片命令
func (th *TelegraphHandler) HandleTelegraphImageCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, commandText string) error {
	action, key, title, _, matchType, err := th.ParseTelegraphCommand(commandText)
	if err != nil {
		return err
	}

	if action != "image" {
		return fmt.Errorf("expected 'image' action, got '%s'", action)
	}

	if message.Photo == nil {
		return fmt.Errorf("no image found in message")
	}

	err = th.HandleImageUpload(bot, message, key, matchType, title)
	if err != nil {
		return fmt.Errorf("failed to create Telegraph image page: %w", err)
	}

	msg := fmt.Sprintf("✅ Telegraph 图文页面已创建：\n📝 键名：%s\n📄 标题：%s\n🔗 类型：%s", key, title, matchType.String())
	return utils.SendTextMessage(bot, message.Chat.ID, msg)
}
