package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"

	"TGFaqBot/config"
	"TGFaqBot/database"
	"TGFaqBot/handlers"
	"TGFaqBot/multichat"
)

// TelegramBot Bot实例管理器
type TelegramBot struct {
	bot             *tgbotapi.BotAPI
	conf            *config.Config
	db              database.Database
	multichatMgr    *multichat.Manager
	state           *handlers.State
	streamer        *handlers.StreamingManager
	monitor         *Monitor
	commandHandler  *handlers.CommandHandler
	callbackHandler *handlers.CallbackHandler
	messageHandler  *handlers.MessageHandler
	adminHandler    *handlers.AdminHandler
	listHandler     *handlers.ListHandler
}

// NewTelegramBot 创建新的Bot实例
func NewTelegramBot(conf *config.Config, db database.Database, multichatMgr *multichat.Manager) (*TelegramBot, error) {
	bot, err := tgbotapi.NewBotAPI(conf.Telegram.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %v", err)
	}

	bot.Debug = conf.Telegram.Debug

	// 初始化组件
	state := handlers.NewState()
	streamer := handlers.NewStreamingManager()
	monitor := NewMonitor(streamer)

	adminHandler := handlers.NewAdminHandler(db, conf, state)
	listHandler := handlers.NewListHandler(db, state)
	commandHandler := handlers.NewCommandHandler(db, conf, adminHandler, listHandler)
	callbackHandler := handlers.NewCallbackHandler(db, conf, state)
	messageHandler := handlers.NewMessageHandler(db, conf, state, streamer, multichatMgr)

	return &TelegramBot{
		bot:             bot,
		conf:            conf,
		db:              db,
		multichatMgr:    multichatMgr,
		state:           state,
		streamer:        streamer,
		monitor:         monitor,
		commandHandler:  commandHandler,
		callbackHandler: callbackHandler,
		messageHandler:  messageHandler,
		adminHandler:    adminHandler,
		listHandler:     listHandler,
	}, nil
}

// Start 启动Bot
func (tb *TelegramBot) Start() error {
	log.Printf("Bot started successfully. Debug mode: %v", tb.bot.Debug)

	// 启动性能优化服务
	tb.monitor.StartOptimizations()

	// 启动超时清理机制
	tb.startTimeoutCleanup()

	// 注册命令
	if err := tb.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %v", err)
	}

	switch tb.conf.Telegram.Mode {
	case "webhook":
		return tb.startWebhook()
	case "getupdates":
		return tb.startPolling()
	default:
		return fmt.Errorf("invalid mode: %s. Must be 'webhook' or 'getupdates'", tb.conf.Telegram.Mode)
	}
}

// registerCommands 注册Bot命令
func (tb *TelegramBot) registerCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "显示介绍信息"},
		{Command: "query", Description: "查询关键词"},
		{Command: "commands", Description: "显示用户权限和可用指令"},
		{Command: "userinfo", Description: "显示用户信息"},
		{Command: "groupinfo", Description: "显示群组信息"},
		{Command: "models", Description: "显示可用的AI模型"},
		{Command: "clearchat", Description: "清除当前对话历史"},
	}

	if len(tb.conf.Admin.AdminIDs) > 0 || len(tb.conf.Admin.SuperAdminIDs) > 0 {
		commands = append(commands, []tgbotapi.BotCommand{
			{Command: "add", Description: "添加条目"},
			{Command: "update", Description: "更新条目"},
			{Command: "delete", Description: "删除条目"},
			{Command: "list", Description: "列出所有条目"},
			{Command: "reload", Description: "重新加载数据库"},
			{Command: "deleteall", Description: "删除所有条目"},
		}...)
	}

	_, err := tb.bot.Request(tgbotapi.NewSetMyCommands(commands...))
	return err
}

