package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"TGFaqBot/config"
	"TGFaqBot/database"
)

type CallbackHandler struct {
	db           database.Database
	conf         *config.Config
	state        *State
	adminHandler *AdminHandler
	listHandler  *ListHandler
}

func NewCallbackHandler(db database.Database, conf *config.Config, state *State) *CallbackHandler {
	return &CallbackHandler{
		db:           db,
		conf:         conf,
		state:        state,
		adminHandler: NewAdminHandler(db, conf, state),
		listHandler:  NewListHandler(db, state),
	}
}

func (h *CallbackHandler) HandleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	data := callbackQuery.Data
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID

	switch {
	case strings.HasPrefix(data, "list_"):
		h.handleListCallback(bot, callbackQuery, data)
	case strings.HasPrefix(data, "entry_"):
		h.handleEntryCallback(bot, callbackQuery, data)
	case strings.HasPrefix(data, "show_update_types_"):
		h.handleShowUpdateTypesCallback(bot, callbackQuery, data, chatID, messageID)
	case strings.HasPrefix(data, "update_type_"):
		h.handleUpdateTypeCallback(bot, callbackQuery, data, chatID, messageID)
	case strings.HasPrefix(data, "confirm_delete_"):
		h.handleConfirmDeleteCallback(bot, callbackQuery, data, chatID, messageID)
	case strings.HasPrefix(data, "confirm_batch_delete_"):
		h.handleConfirmBatchDeleteCallback(bot, callbackQuery, data, chatID, messageID)
	case strings.HasPrefix(data, "delete_"):
		h.handleDeleteCallback(bot, callbackQuery, data, chatID, messageID)
	case strings.HasPrefix(data, "listadmin_"):
		h.handleListAdminCallback(bot, callbackQuery, data)
	case strings.HasPrefix(data, "admin_"):
		h.handleAdminCallback(bot, callbackQuery, data)
	case strings.HasPrefix(data, "deladmin_"):
		h.handleDelAdminCallback(bot, callbackQuery, data, chatID, messageID)
	case data == "confirm_deleteall":
		h.handleConfirmDeleteAllCallback(bot, callbackQuery, chatID, messageID)
	case data == "cancel":
		h.handleCancelCallback(bot, callbackQuery, chatID, messageID)
	case strings.HasPrefix(data, "model:"):
		h.handleModelCallback(bot, callbackQuery, data, chatID, messageID)
	}

	// Acknowledge the callback
	_, err := bot.Request(tgbotapi.NewCallback(callbackQuery.ID, ""))
	if err != nil {
		log.Printf("Error acknowledging callback: %v", err)
	}
}

func (h *CallbackHandler) handleListCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, data string) {
	pageStr := strings.TrimPrefix(data, "list_")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Printf("Error parsing page number: %v", err)
		return
	}

	state, exists := h.state.Get(callbackQuery.Message.Chat.ID)
	if !exists || state.Stage != "listing" {
		log.Printf("Conversation state not found for chat ID: %d", callbackQuery.Message.Chat.ID)
		return
	}
	messageID := state.MessageID

	h.listHandler.HandleListCommandEdit(bot, callbackQuery.Message, page, messageID)
}

func (h *CallbackHandler) handleEntryCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, data string) {
	parts := strings.Split(strings.TrimPrefix(data, "entry_"), "_")
	if len(parts) != 2 {
		log.Printf("Error parsing entry ID and match type: %s", data)
		return
	}

	entryID, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Printf("Error parsing entry ID: %v", err)
		return
	}

	matchType, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Printf("Error parsing match type: %v", err)
		return
	}

	h.listHandler.HandleEntrySelection(bot, callbackQuery.Message, entryID, matchType)
}

func (h *CallbackHandler) handleShowUpdateTypesCallback(bot *tgbotapi.BotAPI, _ *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	parts := strings.Split(strings.TrimPrefix(data, "show_update_types_"), "_")
	if len(parts) != 2 {
		log.Printf("Error parsing entry ID and match type for show_update_types: %s", data)
		return
	}

	entryID, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Printf("Error parsing entry ID for show_update_types: %v", err)
		return
	}

	matchType, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Printf("Error parsing match type for show_update_types: %v", err)
		return
	}

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("精确", fmt.Sprintf("update_type_%d_%d_%d", entryID, matchType, 1)),
			tgbotapi.NewInlineKeyboardButtonData("模糊", fmt.Sprintf("update_type_%d_%d_%d", entryID, matchType, 2)),
			tgbotapi.NewInlineKeyboardButtonData("正则", fmt.Sprintf("update_type_%d_%d_%d", entryID, matchType, 3)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("返回", fmt.Sprintf("entry_%d_%d", entryID, matchType)),
			tgbotapi.NewInlineKeyboardButtonData("取消", "cancel"),
		},
	}

	msgText := "选择新的匹配类型："
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msgText)
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	bot.Send(editMsg)
}

