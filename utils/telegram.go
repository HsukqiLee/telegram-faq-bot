package utils

import (
	"fmt"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramMarkdownMode 定义 Telegram 支持的 Markdown 模式
type TelegramMarkdownMode string

const (
	MarkdownV1 TelegramMarkdownMode = "Markdown"
	MarkdownV2 TelegramMarkdownMode = "MarkdownV2"
	HTML       TelegramMarkdownMode = "HTML"
)

// EscapeMarkdownV1 转义 Markdown V1 特殊字符
func EscapeMarkdownV1(text string) string {
	replacer := strings.NewReplacer(
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
		"[", "\\[",
	)
	return replacer.Replace(text)
}

// ValidateMarkdownV2 验证 MarkdownV2 格式是否正确
func ValidateMarkdownV2(text string) error {
	// 检查配对的格式标记
	formatChars := []rune{'*', '_', '~', '`'}

	for _, char := range formatChars {
		count := strings.Count(text, string(char))
		if count%2 != 0 {
			return fmt.Errorf("unmatched formatting character: %c", char)
		}
	}

	// 检查链接格式 [text](url)
	linkPattern := regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	if matches := linkPattern.FindAllString(text, -1); matches != nil {
		for _, match := range matches {
			if !strings.Contains(match, "](") {
				return fmt.Errorf("invalid link format: %s", match)
			}
		}
	}

	return nil
}

// ConvertStandardMarkdownToTelegram 将标准 Markdown 转换为 Telegram 兼容格式
func ConvertStandardMarkdownToTelegram(text string, mode TelegramMarkdownMode) string {
	switch mode {
	case MarkdownV2:
		// 先处理代码块和内联代码，避免转义其中的内容
		text = preserveCodeBlocks(text)
		// 转义特殊字符（使用已有的函数）
		text = EscapeMarkdownV2(text)
		// 恢复代码块
		text = restoreCodeBlocks(text)
		return text
	case MarkdownV1:
		return EscapeMarkdownV1(text)
	case HTML:
		return convertMarkdownToHTML(text)
	default:
		return text
	}
}

// preserveCodeBlocks 保护代码块不被转义
func preserveCodeBlocks(text string) string {
	// 简单实现：用占位符替换代码块
	codeBlockPattern := regexp.MustCompile("```([\\s\\S]*?)```")
	inlineCodePattern := regexp.MustCompile("`([^`]*)`")

	text = codeBlockPattern.ReplaceAllStringFunc(text, func(match string) string {
		return "CODEBLOCK_PLACEHOLDER_" + strings.ReplaceAll(match, "\n", "NEWLINE_PLACEHOLDER")
	})

	text = inlineCodePattern.ReplaceAllStringFunc(text, func(match string) string {
		return "INLINECODE_PLACEHOLDER_" + strings.ReplaceAll(match, " ", "SPACE_PLACEHOLDER")
	})

	return text
}

// restoreCodeBlocks 恢复代码块
func restoreCodeBlocks(text string) string {
	text = regexp.MustCompile("CODEBLOCK_PLACEHOLDER_(.+?)").ReplaceAllStringFunc(text, func(match string) string {
		content := strings.TrimPrefix(match, "CODEBLOCK_PLACEHOLDER_")
		return strings.ReplaceAll(content, "NEWLINE_PLACEHOLDER", "\n")
	})

	text = regexp.MustCompile("INLINECODE_PLACEHOLDER_(.+?)").ReplaceAllStringFunc(text, func(match string) string {
		content := strings.TrimPrefix(match, "INLINECODE_PLACEHOLDER_")
		return strings.ReplaceAll(content, "SPACE_PLACEHOLDER", " ")
	})

	return text
}

// convertMarkdownToHTML 将基本 Markdown 转换为 HTML
func convertMarkdownToHTML(text string) string {
	replacements := []struct {
		pattern string
		replace string
	}{
		{`\*\*(.*?)\*\*`, `<b>$1</b>`},                     // **bold**
		{`\*(.*?)\*`, `<i>$1</i>`},                         // *italic*
		{`__(.*?)__`, `<b>$1</b>`},                         // __bold__
		{`_(.*?)_`, `<i>$1</i>`},                           // _italic_
		{"`([^`]*)`", `<code>$1</code>`},                   // `code`
		{`~~(.*?)~~`, `<s>$1</s>`},                         // ~~strikethrough~~
		{`\[([^\]]*)\]\(([^)]*)\)`, `<a href="$2">$1</a>`}, // [text](url)
	}

	result := text
	for _, r := range replacements {
		pattern := regexp.MustCompile(r.pattern)
		result = pattern.ReplaceAllString(result, r.replace)
	}

	return result
}

// SendTextMessage 创建并发送文本消息的辅助函数
func SendTextMessage(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bot.Send(msg)
	return err
}

// SendMarkdownMessage 创建并发送Markdown格式消息的辅助函数
func SendMarkdownMessage(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	_, err := bot.Send(msg)
	return err
}

// SendMarkdownV2Message 创建并发送MarkdownV2格式消息的辅助函数
func SendMarkdownV2Message(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	// 验证格式
	if err := ValidateMarkdownV2(text); err != nil {
		return fmt.Errorf("invalid MarkdownV2 format: %w", err)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "MarkdownV2"
	_, err := bot.Send(msg)
	return err
}

// SendSafeMarkdownMessage 发送安全的 Markdown 消息，自动转义特殊字符
func SendSafeMarkdownMessage(bot *tgbotapi.BotAPI, chatID int64, text string, mode TelegramMarkdownMode) error {
	processedText := ConvertStandardMarkdownToTelegram(text, mode)

	msg := tgbotapi.NewMessage(chatID, processedText)
	switch mode {
	case MarkdownV1:
		msg.ParseMode = "Markdown"
	case MarkdownV2:
		msg.ParseMode = "MarkdownV2"
	case HTML:
		msg.ParseMode = "HTML"
	}

	_, err := bot.Send(msg)
	if err != nil {
		// 如果发送失败，尝试使用纯文本模式
		fallbackMsg := tgbotapi.NewMessage(chatID, text)
		_, fallbackErr := bot.Send(fallbackMsg)
		if fallbackErr != nil {
			return fmt.Errorf("both formatted and fallback messages failed: %w, %w", err, fallbackErr)
		}
		return fmt.Errorf("formatted message failed, sent as plain text: %w", err)
	}

	return nil
}

// SendHTMLMessage 创建并发送HTML格式消息的辅助函数
func SendHTMLMessage(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	_, err := bot.Send(msg)
	return err
}

// CreateTextMessage 创建文本消息（不发送）
func CreateTextMessage(chatID int64, text string) tgbotapi.MessageConfig {
	return tgbotapi.NewMessage(chatID, text)
}

// CreateMarkdownMessage 创建Markdown格式消息（不发送）
func CreateMarkdownMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	return msg
}

// CreateSafeMarkdownMessage 创建安全的Markdown格式消息（不发送）
func CreateSafeMarkdownMessage(chatID int64, text string, mode TelegramMarkdownMode) tgbotapi.MessageConfig {
	processedText := ConvertStandardMarkdownToTelegram(text, mode)
	msg := tgbotapi.NewMessage(chatID, processedText)

	switch mode {
	case MarkdownV1:
		msg.ParseMode = "Markdown"
	case MarkdownV2:
		msg.ParseMode = "MarkdownV2"
	case HTML:
		msg.ParseMode = "HTML"
	}

	return msg
}
