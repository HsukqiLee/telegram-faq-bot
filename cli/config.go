package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// generateConfigFile 生成配置文件（交互式）
func generateConfigFile(outputPath string, force bool) error {
	// 检查文件是否已存在
	if _, err := os.Stat(outputPath); err == nil && !force {
		return fmt.Errorf("配置文件已存在: %s，使用 --force 强制覆盖", outputPath)
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🤖 欢迎使用 Telegram FAQ Bot 配置向导！")
	fmt.Println("📝 我将帮助您生成一个完整的配置文件")
	fmt.Println()

	config := make(map[string]interface{})

	// 配置 Telegram Bot
	if err := configureTelegram(scanner, config); err != nil {
		return err
	}

	// 配置 AI 服务
	if err := configureAI(scanner, config); err != nil {
		return err
	}

	// 配置数据库
	if err := configureDatabase(scanner, config); err != nil {
		return err
	}

	// 配置 Redis（可选）
	if err := configureRedis(scanner, config); err != nil {
		return err
	}

	// 配置管理员
	if err := configureAdmin(scanner, config); err != nil {
		return err
	}

	// 保存配置文件
	return saveConfig(config, outputPath)
}

// configureTelegram 配置 Telegram Bot 设置
func configureTelegram(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("🔧 Telegram Bot 配置")
	fmt.Println(strings.Repeat("=", 40))

	telegram := make(map[string]interface{})

	// Bot Token
	for {
		token, err := promptInput(scanner, "请输入您的 Telegram Bot Token", "", true)
		if err != nil {
			return err
		}

		if len(token) < 10 || !strings.Contains(token, ":") {
			fmt.Println("❌ Bot Token 格式不正确，应该类似: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz")
			if !promptRetry(scanner) {
				return fmt.Errorf("用户取消配置")
			}
			continue
		}

		telegram["token"] = token
		break
	}

	// 运行模式
	mode, err := promptChoice(scanner, "选择运行模式", []string{"getupdates", "webhook"}, "getupdates")
	if err != nil {
		return err
	}
	telegram["mode"] = mode

	if mode == "webhook" {
		// Webhook 配置
		webhookURL, err := promptInput(scanner, "请输入 Webhook URL", "", true)
		if err != nil {
			return err
		}
		telegram["webhook_url"] = webhookURL

		webhookPort, err := promptInt(scanner, "请输入 Webhook 端口", 8443, 1, 65535)
		if err != nil {
			return err
		}
		telegram["webhook_port"] = webhookPort
	} else {
		telegram["webhook_url"] = ""
		telegram["webhook_port"] = 8443
	}

	// 调试模式
	debug, err := promptBool(scanner, "是否启用调试模式", true)
	if err != nil {
		return err
	}
	telegram["debug"] = debug

	// 介绍信息
	intro, err := promptInput(scanner, "请输入机器人的介绍信息",
		"👋 欢迎使用Telegram FAQ Bot！\n\n💬 直接发送消息与AI对话\n📝 输入关键词可查询FAQ\n⚙️ 使用 /commands 查看所有可用命令\n🤖 使用 /models 选择AI模型\n🔄 使用 /retry 重新生成回复", false)
	if err != nil {
		return err
	}
	telegram["introduction"] = intro

	config["telegram"] = telegram
	fmt.Println("✅ Telegram 配置完成")
	fmt.Println()
	return nil
}

// configureAI 配置 AI 服务
func configureAI(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("🤖 AI 服务配置")
	fmt.Println(strings.Repeat("=", 40))

	chat := make(map[string]interface{})

	// 基础设置
	prefix, err := promptInput(scanner, "请输入消息前缀（可选）", "", false)
	if err != nil {
		return err
	}
	chat["prefix"] = prefix

	systemPrompt, err := promptInput(scanner, "请输入系统提示词",
		"你是一个有用的AI助手。请用中文回答问题，回答要准确、简洁、有帮助。", false)
	if err != nil {
		return err
	}
	chat["system_prompt"] = systemPrompt

	historyLength, err := promptInt(scanner, "请输入对话历史长度", 5, 1, 50)
	if err != nil {
		return err
	}
	chat["history_length"] = historyLength

	historyTimeout, err := promptInt(scanner, "请输入对话超时时间（分钟）", 30, 1, 1440)
	if err != nil {
		return err
	}
	chat["history_timeout_minutes"] = historyTimeout

	timeout, err := promptInt(scanner, "请输入 AI 响应超时时间（秒）", 60, 10, 300)
	if err != nil {
		return err
	}
	chat["timeout"] = timeout

	// 配置 AI 提供商
	providers := []string{"OpenAI", "Anthropic", "Gemini", "Ollama"}
	selectedProviders, err := promptMultiChoice(scanner, "选择要启用的 AI 提供商", providers)
	if err != nil {
		return err
	}

	// OpenAI 配置
	openai := make(map[string]interface{})
	if contains(selectedProviders, "OpenAI") {
		if err := configureOpenAI(scanner, openai); err != nil {
			return err
		}
		openai["enabled"] = true
	} else {
		openai["enabled"] = false
		openai["api_key"] = "your_openai_api_key"
		openai["api_url"] = "https://api.openai.com/v1"
		openai["default_model"] = "gpt-3.5-turbo"
		openai["disabled_models"] = []string{}
		openai["system_prompt"] = ""
		openai["timeout"] = 0
	}
	chat["openai"] = openai

	// Anthropic 配置
	anthropic := make(map[string]interface{})
	if contains(selectedProviders, "Anthropic") {
		if err := configureAnthropic(scanner, anthropic); err != nil {
			return err
		}
		anthropic["enabled"] = true
	} else {
		anthropic["enabled"] = false
		anthropic["api_key"] = "your_anthropic_api_key"
		anthropic["api_url"] = "https://api.anthropic.com"
		anthropic["default_model"] = "claude-3-sonnet-20240229"
		anthropic["disabled_models"] = []string{}
		anthropic["system_prompt"] = ""
		anthropic["timeout"] = 0
	}
	chat["anthropic"] = anthropic

	// Gemini 配置
	gemini := make(map[string]interface{})
	if contains(selectedProviders, "Gemini") {
		if err := configureGemini(scanner, gemini); err != nil {
			return err
		}
		gemini["enabled"] = true
	} else {
		gemini["enabled"] = false
		gemini["api_key"] = "your_gemini_api_key"
		gemini["api_url"] = "https://generativelanguage.googleapis.com/v1beta"
		gemini["default_model"] = "gemini-pro"
		gemini["disabled_models"] = []string{}
		gemini["system_prompt"] = ""
		gemini["timeout"] = 0
	}
	chat["gemini"] = gemini

	// Ollama 配置
	ollama := make(map[string]interface{})
	if contains(selectedProviders, "Ollama") {
		if err := configureOllama(scanner, ollama); err != nil {
			return err
		}
		ollama["enabled"] = true
	} else {
		ollama["enabled"] = false
		ollama["api_url"] = "http://localhost:11434"
		ollama["default_model"] = "llama2"
		ollama["disabled_models"] = []string{}
		ollama["system_prompt"] = ""
		ollama["timeout"] = 0
	}
	chat["ollama"] = ollama

	config["chat"] = chat
	fmt.Println("✅ AI 服务配置完成")
	fmt.Println()
	return nil
}

// configureDatabase 配置数据库
func configureDatabase(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("💾 数据库配置")
	fmt.Println(strings.Repeat("=", 40))

	database := make(map[string]interface{})

	dbTypes := []string{"json", "sqlite", "mysql", "postgresql"}
	dbType, err := promptChoice(scanner, "选择数据库类型", dbTypes, "json")
	if err != nil {
		return err
	}
	database["type"] = dbType

	switch dbType {
	case "json":
		filename, err := promptInput(scanner, "请输入 JSON 文件名", "data.json", false)
		if err != nil {
			return err
		}
		database["json"] = map[string]interface{}{"filename": filename}
		database["sqlite"] = map[string]interface{}{"filename": "bot_data.db"}
		database["mysql"] = getDefaultMySQLConfig()
		database["postgresql"] = getDefaultPostgreSQLConfig()

	case "sqlite":
		filename, err := promptInput(scanner, "请输入 SQLite 文件名", "bot_data.db", false)
		if err != nil {
			return err
		}
		database["sqlite"] = map[string]interface{}{"filename": filename}
		database["json"] = map[string]interface{}{"filename": "data.json"}
		database["mysql"] = getDefaultMySQLConfig()
		database["postgresql"] = getDefaultPostgreSQLConfig()

	case "mysql":
		mysqlConfig := make(map[string]interface{})
		if err := configureMysQL(scanner, mysqlConfig); err != nil {
			return err
		}
		database["mysql"] = mysqlConfig
		database["json"] = map[string]interface{}{"filename": "data.json"}
		database["sqlite"] = map[string]interface{}{"filename": "bot_data.db"}
		database["postgresql"] = getDefaultPostgreSQLConfig()

	case "postgresql":
		pgConfig := make(map[string]interface{})
		if err := configurePostgreSQL(scanner, pgConfig); err != nil {
			return err
		}
		database["postgresql"] = pgConfig
		database["json"] = map[string]interface{}{"filename": "data.json"}
		database["sqlite"] = map[string]interface{}{"filename": "bot_data.db"}
		database["mysql"] = getDefaultMySQLConfig()
	}

	config["database"] = database
	fmt.Println("✅ 数据库配置完成")
	fmt.Println()
	return nil
}

// configureRedis 配置 Redis
func configureRedis(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("🔄 Redis 配置（可选）")
	fmt.Println(strings.Repeat("=", 40))

	redis := make(map[string]interface{})

	enabled, err := promptBool(scanner, "是否启用 Redis", false)
	if err != nil {
		return err
	}
	redis["enabled"] = enabled

	if enabled {
		host, err := promptInput(scanner, "请输入 Redis 主机地址", "localhost", false)
		if err != nil {
			return err
		}
		redis["host"] = host

		port, err := promptInt(scanner, "请输入 Redis 端口", 6379, 1, 65535)
		if err != nil {
			return err
		}
		redis["port"] = port

		password, err := promptInput(scanner, "请输入 Redis 密码（可选）", "", false)
		if err != nil {
			return err
		}
		redis["password"] = password

		database, err := promptInt(scanner, "请输入 Redis 数据库编号", 0, 0, 15)
		if err != nil {
			return err
		}
		redis["database"] = database

		ttl, err := promptInt(scanner, "请输入 TTL 时间（秒）", 1800, 60, 86400)
		if err != nil {
			return err
		}
		redis["ttl"] = ttl

		aiCacheEnabled, err := promptBool(scanner, "是否启用 AI 缓存", false)
		if err != nil {
			return err
		}
		redis["ai_cache_enabled"] = aiCacheEnabled

		if aiCacheEnabled {
			aiCacheTTL, err := promptInt(scanner, "请输入 AI 缓存 TTL 时间（秒）", 3600, 300, 86400)
			if err != nil {
				return err
			}
			redis["ai_cache_ttl"] = aiCacheTTL
		} else {
			redis["ai_cache_ttl"] = 3600
		}
	} else {
		redis["host"] = "localhost"
		redis["port"] = 6379
		redis["password"] = ""
		redis["database"] = 0
		redis["ttl"] = 1800
		redis["ai_cache_enabled"] = false
		redis["ai_cache_ttl"] = 3600
	}

	config["redis"] = redis
	fmt.Println("✅ Redis 配置完成")
	fmt.Println()
	return nil
}

// configureAdmin 配置管理员
func configureAdmin(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("👑 管理员配置")
	fmt.Println(strings.Repeat("=", 40))

	admin := make(map[string]interface{})

	superAdminID, err := promptInt64(scanner, "请输入超级管理员 Telegram ID", 123456789, 1, 9999999999)
	if err != nil {
		return err
	}
	admin["super_admin_ids"] = []int64{superAdminID}
	admin["admin_ids"] = []int64{}
	admin["allowed_group_ids"] = []int64{}

	config["admin"] = admin
	fmt.Println("✅ 管理员配置完成")
	fmt.Println()
	return nil
}

// saveConfig 保存配置文件
func saveConfig(config map[string]interface{}, outputPath string) error {
	// 创建目录（如果不存在）
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 写入配置文件
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	fmt.Printf("🎉 配置文件已生成: %s\n", outputPath)
	fmt.Println("📝 您现在可以启动机器人了！")
	return nil
}

// 输入提示函数
func promptInput(scanner *bufio.Scanner, prompt, defaultValue string, required bool) (string, error) {
	for {
		if defaultValue != "" {
			fmt.Printf("%s [%s]: ", prompt, defaultValue)
		} else {
			fmt.Printf("%s: ", prompt)
		}

		if !scanner.Scan() {
			return "", fmt.Errorf("读取输入失败")
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			if defaultValue != "" {
				return defaultValue, nil
			}
			if required {
				fmt.Println("❌ 此项为必填项，请重新输入")
				if !promptRetry(scanner) {
					return "", fmt.Errorf("用户取消输入")
				}
				continue
			}
			return "", nil
		}
		return input, nil
	}
}

func promptInt(scanner *bufio.Scanner, prompt string, defaultValue, min, max int) (int, error) {
	for {
		input, err := promptInput(scanner, fmt.Sprintf("%s (%d-%d)", prompt, min, max), strconv.Itoa(defaultValue), false)
		if err != nil {
			return 0, err
		}

		if input == "" {
			return defaultValue, nil
		}

		value, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("❌ 请输入有效的数字 (%d-%d)\n", min, max)
			if !promptRetry(scanner) {
				return 0, fmt.Errorf("用户取消输入")
			}
			continue
		}

		if value < min || value > max {
			fmt.Printf("❌ 数值应在 %d-%d 范围内\n", min, max)
			if !promptRetry(scanner) {
				return 0, fmt.Errorf("用户取消输入")
			}
			continue
		}

		return value, nil
	}
}

