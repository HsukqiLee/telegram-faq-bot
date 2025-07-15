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
	
	columnList := "id, `key`, `value`"
	if len(columns) > 0 {
		columnList = ""
		for i, col := range columns {
			if i > 0 {
				columnList += ", "
			}
			columnList += col
		}
	}
	
	var query string
	switch tableName {
	case "exact":
		query = fmt.Sprintf("SELECT %s FROM exact WHERE id = ?", columnList)
	case "contains":
		query = fmt.Sprintf("SELECT %s FROM contains WHERE id = ?", columnList)
	case "regex":
		query = fmt.Sprintf("SELECT %s FROM regex WHERE id = ?", columnList)
	case "prefix":
		query = fmt.Sprintf("SELECT %s FROM prefix WHERE id = ?", columnList)
	case "suffix":
		query = fmt.Sprintf("SELECT %s FROM suffix WHERE id = ?", columnList)
	default:
		return "", fmt.Errorf("unsupported table: %s", tableName)
	}
	
	return query, nil
}

// BuildSelectAll 构建查询所有记录的SQL语句
func (qb *SQLQueryBuilder) BuildSelectAll(tableName string, columns []string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}
	
	columnList := "id, `key`, `value`"
	if len(columns) > 0 {
		columnList = ""
		for i, col := range columns {
			if i > 0 {
				columnList += ", "
			}
			columnList += col
		}
	}
	
	var query string
	switch tableName {
	case "exact":
		query = fmt.Sprintf("SELECT %s FROM exact", columnList)
	case "contains":
		query = fmt.Sprintf("SELECT %s FROM contains", columnList)
	case "regex":
		query = fmt.Sprintf("SELECT %s FROM regex", columnList)
	case "prefix":
		query = fmt.Sprintf("SELECT %s FROM prefix", columnList)
	case "suffix":
		query = fmt.Sprintf("SELECT %s FROM suffix", columnList)
	default:
		return "", fmt.Errorf("unsupported table: %s", tableName)
	}
	
	return query, nil
}

// BuildInsert 构建插入记录的SQL语句
func (qb *SQLQueryBuilder) BuildInsert(tableName string, columns []string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}
	
	columnList := "`key`, `value`"
	placeholders := "?, ?"
	
	if len(columns) > 0 {
		columnList = ""
		placeholders = ""
		for i, col := range columns {
			if i > 0 {
				columnList += ", "
				placeholders += ", "
			}
			columnList += col
			placeholders += "?"
		}
	}
	
	var query string
	switch tableName {
	case "exact":
		query = fmt.Sprintf("INSERT INTO exact (%s) VALUES (%s)", columnList, placeholders)
	case "contains":
		query = fmt.Sprintf("INSERT INTO contains (%s) VALUES (%s)", columnList, placeholders)
	case "regex":
		query = fmt.Sprintf("INSERT INTO regex (%s) VALUES (%s)", columnList, placeholders)
	case "prefix":
		query = fmt.Sprintf("INSERT INTO prefix (%s) VALUES (%s)", columnList, placeholders)
	case "suffix":
		query = fmt.Sprintf("INSERT INTO suffix (%s) VALUES (%s)", columnList, placeholders)
	default:
		return "", fmt.Errorf("unsupported table: %s", tableName)
	}
	
	return query, nil
}

// BuildUpdate 构建更新记录的SQL语句
func (qb *SQLQueryBuilder) BuildUpdate(tableName string, setColumns []string, whereColumn string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}
	
	setClause := "`value` = ?"
	if len(setColumns) > 0 {
		setClause = ""
		for i, col := range setColumns {
			if i > 0 {
				setClause += ", "
			}
			setClause += fmt.Sprintf("%s = ?", col)
		}
	}
	
	if whereColumn == "" {
		whereColumn = "`key`"
	}
	
	var query string
	switch tableName {
	case "exact":
		query = fmt.Sprintf("UPDATE exact SET %s WHERE %s = ?", setClause, whereColumn)
	case "contains":
		query = fmt.Sprintf("UPDATE contains SET %s WHERE %s = ?", setClause, whereColumn)
	case "regex":
		query = fmt.Sprintf("UPDATE regex SET %s WHERE %s = ?", setClause, whereColumn)
	case "prefix":
		query = fmt.Sprintf("UPDATE prefix SET %s WHERE %s = ?", setClause, whereColumn)
	case "suffix":
		query = fmt.Sprintf("UPDATE suffix SET %s WHERE %s = ?", setClause, whereColumn)
	default:
		return "", fmt.Errorf("unsupported table: %s", tableName)
	}
	
	return query, nil
}

// BuildDelete 构建删除记录的SQL语句
func (qb *SQLQueryBuilder) BuildDelete(tableName string, whereColumn string) (string, error) {
	if err := qb.validator.ValidateTable(tableName); err != nil {
		return "", err
	}
	
	if whereColumn == "" {
		whereColumn = "`key`"
	}
	
	var query string
	switch tableName {
	case "exact":
		query = fmt.Sprintf("DELETE FROM exact WHERE %s = ?", whereColumn)
	case "contains":
		query = fmt.Sprintf("DELETE FROM contains WHERE %s = ?", whereColumn)
	case "regex":
		query = fmt.Sprintf("DELETE FROM regex WHERE %s = ?", whereColumn)
	case "prefix":
		query = fmt.Sprintf("DELETE FROM prefix WHERE %s = ?", whereColumn)
	case "suffix":
		query = fmt.Sprintf("DELETE FROM suffix WHERE %s = ?", whereColumn)
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

	columnList := "id, `key`, `value`"
	if len(columns) == 6 {
		columnList = "id, key, value, content_type, telegraph_url, telegraph_path"
	}

	switch tableName {
	case "exact":
		sqlQuery = fmt.Sprintf("SELECT %s FROM exact WHERE `key` = ?", columnList)
		rows, err = ops.db.Query(sqlQuery, query)
	case "contains":
		if len(columns) == 6 {
			// SQLite syntax
			sqlQuery = fmt.Sprintf("SELECT %s FROM contains WHERE `key` LIKE '%%' || ? || '%%'", columnList)
		} else {
			// MySQL syntax
			sqlQuery = fmt.Sprintf("SELECT %s FROM contains WHERE `key` LIKE CONCAT('%%', ?, '%%')", columnList)
		}
		rows, err = ops.db.Query(sqlQuery, query)
	case "regex":
		// Regex matching needs to be done in application code
		sqlQuery = fmt.Sprintf("SELECT %s FROM regex", columnList)
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
