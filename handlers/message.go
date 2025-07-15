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
	"TGFaqBot/multichat/provider"
	"TGFaqBot/utils"
)

type MessageHandler struct {
	db               database.Database
	conf             *config.Config
	state            *State
	streamer         *StreamingManager
	multichatManager *multichat.Manager
	telegraphHandler *TelegraphHandler
	prefManager      *PreferenceManager
}

func NewMessageHandler(db database.Database, conf *config.Config, state *State, streamer *StreamingManager, multichatMgr *multichat.Manager, prefManager *PreferenceManager) *MessageHandler {
	return &MessageHandler{
		db:               db,
		conf:             conf,
		state:            state,
		streamer:         streamer,
		multichatManager: multichatMgr,
		telegraphHandler: NewTelegraphHandler(db),
		prefManager:      prefManager,
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
	case "awaiting_value":
		// 用户直接输入新的内容（类型已在按钮中选择）
		h.handleValueInput(bot, message, state)

	case "awaiting_type_and_value":
		// 用户应输入: 类型+空格+内容，例如 "2 新内容"
		parts := strings.SplitN(strings.TrimSpace(message.Text), " ", 2)
		if len(parts) < 2 {
			bot.Send(tgbotapi.NewMessage(chatID, "请输入类型和新内容，例如：2 新内容"))
			return
		}
		newType, err := strconv.Atoi(parts[0])
		if err != nil || (newType < 1 || newType > 5) {
			bot.Send(tgbotapi.NewMessage(chatID, "类型输入不合法，请输入1-5之间的数字。例如：2 新内容"))
			return
		}
		state.NewType = newType
		// 直接调用handleValueInput，内容为parts[1]
		fakeMsg := *message
		fakeMsg.Text = parts[1]
		h.handleValueInput(bot, &fakeMsg, state)

	case "awaiting_telegraph_text_content":
		// 处理 Telegraph 文本内容
		h.handleTelegraphTextContent(bot, message, state)

	case "awaiting_telegraph_image":
		// 处理 Telegraph 图片上传
		h.handleTelegraphImageContent(bot, message, state)
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
	oldTypeValue, err := database.MatchTypeFromInt(oldType)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "类型转换错误"))
		h.state.Delete(chatID)
		return
	}

	entry, err := h.db.QueryByID(entryID, oldTypeValue)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "未找到条目"))
		h.state.Delete(chatID)
		return
	}

	// Update the entry
	newTypeValue, err := database.MatchTypeFromInt(newType)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "类型转换错误"))
		h.state.Delete(chatID)
		return
	}

	err = h.db.UpdateEntry(entry.Key, oldTypeValue, newTypeValue, newValue)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "更新失败"))
		h.state.Delete(chatID)
		return
	}

	matchTypeText := utils.GetMatchTypeText(newTypeValue)

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

		// 获取AI响应 - 使用真正的流式响应
		_, shouldReset, stats, err := h.getAIReplyWithStreaming(userMessage, message, func(partialContent string, isComplete bool) bool {
			// 更新流式消息
			h.streamer.UpdateStream(bot, streamKey, partialContent, isComplete, nil)
			return true // 继续流式传输
		})

		// 在流式响应完成后，追加统计信息
		if err == nil && stats != nil {
			h.streamer.AppendStats(bot, streamKey, stats)
		}

		if err != nil {
			// 检查用户是否是管理员
			isAdmin := IsAdminUser(message.From.ID, h.conf)

			var errorMsg string
			if isAdmin {
				// 管理员显示详细错误信息，包括诊断信息
				providerCount := h.multichatManager.GetProviderCount()
				providerNames := h.multichatManager.GetProviderNames()
				diagnosticInfo := h.multichatManager.GetDiagnosticInfo()

				errorMsg = fmt.Sprintf("❌ AI服务错误（管理员详情）：\n\n🔍 错误详情：\n%s\n\n📊 系统诊断：\n• 可用提供商数量：%d\n• 提供商列表：%v\n• 详细信息：%s\n\n👤 用户信息：\n• 用户ID：%d\n• 聊天ID：%d\n• 时间：%s",
					err.Error(),
					providerCount,
					providerNames,
					diagnosticInfo,
					message.From.ID,
					chatID,
					time.Now().Format("2006-01-02 15:04:05"))
			} else {
				// 普通用户显示简化错误信息
				errorMsg = "❌ AI服务暂时不可用，请稍后再试"
			}

			// 编辑消息为错误信息
			editMsg := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, errorMsg)
			bot.Send(editMsg)
			return
		}

		// 处理对话重置
		if shouldReset {
			h.multichatManager.ClearConversation(chatID)
			resetMsg := tgbotapi.NewMessage(chatID, "⚠️ 连续对话达到上限，已自动重置对话历史")
			bot.Send(resetMsg)
		}
	}()
}

