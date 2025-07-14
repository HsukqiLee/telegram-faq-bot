package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// EscapeMarkdownV2 escapes special characters for Telegram MarkdownV2
func EscapeMarkdownV2(text string) string {
	// Define special characters that need escaping
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

	// First step: escape all special characters
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}

	// Second step: restore correctly escaped characters
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, "\\\\"+char, "\\"+char)
	}

	return text
}

// FormatResponse formats the OpenAI response with statistics
func FormatResponse(response string, inputTokens, outputTokens, totalInput, totalOutput int,
	duration time.Duration, remainingRounds, remainingMinutes, remainingSeconds int, currentModel string) string {

	formattedResponse := MdToTgmd(FixMarkdown(response))
	stats := fmt.Sprintf("\n\n━━━━━━ 统计信息 ━━━━━━\n"+
		"📊 输入: %d    总输入: %d\n"+
		"📈 输出: %d    总输出: %d\n"+
		"⏱ 处理时间: %.2f秒\n"+
		"🔄 本轮剩余次数: %d\n"+
		"🕒 对话保留时间: %d分钟 %d秒\n"+
		"🤖 模型: %s\n"+
		"━━━━━━━━━━━━━━━━━",
		inputTokens, totalInput, outputTokens, totalOutput, duration.Seconds(),
		remainingRounds, remainingMinutes, remainingSeconds, currentModel)

	formattedResponse += MdToTgmd(stats)

	return formattedResponse
}

// ReplaceMatches replaces regex matches in text
func ReplaceMatches(text, pattern, replacement string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatchIndex(text, -1)

	if len(matches) == 0 {
		return text
	}

	result := ""
	lastIndex := 0

	for _, match := range matches {
		startIndex := match[0]
		endIndex := match[1]

		submatch := text[match[2]:match[3]]
		result += text[lastIndex:startIndex] + strings.Replace(replacement, "$1", submatch, 1)

		lastIndex = endIndex
	}

	result += text[lastIndex:]
	return result
}

// MdToTgmd converts standard Markdown to Telegram MarkdownV2
func MdToTgmd(text string) string {
	// Pre-process: escape special characters
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}

	// Handle code blocks
	codeBlockRegex := regexp.MustCompile("(?s)\\\\`\\\\`\\\\`(.*?)\\\\`\\\\`\\\\`")
	text = codeBlockRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Remove escape characters from code block content
		inner := strings.Trim(match, "\\`")
		inner = strings.ReplaceAll(inner, "\\", "")
		return "```" + inner + "```"
	})

	// Handle inline code
	inlineCodeRegex := regexp.MustCompile("\\\\`(.*?)\\\\`")
	text = inlineCodeRegex.ReplaceAllString(text, "`$1`")

	// Handle bold italic
	text = ReplaceMatches(text, `\\\*\\\*\\\*(.*?)\\\*\\\*\\\*`, "*_$1_*")

	// Handle bold
	text = ReplaceMatches(text, `\\\*\\\*(.*?)\\\*\\\*`, "*$1*")

	// Handle italic
	text = ReplaceMatches(text, `\\\*(.*?)\\\*`, "_$1_")

	// Handle strikethrough
	strikethroughRegex := regexp.MustCompile(`\\\~\\\~(.*?)\\\~\\\~`)
	text = strikethroughRegex.ReplaceAllString(text, "~$1~")

	// Handle links
	linkRegex := regexp.MustCompile(`\\\[(.*?)\\\]\\\((.*?)\\\)`)
	text = linkRegex.ReplaceAllString(text, "[$1]($2)")

	// Handle headers
	headerRegex := regexp.MustCompile(`(?m)^((?:\\#)+)\s(.+)`)
	text = headerRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := headerRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		// Calculate header level
		level := strings.Count(parts[1], "\\#") / 2

		// Calculate indentation safely
		var indent string
		if level > 1 {
			indent = strings.Repeat("  ", level-1)
		} else {
			indent = ""
		}

		return fmt.Sprintf("*%s*◆ *%s*", indent, parts[2])
	})

	return text
}

// GetUnclosedMarkdownTag finds unclosed markdown tags
func GetUnclosedMarkdownTag(markdown string) string {
	// Order is important!
	var tags = []string{
		"```",
		"`",
		"*",
		"_",
	}
	var currentTag = ""

	markdownRunes := []rune(markdown)

	var i = 0
outer:
	for i < len(markdownRunes) {
		// Skip escaped characters (only outside tags)
		if markdownRunes[i] == '\\' && currentTag == "" {
			i += 2
			continue
		}
		if currentTag != "" {
			if strings.HasPrefix(string(markdownRunes[i:]), currentTag) {
				// Turn a tag off
				i += len(currentTag)
				currentTag = ""
				continue
			}
		} else {
			for _, tag := range tags {
				if strings.HasPrefix(string(markdownRunes[i:]), tag) {
					// Turn a tag on
					currentTag = tag
					i += len(currentTag)
					continue outer
				}
			}
		}
		i++
	}

	return currentTag
}

// FixMarkdown fixes unclosed markdown tags
func FixMarkdown(markdown string) string {
	for {
		tag := GetUnclosedMarkdownTag(markdown)
		if tag == "" {
			return markdown
		}
		markdown += tag
	}
}
