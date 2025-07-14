package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"TGFaqBot/database"
	"TGFaqBot/utils"
)

type ListHandler struct {
	db    database.Database
	state *State
}

func NewListHandler(db database.Database, state *State) *ListHandler {
	return &ListHandler{
		db:    db,
		state: state,
	}
}

func (h *ListHandler) HandleListCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, page int) {
	args := message.CommandArguments()
	var matchTypes []int

	if args != "" {
		typeStrings := strings.Split(args, " ")
		for _, typeString := range typeStrings {
			matchType, err := strconv.Atoi(typeString)
			if err != nil || (matchType != 1 && matchType != 2 && matchType != 3) {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "匹配类型错误，请使用 1, 2, 或 3"))
				return
			}
			matchTypes = append(matchTypes, matchType)
		}
	}

	entries, err := h.db.ListSpecificEntries(matchTypes...)
	if err != nil {
		log.Printf("Error listing entries: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无法获取条目列表"))
		return
	}

	const pageSize = 5
	start := page * pageSize

	if start >= len(entries) {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "没有任何条目"))
		return
	}

	end := start + pageSize
	if end > len(entries) {
		end = len(entries)
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for i := start; i < end; i++ {
		entry := entries[i]
		matchTypeText := utils.GetMatchTypeText(entry.MatchType)
		buttonText := fmt.Sprintf("%s(%s)", entry.Key, matchTypeText)
		callbackData := fmt.Sprintf("entry_%d_%d", entry.ID, entry.MatchType)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}

	// Add pagination buttons
	var navButtons []tgbotapi.InlineKeyboardButton
	if page > 0 {
		prevButton := tgbotapi.NewInlineKeyboardButtonData("上一页", fmt.Sprintf("list_%d", page-1))
		navButtons = append(navButtons, prevButton)
	}
	if end < len(entries) {
		nextButton := tgbotapi.NewInlineKeyboardButtonData("下一页", fmt.Sprintf("list_%d", page+1))
		navButtons = append(navButtons, nextButton)
	}

	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}

	cancelButton := tgbotapi.NewInlineKeyboardButtonData("取消", "cancel")
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{cancelButton})

	msg := tgbotapi.NewMessage(message.Chat.ID, "选择一个条目进行操作：")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	sentMessage, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}

	// Store the message ID in the conversation state
	h.state.Set(message.Chat.ID, &Conversation{
		Stage:     "listing",
		MessageID: sentMessage.MessageID,
	})
}

func (h *ListHandler) HandleListCommandEdit(bot *tgbotapi.BotAPI, message *tgbotapi.Message, page int, messageID int) {
	args := message.CommandArguments()
	var matchTypes []int

	if args != "" {
		typeStrings := strings.Split(args, " ")
		for _, typeString := range typeStrings {
			matchType, err := strconv.Atoi(typeString)
			if err != nil || (matchType != 1 && matchType != 2 && matchType != 3) {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "匹配类型错误，请使用 1, 2, 或 3"))
				return
			}
			matchTypes = append(matchTypes, matchType)
		}
	}

	entries, err := h.db.ListSpecificEntries(matchTypes...)
	if err != nil {
		log.Printf("Error listing entries: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无法获取条目列表"))
		return
	}

	const pageSize = 5
	start := page * pageSize

	if start >= len(entries) {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "没有任何条目"))
		return
	}

	end := utils.Min(start+pageSize, len(entries))

	var buttons [][]tgbotapi.InlineKeyboardButton
	for i := start; i < end; i++ {
		entry := entries[i]
		matchTypeText := utils.GetMatchTypeText(entry.MatchType)
		buttonText := fmt.Sprintf("%s(%s)", entry.Key, matchTypeText)
		callbackData := fmt.Sprintf("entry_%d_%d", entry.ID, entry.MatchType)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}

	// Add pagination buttons
	var navButtons []tgbotapi.InlineKeyboardButton
	if page > 0 {
		prevButton := tgbotapi.NewInlineKeyboardButtonData("上一页", fmt.Sprintf("list_%d", page-1))
		navButtons = append(navButtons, prevButton)
	}
	if end < len(entries) {
		nextButton := tgbotapi.NewInlineKeyboardButtonData("下一页", fmt.Sprintf("list_%d", page+1))
		navButtons = append(navButtons, nextButton)
	}

	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}

	cancelButton := tgbotapi.NewInlineKeyboardButtonData("取消", "cancel")
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{cancelButton})

	msgText := "选择一个条目进行操作："
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, messageID, msgText)
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	bot.Send(editMsg)
}

func (h *ListHandler) HandleEntrySelection(bot *tgbotapi.BotAPI, message *tgbotapi.Message, entryID int, matchType int) {
	entry, err := h.db.QueryByID(entryID, matchType)
	if err != nil {
		log.Printf("Error querying database: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无法获取条目"))
		return
	}

	if entry == nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "未找到条目"))
		return
	}

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("更新", fmt.Sprintf("show_update_types_%d_%d", entry.ID, matchType)),
			tgbotapi.NewInlineKeyboardButtonData("删除", fmt.Sprintf("delete_%d_%d", entry.ID, matchType)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("返回", fmt.Sprintf("list_%d", 0)),
			tgbotapi.NewInlineKeyboardButtonData("取消", "cancel"),
		},
	}

	matchTypeText := utils.GetMatchTypeText(entry.MatchType)
	msgText := fmt.Sprintf("选择操作：\nKey: %s\nValue: %s\n类型：%s", entry.Key, entry.Value, matchTypeText)
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, msgText)
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	bot.Send(editMsg)
}
