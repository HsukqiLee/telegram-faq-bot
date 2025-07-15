package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"TGFaqBot/config"
	"TGFaqBot/database"
	"TGFaqBot/multichat"
	"TGFaqBot/utils"
)

type CommandHandler struct {
	db               database.Database
	conf             *config.Config
	adminHandler     *AdminHandler
	listHandler      *ListHandler
	rateLimiter      *utils.RateLimiter
	multichatManager *multichat.Manager
	state            *State
	streamer         *StreamingManager
	prefManager      *PreferenceManager
}

func NewCommandHandler(db database.Database, conf *config.Config, adminHandler *AdminHandler, listHandler *ListHandler, multichatManager *multichat.Manager, state *State, streamer *StreamingManager, prefManager *PreferenceManager) *CommandHandler {
	return &CommandHandler{
		db:               db,
		conf:             conf,
		adminHandler:     adminHandler,
		listHandler:      listHandler,
		multichatManager: multichatManager,
		state:            state,
		streamer:         streamer,
		prefManager:      prefManager,
		rateLimiter:      utils.NewRateLimiter(),
	}
}

func (h *CommandHandler) HandleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// 检查速率限制（对非管理员用户）
	if !IsAdminUser(message.From.ID, h.conf) {
		if !h.rateLimiter.Allow(message.From.ID, 10, time.Minute) {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "请求过于频繁，请稍后再试"))
			return
		}
	}

	isAdmin := IsAdminUser(message.From.ID, h.conf)
	isSuperAdmin := IsSuperAdminUser(message.From.ID, h.conf)

	switch message.Command() {
	case "start":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, h.conf.Telegram.Introduction))
	case "query":
		h.handleQueryCommand(bot, message)
	case "userinfo":
		h.handleUserInfoCommand(bot, message)
	case "groupinfo":
		h.handleGroupInfoCommand(bot, message)
	case "clearchat":
		h.handleClearChatCommand(bot, message)
	case "models":
		h.handleModelsCommand(bot, message)
	case "retry":
		h.handleRetryCommand(bot, message)
	case "add", "update", "delete":
		if isAdmin {
			h.adminHandler.HandleAdminCommand(bot, message)
		} else {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无权限"))
		}
	case "batchdelete":
		if isAdmin {
			h.handleBatchDeleteCommand(bot, message)
		} else {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无权限"))
		}
	case "list":
		if isAdmin {
			h.listHandler.HandleListCommand(bot, message, 0)
		} else {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无权限"))
		}
	case "reload":
		if isAdmin {
			h.handleReloadCommand(bot, message)
		} else {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无权限"))
		}
	case "deleteall":
		if isAdmin {
			h.handleDeleteAllCommand(bot, message)
		} else {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无权限"))
		}
	case "commands":
		h.handleShowCommand(bot, message)
	case "tgtext":
		if isAdmin {
			h.handleTelegraphTextCommand(bot, message)
		} else {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无权限"))
		}
	case "tgimage":
		if isAdmin {
			h.handleTelegraphImageCommand(bot, message)
		} else {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无权限"))
		}
	case "addadmin", "deladmin", "addgroup", "delgroup", "listadmin":
		if isSuperAdmin {
			h.adminHandler.HandleSuperAdminCommand(bot, message)
		} else {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "无权限"))
		}
	default:
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "未知命令"))
	}
}

func (h *CommandHandler) handleQueryCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	if args == "" {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "格式错误，请使用：/query 关键词"))
		return
	}

	results, err := h.db.Query(args)
	if err != nil {
		log.Printf("Error querying database: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "查询失败"))
		return
	}

	if len(results) == 0 {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "未找到匹配结果"))
		return
	}

	for _, result := range results {
		msg := tgbotapi.NewMessage(message.Chat.ID, result.Value)
		msg.ParseMode = "HTML"
		bot.Send(msg)
	}
}

func (h *CommandHandler) handleUserInfoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	user := message.From
	userInfo := fmt.Sprintf(
		"用户ID: %d\n"+
			"用户名: %s\n"+
			"名: %s\n"+
			"姓: %s\n"+
			"语言代码: %s\n"+
			"是否是机器人: %t\n",
		user.ID,
		user.UserName,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
		user.IsBot,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, userInfo)
	bot.Send(msg)
}

func (h *CommandHandler) handleGroupInfoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if message.Chat.Type == "private" {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "此对话不是群组"))
		return
	}

	groupID := message.Chat.ID
	groupTitle := message.Chat.Title

	groupInfo := fmt.Sprintf(
		"群组ID: %d\n"+
			"群组名称: %s\n",
		groupID,
		groupTitle,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, groupInfo)
	bot.Send(msg)
}

