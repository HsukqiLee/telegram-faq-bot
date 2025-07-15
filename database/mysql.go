package database

import (
	"TGFaqBot/config"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDB struct {
	cfg        config.MySQLConfig
	db         *sql.DB
	commonOps  *CommonSQLOperations
}

func NewMySQLDB(cfg config.MySQLConfig) (*MySQLDB, error) {
	db := &MySQLDB{cfg: cfg}
	if err := db.Reload(); err != nil {
		return nil, err
	}
	db.commonOps = NewCommonSQLOperations(db.db)
	return db, nil
}

// Implement the combined functions
func (m *MySQLDB) Query(query string) ([]Entry, error) {
	// Combine results from all match types
	exact, err := m.QueryExact(query)
	if err != nil {
		return nil, err
	}

	contains, err := m.QueryContains(query)
	if err != nil {
		return nil, err
	}

	regex, err := m.QueryRegex(query)
	if err != nil {
		return nil, err
	}

	// Concatenate the results
	allResults := append(exact, contains...)
	allResults = append(allResults, regex...)

	return allResults, nil
}

func (m *MySQLDB) QueryByID(id int, matchType MatchType) (*Entry, error) {
	return m.commonOps.QueryByID(id, matchType, nil)
}

func (m *MySQLDB) AddEntry(key string, matchType MatchType, value string) error {
	switch matchType {
	case MatchExact: // Exact
		return m.AddEntryExact(key, value)
	case MatchContains: // Contains
		return m.AddEntryContains(key, value)
	case MatchRegex: // Regex
		return m.AddEntryRegex(key, value)
	default:
		return fmt.Errorf("invalid match type: %s", matchType)
	}
}

func (m *MySQLDB) UpdateEntry(key string, oldType MatchType, newType MatchType, value string) error {
	if oldType == newType {
		// Same type, use existing UpdateEntryXXX functions
		switch oldType {
		case MatchExact:
			return m.UpdateEntryExact(key, value)
		case MatchContains:
			return m.UpdateEntryContains(key, value)
		case MatchRegex:
			return m.UpdateEntryRegex(key, value)
		default:
			return fmt.Errorf("invalid match type: %s", oldType)
		}
	} else {
		// Different types, delete from old and add to new
		if err := m.DeleteEntry(key, oldType); err != nil {
			return err
		}
		return m.AddEntry(key, newType, value)
	}
}

func (m *MySQLDB) DeleteEntry(key string, matchType MatchType) error {
	switch matchType {
	case MatchExact: // Exact
		return m.DeleteEntryExact(key)
	case MatchContains: // Contains
		return m.DeleteEntryContains(key)
	case MatchRegex: // Regex
		return m.DeleteEntryRegex(key)
	default:
		return fmt.Errorf("invalid match type: %s", matchType)
	}
}

func (m *MySQLDB) ListEntries(table string) ([]Entry, error) {
	return m.commonOps.ListEntries(table, nil)
}

func (m *MySQLDB) ListAllEntries() ([]Entry, error) {
	// 使用单个查询获取所有条目，按匹配类型排序，比分别查询更高效
	query := `SELECT id, key_text, value_text, match_type FROM faq_entries ORDER BY match_type, id`

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allEntries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.MatchType); err != nil {
			return nil, err
		}
		allEntries = append(allEntries, entry)
	}

	return allEntries, rows.Err()
}

func (m *MySQLDB) QueryExact(query string) ([]Entry, error) {
	return m.commonOps.QueryWithCondition(query, "exact", nil)
}

func (m *MySQLDB) QueryContains(query string) ([]Entry, error) {
	return m.commonOps.QueryWithCondition(query, "contains", nil)
}

func (m *MySQLDB) QueryRegex(query string) ([]Entry, error) {
	return m.commonOps.QueryWithCondition(query, "regex", nil)
}

func (m *MySQLDB) AddEntryExact(key string, value string) error {
	return m.commonOps.AddEntry(key, value, "exact")
}

func (m *MySQLDB) AddEntryContains(key string, value string) error {
	return m.commonOps.AddEntry(key, value, "contains")
}

func (m *MySQLDB) AddEntryRegex(key string, value string) error {
	return m.commonOps.AddEntry(key, value, "regex")
}

func (m *MySQLDB) UpdateEntryExact(key string, value string) error {
	return m.commonOps.UpdateEntry(key, value, "exact")
}

