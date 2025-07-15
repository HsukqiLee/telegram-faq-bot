package database

import (
	"TGFaqBot/config"
	"fmt"
)

type Entry struct {
	ID            int       `json:"id"`
	Key           string    `json:"key"`
	Value         string    `json:"value"`
	MatchType     MatchType `json:"match_type"`
	ContentType   string    `json:"content_type"`   // "text", "telegraph_text", "telegraph_image"
	TelegraphURL  string    `json:"telegraph_url"`  // Telegraph 页面 URL
	TelegraphPath string    `json:"telegraph_path"` // Telegraph 页面路径
}

// ModelInfo 存储AI模型信息
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

type Database interface {
	Query(query string) ([]Entry, error)
	QueryByID(id int, matchType MatchType) (*Entry, error)
	AddEntry(key string, matchType MatchType, value string) error
	UpdateEntry(key string, oldType MatchType, newType MatchType, value string) error
	DeleteEntry(key string, matchType MatchType) error
	DeleteAllEntries() error
	ListEntries(table string) ([]Entry, error)
	ListSpecificEntries(matchTypes ...MatchType) ([]Entry, error)
	QueryExact(query string) ([]Entry, error)
	QueryContains(query string) ([]Entry, error)
	QueryRegex(query string) ([]Entry, error)
	AddEntryExact(key string, value string) error
	AddEntryContains(key string, value string) error
	AddEntryRegex(key string, value string) error
	UpdateEntryExact(key string, value string) error
	UpdateEntryContains(key string, value string) error
	UpdateEntryRegex(key string, value string) error
	DeleteEntryExact(key string) error
	DeleteEntryContains(key string) error
	DeleteEntryRegex(key string) error
	ListEntriesExact() ([]Entry, error)
	ListEntriesContains() ([]Entry, error)
	ListEntriesRegex() ([]Entry, error)
	ListAllEntries() ([]Entry, error)

	// 模型管理接口
	SaveModels(provider string, models []ModelInfo) error
	GetModels(provider string) ([]ModelInfo, error)
	GetAllModels() (map[string][]ModelInfo, error)
	DeleteModels(provider string) error

	// 模型缓存接口
	SetModelCache(models []config.Model, updatedAt string) error
	GetModelCache() ([]config.Model, string, error)
	ClearModelCache() error

	// Telegraph 内容管理
	AddTelegraphEntry(key string, matchType MatchType, value, contentType, telegraphURL, telegraphPath string) error
	UpdateTelegraphEntry(key string, matchType MatchType, value, contentType, telegraphURL, telegraphPath string) error
	GetTelegraphContent(key string, matchType MatchType) (*Entry, error)

	Reload() error
	Close() error
}

func NewDatabase(cfg config.DatabaseConfig) (Database, error) {
	switch cfg.Type {
	case "json":
		return NewJSONDB(cfg.JSON.Filename)
	case "sqlite":
		return NewSQLiteDB(cfg.SQLite.Filename)
	case "mysql":
		return NewMySQLDB(cfg.MySQL)
	case "postgresql":
		return NewPostgreSQLDB(cfg.PostgreSQL)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// MatchType 定义匹配类型的枚举
type MatchType string

const (
	// MatchExact 精确匹配
	MatchExact MatchType = "exact"
	// MatchContains 包含匹配
	MatchContains MatchType = "contains"
	// MatchRegex 正则表达式匹配
	MatchRegex MatchType = "regex"
	// MatchPrefix 前缀匹配
	MatchPrefix MatchType = "prefix"
	// MatchSuffix 后缀匹配
	MatchSuffix MatchType = "suffix"
)

// String 返回匹配类型的字符串表示
func (mt MatchType) String() string {
	switch mt {
	case MatchExact:
		return "精确匹配"
	case MatchContains:
		return "包含匹配"
	case MatchRegex:
		return "正则匹配"
	case MatchPrefix:
		return "前缀匹配"
	case MatchSuffix:
		return "后缀匹配"
	default:
		return fmt.Sprintf("未知类型(%s)", string(mt))
	}
}

// IsValid 检查匹配类型是否有效
func (mt MatchType) IsValid() bool {
	switch mt {
	case MatchExact, MatchContains, MatchRegex, MatchPrefix, MatchSuffix:
		return true
	default:
		return false
	}
}

// FromInt 从整数创建匹配类型 (向后兼容)
func MatchTypeFromInt(i int) (MatchType, error) {
	switch i {
	case 1:
		return MatchExact, nil
	case 2:
		return MatchContains, nil
	case 3:
		return MatchRegex, nil
	case 4:
		return MatchPrefix, nil
	case 5:
		return MatchSuffix, nil
	default:
		return "", fmt.Errorf("invalid match type: %d (valid values: 1=精确匹配, 2=包含匹配, 3=正则匹配, 4=前缀匹配, 5=后缀匹配)", i)
	}
}

// ToInt 将匹配类型转换为整数 (向后兼容)
func (mt MatchType) ToInt() int {
	switch mt {
	case MatchExact:
		return 1
	case MatchContains:
		return 2
	case MatchRegex:
		return 3
	case MatchPrefix:
		return 4
	case MatchSuffix:
		return 5
	default:
		return 1 // 默认返回精确匹配
	}
}

// intToMatchType 安全地将 int 转换为 MatchType
func intToMatchType(i int) MatchType {
	mt, _ := MatchTypeFromInt(i)
	return mt
}

// GetTableName 返回对应的数据库表名
func (mt MatchType) GetTableName() string {
	switch mt {
	case MatchExact:
		return "exact"
	case MatchContains:
		return "contains"
	case MatchRegex:
		return "regex"
	case MatchPrefix:
		return "prefix"
	case MatchSuffix:
		return "suffix"
	default:
		return ""
	}
}
