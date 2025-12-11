package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type ConcurrentRequestConfig struct {
	URL         string                 `json:"url"`
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Threads     int                    `json:"threads"`
	Iterations  int                    `json:"iterations"`
	RandomParam map[string]string      `json:"random_param,omitempty"` // 例如 {"id":"1-1000"}
}

type RequestResult struct {
	Thread      int    `json:"thread"`
	Iteration   int    `json:"iteration"`
	Status      int    `json:"status"`
	BodyPreview string `json:"body_preview"`
	Error       string `json:"error,omitempty"`
}

// listDatabases 列出所有数据库
func listDatabases(request map[string]interface{}) (*mcp.CallToolResult, error) {
	pattern, _ := request["pattern"].(string)

	query := "SHOW DATABASES"
	if pattern != "" {
		query = fmt.Sprintf("SHOW DATABASES LIKE '%s'", pattern)
	}

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		databases = append(databases, dbName)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"databases": databases,
		"count":     len(databases),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// listTables 列出数据库中的所有表
func listTables(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	query := fmt.Sprintf("SHOW TABLES FROM `%s`", database)
	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tables = append(tables, tableName)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"database": database,
		"tables":   tables,
		"count":    len(tables),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// describeTable 查看表结构
func describeTable(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	table, ok := request["table"].(string)
	if !ok || table == "" {
		return mcp.NewToolResultError("table 参数是必需的"), nil
	}

	query := fmt.Sprintf("DESCRIBE `%s`.`%s`", database, table)
	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	type Column struct {
		Field   string         `json:"field"`
		Type    string         `json:"type"`
		Null    string         `json:"null"`
		Key     string         `json:"key"`
		Default sql.NullString `json:"default"`
		Extra   string         `json:"extra"`
	}

	var columns []Column
	for rows.Next() {
		var col Column
		if err := rows.Scan(&col.Field, &col.Type, &col.Null, &col.Key, &col.Default, &col.Extra); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		columns = append(columns, col)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"database": database,
		"table":    table,
		"columns":  columns,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// executeQuery 执行查询
func executeQuery(request map[string]interface{}) (*mcp.CallToolResult, error) {
	query, ok := request["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query 参数是必需的"), nil
	}

	// 安全检查：只允许 SELECT 语句
	trimmedQuery := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(trimmedQuery, "SELECT") && !strings.HasPrefix(trimmedQuery, "SHOW") && !strings.HasPrefix(trimmedQuery, "DESCRIBE") {
		return mcp.NewToolResultError("只允许执行 SELECT、SHOW 和 DESCRIBE 查询"), nil
	}

	limit := 100
	if l, ok := request["limit"].(float64); ok {
		limit = int(l)
	}

	// 添加 LIMIT 子句
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 读取数据
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"query":   query,
		"columns": columns,
		"rows":    results,
		"count":   len(results),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// showIndexes 查看表索引
func showIndexes(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	table, ok := request["table"].(string)
	if !ok || table == "" {
		return mcp.NewToolResultError("table 参数是必需的"), nil
	}

	query := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", database, table)
	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 读取数据
	var indexes []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		indexes = append(indexes, row)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"database": database,
		"table":    table,
		"indexes":  indexes,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// getTableStats 获取表统计信息
func getTableStats(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	table, _ := request["table"].(string)

	var query string
	if table != "" {
		query = fmt.Sprintf(`
			SELECT 
				TABLE_NAME as table_name,
				TABLE_ROWS as row_count,
				ROUND(DATA_LENGTH / 1024 / 1024, 2) as data_size_mb,
				ROUND(INDEX_LENGTH / 1024 / 1024, 2) as index_size_mb,
				ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) as total_size_mb,
				ENGINE as engine,
				TABLE_COLLATION as collation,
				CREATE_TIME as created_at,
				UPDATE_TIME as updated_at
			FROM information_schema.TABLES
			WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'
		`, database, table)
	} else {
		query = fmt.Sprintf(`
			SELECT 
				TABLE_NAME as table_name,
				TABLE_ROWS as row_count,
				ROUND(DATA_LENGTH / 1024 / 1024, 2) as data_size_mb,
				ROUND(INDEX_LENGTH / 1024 / 1024, 2) as index_size_mb,
				ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) as total_size_mb,
				ENGINE as engine,
				TABLE_COLLATION as collation,
				CREATE_TIME as created_at,
				UPDATE_TIME as updated_at
			FROM information_schema.TABLES
			WHERE TABLE_SCHEMA = '%s'
			ORDER BY (DATA_LENGTH + INDEX_LENGTH) DESC
		`, database)
	}

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var tableName, engine, collation string
		var rowCount sql.NullInt64
		var dataSizeMB, indexSizeMB, totalSizeMB sql.NullFloat64
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(&tableName, &rowCount, &dataSizeMB, &indexSizeMB, &totalSizeMB,
			&engine, &collation, &createdAt, &updatedAt); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		stat := map[string]interface{}{
			"table_name":    tableName,
			"row_count":     rowCount.Int64,
			"data_size_mb":  dataSizeMB.Float64,
			"index_size_mb": indexSizeMB.Float64,
			"total_size_mb": totalSizeMB.Float64,
			"engine":        engine,
			"collation":     collation,
		}
		if createdAt.Valid {
			stat["created_at"] = createdAt.Time.Format("2006-01-02 15:04:05")
		}
		if updatedAt.Valid {
			stat["updated_at"] = updatedAt.Time.Format("2006-01-02 15:04:05")
		}
		stats = append(stats, stat)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"database": database,
		"stats":    stats,
		"count":    len(stats),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// showForeignKeys 查看外键关系
func showForeignKeys(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	table, ok := request["table"].(string)
	if !ok || table == "" {
		return mcp.NewToolResultError("table 参数是必需的"), nil
	}

	query := fmt.Sprintf(`
		SELECT 
			CONSTRAINT_NAME as constraint_name,
			COLUMN_NAME as column_name,
			REFERENCED_TABLE_NAME as referenced_table,
			REFERENCED_COLUMN_NAME as referenced_column
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = '%s' 
			AND TABLE_NAME = '%s'
			AND REFERENCED_TABLE_NAME IS NOT NULL
	`, database, table)

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var foreignKeys []map[string]interface{}
	for rows.Next() {
		var constraintName, columnName, referencedTable, referencedColumn string
		if err := rows.Scan(&constraintName, &columnName, &referencedTable, &referencedColumn); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		foreignKeys = append(foreignKeys, map[string]interface{}{
			"constraint_name":   constraintName,
			"column_name":       columnName,
			"referenced_table":  referencedTable,
			"referenced_column": referencedColumn,
		})
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"database":     database,
		"table":        table,
		"foreign_keys": foreignKeys,
		"count":        len(foreignKeys),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// searchSchema 搜索表或字段
func searchSchema(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	keyword, ok := request["keyword"].(string)
	if !ok || keyword == "" {
		return mcp.NewToolResultError("keyword 参数是必需的"), nil
	}

	searchType, _ := request["search_type"].(string)
	if searchType == "" {
		searchType = "both"
	}

	results := make(map[string]interface{})

	// 搜索表名
	if searchType == "table" || searchType == "both" {
		query := fmt.Sprintf(`
			SELECT TABLE_NAME
			FROM information_schema.TABLES
			WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME LIKE '%%%s%%'
		`, database, keyword)

		rows, err := db.Query(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("搜索表名失败: %v", err)), nil
		}
		defer rows.Close()

		var tables []string
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			tables = append(tables, tableName)
		}
		results["tables"] = tables
	}

	// 搜索字段名
	if searchType == "column" || searchType == "both" {
		query := fmt.Sprintf(`
			SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = '%s' AND COLUMN_NAME LIKE '%%%s%%'
		`, database, keyword)

		rows, err := db.Query(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("搜索字段名失败: %v", err)), nil
		}
		defer rows.Close()

		var columns []map[string]interface{}
		for rows.Next() {
			var tableName, columnName, dataType string
			if err := rows.Scan(&tableName, &columnName, &dataType); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			columns = append(columns, map[string]interface{}{
				"table":  tableName,
				"column": columnName,
				"type":   dataType,
			})
		}
		results["columns"] = columns
	}

	results["database"] = database
	results["keyword"] = keyword
	results["search_type"] = searchType

	result, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// showCreateTable 生成建表语句
func showCreateTable(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	table, ok := request["table"].(string)
	if !ok || table == "" {
		return mcp.NewToolResultError("table 参数是必需的"), nil
	}

	query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, table)
	row := db.QueryRow(query)

	var tableName, createSQL string
	if err := row.Scan(&tableName, &createSQL); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"database":   database,
		"table":      table,
		"create_sql": createSQL,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// analyzeColumn 分析字段数据分布
func analyzeColumn(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	table, ok := request["table"].(string)
	if !ok || table == "" {
		return mcp.NewToolResultError("table 参数是必需的"), nil
	}

	column, ok := request["column"].(string)
	if !ok || column == "" {
		return mcp.NewToolResultError("column 参数是必需的"), nil
	}

	// 获取统计信息
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_count,
			COUNT(DISTINCT %s) as unique_count,
			COUNT(%s) as non_null_count,
			COUNT(*) - COUNT(%s) as null_count
		FROM %s.%s
	`, column, column, column, database, table)

	row := db.QueryRow(query)
	var totalCount, uniqueCount, nonNullCount, nullCount int64
	if err := row.Scan(&totalCount, &uniqueCount, &nonNullCount, &nullCount); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}

	analysis := map[string]interface{}{
		"database":       database,
		"table":          table,
		"column":         column,
		"total_count":    totalCount,
		"unique_count":   uniqueCount,
		"non_null_count": nonNullCount,
		"null_count":     nullCount,
	}

	// 尝试获取最大值和最小值（仅对数值和日期类型）
	minMaxQuery := fmt.Sprintf("SELECT MIN(%s), MAX(%s) FROM %s.%s", column, column, database, table)
	minMaxRow := db.QueryRow(minMaxQuery)
	var minVal, maxVal sql.NullString
	if err := minMaxRow.Scan(&minVal, &maxVal); err == nil {
		if minVal.Valid {
			analysis["min_value"] = minVal.String
		}
		if maxVal.Valid {
			analysis["max_value"] = maxVal.String
		}
	}

	// 获取最常见的值（Top 10）
	topValuesQuery := fmt.Sprintf(`
		SELECT %s, COUNT(*) as count
		FROM %s.%s
		WHERE %s IS NOT NULL
		GROUP BY %s
		ORDER BY count DESC
		LIMIT 10
	`, column, database, table, column, column)

	rows, err := db.Query(topValuesQuery)
	if err == nil {
		defer rows.Close()
		var topValues []map[string]interface{}
		for rows.Next() {
			var value string
			var count int64
			if err := rows.Scan(&value, &count); err == nil {
				topValues = append(topValues, map[string]interface{}{
					"value": value,
					"count": count,
				})
			}
		}
		analysis["top_values"] = topValues
	}

	result, _ := json.MarshalIndent(analysis, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// showTriggers 查看触发器
func showTriggers(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	table, _ := request["table"].(string)

	var query string
	if table != "" {
		query = fmt.Sprintf(`
			SELECT 
				TRIGGER_NAME,
				EVENT_MANIPULATION,
				ACTION_TIMING,
				ACTION_STATEMENT
			FROM information_schema.TRIGGERS
			WHERE TRIGGER_SCHEMA = '%s' AND EVENT_OBJECT_TABLE = '%s'
		`, database, table)
	} else {
		query = fmt.Sprintf(`
			SELECT 
				TRIGGER_NAME,
				EVENT_OBJECT_TABLE,
				EVENT_MANIPULATION,
				ACTION_TIMING,
				ACTION_STATEMENT
			FROM information_schema.TRIGGERS
			WHERE TRIGGER_SCHEMA = '%s'
		`, database)
	}

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var triggers []map[string]interface{}
	for rows.Next() {
		if table != "" {
			var triggerName, eventManipulation, actionTiming, actionStatement string
			if err := rows.Scan(&triggerName, &eventManipulation, &actionTiming, &actionStatement); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			triggers = append(triggers, map[string]interface{}{
				"trigger_name":       triggerName,
				"event_manipulation": eventManipulation,
				"action_timing":      actionTiming,
				"action_statement":   actionStatement,
			})
		} else {
			var triggerName, tableName, eventManipulation, actionTiming, actionStatement string
			if err := rows.Scan(&triggerName, &tableName, &eventManipulation, &actionTiming, &actionStatement); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			triggers = append(triggers, map[string]interface{}{
				"trigger_name":       triggerName,
				"table":              tableName,
				"event_manipulation": eventManipulation,
				"action_timing":      actionTiming,
				"action_statement":   actionStatement,
			})
		}
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"database": database,
		"table":    table,
		"triggers": triggers,
		"count":    len(triggers),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// showVariables 查看系统变量
func showVariables(request map[string]interface{}) (*mcp.CallToolResult, error) {
	pattern, _ := request["pattern"].(string)

	var query string
	if pattern != "" {
		query = fmt.Sprintf("SHOW VARIABLES LIKE '%s'", pattern)
	} else {
		query = "SHOW VARIABLES"
	}

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var variables []map[string]interface{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		variables = append(variables, map[string]interface{}{
			"name":  name,
			"value": value,
		})
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"variables": variables,
		"count":     len(variables),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// showStatus 查看数据库状态
func showStatus(request map[string]interface{}) (*mcp.CallToolResult, error) {
	pattern, _ := request["pattern"].(string)

	var query string
	if pattern != "" {
		query = fmt.Sprintf("SHOW STATUS LIKE '%s'", pattern)
	} else {
		query = "SHOW STATUS"
	}

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var status []map[string]interface{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		status = append(status, map[string]interface{}{
			"name":  name,
			"value": value,
		})
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"status": status,
		"count":  len(status),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// showProcesslist 查看正在执行的查询
func showProcesslist(request map[string]interface{}) (*mcp.CallToolResult, error) {
	query := "SHOW FULL PROCESSLIST"

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var processes []map[string]interface{}
	for rows.Next() {
		var id int64
		var user, host, dbName sql.NullString
		var command, state sql.NullString
		var time int64
		var info sql.NullString

		if err := rows.Scan(&id, &user, &host, &dbName, &command, &time, &state, &info); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		process := map[string]interface{}{
			"id":   id,
			"time": time,
		}
		if user.Valid {
			process["user"] = user.String
		}
		if host.Valid {
			process["host"] = host.String
		}
		if dbName.Valid {
			process["database"] = dbName.String
		}
		if command.Valid {
			process["command"] = command.String
		}
		if state.Valid {
			process["state"] = state.String
		}
		if info.Valid {
			process["info"] = info.String
		}

		processes = append(processes, process)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"processes": processes,
		"count":     len(processes),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// showTableCharset 查看表的字符集和排序规则
func showTableCharset(request map[string]interface{}) (*mcp.CallToolResult, error) {
	database, ok := request["database"].(string)
	if !ok || database == "" {
		return mcp.NewToolResultError("database 参数是必需的"), nil
	}

	table, ok := request["table"].(string)
	if !ok || table == "" {
		return mcp.NewToolResultError("table 参数是必需的"), nil
	}

	query := fmt.Sprintf(`
		SELECT 
			COLUMN_NAME,
			CHARACTER_SET_NAME,
			COLLATION_NAME
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'
			AND CHARACTER_SET_NAME IS NOT NULL
	`, database, table)

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var columnName, charsetName, collationName string
		if err := rows.Scan(&columnName, &charsetName, &collationName); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		columns = append(columns, map[string]interface{}{
			"column":    columnName,
			"charset":   charsetName,
			"collation": collationName,
		})
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"database": database,
		"table":    table,
		"columns":  columns,
		"count":    len(columns),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}
func forceBrowserReadHandler(req map[string]interface{}) (*mcp.CallToolResult, error) {
	url := req["url"].(string)
	if url == "" {
		return mcp.NewToolResultError("url 参数是必需的"), nil
	}
	result, _ := json.MarshalIndent(map[string]interface{}{
		"message": "请使用 chrome-devtools-mcp 来打开并读取此网页",
		"url":     url,
		"next_steps": []string{
			"browser.navigate(url)",
			"browser.wait_for_dom()",
			"browser.get_html()",
		},
	}, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// documentGeneratorHandler 按照指定格式生成模块文档 Markdown
func documentGeneratorHandler(input map[string]interface{}) (*mcp.CallToolResult, error) {
	title, _ := input["title"].(string) // 模块名称
	moduleName, _ := input["moduleName"].(string)
	application, _ := input["application"].(string)
	lastUpdate, _ := input["lastUpdate"].(string)
	description, _ := input["description"].(string)
	tables, _ := input["tables"].([]map[string]interface{}) // 数据库表列表

	var md strings.Builder

	// 顶部模块信息
	md.WriteString(fmt.Sprintf("# %s\n\n", title))
	md.WriteString(fmt.Sprintf("**模块名称：** %s\n", moduleName))
	md.WriteString(fmt.Sprintf("**所属应用：** %s\n", application))
	md.WriteString(fmt.Sprintf("**最后更新：** %s\n\n", lastUpdate))
	md.WriteString("---\n\n")

	// 1. 模块概述
	md.WriteString("## 1. 模块概述\n\n")

	md.WriteString("### 1.1 功能说明\n")
	if description != "" {
		for _, line := range strings.Split(description, "\n") {
			md.WriteString(fmt.Sprintf("- %s\n", line))
		}
	} else {
		md.WriteString("- 功能描述待补充\n")
	}
	md.WriteString("\n")

	md.WriteString("### 1.2 数据来源与发放方式\n\n")
	md.WriteString("**数据来源：**\n- 待补充\n\n")
	md.WriteString("**发放方式：**\n- 待补充\n\n")

	md.WriteString("### 1.3 功能权限\n")
	md.WriteString("- 待补充\n\n")
	md.WriteString("---\n\n")

	// 2. 数据库表结构
	md.WriteString("## 2. 数据库表结构\n\n")

	for _, table := range tables {
		tableName, _ := table["name"].(string)
		tableComment, _ := table["comment"].(string)
		createSQL, _ := table["createSQL"].(string)
		association, _ := table["association"].(string)

		md.WriteString(fmt.Sprintf("### 2.%s `%s` — %s\n\n", table["index"], tableName, tableComment))
		md.WriteString("#### 表结构说明\n")
		md.WriteString(fmt.Sprintf("- %s\n\n", tableComment))
		md.WriteString("#### 建表语句\n\n")
		md.WriteString("```sql\n")
		md.WriteString(createSQL + "\n")
		md.WriteString("```\n\n")
		if association != "" {
			md.WriteString("**关联说明：**\n")
			md.WriteString(association + "\n\n")
		}
		md.WriteString("---\n\n")
	}

	// 3. API 接口设计
	md.WriteString("## 3. API 接口设计\n\n")
	md.WriteString("### 3.1 接口列表\n")
	md.WriteString("- 待补充\n\n")
	md.WriteString("---\n\n")

	// 4. 核心流程
	md.WriteString("### 4.1 核心流程\n")
	md.WriteString("- 待补充\n\n")
	md.WriteString("---\n")

	return mcp.NewToolResultText(md.String()), nil
}

func concurrentRequestHandler(input map[string]interface{}) (*mcp.CallToolResult, error) {
	// 解析参数
	data, _ := json.Marshal(input)
	var cfg ConcurrentRequestConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("参数解析错误: %v", err)), nil
	}

	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.Threads < 1 {
		cfg.Threads = 1
	}
	if cfg.Iterations < 1 {
		cfg.Iterations = 1
	}

	results := make([]RequestResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for t := 1; t <= cfg.Threads; t++ {
		wg.Add(1)
		go func(threadNum int) {
			defer wg.Done()
			client := &http.Client{Timeout: 10 * time.Second}

			for i := 1; i <= cfg.Iterations; i++ {
				// 构建参数
				params := make(map[string]interface{})
				for k, v := range cfg.Params {
					params[k] = v
				}
				// 处理随机参数
				for k, rule := range cfg.RandomParam {
					// 简单规则： "1-100" → 随机生成数字
					var min, max int
					_, err := fmt.Sscanf(rule, "%d-%d", &min, &max)
					if err == nil {
						params[k] = rand.Intn(max-min+1) + min
					} else {
						params[k] = rule
					}
				}

				var reqBody []byte
				if cfg.Method == "POST" || cfg.Method == "PUT" {
					reqBody, _ = json.Marshal(params)
				}
				req, err := http.NewRequest(cfg.Method, cfg.URL, bytes.NewReader(reqBody))
				if err != nil {
					mu.Lock()
					results = append(results, RequestResult{
						Thread:    threadNum,
						Iteration: i,
						Error:     err.Error(),
					})
					mu.Unlock()
					continue
				}

				// 添加 Headers
				for hk, hv := range cfg.Headers {
					req.Header.Set(hk, hv)
				}
				if cfg.Method == "POST" || cfg.Method == "PUT" {
					req.Header.Set("Content-Type", "application/json")
				}

				resp, err := client.Do(req)
				status := 0
				bodyPreview := ""
				if err == nil {
					status = resp.StatusCode
					body, _ := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					if len(body) > 100 {
						bodyPreview = string(body[:100])
					} else {
						bodyPreview = string(body)
					}
				} else {
					bodyPreview = ""
				}

				mu.Lock()
				results = append(results, RequestResult{
					Thread:      threadNum,
					Iteration:   i,
					Status:      status,
					BodyPreview: bodyPreview,
					Error: func() string {
						if err != nil {
							return err.Error()
						} else {
							return ""
						}
					}(),
				})
				mu.Unlock()
			}
		}(t)
	}

	wg.Wait()

	// 汇总统计
	successCount := 0
	failureCount := 0
	for _, r := range results {
		if r.Error == "" && r.Status >= 200 && r.Status < 300 {
			successCount++
		} else {
			failureCount++
		}
	}

	resultsJSON, _ := json.Marshal(map[string]interface{}{
		"success_count": successCount,
		"failure_count": failureCount,
		"responses":     results,
	})

	return mcp.NewToolResultText(string(resultsJSON)), nil

}