func (m *MySQLDB) UpdateEntryContains(key string, value string) error {
	return m.commonOps.UpdateEntry(key, value, "contains")
}

func (m *MySQLDB) UpdateEntryRegex(key string, value string) error {
	return m.commonOps.UpdateEntry(key, value, "regex")
}

func (m *MySQLDB) DeleteEntryExact(key string) error {
	return m.commonOps.DeleteEntry(key, "exact")
}

func (m *MySQLDB) DeleteEntryContains(key string) error {
	return m.commonOps.DeleteEntry(key, "contains")
}

func (m *MySQLDB) DeleteEntryRegex(key string) error {
	return m.commonOps.DeleteEntry(key, "regex")
}

func (m *MySQLDB) ListEntriesExact() ([]Entry, error) {
	return m.commonOps.ListEntries("exact", nil)
}

func (m *MySQLDB) ListEntriesContains() ([]Entry, error) {
	return m.commonOps.ListEntries("contains", nil)
}

func (m *MySQLDB) ListEntriesRegex() ([]Entry, error) {
	return m.commonOps.ListEntries("regex", nil)
}

func (m *MySQLDB) ListSpecificEntries(matchTypes ...MatchType) ([]Entry, error) {
	if len(matchTypes) == 0 {
		// List all entries if no match types are specified
		return m.ListAllEntries()
	}

	var allEntries []Entry
	for _, matchType := range matchTypes {
		var entries []Entry
		var err error

		switch matchType {
		case MatchExact:
			entries, err = m.ListEntriesExact()
		case MatchContains:
			entries, err = m.ListEntriesContains()
		case MatchRegex:
			entries, err = m.ListEntriesRegex()
		default:
			return nil, fmt.Errorf("invalid match type: %s", matchType)
		}

		if err != nil {
			return nil, err
		}

		// Set the MatchType for each entry
		for i := range entries {
			entries[i].MatchType = matchType
		}

		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

func (m *MySQLDB) DeleteAllEntries() error {
	_, err := m.db.Exec("DELETE FROM exact")
	if err != nil {
		return err
	}
	_, err = m.db.Exec("DELETE FROM contains")
	if err != nil {
		return err
	}
	_, err = m.db.Exec("DELETE FROM regex")
	if err != nil {
		return err
	}
	return nil
}

func (m *MySQLDB) Reload() error {
	var err error
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s",
		m.cfg.User, m.cfg.Password, m.cfg.Host, m.cfg.Port, m.cfg.Database, m.cfg.SSLMode)

	m.db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		return err
	}

	// Test the connection
	if err = m.db.Ping(); err != nil {
		return err
	}

	// Create tables if not exists
	_, err = m.db.Exec(`
        CREATE TABLE IF NOT EXISTS exact (
            id INTEGER PRIMARY KEY AUTO_INCREMENT,
            ` + "`key`" + ` TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS contains (
            id INTEGER PRIMARY KEY AUTO_INCREMENT,
            ` + "`key`" + ` TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS regex (
            id INTEGER PRIMARY KEY AUTO_INCREMENT,
            ` + "`key`" + ` TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS prefix (
            id INTEGER PRIMARY KEY AUTO_INCREMENT,
            ` + "`key`" + ` TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS suffix (
            id INTEGER PRIMARY KEY AUTO_INCREMENT,
            ` + "`key`" + ` TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS ai_models (
            id INTEGER PRIMARY KEY AUTO_INCREMENT,
            provider VARCHAR(100) NOT NULL,
            model_id VARCHAR(255) NOT NULL,
            model_name VARCHAR(255) NOT NULL,
            description TEXT DEFAULT '',
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            UNIQUE KEY unique_provider_model (provider, model_id)
        );
        CREATE TABLE IF NOT EXISTS user_preferences (
            user_id BIGINT PRIMARY KEY,
            preferred_model_id VARCHAR(255) NOT NULL,
            preferred_provider VARCHAR(100) NOT NULL,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        );
    `)
	if err != nil {
		return err
	}
	
	// 重新初始化common operations
	m.commonOps = NewCommonSQLOperations(m.db)
	return nil
}

func (m *MySQLDB) Close() error {
	return m.db.Close()
}