func (h *CommandHandler) handleClearChatCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	h.multichatManager.ClearConversation(chatID)
	bot.Send(tgbotapi.NewMessage(chatID, "✅ 对话历史已清除"))
}

func (h *CommandHandler) handleModelsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// 检查是否有页码参数
	args := strings.Fields(message.Text)
	page := 1
	if len(args) > 1 {
		if p, err := strconv.Atoi(args[1]); err == nil && p > 0 {
			page = p
		}
	}

	h.sendModelsPage(bot, message.Chat.ID, 0, page) // messageID为0表示发送新消息
}

func (h *CommandHandler) sendModelsPage(bot *tgbotapi.BotAPI, chatID int64, messageID int, page int) {
	allModels, err := h.db.GetAllModels()
	if err != nil {
		var msg tgbotapi.Chattable
		if messageID > 0 {
			msg = tgbotapi.NewEditMessageText(chatID, messageID, "❌ 获取模型列表失败: "+err.Error())
		} else {
			msg = tgbotapi.NewMessage(chatID, "❌ 获取模型列表失败: "+err.Error())
		}
		bot.Send(msg)
		return
	}

	if len(allModels) == 0 {
		var msg tgbotapi.Chattable
		if messageID > 0 {
			msg = tgbotapi.NewEditMessageText(chatID, messageID, "📄 暂无可用模型，请先刷新模型列表")
		} else {
			msg = tgbotapi.NewMessage(chatID, "📄 暂无可用模型，请先刷新模型列表")
		}
		bot.Send(msg)
		return
	}

	// 将所有模型展平成一个列表
	var allModelsList []database.ModelInfo
	var providerMap = make(map[string]string) // 模型ID到提供商的映射

	for provider, models := range allModels {
		for _, model := range models {
			allModelsList = append(allModelsList, model)
			providerMap[model.ID] = provider
		}
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

	// 发送或编辑消息
	if messageID > 0 {
		// 编辑现有消息
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, response.String())
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
		bot.Send(editMsg)
	} else {
		// 发送新消息
		msg := tgbotapi.NewMessage(chatID, response.String())
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		bot.Send(msg)
	}
}

func (h *CommandHandler) handleReloadCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if err := h.db.Reload(); err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "重新加载数据库失败"))
	} else {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "数据库重新加载成功"))
	}

	newConfig, err := config.LoadConfig("config.json")
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "重新加载配置失败"))
	} else {
		*h.conf = *newConfig
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "配置重新加载成功"))
	}
}

func (h *CommandHandler) handleDeleteAllCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData("确认", "confirm_deleteall")},
		{tgbotapi.NewInlineKeyboardButtonData("取消", "cancel")},
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "确认要删除所有条目吗？此操作不可恢复。")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	bot.Send(msg)
}

func (h *CommandHandler) handleShowCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	isAdmin := IsAdminUser(message.From.ID, h.conf)
	isSuperAdmin := IsSuperAdminUser(message.From.ID, h.conf)

	commands := []string{
		"/start - 显示介绍信息",
		"/query - 查询关键词",
		"/commands - 显示用户权限和可用指令",
		"/userinfo - 查询我的信息",
		"/groupinfo - 查询群组信息",
		"/clearchat - 清除会话历史",
		"/models - 查看和选择AI模型",
		"/retry - 重新生成上一次AI回复",
	}

	userType := "普通用户"
	if isAdmin {
		userType = "管理员"
		commands = append(commands, []string{
			"/add - 添加条目",
			"/update - 更新条目",
			"/delete - 删除条目",
			"/list - 列出所有条目",
			"/reload - 重新加载数据库",
			"/deleteall - 删除所有条目",
		}...)
	}

	if isSuperAdmin {
		userType = "超级管理员"
		commands = append(commands, []string{
			"/addadmin - 添加管理员",
			"/deladmin - 删除管理员",
			"/listadmin - 列出管理员",
			"/addgroup - 添加群组",
			"/delgroup - 删除群组",
		}...)
	}

	response := fmt.Sprintf("用户权限：%s\n可用指令：\n%s", userType, strings.Join(commands, "\n"))
	bot.Send(tgbotapi.NewMessage(message.Chat.ID, response))
}

