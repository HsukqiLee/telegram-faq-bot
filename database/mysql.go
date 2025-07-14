package database

import (
	"TGFaqBot/config"
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDB struct {
	cfg config.MySQLConfig
	db  *sql.DB
}

func NewMySQLDB(cfg config.MySQLConfig) (*MySQLDB, error) {
	db := &MySQLDB{cfg: cfg}
	if err := db.Reload(); err != nil {
		return nil, err
	}
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

func (m *MySQLDB) QueryByID(id int, matchType int) (*Entry, error) {
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

	query := fmt.Sprintf("SELECT id, `key`, `value` FROM %s WHERE id = ?", tableName)
	row := m.db.QueryRow(query, id)

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

func (m *MySQLDB) AddEntry(key string, matchType int, value string) error {
	switch matchType {
	case 1: // Exact
		return m.AddEntryExact(key, value)
	case 2: // Contains
		return m.AddEntryContains(key, value)
	case 3: // Regex
		return m.AddEntryRegex(key, value)
	default:
		return fmt.Errorf("invalid match type: %d", matchType)
	}
}

func (m *MySQLDB) UpdateEntry(key string, oldType int, newType int, value string) error {
	if oldType == newType {
		// Same type, use existing UpdateEntryXXX functions
		switch oldType {
		case 1:
			return m.UpdateEntryExact(key, value)
		case 2:
			return m.UpdateEntryContains(key, value)
		case 3:
			return m.UpdateEntryRegex(key, value)
		default:
			return fmt.Errorf("invalid match type: %d", oldType)
		}
	} else {
		// Different types, delete from old and add to new
		if err := m.DeleteEntry(key, oldType); err != nil {
			return err
		}
		return m.AddEntry(key, newType, value)
	}
}

func (m *MySQLDB) DeleteEntry(key string, matchType int) error {
	switch matchType {
	case 1: // Exact
		return m.DeleteEntryExact(key)
	case 2: // Contains
		return m.DeleteEntryContains(key)
	case 3: // Regex
		return m.DeleteEntryRegex(key)
	default:
		return fmt.Errorf("invalid match type: %d", matchType)
	}
}

func (m *MySQLDB) ListEntries(table string) ([]Entry, error) {
	query := fmt.Sprintf("SELECT id, `key`, `value` FROM %s", table)
	rows, err := m.db.Query(query)
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

func (m *MySQLDB) ListAllEntries() ([]Entry, error) {
	var allEntries []Entry

	exactEntries, err := m.ListEntriesExact()
	if err != nil {
		return nil, err
	}
	for i := range exactEntries {
		exactEntries[i].MatchType = 1
	}
	allEntries = append(allEntries, exactEntries...)

	containsEntries, err := m.ListEntriesContains()
	if err != nil {
		return nil, err
	}
	for i := range containsEntries {
		containsEntries[i].MatchType = 2
	}
	allEntries = append(allEntries, containsEntries...)

	regexEntries, err := m.ListEntriesRegex()
	if err != nil {
		return nil, err
	}
	for i := range regexEntries {
		regexEntries[i].MatchType = 3
	}
	allEntries = append(allEntries, regexEntries...)

	return allEntries, nil
}

func (m *MySQLDB) QueryExact(query string) ([]Entry, error) {
	return m.query(query, "exact")
}

func (m *MySQLDB) QueryContains(query string) ([]Entry, error) {
	return m.query(query, "contains")
}

func (m *MySQLDB) QueryRegex(query string) ([]Entry, error) {
	return m.query(query, "regex")
}

func (m *MySQLDB) query(query string, table string) ([]Entry, error) {
	var rows *sql.Rows
	var err error

	switch table {
	case "exact":
		rows, err = m.db.Query("SELECT id, `key`, `value` FROM exact WHERE `key` = ?", query)
	case "contains":
		rows, err = m.db.Query("SELECT id, `key`, `value` FROM contains WHERE `key` LIKE CONCAT('%', ?, '%')", query)
	case "regex":
		rows, err = m.db.Query("SELECT id, `key`, `value` FROM regex") // Regex matching in code
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

func (m *MySQLDB) AddEntryExact(key string, value string) error {
	return m.addEntry(key, value, "exact")
}

func (m *MySQLDB) AddEntryContains(key string, value string) error {
	return m.addEntry(key, value, "contains")
}

func (m *MySQLDB) AddEntryRegex(key string, value string) error {
	return m.addEntry(key, value, "regex")
}

func (m *MySQLDB) addEntry(key string, value string, table string) error {
	_, err := m.db.Exec(fmt.Sprintf("INSERT INTO %s (`key`, `value`) VALUES (?, ?)", table), key, value)
	return err
}

func (m *MySQLDB) UpdateEntryExact(key string, value string) error {
	return m.updateEntry(key, value, "exact")
}

func (m *MySQLDB) UpdateEntryContains(key string, value string) error {
	return m.updateEntry(key, value, "contains")
}

func (m *MySQLDB) UpdateEntryRegex(key string, value string) error {
	return m.updateEntry(key, value, "regex")
}

func (m *MySQLDB) updateEntry(key string, value string, table string) error {
	_, err := m.db.Exec(fmt.Sprintf("UPDATE %s SET `value` = ? WHERE `key` = ?", table), value, key)
	return err
}

func (m *MySQLDB) DeleteEntryExact(key string) error {
	return m.deleteEntry(key, "exact")
}

func (m *MySQLDB) DeleteEntryContains(key string) error {
	return m.deleteEntry(key, "contains")
}

func (m *MySQLDB) DeleteEntryRegex(key string) error {
	return m.deleteEntry(key, "regex")
}

func (m *MySQLDB) deleteEntry(key string, table string) error {
	_, err := m.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE `key` = ?", table), key)
	return err
}

func (m *MySQLDB) ListEntriesExact() ([]Entry, error) {
	return m.listEntries("exact")
}

func (m *MySQLDB) ListEntriesContains() ([]Entry, error) {
	return m.listEntries("contains")
}

func (m *MySQLDB) ListEntriesRegex() ([]Entry, error) {
	return m.listEntries("regex")
}

func (m *MySQLDB) listEntries(table string) ([]Entry, error) {
	rows, err := m.db.Query(fmt.Sprintf("SELECT id, `key`, `value` FROM %s", table))
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
		entries = append(entries, entry)
	}
	return entries, nil
}

func (m *MySQLDB) ListSpecificEntries(matchTypes ...int) ([]Entry, error) {
	if len(matchTypes) == 0 {
		// List all entries if no match types are specified
		return m.ListAllEntries()
	}

	var allEntries []Entry
	for _, matchType := range matchTypes {
		var entries []Entry
		var err error

		switch matchType {
		case 1:
			entries, err = m.ListEntriesExact()
		case 2:
			entries, err = m.ListEntriesContains()
		case 3:
			entries, err = m.ListEntriesRegex()
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
            key TEXT NOT NULL,
            value TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS contains (
            id INTEGER PRIMARY KEY AUTO_INCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS regex (
            id INTEGER PRIMARY KEY AUTO_INCREMENT,
            key TEXT NOT NULL,
            value TEXT NOT NULL
        );
    `)
	if err != nil {
		return err
	}
	return nil
}

func (m *MySQLDB) Close() error {
	return m.db.Close()
}

// 模型管理功能占位符实现（可在未来完善）
func (m *MySQLDB) SaveModels(provider string, models []ModelInfo) error {
	// TODO: 实现MySQL的模型存储功能
	// 暂时返回成功，模型数据可以缓存在内存中
	return nil
}

func (m *MySQLDB) GetModels(provider string) ([]ModelInfo, error) {
	// TODO: 实现MySQL的模型获取功能
	return []ModelInfo{}, nil
}

func (m *MySQLDB) GetAllModels() (map[string][]ModelInfo, error) {
	// TODO: 实现MySQL的所有模型获取功能
	return make(map[string][]ModelInfo), nil
}

func (m *MySQLDB) DeleteModels(provider string) error {
	// TODO: 实现MySQL的模型删除功能
	return nil
}