func (h *CallbackHandler) handleUpdateTypeCallback(bot *tgbotapi.BotAPI, _ *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	parts := strings.Split(strings.TrimPrefix(data, "update_type_"), "_")
	if len(parts) != 3 {
		log.Printf("Error parsing entry ID, old match type, and new match type for update: %s", data)
		return
	}

	entryID, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Printf("Error parsing entry ID for update: %v", err)
		return
	}

	oldType, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Printf("Error parsing old match type for update: %v", err)
		return
	}

	newType, err := strconv.Atoi(parts[2])
	if err != nil {
		log.Printf("Error parsing new match type for update: %v", err)
		return
	}

	// Start conversation to get new value
	h.state.Set(chatID, &Conversation{
		Stage:     "awaiting_value",
		EntryID:   entryID,
		NewType:   newType,
		OldType:   oldType,
		MessageID: messageID,
	})

	// 获取当前条目信息用于显示预览
	entry, err := h.db.QueryByID(entryID, oldType)
	var currentInfo string
	if err == nil && entry != nil {
		currentInfo = fmt.Sprintf("\n\n📝 当前内容:\nKey: %s\nValue: %s", entry.Key, entry.Value)
	}

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "请输入新的内容："+currentInfo)
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("取消", "cancel")
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{cancelButton}}}
	bot.Send(editMsg)
}

func (h *CallbackHandler) handleDeleteCallback(bot *tgbotapi.BotAPI, _ *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	parts := strings.Split(strings.TrimPrefix(data, "delete_"), "_")
	if len(parts) != 2 {
		log.Printf("Error parsing entry ID and match type for delete: %s", data)
		return
	}

	entryID, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Printf("Error parsing entry ID for delete: %v", err)
		return
	}

	matchType, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Printf("Error parsing match type for delete: %v", err)
		return
	}

	entry, err := h.db.QueryByID(entryID, matchType)
	if err != nil {
		log.Printf("Error querying database: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "无法获取条目"))
		return
	}

	if entry == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "未找到条目"))
		return
	}

	// 显示确认删除界面
	confirmMsg := fmt.Sprintf("⚠️ 确认删除以下条目吗？\n\nKey: %s\nValue: %s\n\n此操作不可撤销！", entry.Key, entry.Value)

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ 确认删除", fmt.Sprintf("confirm_delete_%d_%d", entryID, matchType)),
			tgbotapi.NewInlineKeyboardButtonData("❌ 取消", fmt.Sprintf("entry_%d_%d", entryID, matchType)),
		},
	}

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, confirmMsg)
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	bot.Send(editMsg)
}

func (h *CallbackHandler) handleConfirmDeleteCallback(bot *tgbotapi.BotAPI, _ *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	parts := strings.Split(strings.TrimPrefix(data, "confirm_delete_"), "_")
	if len(parts) != 2 {
		log.Printf("Error parsing entry ID and match type for confirm delete: %s", data)
		return
	}

	entryID, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Printf("Error parsing entry ID for confirm delete: %v", err)
		return
	}

	matchType, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Printf("Error parsing match type for confirm delete: %v", err)
		return
	}

	// 获取条目信息用于记录
	entry, err := h.db.QueryByID(entryID, matchType)
	if err != nil {
		log.Printf("Error querying database: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "无法获取条目"))
		return
	}

	if entry == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "未找到条目"))
		return
	}

	// 执行删除操作
	err = h.db.DeleteEntry(entry.Key, matchType)
	if err != nil {
		log.Printf("Error deleting entry: %v", err)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ 删除失败："+err.Error())
		bot.Send(editMsg)
		return
	}

	// 删除成功
	successMsg := fmt.Sprintf("✅ 删除成功！\n\n已删除条目：\nKey: %s\nValue: %s", entry.Key, entry.Value)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, successMsg)
	returnButton := tgbotapi.NewInlineKeyboardButtonData("返回列表", "list_0")
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{returnButton}}}
	bot.Send(editMsg)
}

