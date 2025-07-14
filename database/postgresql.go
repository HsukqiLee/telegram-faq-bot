package database

import (
	"TGFaqBot/config"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type PostgreSQLDB struct {
	db *sql.DB
}

func NewPostgreSQLDB(cfg config.PostgreSQLConfig) (*PostgreSQLDB, error) {
	// 构建连接字符串
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	pgdb := &PostgreSQLDB{db: db}
	if err := pgdb.createTables(); err != nil {
		return nil, err
	}

	return pgdb, nil
}

func (p *PostgreSQLDB) createTables() error {
	// 创建FAQ表
	createFAQTable := `
	CREATE TABLE IF NOT EXISTS faq_entries (
		id SERIAL PRIMARY KEY,
		key_text VARCHAR(255) NOT NULL,
		value_text TEXT NOT NULL,
		match_type INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_faq_key ON faq_entries(key_text);
	CREATE INDEX IF NOT EXISTS idx_faq_match_type ON faq_entries(match_type);
	`

	// 创建模型表
	createModelTable := `
	CREATE TABLE IF NOT EXISTS ai_models (
		id SERIAL PRIMARY KEY,
		provider VARCHAR(50) NOT NULL,
		model_id VARCHAR(255) NOT NULL,
		model_name VARCHAR(255) NOT NULL,
		description TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(provider, model_id)
	);
	
	CREATE INDEX IF NOT EXISTS idx_models_provider ON ai_models(provider);
	`

	if _, err := p.db.Exec(createFAQTable); err != nil {
		return fmt.Errorf("failed to create FAQ table: %v", err)
	}

	if _, err := p.db.Exec(createModelTable); err != nil {
		return fmt.Errorf("failed to create model table: %v", err)
	}

	return nil
}

// FAQ查询方法
func (p *PostgreSQLDB) Query(query string) ([]Entry, error) {
	var allEntries []Entry

	// 精确匹配
	exactEntries, err := p.QueryExact(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, exactEntries...)

	// 包含匹配
	containsEntries, err := p.QueryContains(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, containsEntries...)

	// 正则匹配
	regexEntries, err := p.QueryRegex(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, regexEntries...)

	return allEntries, nil
}

func (p *PostgreSQLDB) QueryByID(id int, matchType int) (*Entry, error) {
	query := `SELECT id, key_text, value_text, match_type FROM faq_entries WHERE id = $1 AND match_type = $2`
	row := p.db.QueryRow(query, id, matchType)

	var entry Entry
	err := row.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.MatchType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &entry, nil
}

func (p *PostgreSQLDB) QueryExact(query string) ([]Entry, error) {
	sqlQuery := `SELECT id, key_text, value_text, match_type FROM faq_entries WHERE match_type = 1 AND key_text = $1`
	return p.queryWithSQL(sqlQuery, query)
}

func (p *PostgreSQLDB) QueryContains(query string) ([]Entry, error) {
	sqlQuery := `SELECT id, key_text, value_text, match_type FROM faq_entries WHERE match_type = 2 AND key_text ILIKE $1`
	return p.queryWithSQL(sqlQuery, "%"+query+"%")
}

func (p *PostgreSQLDB) QueryRegex(query string) ([]Entry, error) {
	sqlQuery := `SELECT id, key_text, value_text, match_type FROM faq_entries WHERE match_type = 3`
	rows, err := p.db.Query(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.MatchType); err != nil {
			return nil, err
		}

		// 在应用层进行正则匹配
		matched, err := regexp.MatchString(entry.Key, query)
		if err != nil {
			continue // 跳过无效的正则表达式
		}
		if matched {
			entries = append(entries, entry)
		}
	}

	return entries, rows.Err()
}

func (p *PostgreSQLDB) queryWithSQL(sqlQuery string, args ...interface{}) ([]Entry, error) {
	rows, err := p.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.MatchType); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// FAQ管理方法
func (p *PostgreSQLDB) AddEntry(key string, matchType int, value string) error {
	query := `INSERT INTO faq_entries (key_text, value_text, match_type) VALUES ($1, $2, $3)`
	_, err := p.db.Exec(query, key, value, matchType)
	return err
}

func (p *PostgreSQLDB) UpdateEntry(key string, oldType int, newType int, value string) error {
	query := `UPDATE faq_entries SET value_text = $1, match_type = $2, updated_at = CURRENT_TIMESTAMP WHERE key_text = $3 AND match_type = $4`
	_, err := p.db.Exec(query, value, newType, key, oldType)
	return err
}

func (p *PostgreSQLDB) DeleteEntry(key string, matchType int) error {
	query := `DELETE FROM faq_entries WHERE key_text = $1 AND match_type = $2`
	_, err := p.db.Exec(query, key, matchType)
	return err
}

func (p *PostgreSQLDB) DeleteAllEntries() error {
	query := `DELETE FROM faq_entries`
	_, err := p.db.Exec(query)
	return err
}

