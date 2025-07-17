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
)

type CallbackHandler struct {
	db           database.Database
	conf         *config.Config
	state        *State
	adminHandler *AdminHandler
	listHandler  *ListHandler
	prefManager  *PreferenceManager
	multichatMgr *multichat.Manager
}

func NewCallbackHandler(db database.Database, conf *config.Config, state *State, prefManager *PreferenceManager, multichatMgr *multichat.Manager) *CallbackHandler {
	return &CallbackHandler{
		db:           db,
		conf:         conf,
		state:        state,
		adminHandler: NewAdminHandler(db, conf, state),
		listHandler:  NewListHandler(db, state),
		prefManager:  prefManager,
		multichatMgr: multichatMgr,
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
	case strings.HasPrefix(data, "models_page_"):
		h.handleModelsPageCallback(bot, callbackQuery, data, chatID, messageID)
	case data == "refresh_models":
		h.handleRefreshModelsCallback(bot, callbackQuery, chatID, messageID)
	case data == "models_current":
		// 当前页按钮，不做任何操作
		return
	case strings.HasPrefix(data, "select_model_"):
		h.handleSelectModelCallback(bot, callbackQuery, data, chatID, messageID)
	case data == "clear_model_preference":
		h.handleClearModelPreferenceCallback(bot, callbackQuery, chatID, messageID)
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
			tgbotapi.NewInlineKeyboardButtonData("前缀", fmt.Sprintf("update_type_%d_%d_%d", entryID, matchType, 4)),
			tgbotapi.NewInlineKeyboardButtonData("后缀", fmt.Sprintf("update_type_%d_%d_%d", entryID, matchType, 5)),
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
	oldTypeValue, err := database.MatchTypeFromInt(oldType)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "类型转换错误"))
		return
	}

	entry, err := h.db.QueryByID(entryID, oldTypeValue)
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

	matchTypeValue, err := database.MatchTypeFromInt(matchType)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "匹配类型转换错误"))
		return
	}

	entry, err := h.db.QueryByID(entryID, matchTypeValue)
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

	matchTypeValue, err := database.MatchTypeFromInt(matchType)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "匹配类型转换错误"))
		return
	}

	// 获取条目信息用于记录
	entry, err := h.db.QueryByID(entryID, matchTypeValue)
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
	err = h.db.DeleteEntry(entry.Key, matchTypeValue)
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

	matchTypeValue, err := database.MatchTypeFromInt(matchType)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "匹配类型转换错误"))
		return
	}

	var pattern string
	if len(parts) > 1 {
		pattern = parts[1]
	}

	// 重新获取符合条件的条目（防止数据变化）
	var entries []database.Entry
	if pattern == "" {
		entries, err = h.db.ListSpecificEntries(matchTypeValue)
	} else {
		allEntries, err := h.db.ListSpecificEntries(matchTypeValue)
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
		h.adminHandler.HandleListAdminCommandEdit(bot, callbackQuery.Message, page, 0)
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

func (h *CallbackHandler) handleModelsPageCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	pageStr := strings.TrimPrefix(data, "models_page_")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Printf("Error parsing models page number: %v", err)
		return
	}

	h.sendModelsPage(bot, chatID, messageID, page)

	// 确认回调
	bot.Request(tgbotapi.NewCallback(callbackQuery.ID, ""))
}

func (h *CallbackHandler) handleRefreshModelsCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, chatID int64, messageID int) {
	h.sendModelsPage(bot, chatID, messageID, 1)

	// 确认回调
	bot.Request(tgbotapi.NewCallback(callbackQuery.ID, "模型列表已刷新"))
}