func (h *CommandHandler) handleBatchDeleteCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	if args == "" {
		helpMsg := `批量删除命令格式：
/batchdelete <匹配类型> [关键词模式]

匹配类型：
• exact: 精确匹配
• contains: 包含匹配  
• regex: 正则匹配
• prefix: 前缀匹配
• suffix: 后缀匹配

示例：
/batchdelete contains test  # 删除所有包含"test"的条目
/batchdelete exact          # 删除所有精确匹配类型的条目`
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, helpMsg))
		return
	}

	parts := strings.SplitN(args, " ", 2)
	matchTypeStr := parts[0]

	// 验证并转换匹配类型
	var matchType database.MatchType
	switch matchTypeStr {
	case "exact":
		matchType = database.MatchExact
	case "contains":
		matchType = database.MatchContains
	case "regex":
		matchType = database.MatchRegex
	case "prefix":
		matchType = database.MatchPrefix
	case "suffix":
		matchType = database.MatchSuffix
	default:
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "匹配类型错误，请使用 exact, contains, regex, prefix, suffix"))
		return
	}

	var pattern string
	if len(parts) > 1 {
		pattern = parts[1]
	}

	// 获取符合条件的条目
	var entries []database.Entry
	var err error
	if pattern == "" {
		// 获取指定类型的所有条目
		entries, err = h.db.ListSpecificEntries(matchType)
	} else {
		// 根据模式筛选条目
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
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "获取条目列表失败"))
		return
	}

	if len(entries) == 0 {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "没有找到符合条件的条目"))
		return
	}

	// 显示确认信息
	confirmMsg := fmt.Sprintf("⚠️ 即将删除 %d 个条目，确认继续吗？\n\n", len(entries))

	// 显示前5个条目作为预览
	previewCount := 5
	if len(entries) < previewCount {
		previewCount = len(entries)
	}

	confirmMsg += "预览（前5个）：\n"
	for i := 0; i < previewCount; i++ {
		confirmMsg += fmt.Sprintf("• %s\n", entries[i].Key)
	}

	if len(entries) > previewCount {
		confirmMsg += fmt.Sprintf("... 还有 %d 个条目\n", len(entries)-previewCount)
	}

	confirmMsg += "\n⚠️ 此操作不可撤销！"

	// 创建确认按钮
	confirmButton := tgbotapi.NewInlineKeyboardButtonData("✅ 确认批量删除", fmt.Sprintf("confirm_batch_delete_%d_%s", matchType.ToInt(), pattern))
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("❌ 取消", "cancel")

	msg := tgbotapi.NewMessage(message.Chat.ID, confirmMsg)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{confirmButton},
		[]tgbotapi.InlineKeyboardButton{cancelButton},
	)

	bot.Send(msg)
}

// handleTelegraphTextCommand 处理 Telegraph 文本命令
func (h *CommandHandler) handleTelegraphTextCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := strings.TrimSpace(strings.TrimPrefix(message.Text, "/tgtext"))
	if args == "" {
		helpMsg := `📝 Telegraph 文本页面创建命令格式：
/tgtext <匹配类型> <键名> <标题> <内容>

参数说明：
• 匹配类型：exact=精确匹配, contains=包含匹配, regex=正则匹配
• 键名：触发词
• 标题：Telegraph 页面标题
• 内容：页面文本内容

示例：
/tgtext exact help 帮助文档 这是详细的帮助文档内容...`
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, helpMsg))
		return
	}

	parts := strings.SplitN(args, " ", 4)
	if len(parts) < 4 {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ 参数不足，请使用格式：/tgtext <匹配类型> <键名> <标题> <内容>"))
		return
	}

	matchTypeStr := parts[0]
	// 验证并转换匹配类型
	var matchType database.MatchType
	switch matchTypeStr {
	case "exact":
		matchType = database.MatchExact
	case "contains":
		matchType = database.MatchContains
	case "regex":
		matchType = database.MatchRegex
	case "prefix":
		matchType = database.MatchPrefix
	case "suffix":
		matchType = database.MatchSuffix
	default:
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ 匹配类型错误，请使用 exact, contains, regex, prefix, suffix"))
		return
	}

	key := parts[1]
	title := parts[2]
	content := parts[3]

	// 创建 Telegraph 处理器
	telegraphHandler := NewTelegraphHandler(h.db)
	err := telegraphHandler.HandleTextUpload(key, matchType, title, content)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ 创建 Telegraph 页面失败：%v", err)))
		return
	}

	msg := fmt.Sprintf("✅ Telegraph 文本页面已创建：\n📝 键名：%s\n📄 标题：%s\n🔗 类型：%s", key, title, matchType)
	bot.Send(tgbotapi.NewMessage(message.Chat.ID, msg))
}

