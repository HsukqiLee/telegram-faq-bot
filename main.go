package main

import (
	"log"

	"TGFaqBot/bot"
	"TGFaqBot/config"
	"TGFaqBot/database"
	"TGFaqBot/multichat"
)

func main() {
	// 加载配置
	conf, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if conf.Chat.SystemPrompt != "" {
		log.Printf("Initializing chat with system prompt: %s", conf.Chat.SystemPrompt)
	}

	// 初始化数据库
	db, err := database.NewDatabase(conf.Database)
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// 初始化多渠道管理器（会自动获取并缓存模型列表）
	manager := multichat.NewManager(conf, "config.json", db)

	// 创建并启动Bot
	telegramBot, err := bot.NewTelegramBot(conf, db, manager)
	if err != nil {
		log.Fatalf("Failed to create telegram bot: %v", err)
	}
	defer telegramBot.Close()

	// 启动Bot
	if err := telegramBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
}