func (h *CallbackHandler) sendModelsPage(bot *tgbotapi.BotAPI, chatID int64, messageID int, page int) {
	allModels, err := h.db.GetAllModels()
	if err != nil {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ 获取模型列表失败: "+err.Error())
		bot.Send(editMsg)
		return
	}

	if len(allModels) == 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "📄 暂无可用模型，请先刷新模型列表")
		bot.Send(editMsg)
		return
	}

	// 获取当前可用的提供商列表
	availableProviders := h.multichatMgr.GetAvailableProviders()
	if len(availableProviders) == 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "❌ 没有可用的AI提供商")
		bot.Send(editMsg)
		return
	}

	// 只包含可用提供商的模型
	var allModelsList []database.ModelInfo
	var providerMap = make(map[string]string) // 模型ID到提供商的映射

	for _, provider := range availableProviders {
		if models, exists := allModels[provider]; exists {
			for _, model := range models {
				allModelsList = append(allModelsList, model)
				providerMap[model.ID] = provider
			}
		}
	}

	if len(allModelsList) == 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, "📄 当前可用提供商没有模型，请刷新模型列表")
		bot.Send(editMsg)
		return
	}

	// 分页设置
	const modelsPerPage = 20
	totalModels := len(allModelsList)
	totalPages := (totalModels + modelsPerPage - 1) / modelsPerPage

	if page > totalPages {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	// 计算当前页的模型范围
	startIdx := (page - 1) * modelsPerPage
	endIdx := startIdx + modelsPerPage
	if endIdx > totalModels {
		endIdx = totalModels
	}

	// 构建响应消息
	var response strings.Builder
	response.WriteString(fmt.Sprintf("🤖 可用模型列表 (第 %d/%d 页)\n", page, totalPages))

	// 显示当前偏好
	if pref := h.prefManager.GetChatPreference(chatID); pref != nil {
		response.WriteString(fmt.Sprintf("当前偏好：%s (%s)\n", pref.ModelName, strings.ToUpper(pref.Provider)))
	} else {
		response.WriteString("当前偏好：未设置（使用默认策略）\n")
	}

	response.WriteString("点击模型名称来选择使用\n\n")

	// 构建模型选择按钮
	var buttons [][]tgbotapi.InlineKeyboardButton
	var modelButtons []tgbotapi.InlineKeyboardButton

	currentProvider := ""
	buttonCount := 0
	for i := startIdx; i < endIdx; i++ {
		model := allModelsList[i]
		provider := providerMap[model.ID]

		// 如果是新的提供商，添加提供商标题
		if provider != currentProvider {
			// 如果有未完成的按钮行，先添加到buttons中
			if len(modelButtons) > 0 {
				buttons = append(buttons, modelButtons)
				modelButtons = nil
			}

			if currentProvider != "" {
				response.WriteString("\n")
			}
			response.WriteString(fmt.Sprintf("**%s**\n", strings.ToUpper(provider)))
			currentProvider = provider
		}

		// 添加模型信息到消息文本
		response.WriteString(fmt.Sprintf("  • %s", model.Name))
		if model.Description != "" {
			response.WriteString(fmt.Sprintf(" - %s", model.Description))
		}
		response.WriteString("\n")

		// 创建模型选择按钮（简化名称以适应按钮宽度）
		buttonText := model.Name
		if len(buttonText) > 20 {
			buttonText = buttonText[:17] + "..."
		}
		modelButtons = append(modelButtons,
			tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("select_model_%s", model.ID)))
		buttonCount++

		// 每行最多2个按钮
		if len(modelButtons) >= 2 {
			buttons = append(buttons, modelButtons)
			modelButtons = nil
		}
	}

	// 添加剩余的模型按钮
	if len(modelButtons) > 0 {
		buttons = append(buttons, modelButtons)
	}
	var pageButtons []tgbotapi.InlineKeyboardButton

	if page > 1 {
		pageButtons = append(pageButtons,
			tgbotapi.NewInlineKeyboardButtonData("⬅️ 上一页", fmt.Sprintf("models_page_%d", page-1)))
	}

	pageButtons = append(pageButtons,
		tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", page, totalPages), "models_current"))

	if page < totalPages {
		pageButtons = append(pageButtons,
			tgbotapi.NewInlineKeyboardButtonData("下一页 ➡️", fmt.Sprintf("models_page_%d", page+1)))
	}

	buttons = append(buttons, pageButtons)

	// 添加刷新按钮
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔄 刷新模型列表", "refresh_models"),
	})

	// 编辑消息
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, response.String())
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	bot.Send(editMsg)
}