// startWebhook 启动Webhook模式
func (tb *TelegramBot) startWebhook() error {
	// Set Webhook
	webhookConfig, _ := tgbotapi.NewWebhook(tb.conf.Telegram.WebhookURL)
	_, err := tb.bot.Request(webhookConfig)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %v", err)
	}

	info, err := tb.bot.GetWebhookInfo()
	if err != nil {
		return fmt.Errorf("failed to get webhook info: %v", err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}

	// Start HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		update := tgbotapi.Update{}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			log.Printf("Error decoding update: %v", err)
			return
		}

		tb.processUpdate(&update)
	}).Methods("POST")

	return http.ListenAndServe(fmt.Sprintf(":%d", tb.conf.Telegram.WebhookPort), router)
}

// startTimeoutCleanup 启动超时清理机制
func (tb *TelegramBot) startTimeoutCleanup() {
	go func() {
		ticker := time.NewTicker(2 * time.Minute) // 每2分钟检查一次
		defer ticker.Stop()

		for range ticker.C {
			expiredChats := tb.state.CleanupExpired(5 * time.Minute) // 5分钟超时
			for _, chatID := range expiredChats {
				// 通知用户操作已超时
				msg := tgbotapi.NewMessage(chatID, "⏰ 操作超时已自动取消，如需继续请重新开始操作")
				tb.bot.Send(msg)
				log.Printf("Cleaned up expired conversation state for chat %d", chatID)
			}
		}
	}()
}

// startPolling 启动轮询模式
func (tb *TelegramBot) startPolling() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := tb.bot.GetUpdatesChan(u)

	for update := range updates {
		tb.processUpdate(&update)
	}

	return nil
}

// processUpdate 处理更新
func (tb *TelegramBot) processUpdate(update *tgbotapi.Update) {
	if update.Message != nil {
		tb.processMessage(update.Message)
	} else if update.CallbackQuery != nil {
		tb.callbackHandler.HandleCallbackQuery(tb.bot, update.CallbackQuery)
	}
}

// processMessage 处理消息
func (tb *TelegramBot) processMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	isGroup := message.Chat.Type == "group" || message.Chat.Type == "supergroup"
	isAllowedGroup := false

	if isGroup {
		for _, groupID := range tb.conf.Admin.AllowedGroupIDs {
			if groupID == chatID {
				isAllowedGroup = true
				break
			}
		}
	}

	// Check if the group is allowed or if it's a private chat
	if !isGroup || isAllowedGroup {
		// Check if the message mentions the bot (only in allowed groups)
		if !isGroup || tb.mentionsBot(message) || message.IsCommand() {
			if message.IsCommand() {
				tb.commandHandler.HandleCommand(tb.bot, message)
			} else {
				// Handle messages concurrently to avoid blocking
				go tb.messageHandler.HandleMessage(tb.bot, message)
			}
		}
	} else {
		// If the group is not allowed, only process certain commands
		allowedCommands := []string{"start", "commands", "userinfo", "groupinfo", "addgroup"}
		if message.IsCommand() && tb.isCommandAllowed(message.Command(), allowedCommands) {
			tb.commandHandler.HandleCommand(tb.bot, message)
		}
	}
}

// mentionsBot 检查消息是否提及Bot
func (tb *TelegramBot) mentionsBot(message *tgbotapi.Message) bool {
	for _, entity := range message.Entities {
		if entity.Type == "mention" {
			offset := entity.Offset
			length := entity.Length
			if offset < len(message.Text) && offset+length <= len(message.Text) {
				mention := message.Text[offset : offset+length]
				if mention == "@"+tb.bot.Self.UserName {
					return true
				}
			}
		}
	}
	return false
}

// isCommandAllowed 检查命令是否在允许列表中
func (tb *TelegramBot) isCommandAllowed(command string, allowedCommands []string) bool {
	for _, allowed := range allowedCommands {
		if command == allowed {
			return true
		}
	}
	return false
}

// cleanupExpiredSessions 清理过期会话 - 已移动到 startTimeoutCleanup 中

// GetBot 获取Bot实例
func (tb *TelegramBot) GetBot() *tgbotapi.BotAPI {
	return tb.bot
}

// Close 关闭Bot
func (tb *TelegramBot) Close() error {
	if tb.db != nil {
		return tb.db.Close()
	}
	return nil
}
