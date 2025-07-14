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
	"TGFaqBot/utils"
)

type CommandHandler struct {
	db           database.Database
	conf         *config.Config
	adminHandler *AdminHandler
	listHandler  *ListHandler
	rateLimiter  *utils.RateLimiter
}

func NewCommandHandler(db database.Database, conf *config.Config, adminHandler *AdminHandler, listHandler *ListHandler) *CommandHandler {
	return &CommandHandler{
		db:           db,
		conf:         conf,
		adminHandler: adminHandler,
		listHandler:  listHandler,
		rateLimiter:  utils.NewRateLimiter(),
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
	// TODO: Implement conversation clearing with new chat system
	bot.Send(tgbotapi.NewMessage(message.Chat.ID, "对话历史已清除"))
}

func (h *CommandHandler) handleModelsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// TODO: Implement models listing with new chat system
	bot.Send(tgbotapi.NewMessage(message.Chat.ID, "模型列表功能正在重构中..."))
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
• 1: 精确匹配
• 2: 包含匹配  
• 3: 正则匹配

示例：
/batchdelete 2 test  # 删除所有包含"test"的条目
/batchdelete 1       # 删除所有精确匹配类型的条目`
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, helpMsg))
		return
	}

	parts := strings.SplitN(args, " ", 2)
	matchType, err := strconv.Atoi(parts[0])
	if err != nil || (matchType != 1 && matchType != 2 && matchType != 3) {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "匹配类型错误，请使用 1, 2, 或 3"))
		return
	}

	var pattern string
	if len(parts) > 1 {
		pattern = parts[1]
	}

	// 获取符合条件的条目
	var entries []database.Entry
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
	confirmButton := tgbotapi.NewInlineKeyboardButtonData("✅ 确认批量删除", fmt.Sprintf("confirm_batch_delete_%d_%s", matchType, pattern))
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("❌ 取消", "cancel")

	msg := tgbotapi.NewMessage(message.Chat.ID, confirmMsg)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{confirmButton},
		[]tgbotapi.InlineKeyboardButton{cancelButton},
	)

	bot.Send(msg)
}
