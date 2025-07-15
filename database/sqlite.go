package database

import (
	"database/sql"
	"fmt"
	"regexp"

	"TGFaqBot/config"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDB struct {
	filename string
	db       *sql.DB
}

func NewSQLiteDB(filename string) (*SQLiteDB, error) {
	db := &SQLiteDB{filename: filename}
	if err := db.Reload(); err != nil {
		return nil, err
	}
	return db, nil
}

// Implement the combined functions
func (s *SQLiteDB) Query(query string) ([]Entry, error) {
	var allEntries []Entry

	exactEntries, err := s.QueryExact(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, exactEntries...)

	containsEntries, err := s.QueryContains(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, containsEntries...)

	regexEntries, err := s.QueryRegex(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, regexEntries...)

	return allEntries, nil
}

func (s *SQLiteDB) QueryByID(id int, matchType MatchType) (*Entry, error) {
	tableName := matchType.GetTableName()
	if tableName == "" {
		return nil, fmt.Errorf("invalid match type: %s", matchType)
	}

	// 使用白名单验证表名以防止SQL注入
	validTables := map[string]bool{
		"exact":    true,
		"contains": true,
		"regex":    true,
		"prefix":   true,
		"suffix":   true,
	}
	if !validTables[tableName] {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	var query string
	switch tableName {
	case "exact":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM exact WHERE id = ?"
	case "contains":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM contains WHERE id = ?"
	case "regex":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM regex WHERE id = ?"
	case "prefix":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM prefix WHERE id = ?"
	case "suffix":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM suffix WHERE id = ?"
	}
	row := s.db.QueryRow(query, id)

	var entry Entry
	err := row.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entry with ID %d not found in %s", id, tableName)
		}
		return nil, err
	}

	entry.MatchType = matchType // Set the MatchType before returning
	return &entry, nil
}

func (s *SQLiteDB) AddEntry(key string, matchType MatchType, value string) error {
	switch matchType {
	case MatchExact:
		return s.AddEntryExact(key, value)
	case MatchContains:
		return s.AddEntryContains(key, value)
	case MatchRegex:
		return s.AddEntryRegex(key, value)
	default:
		return fmt.Errorf("invalid match type: %s", matchType)
	}
}

func (s *SQLiteDB) UpdateEntry(key string, oldType MatchType, newType MatchType, value string) error {
	if oldType == newType {
		// Same type, use existing UpdateEntryXXX functions
		switch oldType {
		case MatchExact:
			return s.UpdateEntryExact(key, value)
		case MatchContains:
			return s.UpdateEntryContains(key, value)
		case MatchRegex:
			return s.UpdateEntryRegex(key, value)
		default:
			return fmt.Errorf("invalid match type: %s", oldType)
		}
	} else {
		// Different types, delete from old and add to new
		if err := s.DeleteEntry(key, oldType); err != nil {
			return err
		}
		return s.AddEntry(key, newType, value)
	}
}

func (s *SQLiteDB) DeleteEntry(key string, matchType MatchType) error {
	switch matchType {
	case MatchExact:
		return s.DeleteEntryExact(key)
	case MatchContains:
		return s.DeleteEntryContains(key)
	case MatchRegex:
		return s.DeleteEntryRegex(key)
	default:
		return fmt.Errorf("invalid match type: %s", matchType)
	}
}

func (s *SQLiteDB) ListEntries(table string) ([]Entry, error) {
	switch table {
	case "exact":
		return s.ListEntriesExact()
	case "contains":
		return s.ListEntriesContains()
	case "regex":
		return s.ListEntriesRegex()
	default:
		return nil, fmt.Errorf("invalid match type: %s", table)
	}
}

func (s *SQLiteDB) ListAllEntries() ([]Entry, error) {
	// 使用UNION ALL查询一次性获取所有条目，比分别查询三个表更高效
	query := `
		SELECT id, key, value, content_type, telegraph_url, telegraph_path, 1 as match_type FROM exact
		UNION ALL
		SELECT id, key, value, content_type, telegraph_url, telegraph_path, 2 as match_type FROM contains
		UNION ALL
		SELECT id, key, value, content_type, telegraph_url, telegraph_path, 3 as match_type FROM regex
		ORDER BY match_type, id`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allEntries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath, &entry.MatchType); err != nil {
			return nil, err
		}
		allEntries = append(allEntries, entry)
	}

	return allEntries, rows.Err()
}

func (s *SQLiteDB) QueryExact(query string) ([]Entry, error) {
	return s.query(query, "exact")
}

func (s *SQLiteDB) QueryContains(query string) ([]Entry, error) {
	return s.query(query, "contains")
}

func (s *SQLiteDB) QueryRegex(query string) ([]Entry, error) {
	return s.query(query, "regex")
}