// 模型管理功能实现
func (m *MySQLDB) SaveModels(provider string, models []ModelInfo) error {
	// 开始事务
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// 先删除该提供商的旧模型
	_, err = tx.Exec("DELETE FROM ai_models WHERE provider = ?", provider)
	if err != nil {
		return fmt.Errorf("failed to delete old models: %v", err)
	}

	// 插入新模型
	stmt, err := tx.Prepare("INSERT INTO ai_models (provider, model_id, model_name, description, updated_at) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for _, model := range models {
		_, err = stmt.Exec(provider, model.ID, model.Name, model.Description, model.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert model %s: %v", model.ID, err)
		}
	}

	return tx.Commit()
}

func (m *MySQLDB) GetModels(provider string) ([]ModelInfo, error) {
	rows, err := m.db.Query("SELECT model_id, model_name, provider, description, updated_at FROM ai_models WHERE provider = ? ORDER BY model_id", provider)
	if err != nil {
		return nil, fmt.Errorf("failed to query models: %v", err)
	}
	defer rows.Close()

	var models []ModelInfo
	for rows.Next() {
		var model ModelInfo
		err := rows.Scan(&model.ID, &model.Name, &model.Provider, &model.Description, &model.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model: %v", err)
		}
		models = append(models, model)
	}

	return models, rows.Err()
}

func (m *MySQLDB) GetAllModels() (map[string][]ModelInfo, error) {
	rows, err := m.db.Query("SELECT provider, model_id, model_name, description, updated_at FROM ai_models ORDER BY provider, model_id")
	if err != nil {
		return nil, fmt.Errorf("failed to query all models: %v", err)
	}
	defer rows.Close()

	result := make(map[string][]ModelInfo)
	for rows.Next() {
		var model ModelInfo
		err := rows.Scan(&model.Provider, &model.ID, &model.Name, &model.Description, &model.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model: %v", err)
		}
		result[model.Provider] = append(result[model.Provider], model)
	}

	return result, rows.Err()
}

func (m *MySQLDB) DeleteModels(provider string) error {
	_, err := m.db.Exec("DELETE FROM ai_models WHERE provider = ?", provider)
	if err != nil {
		return fmt.Errorf("failed to delete models for provider %s: %v", provider, err)
	}
	return nil
}

// Telegraph 内容管理方法
func (m *MySQLDB) AddTelegraphEntry(key string, matchType MatchType, value string, contentType string, telegraphURL string, telegraphPath string) error {
	// 暂时简化实现，将 Telegraph URL 存储在 value 字段中
	return m.AddEntry(key, matchType, telegraphURL)
}

func (m *MySQLDB) UpdateTelegraphEntry(key string, matchType MatchType, value string, contentType string, telegraphURL string, telegraphPath string) error {
	// 暂时简化实现
	return m.UpdateEntry(key, matchType, matchType, telegraphURL)
}

func (m *MySQLDB) GetTelegraphContent(key string, matchType MatchType) (*Entry, error) {
	// 暂时使用现有的查询方法
	return m.QueryByID(1, matchType) // 默认ID为1
}

// 模型缓存接口实现
func (m *MySQLDB) SetModelCache(models []config.Model, updatedAt string) error {
	// 简单实现：使用ai_models表的特殊provider来存储缓存
	// 清空现有缓存
	_, err := m.db.Exec("DELETE FROM ai_models WHERE provider = '__cache__'")
	if err != nil {
		return err
	}

	// 插入新的缓存数据
	for _, model := range models {
		_, err = m.db.Exec(`
			INSERT INTO ai_models (provider, model_id, model_name, description, updated_at) 
			VALUES ('__cache__', ?, ?, ?, ?)`,
			model.ID, model.Name, model.Provider, updatedAt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MySQLDB) GetModelCache() ([]config.Model, string, error) {
	rows, err := m.db.Query("SELECT model_id, model_name, description, updated_at FROM ai_models WHERE provider = '__cache__'")
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var models []config.Model
	var cacheTime string

	for rows.Next() {
		var model config.Model
		var modelCacheTime string
		err := rows.Scan(&model.ID, &model.Name, &model.Provider, &modelCacheTime)
		if err != nil {
			return nil, "", err
		}
		models = append(models, model)
		if cacheTime == "" {
			cacheTime = modelCacheTime
		}
	}

	return models, cacheTime, rows.Err()
}

func (m *MySQLDB) ClearModelCache() error {
	_, err := m.db.Exec("DELETE FROM ai_models WHERE provider = '__cache__'")
	return err
}
