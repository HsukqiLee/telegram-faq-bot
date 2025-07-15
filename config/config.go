package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type Config struct {
	Telegram TelegramConfig `json:"telegram"`
	Chat     ChatConfig     `json:"chat"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis,omitempty"`
	Admin    AdminConfig    `json:"admin"`
}

type TelegramConfig struct {
	Token        string `json:"token"`
	WebhookURL   string `json:"webhook_url"`
	WebhookPort  int    `json:"webhook_port"`
	Debug        bool   `json:"debug"`
	Introduction string `json:"introduction"`
	Mode         string `json:"mode"`
}

type ChatConfig struct {
	Prefix                string          `json:"prefix"`                  // 聊天前缀，为空时默认触发
	SystemPrompt          string          `json:"system_prompt"`           // 全局系统提示词
	HistoryLength         int             `json:"history_length"`          // 全局历史记录长度
	HistoryTimeoutMinutes int             `json:"history_timeout_minutes"` // 全局历史超时分钟数
	Timeout               int64           `json:"timeout"`                 // 全局超时时间
	OpenAI                *ProviderConfig `json:"openai,omitempty"`
	Anthropic             *ProviderConfig `json:"anthropic,omitempty"`
	Gemini                *ProviderConfig `json:"gemini,omitempty"`
	Ollama                *ProviderConfig `json:"ollama,omitempty"`
}

type ProviderConfig struct {
	Enabled        bool     `json:"enabled"`
	APIKey         string   `json:"api_key"`
	APIURL         string   `json:"api_url"`
	DefaultModel   string   `json:"default_model"`
	DisabledModels []string `json:"disabled_models,omitempty"`
	SystemPrompt   string   `json:"system_prompt,omitempty"` // 可选，覆盖全局设置
	Timeout        int64    `json:"timeout,omitempty"`       // 可选，覆盖全局设置
}

type Model struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider"`
}

type OpenAIConfig struct {
	APIKey                string   `json:"api_key"`
	APIURL                string   `json:"api_url"`
	DefaultModel          string   `json:"default_model"`
	DisabledModels        []string `json:"disabled_models"`
	Timeout               int64    `json:"timeout"`
	SystemPrompt          string   `json:"system_prompt"`
	HistoryLength         int      `json:"history_length"`
	HistoryTimeoutMinutes int      `json:"history_timeout_minutes"`
}

type DatabaseConfig struct {
	Type       string           `json:"type"`
	JSON       JSONConfig       `json:"json,omitempty"`
	SQLite     SQLiteConfig     `json:"sqlite,omitempty"`
	MySQL      MySQLConfig      `json:"mysql,omitempty"`
	PostgreSQL PostgreSQLConfig `json:"postgresql,omitempty"`
}

type JSONConfig struct {
	Filename string `json:"filename"`
}

type SQLiteConfig struct {
	Filename string `json:"filename"`
}

type MySQLConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode,omitempty"` // disable, true, false, skip-verify, preferred
}

type PostgreSQLConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode,omitempty"` // disable, require, verify-ca, verify-full
}

type RedisConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password,omitempty"`
	Database int    `json:"database"`
	TTL      int    `json:"ttl"` // TTL in seconds for conversation data
	
	// AI对话缓存配置
	AICacheEnabled bool `json:"ai_cache_enabled"` // 是否开启AI对话缓存
	AICacheTTL     int  `json:"ai_cache_ttl"`     // AI对话缓存过期时间(秒)
}

