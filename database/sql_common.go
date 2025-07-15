package database

import (
	"database/sql"
	"fmt"
	"regexp"
)

// SQLDatabaseBase 提供SQL数据库的通用功能 (已移除未使用的字段)

// TableValidator 表名验证器
type TableValidator struct {
	validTables map[string]bool
}

// NewTableValidator 创建新的表名验证器
func NewTableValidator() *TableValidator {
	return &TableValidator{
		validTables: map[string]bool{
			"exact":    true,
			"contains": true,
			"regex":    true,
			"prefix":   true,
			"suffix":   true,
		},
	}
}

// ValidateTable 验证表名是否有效
func (tv *TableValidator) ValidateTable(tableName string) error {
	if !tv.validTables[tableName] {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	return nil
}

// SQLQueryBuilder 构建安全的SQL查询语句
type SQLQueryBuilder struct {
	validator *TableValidator
}

// NewSQLQueryBuilder 创建新的SQL查询构建器
func NewSQLQueryBuilder() *SQLQueryBuilder {
	return &SQLQueryBuilder{
		validator: NewTableValidator(),
	}
}

// BuildSelectByID 构建按ID查询的SQL语句
func (qb *SQLQueryBuilder) BuildSelectByID(tableName string, columns []string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}

	// 使用预定义的安全查询模式
	var query string
	if len(columns) == 6 {
		// 完整列查询（适用于SQLite扩展字段）
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
		default:
			return "", fmt.Errorf("unsupported table: %s", tableName)
		}
	} else {
		// 基本列查询
		switch tableName {
		case "exact":
			query = "SELECT id, `key`, `value` FROM exact WHERE id = ?"
		case "contains":
			query = "SELECT id, `key`, `value` FROM contains WHERE id = ?"
		case "regex":
			query = "SELECT id, `key`, `value` FROM regex WHERE id = ?"
		case "prefix":
			query = "SELECT id, `key`, `value` FROM prefix WHERE id = ?"
		case "suffix":
			query = "SELECT id, `key`, `value` FROM suffix WHERE id = ?"
		default:
			return "", fmt.Errorf("unsupported table: %s", tableName)
		}
	}

	return query, nil
}

// BuildSelectAll 构建查询所有记录的SQL语句
func (qb *SQLQueryBuilder) BuildSelectAll(tableName string, columns []string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}

	// 使用预定义的安全查询模式
	var query string
	if len(columns) == 6 {
		// 完整列查询（适用于SQLite扩展字段）
		switch tableName {
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
		default:
			return "", fmt.Errorf("unsupported table: %s", tableName)
		}
	} else {
		// 基本列查询
		switch tableName {
		case "exact":
			query = "SELECT id, `key`, `value` FROM exact"
		case "contains":
			query = "SELECT id, `key`, `value` FROM contains"
		case "regex":
			query = "SELECT id, `key`, `value` FROM regex"
		case "prefix":
			query = "SELECT id, `key`, `value` FROM prefix"
		case "suffix":
			query = "SELECT id, `key`, `value` FROM suffix"
		default:
			return "", fmt.Errorf("unsupported table: %s", tableName)
		}
	}

	return query, nil
}

// BuildInsert 构建插入记录的SQL语句
func (qb *SQLQueryBuilder) BuildInsert(tableName string, columns []string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}

	// 使用预定义的安全查询模式
	var query string
	if len(columns) == 5 {
		// 完整插入（包括Telegraph字段）
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
		default:
			return "", fmt.Errorf("unsupported table: %s", tableName)
		}
	} else {
		// 基本插入（key, value）
		switch tableName {
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
		default:
			return "", fmt.Errorf("unsupported table: %s", tableName)
		}
	}

	return query, nil
}

// BuildUpdate 构建更新记录的SQL语句
func (qb *SQLQueryBuilder) BuildUpdate(tableName string, setColumns []string, whereColumn string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}

	// 使用预定义的安全查询模式
	var query string
	if len(setColumns) == 4 {
		// 完整更新（包括Telegraph字段）
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
		default:
			return "", fmt.Errorf("unsupported table: %s", tableName)
		}
	} else {
		// 基本更新（只更新value）
		switch tableName {
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
		default:
			return "", fmt.Errorf("unsupported table: %s", tableName)
		}
	}

	return query, nil
}

// BuildDelete 构建删除记录的SQL语句
func (qb *SQLQueryBuilder) BuildDelete(tableName string, whereColumn string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}

	// 使用预定义的安全查询模式（总是使用key作为条件）
	var query string
	switch tableName {
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
	default:
		return "", fmt.Errorf("unsupported table: %s", tableName)
	}

	return query, nil
}

// CommonSQLOperations 通用的SQL操作
type CommonSQLOperations struct {
	db      *sql.DB
	builder *SQLQueryBuilder
}

// NewCommonSQLOperations 创建通用SQL操作实例
func NewCommonSQLOperations(db *sql.DB) *CommonSQLOperations {
	return &CommonSQLOperations{
		db:      db,
		builder: NewSQLQueryBuilder(),
	}
}

// QueryByID 通用的按ID查询方法
func (ops *CommonSQLOperations) QueryByID(id int, matchType MatchType, columns []string) (*Entry, error) {
	tableName := matchType.GetTableName()
	if tableName == "" {
		return nil, fmt.Errorf("invalid match type: %s", matchType)
	}

	query, err := ops.builder.BuildSelectByID(tableName, columns)
	if err != nil {
		return nil, err
	}

	row := ops.db.QueryRow(query, id)

	var entry Entry
	if len(columns) == 0 || len(columns) == 3 {
		// 基本查询：id, key, value
		err = row.Scan(&entry.ID, &entry.Key, &entry.Value)
	} else if len(columns) == 6 {
		// 完整查询：id, key, value, content_type, telegraph_url, telegraph_path
		err = row.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath)
	} else {
		return nil, fmt.Errorf("unsupported column count: %d", len(columns))
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entry with ID %d not found in %s", id, tableName)
		}
		return nil, err
	}

	entry.MatchType = matchType
	return &entry, nil
}