func promptInt64(scanner *bufio.Scanner, prompt string, defaultValue, min, max int64) (int64, error) {
	for {
		input, err := promptInput(scanner, fmt.Sprintf("%s (%d-%d)", prompt, min, max), strconv.FormatInt(defaultValue, 10), false)
		if err != nil {
			return 0, err
		}

		if input == "" {
			return defaultValue, nil
		}

		value, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			fmt.Printf("❌ 请输入有效的数字 (%d-%d)\n", min, max)
			if !promptRetry(scanner) {
				return 0, fmt.Errorf("用户取消输入")
			}
			continue
		}

		if value < min || value > max {
			fmt.Printf("❌ 数值应在 %d-%d 范围内\n", min, max)
			if !promptRetry(scanner) {
				return 0, fmt.Errorf("用户取消输入")
			}
			continue
		}

		return value, nil
	}
}

func promptBool(scanner *bufio.Scanner, prompt string, defaultValue bool) (bool, error) {
	defaultStr := "n"
	if defaultValue {
		defaultStr = "y"
	}

	for {
		input, err := promptInput(scanner, fmt.Sprintf("%s (y/n)", prompt), defaultStr, false)
		if err != nil {
			return false, err
		}

		input = strings.ToLower(input)
		switch input {
		case "y", "yes", "true", "1":
			return true, nil
		case "n", "no", "false", "0":
			return false, nil
		case "":
			return defaultValue, nil
		default:
			fmt.Println("❌ 请输入 y 或 n")
			if !promptRetry(scanner) {
				return false, fmt.Errorf("用户取消输入")
			}
		}
	}
}