func (h *CallbackHandler) handleConfirmBatchDeleteCallback(bot *tgbotapi.BotAPI, _ *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	// 解析回调数据: confirm_batch_delete_<matchType>_<pattern>
	trimmed := strings.TrimPrefix(data, "confirm_batch_delete_")
	parts := strings.SplitN(trimmed, "_", 2)

	if len(parts) < 1 {
		bot.Send(tgbotapi.NewMessage(chatID, "参数错误"))
		return
	}

	matchType, err := strconv.Atoi(parts[0])
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "匹配类型错误"))
		return
	}

	var pattern string
	if len(parts) > 1 {
		pattern = parts[1]
	}

	// 重新获取符合条件的条目（防止数据变化）
	var entries []database.Entry
	if pattern == "" {
		entries, err = h.db.ListSpecificEntries(matchType)
	} else {
		allEntries, err := h.db.ListSpecificEntries(matchType)
		if err == nil {
			for _, entry := range allEntries {
				if strings.Contains(entry.Key, pattern) || strings.Contains(entry.Value, pattern) {
					entries = append(entries, entry)
				}
			}
		}
	}

	if err != nil {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ 获取条目列表失败："+err.Error())
		bot.Send(editMsg)
		return
	}

	if len(entries) == 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "ℹ️ 没有找到符合条件的条目")
		bot.Send(editMsg)
		return
	}

	// 执行批量删除
	successCount := 0
	var failedEntries []string

	for _, entry := range entries {
		err := h.db.DeleteEntry(entry.Key, entry.MatchType)
		if err != nil {
			failedEntries = append(failedEntries, fmt.Sprintf("%s (%s)", entry.Key, err.Error()))
		} else {
			successCount++
		}
	}

	// 显示结果
	var resultMsg string
	if successCount == len(entries) {
		resultMsg = fmt.Sprintf("✅ 批量删除成功！\n\n共删除了 %d 个条目", successCount)
	} else if successCount > 0 {
		resultMsg = fmt.Sprintf("⚠️ 部分删除成功\n\n成功：%d 个\n失败：%d 个", successCount, len(failedEntries))
		if len(failedEntries) <= 5 {
			resultMsg += "\n\n失败的条目："
			for _, failed := range failedEntries {
				resultMsg += "\n• " + failed
			}
		}
	} else {
		resultMsg = "❌ 批量删除失败\n\n所有条目删除都失败了"
	}

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, resultMsg)
	returnButton := tgbotapi.NewInlineKeyboardButtonData("返回列表", "list_0")
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{returnButton}},
	}
	bot.Send(editMsg)
}

func (h *CallbackHandler) handleListAdminCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, data string) {
	if data == "listadmin_0" {
		state, exists := h.state.Get(callbackQuery.Message.Chat.ID)
		if !exists || state.Stage != "listing_admin" {
			log.Printf("Conversation state not found for chat ID: %d", callbackQuery.Message.Chat.ID)
			return
		}
		messageID := state.MessageID
		h.adminHandler.HandleListAdminCommandEdit(bot, callbackQuery.Message, 0, messageID)
	} else {
		pageStr := strings.TrimPrefix(data, "listadmin_")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			log.Printf("Error parsing page number: %v", err)
			return
		}
		h.adminHandler.HandleListAdminCommand(bot, callbackQuery.Message, page)
	}
}

func (h *CallbackHandler) handleAdminCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, data string) {
	adminIDStr := strings.TrimPrefix(data, "admin_")
	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Printf("Error parsing admin ID: %v", err)
		return
	}
	h.adminHandler.HandleAdminSelection(bot, callbackQuery.Message, adminID)
}

func (h *CallbackHandler) handleDelAdminCallback(bot *tgbotapi.BotAPI, _ *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	adminIDStr := strings.TrimPrefix(data, "deladmin_")
	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Printf("Error parsing admin ID: %v", err)
		return
	}

	h.conf.Admin.AdminIDs = RemoveID(h.conf.Admin.AdminIDs, adminID)
	err = config.SaveConfig("config.json", h.conf)
	if err != nil {
		log.Printf("Error saving config: %v", err)
		bot.Send(tgbotapi.NewEditMessageText(chatID, messageID, "删除管理员失败"))
		return
	}

	msgText := fmt.Sprintf("已删除管理员 %d", adminID)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msgText)
	bot.Send(editMsg)
}

func (h *CallbackHandler) handleConfirmDeleteAllCallback(bot *tgbotapi.BotAPI, _ *tgbotapi.CallbackQuery, chatID int64, messageID int) {
	deleteErr := h.db.DeleteAllEntries()
	if deleteErr != nil {
		log.Printf("Error deleting entry: %v", deleteErr)
		bot.Send(tgbotapi.NewEditMessageText(chatID, messageID, "删除失败"))
		return
	}
	bot.Send(tgbotapi.NewEditMessageText(chatID, messageID, "已清空所有条目"))
}

func (h *CallbackHandler) handleCancelCallback(bot *tgbotapi.BotAPI, _ *tgbotapi.CallbackQuery, chatID int64, messageID int) {
	// 清理状态
	h.state.Delete(chatID)

	// 显示取消消息
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ 操作已取消")

	// 添加返回主菜单的选项
	mainMenuButton := tgbotapi.NewInlineKeyboardButtonData("🏠 返回主菜单", "main_menu")
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{mainMenuButton}},
	}

	bot.Send(editMsg)
}

func (h *CallbackHandler) handleModelCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	modelName := strings.TrimPrefix(data, "model:")

	// Note: In the new multichat system, model selection is handled automatically
	// based on provider configuration and availability

	// Acknowledge the callback
	_, err := bot.Request(tgbotapi.NewCallback(callbackQuery.ID, "模型设置已记录: "+modelName))
	if err != nil {
		log.Printf("Error acknowledging callback: %v", err)
	}

	// Edit the message
	msgText := fmt.Sprintf("模型偏好已记录: %s\n注意：实际使用的模型将根据当前可用的AI提供商自动选择", modelName)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msgText)
	bot.Send(editMsg)
}