// handleTelegraphImageCommand 处理 Telegraph 图片命令
func (h *CommandHandler) handleTelegraphImageCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := strings.TrimSpace(strings.TrimPrefix(message.Text, "/tgimage"))
	if args == "" {
		helpMsg := `🖼️ Telegraph 图文页面创建命令格式：
/tgimage <匹配类型> <键名> <标题>

参数说明：
• 匹配类型：exact=精确匹配, contains=包含匹配, regex=正则匹配
• 键名：触发词
• 标题：Telegraph 页面标题

使用步骤：
1. 发送命令：/tgimage exact photo 图片展示
2. 然后发送一张图片（可以带文字说明）

示例：
/tgimage exact product 产品展示`
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, helpMsg))
		return
	}

	parts := strings.SplitN(args, " ", 3)
	if len(parts) < 3 {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ 参数不足，请使用格式：/tgimage <匹配类型> <键名> <标题>"))
		return
	}

	matchTypeStr := parts[0]
	// 验证并转换匹配类型
	var matchType database.MatchType
	switch matchTypeStr {
	case "exact":
		matchType = database.MatchExact
	case "contains":
		matchType = database.MatchContains
	case "regex":
		matchType = database.MatchRegex
	case "prefix":
		matchType = database.MatchPrefix
	case "suffix":
		matchType = database.MatchSuffix
	default:
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ 匹配类型错误，请使用 exact, contains, regex, prefix, suffix"))
		return
	}

	key := parts[1]
	title := parts[2]

	// 设置对话状态，等待用户发送图片
	h.state.Set(message.Chat.ID, &Conversation{
		Stage:           "awaiting_telegraph_image",
		TelegraphAction: "image",
		TelegraphKey:    key,
		TelegraphTitle:  title,
		MatchType:       matchType,
		CreatedAt:       time.Now(),
	})

	bot.Send(tgbotapi.NewMessage(message.Chat.ID,
		fmt.Sprintf("📤 请发送图片来创建 Telegraph 页面\n📝 键名：%s\n📄 标题：%s\n🔗 类型：%d", key, title, matchType.ToInt())))
}

func (h *CommandHandler) handleRetryCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// 检查是否有有效的会话
	chatID := message.Chat.ID
	lastInput := h.multichatManager.GetConversationManager().GetLastUserInput(chatID)

	if lastInput == "" {
		bot.Send(tgbotapi.NewMessage(chatID, "没有找到可重试的对话内容。"))
		return
	}

	// 获取用户偏好的模型
	var preferredProvider, preferredModel string
	if pref := h.prefManager.GetChatPreference(chatID); pref != nil {
		preferredProvider = pref.Provider
		preferredModel = pref.ModelID
	}

	// 发送"正在思考"消息
	thinkingMsg := tgbotapi.NewMessage(chatID, "🤔 正在重新生成回复...")
	sentMsg, err := bot.Send(thinkingMsg)
	if err != nil {
		log.Printf("Error sending thinking message: %v", err)
		return
	}

	// 创建流式更新管理器
	streamKey := fmt.Sprintf("%d_%d", chatID, sentMsg.MessageID)
	h.streamer.CreateStream(streamKey, chatID, sentMsg.MessageID)

	// 使用回调获取AI回复（重试）
	callback := func(content string, isComplete bool) bool {
		h.streamer.UpdateStream(bot, streamKey, content, isComplete, nil)
		return true // 继续接收更新
	}

	response, inputTokens, outputTokens, duration, remainingRounds, shouldReset, usedProvider, err := h.multichatManager.GetConversationManager().RetryLastMessageWithCallback(
		chatID, preferredProvider, preferredModel, callback,
	)

	if err != nil {
		log.Printf("Error getting retry response: %v", err)
		editMsg := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "❌ 重试失败，请稍后再试")
		bot.Send(editMsg)
		h.streamer.DeleteStream(streamKey)
		return
	}

	// 创建统计信息并追加
	stats := &ChatStats{
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		RemainingRounds: remainingRounds,
		Duration:        duration,
		Provider:        usedProvider,
		Model:           preferredModel,
		TTL:             24 * time.Hour, // 默认24小时TTL
	}

	// 追加统计信息
	h.streamer.AppendStats(bot, streamKey, stats)

	// 清理流式管理器
	defer h.streamer.DeleteStream(streamKey)

	// 记录重试操作
	log.Printf("Retry completed - Chat ID: %d, Provider: %s, Model: %s, Input tokens: %d, Output tokens: %d, Duration: %v, Remaining rounds: %d",
		chatID, usedProvider, preferredModel, inputTokens, outputTokens, duration, remainingRounds)

	// 如果达到对话上限，发送提示
	if shouldReset {
		resetMsg := fmt.Sprintf("\n\n⚠️ 已达到 %d 轮对话上限，会话将重置", h.conf.Chat.HistoryLength)
		finalResponse := response + resetMsg

		editMsg := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, finalResponse)
		// 尝试MarkdownV2格式
		editMsg.ParseMode = "MarkdownV2"
		if _, err := bot.Send(editMsg); err != nil {
			// 如果失败，回退到普通文本
			editMsg.ParseMode = ""
			editMsg.Text = cleanTextForPlain(finalResponse)
			bot.Send(editMsg)
		}

		// 重置对话
		h.multichatManager.GetConversationManager().ClearConversation(chatID, h.conf.Chat.SystemPrompt)
	}
}