func promptChoice(scanner *bufio.Scanner, prompt string, choices []string, defaultChoice string) (string, error) {
	for {
		fmt.Printf("%s:\n", prompt)
		for i, choice := range choices {
			marker := " "
			if choice == defaultChoice {
				marker = "*"
			}
			fmt.Printf("  %s %d. %s\n", marker, i+1, choice)
		}

		input, err := promptInput(scanner, "请选择", "", false)
		if err != nil {
			return "", err
		}

		if input == "" {
			return defaultChoice, nil
		}

		// 尝试按数字选择
		if num, err := strconv.Atoi(input); err == nil {
			if num >= 1 && num <= len(choices) {
				return choices[num-1], nil
			}
		}

		// 尝试按名称选择
		for _, choice := range choices {
			if strings.EqualFold(input, choice) {
				return choice, nil
			}
		}

		fmt.Println("❌ 无效选择，请重新输入")
		if !promptRetry(scanner) {
			return "", fmt.Errorf("用户取消输入")
		}
	}
}

func promptMultiChoice(scanner *bufio.Scanner, prompt string, choices []string) ([]string, error) {
	selected := make([]string, 0)

	for {
		fmt.Printf("%s (可多选，输入完成后输入 'done'):\n", prompt)
		for i, choice := range choices {
			marker := " "
			if contains(selected, choice) {
				marker = "✓"
			}
			fmt.Printf("  %s %d. %s\n", marker, i+1, choice)
		}

		input, err := promptInput(scanner, "请选择（输入序号或名称，'done' 完成选择）", "", true)
		if err != nil {
			return nil, err
		}

		if strings.ToLower(input) == "done" {
			if len(selected) == 0 {
				fmt.Println("❌ 至少选择一个选项")
				continue
			}
			return selected, nil
		}

		// 尝试按数字选择
		if num, err := strconv.Atoi(input); err == nil {
			if num >= 1 && num <= len(choices) {
				choice := choices[num-1]
				if contains(selected, choice) {
					// 取消选择
					selected = remove(selected, choice)
					fmt.Printf("取消选择: %s\n", choice)
				} else {
					// 添加选择
					selected = append(selected, choice)
					fmt.Printf("已选择: %s\n", choice)
				}
				continue
			}
		}

		// 尝试按名称选择
		found := false
		for _, choice := range choices {
			if strings.EqualFold(input, choice) {
				if contains(selected, choice) {
					selected = remove(selected, choice)
					fmt.Printf("取消选择: %s\n", choice)
				} else {
					selected = append(selected, choice)
					fmt.Printf("已选择: %s\n", choice)
				}
				found = true
				break
			}
		}

		if !found {
			fmt.Println("❌ 无效选择，请重新输入")
		}
	}
}