type AdminConfig struct {
	SuperAdminIDs   []int64 `json:"super_admin_ids"`
	AdminIDs        []int64 `json:"admin_ids"`
	AllowedGroupIDs []int64 `json:"allowed_group_ids"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}

	// 设置Chat配置的默认值
	if config.Chat.HistoryLength == 0 {
		config.Chat.HistoryLength = 10
	}
	if config.Chat.HistoryTimeoutMinutes == 0 {
		config.Chat.HistoryTimeoutMinutes = 30
	}
	if config.Chat.Timeout == 0 {
		config.Chat.Timeout = 60
	}

	// 加载环境变量覆盖配置
	config.LoadEnvVariables()

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &config, nil
}

// LoadEnvVariables 从环境变量加载配置，并在使用配置文件值时发出警告
func (c *Config) LoadEnvVariables() {
	// Telegram Token
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		c.Telegram.Token = token
	} else if c.Telegram.Token != "" {
		log.Printf("⚠️  WARNING: Using Telegram token from config file. Consider using TELEGRAM_BOT_TOKEN environment variable for better security.")
	}

	// OpenAI API Key
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		if c.Chat.OpenAI != nil {
			c.Chat.OpenAI.APIKey = apiKey
		}
	} else if c.Chat.OpenAI != nil && c.Chat.OpenAI.APIKey != "" {
		log.Printf("⚠️  WARNING: Using OpenAI API key from config file. Consider using OPENAI_API_KEY environment variable for better security.")
	}

	// Anthropic API Key
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		if c.Chat.Anthropic != nil {
			c.Chat.Anthropic.APIKey = apiKey
		}
	} else if c.Chat.Anthropic != nil && c.Chat.Anthropic.APIKey != "" {
		log.Printf("⚠️  WARNING: Using Anthropic API key from config file. Consider using ANTHROPIC_API_KEY environment variable for better security.")
	}

	// Gemini API Key
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		if c.Chat.Gemini != nil {
			c.Chat.Gemini.APIKey = apiKey
		}
	} else if c.Chat.Gemini != nil && c.Chat.Gemini.APIKey != "" {
		log.Printf("⚠️  WARNING: Using Gemini API key from config file. Consider using GEMINI_API_KEY environment variable for better security.")
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证Telegram配置
	if c.Telegram.Token == "" {
		return errors.New("telegram bot token is required")
	}

	if !strings.HasPrefix(c.Telegram.Token, "bot") && len(strings.Split(c.Telegram.Token, ":")) != 2 {
		return errors.New("invalid telegram bot token format")
	}

	// 验证模式
	if c.Telegram.Mode != "webhook" && c.Telegram.Mode != "getupdates" {
		return errors.New("telegram mode must be 'webhook' or 'getupdates'")
	}

	// Webhook模式需要URL
	if c.Telegram.Mode == "webhook" && c.Telegram.WebhookURL == "" {
		return errors.New("webhook_url is required when mode is 'webhook'")
	}

	// 验证管理员
	if len(c.Admin.SuperAdminIDs) == 0 {
		return errors.New("at least one super admin is required")
	}

	// 验证数据库配置
	if err := c.validateDatabase(); err != nil {
		return fmt.Errorf("database config error: %v", err)
	}

	// 验证AI配置
	if err := c.validateAIProviders(); err != nil {
		return fmt.Errorf("AI provider config error: %v", err)
	}

	return nil
}

func (c *Config) validateDatabase() error {
	switch c.Database.Type {
	case "json":
		if c.Database.JSON.Filename == "" {
			return errors.New("json filename is required")
		}
	case "sqlite":
		if c.Database.SQLite.Filename == "" {
			return errors.New("sqlite filename is required")
		}
	case "mysql":
		if c.Database.MySQL.Host == "" || c.Database.MySQL.User == "" {
			return errors.New("mysql host and user are required")
		}
		// 设置默认端口
		if c.Database.MySQL.Port == 0 {
			c.Database.MySQL.Port = 3306
		}
		// 设置默认SSL模式
		if c.Database.MySQL.SSLMode == "" {
			c.Database.MySQL.SSLMode = "false"
		}
	case "postgresql":
		if c.Database.PostgreSQL.Host == "" || c.Database.PostgreSQL.User == "" || c.Database.PostgreSQL.Database == "" {
			return errors.New("postgresql host, user and database are required")
		}
		// 设置默认端口
		if c.Database.PostgreSQL.Port == 0 {
			c.Database.PostgreSQL.Port = 5432
		}
		// 设置默认SSL模式
		if c.Database.PostgreSQL.SSLMode == "" {
			c.Database.PostgreSQL.SSLMode = "disable"
		}
	default:
		return fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}
	return nil
}

func (c *Config) validateAIProviders() error {
	enabledCount := 0

	if c.Chat.OpenAI != nil && c.Chat.OpenAI.Enabled {
		if c.Chat.OpenAI.APIKey == "" {
			return errors.New("OpenAI API key is required when enabled")
		}
		enabledCount++
	}

	if c.Chat.Anthropic != nil && c.Chat.Anthropic.Enabled {
		if c.Chat.Anthropic.APIKey == "" {
			return errors.New("anthropic API key is required when enabled")
		}
		enabledCount++
	}

	if c.Chat.Gemini != nil && c.Chat.Gemini.Enabled {
		if c.Chat.Gemini.APIKey == "" {
			return errors.New("gemini API key is required when enabled")
		}
		enabledCount++
	}

	// 允许没有AI提供商启用的情况，此时将显示相应的警告消息
	if enabledCount == 0 {
		log.Println("⚠️ WARNING: No AI providers are enabled. AI chat functionality will be disabled.")
	}

	return nil
}

func SaveConfig(filename string, config *Config) error {
	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, bytes, 0644)
}

// GetEnabledProviders 返回所有启用的提供商
func (c *ChatConfig) GetEnabledProviders() map[string]*ProviderConfig {
	providers := make(map[string]*ProviderConfig)

	if c.OpenAI != nil && c.OpenAI.Enabled {
		providers["openai"] = c.OpenAI
	}
	if c.Anthropic != nil && c.Anthropic.Enabled {
		providers["anthropic"] = c.Anthropic
	}
	if c.Gemini != nil && c.Gemini.Enabled {
		providers["gemini"] = c.Gemini
	}
	if c.Ollama != nil && c.Ollama.Enabled {
		providers["ollama"] = c.Ollama
	}

	return providers
}

// GetProviderTimeout 获取提供商的超时设置，如果未设置则使用全局设置
func (p *ProviderConfig) GetTimeout(globalTimeout int64) int64 {
	if p.Timeout > 0 {
		return p.Timeout
	}
	return globalTimeout
}

// GetProviderSystemPrompt 获取提供商的系统提示词，如果未设置则使用全局设置
func (p *ProviderConfig) GetSystemPrompt(globalPrompt string) string {
	if p.SystemPrompt != "" {
		return p.SystemPrompt
	}
	return globalPrompt
}