func (h *CallbackHandler) handleSelectModelCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, data string, chatID int64, messageID int) {
	modelID := strings.TrimPrefix(data, "select_model_")

	// 获取模型信息进行显示
	allModels, err := h.db.GetAllModels()
	if err != nil {
		bot.Request(tgbotapi.NewCallback(callbackQuery.ID, "获取模型信息失败"))
		return
	}

	var selectedModel *database.ModelInfo
	var selectedProvider string

	// 查找选中的模型
	for provider, models := range allModels {
		for _, model := range models {
			if model.ID == modelID {
				selectedModel = &model
				selectedProvider = provider
				break
			}
		}
		if selectedModel != nil {
			break
		}
	}

	if selectedModel == nil {
		bot.Request(tgbotapi.NewCallback(callbackQuery.ID, "未找到选中的模型"))
		return
	}

	// 验证提供商是否可用
	if !h.multichatMgr.IsProviderAvailable(selectedProvider) {
		bot.Request(tgbotapi.NewCallback(callbackQuery.ID, "该模型的提供商当前不可用"))

		// 显示错误信息
		msgText := fmt.Sprintf("❌ 模型选择失败\n\n🤖 模型：%s\n🏢 提供商：%s\n\n⚠️ 该提供商当前不可用，请选择其他模型",
			selectedModel.Name, strings.ToUpper(selectedProvider))

		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msgText)
		backButton := tgbotapi.NewInlineKeyboardButtonData("← 返回模型列表", "models_page_1")
		editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{backButton}},
		}
		bot.Send(editMsg)
		return
	}

	// 存储聊天的模型偏好
	h.prefManager.SetChatPreference(chatID, selectedModel.ID, selectedProvider, selectedModel.Name)

	// 显示模型选择确认
	msgText := fmt.Sprintf("✅ 已选择模型：\n\n🤖 **%s**\n🏢 提供商：%s",
		selectedModel.Name, strings.ToUpper(selectedProvider))

	if selectedModel.Description != "" {
		msgText += fmt.Sprintf("\n📋 描述：%s", selectedModel.Description)
	}

	msgText += "\n\n💡 此模型偏好已保存到当前聊天会话。当有多个AI提供商可用时，系统将优先尝试使用您选择的模型。"

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msgText)
	editMsg.ParseMode = "Markdown"

	// 添加返回按钮和清除偏好按钮
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("← 返回模型列表", "models_page_1"),
			tgbotapi.NewInlineKeyboardButtonData("🗑️ 清除偏好", "clear_model_preference"),
		},
	}
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}

	bot.Send(editMsg)

	// 确认回调
	bot.Request(tgbotapi.NewCallback(callbackQuery.ID, fmt.Sprintf("已选择：%s", selectedModel.Name)))
}

func (h *CallbackHandler) handleClearModelPreferenceCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, chatID int64, messageID int) {
	// 清除聊天的模型偏好
	h.prefManager.ClearChatPreference(chatID)

	// 显示确认消息
	msgText := "🗑️ 已清除模型偏好\n\n💡 系统将使用默认的多AI提供商策略来选择模型"

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, msgText)

	// 添加返回按钮
	backButton := tgbotapi.NewInlineKeyboardButtonData("← 返回模型列表", "models_page_1")
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{backButton}},
	}

	bot.Send(editMsg)

	// 确认回调
	bot.Request(tgbotapi.NewCallback(callbackQuery.ID, "已清除模型偏好"))
}