func promptRetry(scanner *bufio.Scanner) bool {
	for {
		fmt.Print("是否重新输入？(y/n) [y]: ")
		if !scanner.Scan() {
			return false
		}

		input := strings.ToLower(strings.TrimSpace(scanner.Text()))
		switch input {
		case "", "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("请输入 y 或 n")
		}
	}
}

// AI 提供商配置函数
func configureOpenAI(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("  🔑 OpenAI 配置")

	apiKey, err := promptInput(scanner, "  请输入 OpenAI API Key", "", true)
	if err != nil {
		return err
	}
	config["api_key"] = apiKey

	apiURL, err := promptInput(scanner, "  请输入 API URL", "https://api.openai.com/v1", false)
	if err != nil {
		return err
	}
	config["api_url"] = apiURL

	defaultModel, err := promptInput(scanner, "  请输入默认模型", "gpt-3.5-turbo", false)
	if err != nil {
		return err
	}
	config["default_model"] = defaultModel

	config["disabled_models"] = []string{}
	config["system_prompt"] = ""
	config["timeout"] = 0

	return nil
}

func configureAnthropic(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("  🔑 Anthropic 配置")

	apiKey, err := promptInput(scanner, "  请输入 Anthropic API Key", "", true)
	if err != nil {
		return err
	}
	config["api_key"] = apiKey

	apiURL, err := promptInput(scanner, "  请输入 API URL", "https://api.anthropic.com", false)
	if err != nil {
		return err
	}
	config["api_url"] = apiURL

	defaultModel, err := promptInput(scanner, "  请输入默认模型", "claude-3-sonnet-20240229", false)
	if err != nil {
		return err
	}
	config["default_model"] = defaultModel

	config["disabled_models"] = []string{}
	config["system_prompt"] = ""
	config["timeout"] = 0

	return nil
}