func (s *SQLiteDB) query(query string, table string) ([]Entry, error) {
	var rows *sql.Rows
	var err error

	switch table {
	case "exact":
		rows, err = s.db.Query("SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM exact WHERE `key` = ?", query)
	case "contains":
		rows, err = s.db.Query("SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM contains WHERE `key` LIKE '%' || ? || '%'", query)
	case "regex":
		rows, err = s.db.Query("SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM regex") // Regex matching in code
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var entries []Entry
		for rows.Next() {
			var entry Entry
			if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath); err != nil {
				return nil, err
			}
			matched, _ := regexp.MatchString(query, entry.Key)
			if matched {
				entry.MatchType = MatchRegex
				entries = append(entries, entry)
			}
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("invalid table name: %s", table)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath); err != nil {
			return nil, err
		}
		switch table {
		case "exact":
			entry.MatchType = MatchExact
		case "contains":
			entry.MatchType = MatchContains
		case "regex":
			entry.MatchType = MatchRegex
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *SQLiteDB) AddEntryExact(key string, value string) error {
	return s.addEntry(key, value, "exact")
}

func (s *SQLiteDB) AddEntryContains(key string, value string) error {
	return s.addEntry(key, value, "contains")
}

func (s *SQLiteDB) AddEntryRegex(key string, value string) error {
	return s.addEntry(key, value, "regex")
}

func (s *SQLiteDB) addEntry(key string, value string, table string) error {
	// 使用白名单验证表名以防止SQL注入
	validTables := map[string]bool{
		"exact":    true,
		"contains": true,
		"regex":    true,
		"prefix":   true,
		"suffix":   true,
	}
	if !validTables[table] {
		return fmt.Errorf("invalid table name: %s", table)
	}

	var query string
	switch table {
	case "exact":
		query = "INSERT INTO exact (`key`, `value`) VALUES (?, ?)"
	case "contains":
		query = "INSERT INTO contains (`key`, `value`) VALUES (?, ?)"
	case "regex":
		query = "INSERT INTO regex (`key`, `value`) VALUES (?, ?)"
	case "prefix":
		query = "INSERT INTO prefix (`key`, `value`) VALUES (?, ?)"
	case "suffix":
		query = "INSERT INTO suffix (`key`, `value`) VALUES (?, ?)"
	}
	_, err := s.db.Exec(query, key, value)
	return err
}

func (s *SQLiteDB) UpdateEntryExact(key string, value string) error {
	return s.updateEntry(key, value, "exact")
}

func (s *SQLiteDB) UpdateEntryContains(key string, value string) error {
	return s.updateEntry(key, value, "contains")
}

func (s *SQLiteDB) UpdateEntryRegex(key string, value string) error {
	return s.updateEntry(key, value, "regex")
}

func (s *SQLiteDB) updateEntry(key string, value string, table string) error {
	// 使用白名单验证表名以防止SQL注入
	validTables := map[string]bool{
		"exact":    true,
		"contains": true,
		"regex":    true,
		"prefix":   true,
		"suffix":   true,
	}
	if !validTables[table] {
		return fmt.Errorf("invalid table name: %s", table)
	}

	var query string
	switch table {
	case "exact":
		query = "UPDATE exact SET `value` = ? WHERE `key` = ?"
	case "contains":
		query = "UPDATE contains SET `value` = ? WHERE `key` = ?"
	case "regex":
		query = "UPDATE regex SET `value` = ? WHERE `key` = ?"
	case "prefix":
		query = "UPDATE prefix SET `value` = ? WHERE `key` = ?"
	case "suffix":
		query = "UPDATE suffix SET `value` = ? WHERE `key` = ?"
	}
	_, err := s.db.Exec(query, value, key)
	return err
}

func (s *SQLiteDB) DeleteEntryExact(key string) error {
	return s.deleteEntry(key, "exact")
}

func (s *SQLiteDB) DeleteEntryContains(key string) error {
	return s.deleteEntry(key, "contains")
}

func (s *SQLiteDB) DeleteEntryRegex(key string) error {
	return s.deleteEntry(key, "regex")
}

func (s *SQLiteDB) deleteEntry(key string, table string) error {
	// 使用白名单验证表名以防止SQL注入
	validTables := map[string]bool{
		"exact":    true,
		"contains": true,
		"regex":    true,
		"prefix":   true,
		"suffix":   true,
	}
	if !validTables[table] {
		return fmt.Errorf("invalid table name: %s", table)
	}

	var query string
	switch table {
	case "exact":
		query = "DELETE FROM exact WHERE `key` = ?"
	case "contains":
		query = "DELETE FROM contains WHERE `key` = ?"
	case "regex":
		query = "DELETE FROM regex WHERE `key` = ?"
	case "prefix":
		query = "DELETE FROM prefix WHERE `key` = ?"
	case "suffix":
		query = "DELETE FROM suffix WHERE `key` = ?"
	}
	_, err := s.db.Exec(query, key)
	return err
}

func (s *SQLiteDB) ListEntriesExact() ([]Entry, error) {
	return s.listEntries("exact")
}

func (s *SQLiteDB) ListEntriesContains() ([]Entry, error) {
	return s.listEntries("contains")
}

func (s *SQLiteDB) ListEntriesRegex() ([]Entry, error) {
	return s.listEntries("regex")
}

func (s *SQLiteDB) listEntries(table string) ([]Entry, error) {
	// 使用白名单验证表名以防止SQL注入
	validTables := map[string]bool{
		"exact":    true,
		"contains": true,
		"regex":    true,
		"prefix":   true,
		"suffix":   true,
	}
	if !validTables[table] {
		return nil, fmt.Errorf("invalid table name: %s", table)
	}

	var query string
	switch table {
	case "exact":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM exact"
	case "contains":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM contains"
	case "regex":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM regex"
	case "prefix":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM prefix"
	case "suffix":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM suffix"
	}
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath); err != nil {
			return nil, err
		}
		switch table {
		case "exact":
			entry.MatchType = MatchExact
		case "contains":
			entry.MatchType = MatchContains
		case "regex":
			entry.MatchType = MatchRegex
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *SQLiteDB) ListSpecificEntries(matchTypes ...MatchType) ([]Entry, error) {
	if len(matchTypes) == 0 {
		// List all entries if no match types are specified
		return s.ListAllEntries()
	}

	var allEntries []Entry
	for _, matchType := range matchTypes {
		var entries []Entry
		var err error

		switch matchType {
		case MatchExact:
			entries, err = s.ListEntriesExact()
		case MatchContains:
			entries, err = s.ListEntriesContains()
		case MatchRegex:
			entries, err = s.ListEntriesRegex()
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

func (s *SQLiteDB) DeleteAllEntries() error {
	_, err := s.db.Exec("DELETE FROM exact")
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM contains")
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM regex")
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLiteDB) Reload() error {
	var err error
	s.db, err = sql.Open("sqlite3", s.filename)
	if err != nil {
		return err
	}

	// Create tables if not exists
	_, err = s.db.Exec(`
        CREATE TABLE IF NOT EXISTS exact (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS contains (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS regex (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS prefix (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS suffix (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            content_type TEXT DEFAULT 'text',
            telegraph_url TEXT DEFAULT '',
            telegraph_path TEXT DEFAULT ''
        );
        CREATE TABLE IF NOT EXISTS ai_models (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            provider TEXT NOT NULL,
            model_id TEXT NOT NULL,
            model_name TEXT NOT NULL,
            description TEXT DEFAULT '',
            updated_at TEXT NOT NULL,
            UNIQUE(provider, model_id)
        );
        CREATE TABLE IF NOT EXISTS model_cache (
            id INTEGER PRIMARY KEY,
            model_id TEXT NOT NULL,
            model_name TEXT,
            provider TEXT NOT NULL,
            cache_time TEXT NOT NULL
        );
    `)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// 模型管理功能实现
func (s *SQLiteDB) SaveModels(provider string, models []ModelInfo) error {
	// 开始事务
	tx, err := s.db.Begin()
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

func (s *SQLiteDB) GetModels(provider string) ([]ModelInfo, error) {
	rows, err := s.db.Query("SELECT model_id, model_name, provider, description, updated_at FROM ai_models WHERE provider = ? ORDER BY model_id", provider)
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

func (s *SQLiteDB) GetAllModels() (map[string][]ModelInfo, error) {
	rows, err := s.db.Query("SELECT provider, model_id, model_name, description, updated_at FROM ai_models ORDER BY provider, model_id")
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

func (s *SQLiteDB) DeleteModels(provider string) error {
	_, err := s.db.Exec("DELETE FROM ai_models WHERE provider = ?", provider)
	if err != nil {
		return fmt.Errorf("failed to delete models for provider %s: %v", provider, err)
	}
	return nil
}

// Telegraph 内容管理方法
func (s *SQLiteDB) AddTelegraphEntry(key string, matchType MatchType, value string, contentType string, telegraphURL string, telegraphPath string) error {
	tableName := matchType.GetTableName()
	if tableName == "" {
		return fmt.Errorf("invalid match type: %s", matchType)
	}

	// 使用白名单验证表名以防止SQL注入
	validTables := map[string]bool{
		"exact":    true,
		"contains": true,
		"regex":    true,
		"prefix":   true,
		"suffix":   true,
	}
	if !validTables[tableName] {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	var query string
	switch tableName {
	case "exact":
		query = "INSERT INTO exact (`key`, `value`, `content_type`, `telegraph_url`, `telegraph_path`) VALUES (?, ?, ?, ?, ?)"
	case "contains":
		query = "INSERT INTO contains (`key`, `value`, `content_type`, `telegraph_url`, `telegraph_path`) VALUES (?, ?, ?, ?, ?)"
	case "regex":
		query = "INSERT INTO regex (`key`, `value`, `content_type`, `telegraph_url`, `telegraph_path`) VALUES (?, ?, ?, ?, ?)"
	case "prefix":
		query = "INSERT INTO prefix (`key`, `value`, `content_type`, `telegraph_url`, `telegraph_path`) VALUES (?, ?, ?, ?, ?)"
	case "suffix":
		query = "INSERT INTO suffix (`key`, `value`, `content_type`, `telegraph_url`, `telegraph_path`) VALUES (?, ?, ?, ?, ?)"
	}
	_, err := s.db.Exec(query, key, value, contentType, telegraphURL, telegraphPath)
	return err
}

// UpdateTelegraphEntry 更新 Telegraph 条目
func (s *SQLiteDB) UpdateTelegraphEntry(key string, matchType MatchType, value string, contentType string, telegraphURL string, telegraphPath string) error {
	tableName := matchType.GetTableName()
	if tableName == "" {
		return fmt.Errorf("invalid match type: %s", matchType)
	}

	// 使用白名单验证表名以防止SQL注入
	validTables := map[string]bool{
		"exact":    true,
		"contains": true,
		"regex":    true,
		"prefix":   true,
		"suffix":   true,
	}
	if !validTables[tableName] {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	var query string
	switch tableName {
	case "exact":
		query = "UPDATE exact SET `value` = ?, `content_type` = ?, `telegraph_url` = ?, `telegraph_path` = ? WHERE `key` = ?"
	case "contains":
		query = "UPDATE contains SET `value` = ?, `content_type` = ?, `telegraph_url` = ?, `telegraph_path` = ? WHERE `key` = ?"
	case "regex":
		query = "UPDATE regex SET `value` = ?, `content_type` = ?, `telegraph_url` = ?, `telegraph_path` = ? WHERE `key` = ?"
	case "prefix":
		query = "UPDATE prefix SET `value` = ?, `content_type` = ?, `telegraph_url` = ?, `telegraph_path` = ? WHERE `key` = ?"
	case "suffix":
		query = "UPDATE suffix SET `value` = ?, `content_type` = ?, `telegraph_url` = ?, `telegraph_path` = ? WHERE `key` = ?"
	}
	_, err := s.db.Exec(query, value, contentType, telegraphURL, telegraphPath, key)
	return err
}

// GetTelegraphContent 获取 Telegraph 内容
func (s *SQLiteDB) GetTelegraphContent(key string, matchType MatchType) (*Entry, error) {
	tableName := matchType.GetTableName()
	if tableName == "" {
		return nil, fmt.Errorf("invalid match type: %s", matchType)
	}

	// 使用白名单验证表名以防止SQL注入
	validTables := map[string]bool{
		"exact":    true,
		"contains": true,
		"regex":    true,
		"prefix":   true,
		"suffix":   true,
	}
	if !validTables[tableName] {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	var query string
	switch tableName {
	case "exact":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM exact WHERE `key` = ?"
	case "contains":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM contains WHERE `key` = ?"
	case "regex":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM regex WHERE `key` = ?"
	case "prefix":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM prefix WHERE `key` = ?"
	case "suffix":
		query = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM suffix WHERE `key` = ?"
	}
	row := s.db.QueryRow(query, key)

	var entry Entry
	err := row.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entry with key %s not found in %s", key, tableName)
		}
		return nil, err
	}

	entry.MatchType = matchType
	return &entry, nil
}

// 模型缓存接口实现
func (s *SQLiteDB) SetModelCache(models []config.Model, updatedAt string) error {
	// 清空现有缓存
	_, err := s.db.Exec("DELETE FROM model_cache")
	if err != nil {
		return err
	}

	// 插入新的缓存数据
	for i, model := range models {
		_, err = s.db.Exec(`
			INSERT INTO model_cache (id, model_id, model_name, provider, cache_time) 
			VALUES (?, ?, ?, ?, ?)`,
			i+1, model.ID, model.Name, model.Provider, updatedAt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteDB) GetModelCache() ([]config.Model, string, error) {
	rows, err := s.db.Query("SELECT model_id, model_name, provider, cache_time FROM model_cache ORDER BY id")
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

func (s *SQLiteDB) ClearModelCache() error {
	_, err := s.db.Exec("DELETE FROM model_cache")
	return err
}
