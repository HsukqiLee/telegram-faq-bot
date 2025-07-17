package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"TGFaqBot/config"
	"TGFaqBot/database"
	"TGFaqBot/utils"
)

// IsSuperAdminUser 检查用户是否为超级管理员
func IsSuperAdminUser(userID int64, conf *config.Config) bool {
	for _, id := range conf.Admin.SuperAdminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// IsAdminUser 检查用户是否为管理员（包括超级管理员）
func IsAdminUser(userID int64, conf *config.Config) bool {
	for _, id := range conf.Admin.SuperAdminIDs {
		if id == userID {
			return true
		}
	}
	for _, id := range conf.Admin.AdminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// RemoveID 从切片中移除指定ID
func RemoveID(slice []int64, id int64) []int64 {
	for i, v := range slice {
		if v == id {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

type AdminHandler struct {
	db    database.Database
	conf  *config.Config
	state *State
}

func NewAdminHandler(db database.Database, conf *config.Config, state *State) *AdminHandler {
	return &AdminHandler{
		db:    db,
		conf:  conf,
		state: state,
	}
}

func (h *AdminHandler) HandleAdminCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	parts := strings.SplitN(args, " ", 3)

	if message.Command() == "delete" {
		if len(parts) < 2 {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "格式错误，请使用：/delete key type\n其中，type 的取值可以是：\n• exact: 表示精确匹配\n• contains: 表示包含匹配\n• regex: 表示正则匹配"))
			return
		}
	} else if len(parts) < 3 {
		var helpMsg string
		switch message.Command() {
		case "add":
			helpMsg = "格式错误，请使用：/add key type value\n其中，type 的取值可以是：\n• exact: 表示精确匹配\n• contains: 表示包含匹配\n• regex: 表示正则匹配"
		case "update":
			helpMsg = "格式错误，请使用：/update key oldType newType value\n其中，type 的取值可以是：\n• exact: 表示精确匹配\n• contains: 表示包含匹配\n• regex: 表示正则匹配"
		}
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, helpMsg))
		return
	}

	{
		key := parts[0]
		matchTypeStr := parts[1]

		matchType, errType := utils.ParseMatchType(matchTypeStr)
		if errType != nil {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "type 参数错误，在 /"+message.Command()+" 命令中，type 必须是：\n• exact: 表示精确匹配\n• contains: 表示包含匹配\n• regex: 表示正则匹配\n• prefix: 表示前缀匹配\n• suffix: 表示后缀匹配"))
			return
		}

		switch message.Command() {
		case "add":
			if len(parts) < 3 {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "格式错误，请使用：/add key type value"))
				return
			}
			value := parts[2]

			// Check if the entry already exists
			existingEntries, err := h.db.QueryExact(key)
			if err != nil {
				log.Printf("Error querying database: %v", err)
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "查询失败"))
				return
			}
			if len(existingEntries) > 0 {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "该条目已存在"))
				return
			}

			err = h.db.AddEntry(key, matchType, value)
			if err != nil {
				log.Printf("Error adding entry: %v", err)
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "添加失败"))
				return
			}
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "添加成功"))

		case "update":
			if len(parts) < 4 {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "格式错误，请使用：/update key oldType newType value"))
				return
			}
			newTypeStr := parts[2]
			newValue := parts[3]

			// 验证并转换新匹配类型
			var newType database.MatchType
			switch newTypeStr {
			case "exact":
				newType = database.MatchExact
			case "contains":
				newType = database.MatchContains
			case "regex":
				newType = database.MatchRegex
			case "prefix":
				newType = database.MatchPrefix
			case "suffix":
				newType = database.MatchSuffix
			default:
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "NewType 参数错误，必须是 exact, contains, regex, prefix, suffix"))
				return
			}

			err := h.db.UpdateEntry(key, matchType, newType, newValue)
			if err != nil {
				log.Printf("Error updating entry: %v", err)
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "更新失败"))
				return
			}
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "更新成功"))

		case "delete":
			err := h.db.DeleteEntry(key, matchType)
			if err != nil {
				log.Printf("Error deleting entry: %v", err)
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "删除失败"))
				return
			}
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "删除成功"))
		}
	}
	// end of HandleAdminCommand

	key := parts[0]
	matchTypeStr := parts[1]

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
		helpMsg := fmt.Sprintf("type 参数错误，在 /%s 命令中，type 必须是：\n• exact: 表示精确匹配\n• contains: 表示包含匹配\n• regex: 表示正则匹配\n• prefix: 表示前缀匹配\n• suffix: 表示后缀匹配", message.Command())
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, helpMsg))
		return
	}

	switch message.Command() {
	case "add":
		if len(parts) < 3 {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "格式错误，请使用：/add key type value"))
			return
		}
		value := parts[2]

		// Check if the entry already exists
		existingEntries, err := h.db.QueryExact(key)
		if err != nil {
			log.Printf("Error querying database: %v", err)
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "查询失败"))
			return
		}
		if len(existingEntries) > 0 {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "该条目已存在"))
			return
		}

		err = h.db.AddEntry(key, matchType, value)
		if err != nil {
			log.Printf("Error adding entry: %v", err)
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "添加失败"))
			return
		}
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "添加成功"))

	case "update":
		if len(parts) < 4 {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "格式错误，请使用：/update key oldType newType value"))
			return
		}
		newTypeStr := parts[2]
		newValue := parts[3]

		// 验证并转换新匹配类型
		var newType database.MatchType
		switch newTypeStr {
		case "exact":
			newType = database.MatchExact
		case "contains":
			newType = database.MatchContains
		case "regex":
			newType = database.MatchRegex
		case "prefix":
			newType = database.MatchPrefix
		case "suffix":
			newType = database.MatchSuffix
		default:
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "NewType 参数错误，必须是 exact, contains, regex, prefix, suffix"))
			return
		}

		err := h.db.UpdateEntry(key, matchType, newType, newValue)
		if err != nil {
			log.Printf("Error updating entry: %v", err)
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "更新失败"))
			return
		}
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "更新成功"))

	case "delete":
		err := h.db.DeleteEntry(key, matchType)
		if err != nil {
			log.Printf("Error deleting entry: %v", err)
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "删除失败"))
			return
		}
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "删除成功"))
	}
}