func configureGemini(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("  🔑 Gemini 配置")

	apiKey, err := promptInput(scanner, "  请输入 Gemini API Key", "", true)
	if err != nil {
		return err
	}
	config["api_key"] = apiKey

	apiURL, err := promptInput(scanner, "  请输入 API URL", "https://generativelanguage.googleapis.com/v1beta", false)
	if err != nil {
		return err
	}
	config["api_url"] = apiURL

	defaultModel, err := promptInput(scanner, "  请输入默认模型", "gemini-pro", false)
	if err != nil {
		return err
	}
	config["default_model"] = defaultModel

	config["disabled_models"] = []string{}
	config["system_prompt"] = ""
	config["timeout"] = 0

	return nil
}

func configureOllama(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("  🔑 Ollama 配置")

	apiURL, err := promptInput(scanner, "  请输入 Ollama API URL", "http://localhost:11434", false)
	if err != nil {
		return err
	}
	config["api_url"] = apiURL

	defaultModel, err := promptInput(scanner, "  请输入默认模型", "llama2", false)
	if err != nil {
		return err
	}
	config["default_model"] = defaultModel

	config["disabled_models"] = []string{}
	config["system_prompt"] = ""
	config["timeout"] = 0

	return nil
}

// 数据库配置函数
func configureMysQL(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("  🐬 MySQL 配置")

	host, err := promptInput(scanner, "  请输入 MySQL 主机地址", "localhost", false)
	if err != nil {
		return err
	}
	config["host"] = host

	port, err := promptInt(scanner, "  请输入 MySQL 端口", 3306, 1, 65535)
	if err != nil {
		return err
	}
	config["port"] = port

	user, err := promptInput(scanner, "  请输入 MySQL 用户名", "bot_user", true)
	if err != nil {
		return err
	}
	config["user"] = user

	password, err := promptInput(scanner, "  请输入 MySQL 密码", "", true)
	if err != nil {
		return err
	}
	config["password"] = password

	database, err := promptInput(scanner, "  请输入数据库名", "telegram_bot", true)
	if err != nil {
		return err
	}
	config["database"] = database

	sslmode, err := promptChoice(scanner, "  选择 SSL 模式", []string{"false", "true"}, "false")
	if err != nil {
		return err
	}
	config["sslmode"] = sslmode

	return nil
}

