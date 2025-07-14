package main

import (
	"fmt"
	"log"
	"time"

	"TGFaqBot/bot"
	"TGFaqBot/config"
	"TGFaqBot/database"
	"TGFaqBot/multichat"
)

// fetchAndSaveModels 从AI提供商获取模型列表并保存到数据库
func fetchAndSaveModels(manager *multichat.Manager, db database.Database) error {
	providers := manager.GetAvailableProviders()
	log.Printf("Fetching models from %d enabled providers", len(providers))

	for _, providerName := range providers {
		log.Printf("Fetching models for provider: %s", providerName)

		// 从提供商直接获取模型列表（不依赖缓存）
		models, err := fetchModelsFromProvider(manager, providerName)
		if err != nil {
			log.Printf("Warning: Failed to fetch models for %s: %v", providerName, err)
			continue
		}

		if len(models) == 0 {
			log.Printf("Warning: No models returned for %s, skipping", providerName)
			continue
		}

		// 转换为数据库格式
		var modelInfos []database.ModelInfo
		for _, model := range models {
			modelInfo := database.ModelInfo{
				ID:          model.ID,
				Name:        model.Name,
				Provider:    model.Provider,
				Description: "", // 可在未来添加描述
				UpdatedAt:   time.Now().Format("2006-01-02 15:04:05"),
			}
			modelInfos = append(modelInfos, modelInfo)
		}

		// 保存到数据库
		err = db.SaveModels(providerName, modelInfos)
		if err != nil {
			log.Printf("Warning: Failed to save models for %s: %v", providerName, err)
			continue
		}

		log.Printf("Successfully saved %d models for %s", len(modelInfos), providerName)
	}

	return nil
}

// fetchModelsFromProvider 直接从提供商获取模型列表
func fetchModelsFromProvider(manager *multichat.Manager, providerName string) ([]config.Model, error) {
	// 先尝试获取缓存的模型
	cachedModels := manager.GetCachedModels(providerName)
	if len(cachedModels) > 0 {
		log.Printf("Using cached models for %s", providerName)
		return cachedModels, nil
	}

	// 如果没有缓存，先使用默认模型
	log.Printf("No cached models for %s, using default models", providerName)
	defaultModels := getDefaultModels(providerName)
	if len(defaultModels) > 0 {
		log.Printf("Using %d default models for %s", len(defaultModels), providerName)
		return defaultModels, nil
	}

	return []config.Model{}, fmt.Errorf("no models available for provider %s", providerName)
}

// getDefaultModels 返回默认的模型列表（当API调用失败时使用）
func getDefaultModels(providerName string) []config.Model {
	switch providerName {
	case "openai":
		return []config.Model{
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai"},
			{ID: "gpt-4o", Name: "GPT-4o", Provider: "openai"},
			{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "openai"},
			{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: "openai"},
		}
	case "anthropic":
		return []config.Model{
			{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "anthropic"},
			{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", Provider: "anthropic"},
			{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Provider: "anthropic"},
		}
	case "gemini":
		return []config.Model{
			{ID: "gemini-pro", Name: "Gemini Pro", Provider: "gemini"},
			{ID: "gemini-pro-vision", Name: "Gemini Pro Vision", Provider: "gemini"},
		}
	default:
		return []config.Model{}
	}
}

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

	// 初始化多渠道管理器并获取模型列表
	manager := multichat.NewManager(conf, "config.json")
	err = fetchAndSaveModels(manager, db)
	if err != nil {
		log.Printf("Warning: Failed to fetch models: %v", err)
	}

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