// 列表方法
func (p *PostgreSQLDB) ListEntries(table string) ([]Entry, error) {
	return p.ListAllEntries()
}

func (p *PostgreSQLDB) ListSpecificEntries(matchTypes ...int) ([]Entry, error) {
	if len(matchTypes) == 0 {
		return p.ListAllEntries()
	}

	placeholders := make([]string, len(matchTypes))
	args := make([]interface{}, len(matchTypes))
	for i, mt := range matchTypes {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = mt
	}

	query := fmt.Sprintf(`SELECT id, key_text, value_text, match_type FROM faq_entries WHERE match_type IN (%s) ORDER BY id`,
		strings.Join(placeholders, ","))

	return p.queryWithSQL(query, args...)
}

func (p *PostgreSQLDB) ListAllEntries() ([]Entry, error) {
	query := `SELECT id, key_text, value_text, match_type FROM faq_entries ORDER BY id`
	return p.queryWithSQL(query)
}

// 特定类型的方法
func (p *PostgreSQLDB) AddEntryExact(key string, value string) error {
	return p.AddEntry(key, 1, value)
}

func (p *PostgreSQLDB) AddEntryContains(key string, value string) error {
	return p.AddEntry(key, 2, value)
}

func (p *PostgreSQLDB) AddEntryRegex(key string, value string) error {
	return p.AddEntry(key, 3, value)
}

func (p *PostgreSQLDB) UpdateEntryExact(key string, value string) error {
	return p.UpdateEntry(key, 1, 1, value)
}

func (p *PostgreSQLDB) UpdateEntryContains(key string, value string) error {
	return p.UpdateEntry(key, 2, 2, value)
}

func (p *PostgreSQLDB) UpdateEntryRegex(key string, value string) error {
	return p.UpdateEntry(key, 3, 3, value)
}

func (p *PostgreSQLDB) DeleteEntryExact(key string) error {
	return p.DeleteEntry(key, 1)
}

func (p *PostgreSQLDB) DeleteEntryContains(key string) error {
	return p.DeleteEntry(key, 2)
}

func (p *PostgreSQLDB) DeleteEntryRegex(key string) error {
	return p.DeleteEntry(key, 3)
}

func (p *PostgreSQLDB) ListEntriesExact() ([]Entry, error) {
	return p.ListSpecificEntries(1)
}

func (p *PostgreSQLDB) ListEntriesContains() ([]Entry, error) {
	return p.ListSpecificEntries(2)
}

func (p *PostgreSQLDB) ListEntriesRegex() ([]Entry, error) {
	return p.ListSpecificEntries(3)
}

// 模型管理方法
func (p *PostgreSQLDB) SaveModels(provider string, models []ModelInfo) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除该提供商的旧模型
	if _, err := tx.Exec(`DELETE FROM ai_models WHERE provider = $1`, provider); err != nil {
		return err
	}

	// 插入新模型
	stmt, err := tx.Prepare(`INSERT INTO ai_models (provider, model_id, model_name, description, updated_at) VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, model := range models {
		if _, err := stmt.Exec(provider, model.ID, model.Name, model.Description, time.Now()); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *PostgreSQLDB) GetModels(provider string) ([]ModelInfo, error) {
	query := `SELECT model_id, model_name, provider, description, updated_at FROM ai_models WHERE provider = $1 ORDER BY model_name`
	rows, err := p.db.Query(query, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []ModelInfo
	for rows.Next() {
		var model ModelInfo
		var updatedAt time.Time
		if err := rows.Scan(&model.ID, &model.Name, &model.Provider, &model.Description, &updatedAt); err != nil {
			return nil, err
		}
		model.UpdatedAt = updatedAt.Format(time.RFC3339)
		models = append(models, model)
	}

	return models, rows.Err()
}

func (p *PostgreSQLDB) GetAllModels() (map[string][]ModelInfo, error) {
	query := `SELECT provider, model_id, model_name, description, updated_at FROM ai_models ORDER BY provider, model_name`
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]ModelInfo)
	for rows.Next() {
		var model ModelInfo
		var updatedAt time.Time
		if err := rows.Scan(&model.Provider, &model.ID, &model.Name, &model.Description, &updatedAt); err != nil {
			return nil, err
		}
		model.UpdatedAt = updatedAt.Format(time.RFC3339)
		result[model.Provider] = append(result[model.Provider], model)
	}

	return result, rows.Err()
}

func (p *PostgreSQLDB) DeleteModels(provider string) error {
	query := `DELETE FROM ai_models WHERE provider = $1`
	_, err := p.db.Exec(query, provider)
	return err
}

// 系统方法
func (p *PostgreSQLDB) Reload() error {
	// PostgreSQL不需要重新加载，因为数据是实时的
	return nil
}

func (p *PostgreSQLDB) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