func configurePostgreSQL(scanner *bufio.Scanner, config map[string]interface{}) error {
	fmt.Println("  🐘 PostgreSQL 配置")

	host, err := promptInput(scanner, "  请输入 PostgreSQL 主机地址", "localhost", false)
	if err != nil {
		return err
	}
	config["host"] = host

	port, err := promptInt(scanner, "  请输入 PostgreSQL 端口", 5432, 1, 65535)
	if err != nil {
		return err
	}
	config["port"] = port

	user, err := promptInput(scanner, "  请输入 PostgreSQL 用户名", "bot_user", true)
	if err != nil {
		return err
	}
	config["user"] = user

	password, err := promptInput(scanner, "  请输入 PostgreSQL 密码", "", true)
	if err != nil {
		return err
	}
	config["password"] = password

	database, err := promptInput(scanner, "  请输入数据库名", "telegram_bot", true)
	if err != nil {
		return err
	}
	config["database"] = database

	sslmode, err := promptChoice(scanner, "  选择 SSL 模式", []string{"disable", "require", "verify-ca", "verify-full"}, "disable")
	if err != nil {
		return err
	}
	config["sslmode"] = sslmode

	return nil
}

// 默认配置生成函数
func getDefaultMySQLConfig() map[string]interface{} {
	return map[string]interface{}{
		"host":     "localhost",
		"port":     3306,
		"user":     "bot_user",
		"password": "your_mysql_password",
		"database": "telegram_bot",
		"sslmode":  "false",
	}
}

func getDefaultPostgreSQLConfig() map[string]interface{} {
	return map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"user":     "bot_user",
		"password": "your_postgresql_password",
		"database": "telegram_bot",
		"sslmode":  "disable",
	}
}

// contains 判断字符串切片中是否包含某个元素
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// remove 从字符串切片中移除指定元素
func remove(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != item {
			result = append(result, v)
		}
	}
	return result
}
