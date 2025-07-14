package database

import (
	"database/sql"
	"fmt"
	"regexp"

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

func (s *SQLiteDB) QueryByID(id int, matchType int) (*Entry, error) {
	var tableName string
	switch matchType {
	case 1:
		tableName = "exact"
	case 2:
		tableName = "contains"
	case 3:
		tableName = "regex"
	default:
		return nil, fmt.Errorf("invalid match type: %d", matchType)
	}

	query := fmt.Sprintf("SELECT id, key, value FROM %s WHERE id = ?", tableName)
	row := s.db.QueryRow(query, id)

	var entry Entry
	err := row.Scan(&entry.ID, &entry.Key, &entry.Value)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entry with ID %d not found in %s", id, tableName)
		}
		return nil, err
	}

	entry.MatchType = matchType // Set the MatchType before returning
	return &entry, nil
}

func (s *SQLiteDB) AddEntry(key string, matchType int, value string) error {
	switch matchType {
	case 1:
		return s.AddEntryExact(key, value)
	case 2:
		return s.AddEntryContains(key, value)
	case 3:
		return s.AddEntryRegex(key, value)
	default:
		return fmt.Errorf("invalid match type: %d", matchType)
	}
}

func (s *SQLiteDB) UpdateEntry(key string, oldType int, newType int, value string) error {
	if oldType == newType {
		// Same type, use existing UpdateEntryXXX functions
		switch oldType {
		case 1:
			return s.UpdateEntryExact(key, value)
		case 2:
			return s.UpdateEntryContains(key, value)
		case 3:
			return s.UpdateEntryRegex(key, value)
		default:
			return fmt.Errorf("invalid match type: %d", oldType)
		}
	} else {
		// Different types, delete from old and add to new
		if err := s.DeleteEntry(key, oldType); err != nil {
			return err
		}
		return s.AddEntry(key, newType, value)
	}
}

func (s *SQLiteDB) DeleteEntry(key string, matchType int) error {
	switch matchType {
	case 1:
		return s.DeleteEntryExact(key)
	case 2:
		return s.DeleteEntryContains(key)
	case 3:
		return s.DeleteEntryRegex(key)
	default:
		return fmt.Errorf("invalid match type: %d", matchType)
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
	var allEntries []Entry

	exactEntries, err := s.ListEntriesExact()
	if err != nil {
		return nil, err
	}
	for i := range exactEntries {
		exactEntries[i].MatchType = 1
	}
	allEntries = append(allEntries, exactEntries...)

	containsEntries, err := s.ListEntriesContains()
	if err != nil {
		return nil, err
	}
	for i := range containsEntries {
		containsEntries[i].MatchType = 2
	}
	allEntries = append(allEntries, containsEntries...)

	regexEntries, err := s.ListEntriesRegex()
	if err != nil {
		return nil, err
	}
	for i := range regexEntries {
		regexEntries[i].MatchType = 3
	}
	allEntries = append(allEntries, regexEntries...)

	return allEntries, nil
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
		rows, err = s.db.Query("SELECT id, key, value FROM exact WHERE `key` = ?", query)
	case "contains":
		rows, err = s.db.Query("SELECT id, key, value FROM contains WHERE `key` LIKE '%' || ? || '%'", query)
	case "regex":
		rows, err = s.db.Query("SELECT id, key, value FROM regex") // Regex matching in code
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var entries []Entry
		for rows.Next() {
			var entry Entry
			if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value); err != nil {
				return nil, err
			}
			matched, _ := regexp.MatchString(query, entry.Key)
			if matched {
				entry.MatchType = 3
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
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value); err != nil {
			return nil, err
		}
		switch table {
		case "exact":
			entry.MatchType = 1
		case "contains":
			entry.MatchType = 2
		case "regex":
			entry.MatchType = 3
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
	_, err := s.db.Exec(fmt.Sprintf("INSERT INTO %s (`key`, `value`) VALUES (?, ?)", table), key, value)
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
	_, err := s.db.Exec(fmt.Sprintf("UPDATE %s SET `value` = ? WHERE `key` = ?", table), value, key)
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
	_, err := s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE `key` = ?", table), key)
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
	rows, err := s.db.Query(fmt.Sprintf("SELECT id, key, value FROM %s", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value); err != nil {
			return nil, err
		}
		switch table {
		case "exact":
			entry.MatchType = 1
		case "contains":
			entry.MatchType = 2
		case "regex":
			entry.MatchType = 3
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *SQLiteDB) ListSpecificEntries(matchTypes ...int) ([]Entry, error) {
	if len(matchTypes) == 0 {
		// List all entries if no match types are specified
		return s.ListAllEntries()
	}

	var allEntries []Entry
	for _, matchType := range matchTypes {
		var entries []Entry
		var err error

		switch matchType {
		case 1:
			entries, err = s.ListEntriesExact()
		case 2:
			entries, err = s.ListEntriesContains()
		case 3:
			entries, err = s.ListEntriesRegex()
		default:
			return nil, fmt.Errorf("invalid match type: %d", matchType)
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
            value TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS contains (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS regex (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL
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

// 模型管理功能占位符实现（可在未来完善）
func (s *SQLiteDB) SaveModels(provider string, models []ModelInfo) error {
	// TODO: 实现SQLite的模型存储功能
	// 暂时返回成功，模型数据可以缓存在内存中
	return nil
}

func (s *SQLiteDB) GetModels(provider string) ([]ModelInfo, error) {
	// TODO: 实现SQLite的模型获取功能
	return []ModelInfo{}, nil
}

func (s *SQLiteDB) GetAllModels() (map[string][]ModelInfo, error) {
	// TODO: 实现SQLite的所有模型获取功能
	return make(map[string][]ModelInfo), nil
}

func (s *SQLiteDB) DeleteModels(provider string) error {
	// TODO: 实现SQLite的模型删除功能
	return nil
}
