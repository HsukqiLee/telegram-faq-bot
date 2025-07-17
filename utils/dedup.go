package utils

import (
	"fmt"
	"strings"

	"TGFaqBot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Paginate returns a slice for the given page and pageSize.
func Paginate[T any](slice []T, page, pageSize int) []T {
	if pageSize <= 0 || page < 0 {
		return nil
	}
	start := page * pageSize
	if start >= len(slice) {
		return nil
	}
	end := start + pageSize
	if end > len(slice) {
		end = len(slice)
	}
	return slice[start:end]
}

// BuildPaginationButtons generates pagination and cancel buttons for inline keyboard.
func BuildPaginationButtons(page, total, pageSize int, prefix string, cancelLabel string) [][]tgbotapi.InlineKeyboardButton {
	var navButtons []tgbotapi.InlineKeyboardButton
	if page > 0 {
		prevButton := tgbotapi.NewInlineKeyboardButtonData("上一页", fmt.Sprintf("%s_%d", prefix, page-1))
		navButtons = append(navButtons, prevButton)
	}
	if (page+1)*pageSize < total {
		nextButton := tgbotapi.NewInlineKeyboardButtonData("下一页", fmt.Sprintf("%s_%d", prefix, page+1))
		navButtons = append(navButtons, nextButton)
	}
	if len(navButtons) == 0 && cancelLabel == "" {
		return nil
	}
	buttons := [][]tgbotapi.InlineKeyboardButton{}
	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}
	if cancelLabel != "" {
		cancelButton := tgbotapi.NewInlineKeyboardButtonData(cancelLabel, "cancel")
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{cancelButton})
	}
	return buttons
}

// ParseMatchType parses a string to database.MatchType.
func ParseMatchType(str string) (database.MatchType, error) {
	switch strings.ToLower(str) {
	case "exact":
		return database.MatchExact, nil
	case "contains":
		return database.MatchContains, nil
	case "regex":
		return database.MatchRegex, nil
	case "prefix":
		return database.MatchPrefix, nil
	case "suffix":
		return database.MatchSuffix, nil
	default:
		return "", fmt.Errorf("invalid match type: %s", str)
	}
}