// getAIReplyWithStreaming 获取AI回复，支持流式回调（支持所有提供商）
func (h *MessageHandler) getAIReplyWithStreaming(userMessage string, message *tgbotapi.Message, callback func(string, bool) bool) (string, bool, *ChatStats, error) {
	chatID := message.Chat.ID

	// 记录开始时间
	startTime := time.Now()

	// 获取用户的模型偏好
	var preferredProvider, preferredModel string
	if pref := h.prefManager.GetChatPreference(chatID); pref != nil {
		preferredProvider = pref.Provider
		preferredModel = pref.ModelID
		log.Printf("Using preferred model for chat %d: %s (%s)", chatID, preferredModel, preferredProvider)
	}

	// 使用新的multichat系统获取响应，支持真正的流式回调
	response, inputTokens, outputTokens, duration, remainingRounds, shouldReset, usedProvider, err := h.multichatManager.GetResponseWithCallback(chatID, userMessage, preferredProvider, preferredModel, callback)
	if err != nil {
		return "", false, nil, fmt.Errorf("failed to get AI response: %v", err)
	}

	// 计算实际使用的模型
	actualModel := preferredModel
	if actualModel == "" {
		// 获取实际使用的默认模型ID
		actualModel = h.multichatManager.GetService().GetDefaultModel(usedProvider)
	}

	// 创建统计信息
	stats := &ChatStats{
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		RemainingRounds: remainingRounds,
		Duration:        time.Since(startTime),
		Provider:        usedProvider,
		Model:           actualModel,
		TTL:             24 * time.Hour, // 默认24小时TTL，可以从配置读取
		IsCachedReply:   inputTokens == 0 && outputTokens == 0 && strings.Contains(response, "💾 缓存回复"), // 判断是否为缓存回复
	}

	// 格式化响应（包含token信息等）
	formattedResponse := h.multichatManager.FormatResponse(
		&provider.ChatResponse{
			Content:      response,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Provider:     usedProvider,
		},
		duration,
		0, 0, // totalInput, totalOutput 由Manager内部管理
		remainingRounds, 0, 0, // remainingMinutes, remainingSeconds 由Manager内部管理
	)

	return formattedResponse, shouldReset, stats, nil
}

// handleTelegraphTextContent 处理 Telegraph 文本内容输入
func (h *MessageHandler) handleTelegraphTextContent(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *Conversation) {
	chatID := message.Chat.ID
	content := message.Text

	// 创建 Telegraph 文本页面
	err := h.telegraphHandler.HandleTextUpload(state.TelegraphKey, state.MatchType, state.TelegraphTitle, content)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ 创建 Telegraph 页面失败：%v", err)))
		h.state.Delete(chatID)
		return
	}

	// 发送成功消息
	msg := fmt.Sprintf("✅ Telegraph 文本页面已创建：\n📝 键名：%s\n📄 标题：%s\n🔗 类型：%d",
		state.TelegraphKey, state.TelegraphTitle, state.MatchType.ToInt())
	bot.Send(tgbotapi.NewMessage(chatID, msg))

	// 清除状态
	h.state.Delete(chatID)
}

// handleTelegraphImageContent 处理 Telegraph 图片内容
func (h *MessageHandler) handleTelegraphImageContent(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *Conversation) {
	chatID := message.Chat.ID

	if message.Photo == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ 请发送一张图片"))
		return
	}

	// 处理图片上传
	err := h.telegraphHandler.HandleImageUpload(bot, message, state.TelegraphKey, state.MatchType, state.TelegraphTitle)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ 上传图片失败：%v", err)))
		h.state.Delete(chatID)
		return
	}

	// 发送成功消息
	msg := fmt.Sprintf("✅ Telegraph 图文页面已创建：\n📝 键名：%s\n📄 标题：%s\n🔗 类型：%d",
		state.TelegraphKey, state.TelegraphTitle, state.MatchType.ToInt())
	bot.Send(tgbotapi.NewMessage(chatID, msg))

	// 清除状态
	h.state.Delete(chatID)
}
