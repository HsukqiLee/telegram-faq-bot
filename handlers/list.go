package handlers

import (
	"fmt"
	"log"
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
	var matchTypes []database.MatchType
	if args != "" {
		typeStrings := strings.Split(args, " ")
		for _, typeString := range typeStrings {
			matchType, err := utils.ParseMatchType(typeString)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "匹配类型错误，请使用 exact, contains, regex, prefix, suffix"))
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
	pageEntries := utils.Paginate(entries, page, pageSize)
	if len(pageEntries) == 0 {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "没有任何条目"))
		return
	}
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, entry := range pageEntries {
		matchTypeText := utils.GetMatchTypeText(entry.MatchType)
		buttonText := fmt.Sprintf("%s(%s)", entry.Key, matchTypeText)
		callbackData := fmt.Sprintf("entry_%d_%d", entry.ID, entry.MatchType.ToInt())
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}
	// Add pagination and cancel buttons
	buttons = append(buttons, utils.BuildPaginationButtons(page, len(entries), pageSize, "list", "取消")...)
	msg := tgbotapi.NewMessage(message.Chat.ID, "选择一个条目进行操作：")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	sentMessage, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}
	h.state.Set(message.Chat.ID, &Conversation{
		Stage:     "listing",
		MessageID: sentMessage.MessageID,
	})
}

func (h *ListHandler) HandleListCommandEdit(bot *tgbotapi.BotAPI, message *tgbotapi.Message, page int, messageID int) {
	args := message.CommandArguments()
	var matchTypes []database.MatchType
	if args != "" {
		typeStrings := strings.Split(args, " ")
		for _, typeString := range typeStrings {
			matchType, err := utils.ParseMatchType(typeString)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "匹配类型错误，请使用 exact, contains, regex, prefix, suffix"))
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
	pageEntries := utils.Paginate(entries, page, pageSize)
	if len(pageEntries) == 0 {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "没有任何条目"))
		return
	}
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, entry := range pageEntries {
		matchTypeText := utils.GetMatchTypeText(entry.MatchType)
		buttonText := fmt.Sprintf("%s(%s)", entry.Key, matchTypeText)
		callbackData := fmt.Sprintf("entry_%d_%d", entry.ID, entry.MatchType.ToInt())
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}
	// Add pagination and cancel buttons
	buttons = append(buttons, utils.BuildPaginationButtons(page, len(entries), pageSize, "list", "取消")...)
	msgText := "选择一个条目进行操作："
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, messageID, msgText)
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	bot.Send(editMsg)
}

func (h *ListHandler) HandleEntrySelection(bot *tgbotapi.BotAPI, message *tgbotapi.Message, entryID int, matchType int) {
	matchTypeValue, err := database.MatchTypeFromInt(matchType)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "匹配类型转换错误"))
		return
	}

	entry, err := h.db.QueryByID(entryID, matchTypeValue)
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
