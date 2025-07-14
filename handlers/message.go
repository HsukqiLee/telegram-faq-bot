package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"TGFaqBot/config"
	"TGFaqBot/database"
	"TGFaqBot/multichat"
	"TGFaqBot/utils"
)

type MessageHandler struct {
	db               database.Database
	conf             *config.Config
	state            *State
	streamer         *StreamingManager
	multichatManager *multichat.Manager
}

func NewMessageHandler(db database.Database, conf *config.Config, state *State, streamer *StreamingManager, multichatMgr *multichat.Manager) *MessageHandler {
	return &MessageHandler{
		db:               db,
		conf:             conf,
		state:            state,
		streamer:         streamer,
		multichatManager: multichatMgr,
	}
}

func (h *MessageHandler) HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	state, exists := h.state.Get(chatID)

	if exists {
		h.handleConversationMessage(bot, message, state)
	} else {
		// Handle other messages or commands
		if !strings.HasPrefix(message.Text, "/") {
			h.handleAIMessage(bot, message)
		}
	}
}

func (h *MessageHandler) handleConversationMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *Conversation) {
	chatID := message.Chat.ID

	switch state.Stage {
	case "awaiting_type":
		newType, err := strconv.Atoi(message.Text)
		if err != nil || (newType != 1 && newType != 2 && newType != 3) {
			bot.Send(tgbotapi.NewMessage(chatID, "输入类型不合法，操作中断"))
			h.state.Delete(chatID)
			return
		}

		state.NewType = newType
		editMsg := tgbotapi.NewEditMessageText(chatID, message.MessageID, "请输入新的内容：")
		prevButton := tgbotapi.NewInlineKeyboardButtonData("上一步", fmt.Sprintf("update_%d_%d", state.EntryID, state.OldType))
		cancelButton := tgbotapi.NewInlineKeyboardButtonData("取消", "cancel")
		editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{prevButton, cancelButton}}}
		bot.Send(editMsg)
		state.Stage = "awaiting_value"
		h.state.Set(chatID, state)

	case "awaiting_value":
		h.handleValueInput(bot, message, state)
	}
}

func (h *MessageHandler) handleValueInput(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *Conversation) {
	chatID := message.Chat.ID
	newValue := message.Text
	entryID := state.EntryID
	newType := state.NewType
	oldType := state.OldType
	originalMessageID := state.MessageID

	// Retrieve the entry from the database
	entry, err := h.db.QueryByID(entryID, oldType)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "未找到条目"))
		h.state.Delete(chatID)
		return
	}

	// Update the entry
	err = h.db.UpdateEntry(entry.Key, oldType, newType, newValue)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "更新失败"))
		h.state.Delete(chatID)
		return
	}

	matchTypeText := utils.GetMatchTypeText(newType)

	// Edit the original message to "操作结束"
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, originalMessageID, "操作结束")
	bot.Send(editMsg)

	// Send a new message with the updated information
	newMsgText := fmt.Sprintf("更新成功！\nKey: %s\nValue: %s\n类型：%s", entry.Key, newValue, matchTypeText)
	newMsg := tgbotapi.NewMessage(message.Chat.ID, newMsgText)
	bot.Send(newMsg)

	h.state.Delete(chatID)
}

// handleAIMessage 处理AI对话消息，支持流式输出
func (h *MessageHandler) handleAIMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userMessage := message.Text

	// 检查是否有启用的AI提供商，如果没有则静默忽略
	if !h.multichatManager.HasEnabledProviders() {
		log.Printf("No AI providers enabled, ignoring message from chat %d", chatID)
		return
	}

	// 检查消息发送频率限制（防止spam）
	if !h.streamer.CanSendMessage(chatID) {
		log.Printf("Message throttled for chat %d", chatID)
		return
	}

	// 发送初始"正在输入"消息
	initialMsg := tgbotapi.NewMessage(chatID, "🤔 正在思考...")
	sentMsg, err := bot.Send(initialMsg)
	if err != nil {
		log.Printf("Error sending initial message: %v", err)
		return
	}

	// 创建流式消息管理器
	streamKey := fmt.Sprintf("%d_%d", chatID, sentMsg.MessageID)
	h.streamer.CreateStream(streamKey, chatID, sentMsg.MessageID)

	// 异步获取AI响应
	go func() {
		defer h.streamer.DeleteStream(streamKey)

		// 获取AI响应 - 使用流式响应
		_, shouldReset, err := h.getOpenAIReplyWithStreaming(userMessage, message, func(partialContent string, isComplete bool) bool {
			// 更新流式消息
			h.streamer.UpdateStream(bot, streamKey, partialContent, isComplete)
			return true // 继续流式传输
		})

		if err != nil {
			// 编辑消息为错误信息
			editMsg := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, fmt.Sprintf("❌ 发生错误：%s", err.Error()))
			bot.Send(editMsg)
			return
		}

		// 处理对话重置
		if shouldReset {
			// TODO: Implement conversation clearing with new chat system
			resetMsg := tgbotapi.NewMessage(chatID, "⚠️ 连续对话达到上限，已自动重置对话历史")
			bot.Send(resetMsg)
		}
	}()
}

// getOpenAIReplyWithStreaming 获取OpenAI回复，支持流式回调
func (h *MessageHandler) getOpenAIReplyWithStreaming(userMessage string, message *tgbotapi.Message, callback func(string, bool) bool) (string, bool, error) {
	// TODO: Implement streaming response with new chat system
	_ = userMessage // 参数暂时未使用
	_ = message     // 参数暂时未使用
	_ = callback    // 参数暂时未使用
	return "功能正在重构中...", false, nil
}