// AddEntry 通用的添加记录方法
func (ops *CommonSQLOperations) AddEntry(key, value, tableName string, extraArgs ...interface{}) error {
	var columns []string
	var args []interface{}

	if len(extraArgs) == 0 {
		// 基本插入：key, value
		columns = []string{"`key`", "`value`"}
		args = []interface{}{key, value}
	} else if len(extraArgs) == 3 {
		// 完整插入：key, value, content_type, telegraph_url, telegraph_path
		columns = []string{"`key`", "`value`", "`content_type`", "`telegraph_url`", "`telegraph_path`"}
		args = []interface{}{key, value, extraArgs[0], extraArgs[1], extraArgs[2]}
	} else {
		return fmt.Errorf("invalid number of extra arguments: %d", len(extraArgs))
	}

	query, err := ops.builder.BuildInsert(tableName, columns)
	if err != nil {
		return err
	}

	_, err = ops.db.Exec(query, args...)
	return err
}

// UpdateEntry 通用的更新记录方法
func (ops *CommonSQLOperations) UpdateEntry(key, value, tableName string, extraArgs ...interface{}) error {
	var setColumns []string
	var args []interface{}

	if len(extraArgs) == 0 {
		// 基本更新：value
		setColumns = []string{"`value`"}
		args = []interface{}{value, key}
	} else if len(extraArgs) == 3 {
		// 完整更新：value, content_type, telegraph_url, telegraph_path
		setColumns = []string{"`value`", "`content_type`", "`telegraph_url`", "`telegraph_path`"}
		args = []interface{}{value, extraArgs[0], extraArgs[1], extraArgs[2], key}
	} else {
		return fmt.Errorf("invalid number of extra arguments: %d", len(extraArgs))
	}

	query, err := ops.builder.BuildUpdate(tableName, setColumns, "`key`")
	if err != nil {
		return err
	}

	_, err = ops.db.Exec(query, args...)
	return err
}

// DeleteEntry 通用的删除记录方法
func (ops *CommonSQLOperations) DeleteEntry(key, tableName string) error {
	query, err := ops.builder.BuildDelete(tableName, "`key`")
	if err != nil {
		return err
	}

	_, err = ops.db.Exec(query, key)
	return err
}

// ListEntries 通用的列出记录方法
func (ops *CommonSQLOperations) ListEntries(tableName string, columns []string) ([]Entry, error) {
	query, err := ops.builder.BuildSelectAll(tableName, columns)
	if err != nil {
		return nil, err
	}

	rows, err := ops.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		if len(columns) == 0 || len(columns) == 3 {
			// 基本查询：id, key, value
			if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value); err != nil {
				return nil, err
			}
		} else if len(columns) == 6 {
			// 完整查询：id, key, value, content_type, telegraph_url, telegraph_path
			if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unsupported column count: %d", len(columns))
		}

		// 根据表名设置MatchType
		switch tableName {
		case "exact":
			entry.MatchType = MatchExact
		case "contains":
			entry.MatchType = MatchContains
		case "regex":
			entry.MatchType = MatchRegex
		case "prefix":
			entry.MatchType = MatchPrefix
		case "suffix":
			entry.MatchType = MatchSuffix
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

// QueryWithCondition 通用的条件查询方法
func (ops *CommonSQLOperations) QueryWithCondition(query, tableName string, columns []string) ([]Entry, error) {
	var sqlQuery string
	var rows *sql.Rows
	var err error

	// 使用预定义的查询字符串而不是动态构建以防止SQL注入
	switch tableName {
	case "exact":
		if len(columns) == 6 {
			sqlQuery = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM exact WHERE `key` = ?"
		} else {
			sqlQuery = "SELECT id, `key`, `value` FROM exact WHERE `key` = ?"
		}
		rows, err = ops.db.Query(sqlQuery, query)
	case "contains":
		if len(columns) == 6 {
			// SQLite syntax
			sqlQuery = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM contains WHERE `key` LIKE '%' || ? || '%'"
		} else {
			// MySQL syntax
			sqlQuery = "SELECT id, `key`, `value` FROM contains WHERE `key` LIKE CONCAT('%', ?, '%')"
		}
		rows, err = ops.db.Query(sqlQuery, query)
	case "regex":
		// Regex matching needs to be done in application code
		if len(columns) == 6 {
			sqlQuery = "SELECT id, key, value, content_type, telegraph_url, telegraph_path FROM regex"
		} else {
			sqlQuery = "SELECT id, `key`, `value` FROM regex"
		}
		rows, err = ops.db.Query(sqlQuery)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var entries []Entry
		for rows.Next() {
			var entry Entry
			if len(columns) == 6 {
				if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath); err != nil {
					return nil, err
				}
			} else {
				if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value); err != nil {
					return nil, err
				}
			}
			matched, _ := regexp.MatchString(query, entry.Key)
			if matched {
				entry.MatchType = MatchRegex
				entries = append(entries, entry)
			}
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		if len(columns) == 6 {
			if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.ContentType, &entry.TelegraphURL, &entry.TelegraphPath); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value); err != nil {
				return nil, err
			}
		}

		// 根据表名设置MatchType
		switch tableName {
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
