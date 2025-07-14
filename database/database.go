package database

import (
	"TGFaqBot/config"
	"fmt"
)

type Entry struct {
	ID        int    `json:"id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	MatchType int    `json:"match_type"`
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
	QueryByID(id int, matchType int) (*Entry, error)
	AddEntry(key string, matchType int, value string) error
	UpdateEntry(key string, oldType int, newType int, value string) error
	DeleteEntry(key string, matchType int) error
	DeleteAllEntries() error
	ListEntries(table string) ([]Entry, error)
	ListSpecificEntries(matchTypes ...int) ([]Entry, error)
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
