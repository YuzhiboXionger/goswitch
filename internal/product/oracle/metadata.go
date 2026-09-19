package oracle

import (
	"database/sql"
	"fmt"
	"goswitch/internal/core"
	"strings"
)

// Metadata Oracle 元数据查询
type Metadata struct {
	db *sql.DB
}

// isOracleSpecificDefault 判断是否为 Oracle 特有的默认值（跨数据库时需过滤）
func isOracleSpecificDefault(val string) bool {
	upper := strings.ToUpper(strings.TrimSpace(val))
	oracleDefaults := []string{
		"SYSTIMESTAMP", "SYSDATE", "CURRENT_TIMESTAMP", "CURRENT_DATE",
		"LOCALTIMESTAMP", "SYS_EXTRACT_UTC(SYSTIMESTAMP)",
	}
	for _, d := range oracleDefaults {
		if strings.Contains(upper, d) {
			return true
		}
	}
	return false
}

// ListTables 查询表列表
func (m *Metadata) ListTables(database string) ([]string, error) {
	query := "SELECT table_name FROM user_tables WHERE table_name NOT LIKE 'BIN$%' ORDER BY table_name"
	rows, err := m.db.Query(query)
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
	// 注意：Oracle 的 data_default 是 LONG 类型，不能使用 COALESCE
	query := `
		SELECT
			c.column_name,
			c.data_type,
			NVL(c.data_length, 0),
			NVL(c.data_precision, 0),
			NVL(c.data_scale, 0),
			c.nullable,
			c.data_default,
			NVL(cc.comments, ''),
			CASE WHEN c.identity_column = 'YES' THEN 1 ELSE 0 END as is_identity
		FROM user_tab_columns c
		LEFT JOIN user_col_comments cc ON cc.table_name = c.table_name AND cc.column_name = c.column_name
		WHERE c.table_name = :1
		ORDER BY c.column_id`

	rows, err := m.db.Query(query, strings.ToUpper(table))
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	defer rows.Close()

	var columns []core.ColumnMeta
	for rows.Next() {
		var col core.ColumnMeta
		var nullable string
		var defaultVal sql.NullString
		var comment sql.NullString
		var isIdentity int

		err := rows.Scan(
			&col.Name, &col.DataType, &col.Length, &col.Precision, &col.Scale,
			&nullable, &defaultVal, &comment, &isIdentity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Nullable = nullable == "Y"
		col.AutoIncr = isIdentity == 1
		if defaultVal.Valid && defaultVal.String != "" {
			// 过滤掉 Oracle 特有的默认值函数
			trimmed := strings.TrimSpace(defaultVal.String)
			if !isOracleSpecificDefault(trimmed) {
				col.DefaultValue = &defaultVal.String
			}
		}
		if comment.Valid {
			col.Comment = comment.String
		}

		columns = append(columns, col)
	}
	return columns, rows.Err()
}

// GetPrimaryKeys 查询主键
func (m *Metadata) GetPrimaryKeys(database, table string) ([]string, error) {
	query := `
		SELECT cc.column_name
		FROM user_constraints c
		JOIN user_cons_columns cc ON c.constraint_name = cc.constraint_name
		WHERE c.constraint_type = 'P'
			AND c.table_name = :1
		ORDER BY cc.position`

	rows, err := m.db.Query(query, strings.ToUpper(table))
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

	sb.WriteString(fmt.Sprintf("CREATE TABLE \"%s\" (\n", strings.ToUpper(table)))

	for i, col := range columns {
		colName := strings.ToUpper(col.Name)
		sb.WriteString(fmt.Sprintf("  \"%s\" %s", colName, m.getFieldDefinition(col)))

		if !col.Nullable {
			sb.WriteString(" NOT NULL")
		}

		if col.DefaultValue != nil {
			sb.WriteString(fmt.Sprintf(" DEFAULT %s", *col.DefaultValue))
		}

		if i < len(columns)-1 || len(primaryKeys) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	if len(primaryKeys) > 0 {
		quotedKeys := make([]string, len(primaryKeys))
		for i, k := range primaryKeys {
			quotedKeys[i] = fmt.Sprintf("\"%s\"", strings.ToUpper(k))
		}
		sb.WriteString(fmt.Sprintf("  PRIMARY KEY (%s)\n", strings.Join(quotedKeys, ", ")))
	}

	sb.WriteString(")")

	return sb.String(), nil
}

// getFieldDefinition 获取字段类型定义
// 支持 Oracle 原生类型和 MySQL/PostgreSQL 类型（用于跨数据库迁移）
func (m *Metadata) getFieldDefinition(col core.ColumnMeta) string {
	switch strings.ToLower(col.DataType) {
	// 字符串类型
	case "varchar", "varchar2", "character varying":
		if col.Length > 0 {
			return fmt.Sprintf("VARCHAR2(%d)", col.Length)
		}
		return "VARCHAR2(255)"
	case "char", "character":
		if col.Length > 0 {
			return fmt.Sprintf("CHAR(%d)", col.Length)
		}
		return "CHAR(1)"
	case "nchar":
		if col.Length > 0 {
			return fmt.Sprintf("NCHAR(%d)", col.Length)
		}
		return "NCHAR(1)"
	case "nvarchar2":
		if col.Length > 0 {
			return fmt.Sprintf("NVARCHAR2(%d)", col.Length)
		}
		return "NVARCHAR2(255)"
	case "text", "tinytext", "mediumtext", "longtext", "clob", "nclob":
		return "CLOB"
	case "enum":
		return "VARCHAR2(255)"
	case "set":
		return "CLOB"

	// 数值类型
	case "number", "numeric", "decimal":
		if col.Precision > 0 {
			return fmt.Sprintf("NUMBER(%d,%d)", col.Precision, col.Scale)
		}
		return "NUMBER"
	case "integer", "int", "int4", "mediumint":
		return "NUMBER(10)"
	case "bigint", "int8":
		return "NUMBER(19)"
	case "smallint", "int2":
		return "NUMBER(5)"
	case "tinyint":
		if col.Length == 1 {
			return "NUMBER(1)"
		}
		return "NUMBER(3)"
	case "float", "real", "float4":
		return "BINARY_FLOAT"
	case "double", "double precision", "float8", "binary_double":
		return "BINARY_DOUBLE"
	case "boolean", "bool":
		return "NUMBER(1)"

	// 日期时间类型
	case "date":
		return "DATE"
	case "datetime", "timestamp", "timestamp without time zone":
		return "TIMESTAMP"
	case "timestamp with time zone", "timestamptz":
		return "TIMESTAMP WITH TIME ZONE"
	case "time":
		return "TIMESTAMP"
	case "year":
		return "NUMBER(4)"

	// 二进制类型
	case "blob", "tinyblob", "mediumblob", "longblob", "binary", "varbinary", "bytea", "raw":
		return "BLOB"

	// MongoDB 类型
	case "objectid":
		return "VARCHAR2(24)"
	case "document", "array":
		return "CLOB"
	case "long":
		return "NUMBER(19)"
	case "decimal128":
		return "NUMBER(38,9)"
	case "null":
		return "VARCHAR2(4000)"

	default:
		// 未知类型尝试直接使用
		return strings.ToUpper(col.DataType)
	}
}