func (h *AdminHandler) HandleSuperAdminCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	parts := strings.SplitN(args, " ", 2)

	if len(parts) < 1 && message.Command() != "listadmin" {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "参数错误，请使用：\n/addadmin user_id\n/deladmin user_id\n/addgroup group_id\n/delgroup group_id"))
		return
	}

	var targetID int64
	var err error
	if message.Command() != "listadmin" {
		targetID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "ID 格式错误，请使用：\n/addadmin user_id\n/deladmin user_id\n/addgroup group_id\n/delgroup group_id"))
			return
		}
	}

	switch message.Command() {
	case "addadmin":
		for _, adminID := range h.conf.Admin.AdminIDs {
			if adminID == targetID {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "该管理员已存在"))
				return
			}
		}
		h.conf.Admin.AdminIDs = append(h.conf.Admin.AdminIDs, targetID)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("已添加管理员 %d", targetID)))
	case "deladmin":
		h.conf.Admin.AdminIDs = RemoveID(h.conf.Admin.AdminIDs, targetID)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("已删除管理员 %d", targetID)))
	case "addgroup":
		for _, groupID := range h.conf.Admin.AllowedGroupIDs {
			if groupID == targetID {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, "该群组已存在"))
				return
			}
		}
		h.conf.Admin.AllowedGroupIDs = append(h.conf.Admin.AllowedGroupIDs, targetID)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("已添加群组 %d", targetID)))
	case "delgroup":
		h.conf.Admin.AllowedGroupIDs = RemoveID(h.conf.Admin.AllowedGroupIDs, targetID)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("已删除群组 %d", targetID)))
	case "listadmin":
		h.HandleListAdminCommandEdit(bot, message, 0, 0)
		return
	}

	err = config.SaveConfig("config.json", h.conf) // Save the updated config
	if err != nil {
		log.Printf("Error saving config: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "保存配置失败"))
	}
}

func (h *AdminHandler) HandleListAdminCommandEdit(bot *tgbotapi.BotAPI, message *tgbotapi.Message, page int, messageID int) {
	const pageSize = 5
	adminIDs := h.conf.Admin.AdminIDs
	pageAdmins := utils.Paginate(adminIDs, page, pageSize)
	if len(pageAdmins) == 0 {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "没有管理员"))
		return
	}
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, adminID := range pageAdmins {
		buttonText := fmt.Sprintf("管理员ID: %d", adminID)
		callbackData := fmt.Sprintf("admin_%d", adminID)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}
	buttons = append(buttons, utils.BuildPaginationButtons(page, len(adminIDs), pageSize, "listadmin", "取消")...)
	msgText := "选择一个管理员进行操作："
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, messageID, msgText)
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	bot.Send(editMsg)
}

func (h *AdminHandler) HandleAdminSelection(bot *tgbotapi.BotAPI, message *tgbotapi.Message, adminID int64) {
	chatID := message.Chat.ID

	// Retrieve the message ID from the conversation state
	state, exists := h.state.Get(chatID)
	if !exists || state.Stage != "listing_admin" {
		log.Printf("Conversation state not found for chat ID: %d", chatID)
		return
	}
	messageID := state.MessageID

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData("删除", fmt.Sprintf("deladmin_%d", adminID))},
		{tgbotapi.NewInlineKeyboardButtonData("返回", "listadmin_0")},
	}

	msgText := fmt.Sprintf("选择对管理员 %d 的操作：", adminID)
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, messageID, msgText)
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	bot.Send(editMsg)
}
