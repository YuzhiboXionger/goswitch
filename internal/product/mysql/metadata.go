package mysql

import (
	"database/sql"
	"fmt"
	"goswitch/internal/core"
	"strings"
)

// Metadata MySQL 元数据查询
type Metadata struct {
	db *sql.DB
}

// ListTables 查询表列表
func (m *Metadata) ListTables(database string) ([]string, error) {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE'"
	rows, err := m.db.Query(query, database)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// GetColumns 查询列信息
func (m *Metadata) GetColumns(database, table string) ([]core.ColumnMeta, error) {
	query := `
		SELECT
			column_name,
			data_type,
			COALESCE(character_maximum_length, 0),
			COALESCE(numeric_precision, 0),
			COALESCE(numeric_scale, 0),
			is_nullable,
			column_default,
			COALESCE(column_comment, ''),
			COALESCE(extra, '')
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position`

	rows, err := m.db.Query(query, database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	defer rows.Close()

	var columns []core.ColumnMeta
	for rows.Next() {
		var col core.ColumnMeta
		var extra string
		var nullable string
		var defaultVal sql.NullString

		err := rows.Scan(
			&col.Name, &col.DataType, &col.Length, &col.Precision, &col.Scale,
			&nullable, &defaultVal, &col.Comment, &extra,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Nullable = nullable == "YES"
		col.AutoIncr = strings.Contains(extra, "auto_increment")
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}

		columns = append(columns, col)
	}
	return columns, rows.Err()
}

// GetPrimaryKeys 查询主键
func (m *Metadata) GetPrimaryKeys(database, table string) ([]string, error) {
	query := `
		SELECT column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ? AND table_name = ? AND constraint_name = 'PRIMARY'
		ORDER BY ordinal_position`

	rows, err := m.db.Query(query, database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan primary key: %w", err)
		}
		keys = append(keys, name)
	}
	return keys, rows.Err()
}

// GenerateCreateTableDDL 生成建表 DDL
func (m *Metadata) GenerateCreateTableDDL(database, table string, columns []core.ColumnMeta, primaryKeys []string) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", table))

	for i, col := range columns {
		sb.WriteString(fmt.Sprintf("  `%s` %s", col.Name, m.getFieldDefinition(col)))

		if !col.Nullable {
			sb.WriteString(" NOT NULL")
		}

		if col.DefaultValue != nil {
			sb.WriteString(fmt.Sprintf(" DEFAULT %s", *col.DefaultValue))
		}

		if col.AutoIncr {
			sb.WriteString(" AUTO_INCREMENT")
		}

		if col.Comment != "" {
			sb.WriteString(fmt.Sprintf(" COMMENT '%s'", col.Comment))
		}

		if i < len(columns)-1 || len(primaryKeys) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	if len(primaryKeys) > 0 {
		sb.WriteString(fmt.Sprintf("  PRIMARY KEY (`%s`)\n", strings.Join(primaryKeys, "`, `")))
	}

	sb.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")

	return sb.String(), nil
}

// getFieldDefinition 获取字段类型定义
// 同时支持 MySQL 原生类型、PostgreSQL 和 Oracle 类型（用于跨数据库迁移）
func (m *Metadata) getFieldDefinition(col core.ColumnMeta) string {
	switch strings.ToLower(col.DataType) {
	// 字符串类型
	case "varchar", "varchar2", "character varying":
		if col.Length > 0 {
			return fmt.Sprintf("varchar(%d)", col.Length)
		}
		return "varchar(255)"
	case "char", "character", "nchar":
		if col.Length > 0 {
			return fmt.Sprintf("char(%d)", col.Length)
		}
		return "char(1)"
	case "nvarchar2":
		if col.Length > 0 {
			return fmt.Sprintf("varchar(%d)", col.Length)
		}
		return "varchar(255)"
	case "text", "tinytext", "mediumtext", "longtext", "clob", "nclob":
		return "text"
	case "enum":
		return "varchar(255)"
	case "set":
		return "text"

	// 数值类型
	case "decimal", "numeric":
		if col.Precision > 0 {
			return fmt.Sprintf("decimal(%d,%d)", col.Precision, col.Scale)
		}
		return "decimal(10,2)"
	case "number":
		// Oracle NUMBER 类型映射
		if col.Scale > 0 {
			return fmt.Sprintf("decimal(%d,%d)", col.Precision, col.Scale)
		}
		if col.Precision <= 10 {
			return "int"
		}
		if col.Precision <= 19 {
			return "bigint"
		}
		return "bigint"
	case "integer", "int", "int4":
		return "int"
	case "bigint", "int8":
		return "bigint"
	case "smallint", "int2":
		return "smallint"
	case "tinyint":
		if col.Length == 1 {
			return "tinyint(1)"
		}
		return "tinyint"
	case "mediumint":
		return "mediumint"
	case "float", "real", "float4", "binary_float":
		return "float"
	case "double", "double precision", "float8", "binary_double":
		return "double"
	case "boolean", "bool":
		return "tinyint(1)"

	// 日期时间类型
	case "datetime":
		return "datetime"
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
		return "timestamp"
	case "date":
		// Oracle DATE 包含时间，映射为 datetime
		return "datetime"
	case "time", "time without time zone":
		return "time"
	case "year":
		return "year"

	// 二进制类型
	case "blob", "tinyblob", "mediumblob", "longblob", "binary", "varbinary", "bytea", "raw":
		return "longblob"

	// JSON 类型
	case "json", "jsonb":
		return "json"

	// MongoDB 类型
	case "objectid":
		return "varchar(24)"
	case "document", "array":
		return "json"
	case "long":
		return "bigint"
	case "decimal128":
		return "decimal(38,9)"
	case "null":
		return "text"

	default:
		// 未知类型尝试直接使用
		return col.DataType
	}
}
