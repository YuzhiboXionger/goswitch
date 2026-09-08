package oracle

import (
	"database/sql"
	"fmt"
	"strings"
)

// Manager Oracle 表管理器
type Manager struct {
	db *sql.DB
}

// TableExists 判断表是否存在
func (m *Manager) TableExists(database, table string) (bool, error) {
	query := "SELECT COUNT(*) FROM user_tables WHERE table_name = :1"
	var countStr string
	err := m.db.QueryRow(query, strings.ToUpper(table)).Scan(&countStr)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	// 转换字符串为整数
	var count int
	fmt.Sscanf(countStr, "%d", &count)
	return count > 0, nil
}

// CreateTable 创建表
func (m *Manager) CreateTable(database, table, ddl string) error {
	_, err := m.db.Exec(ddl)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

// DropTable 删除表
func (m *Manager) DropTable(database, table string) error {
	query := fmt.Sprintf("DROP TABLE \"%s\" CASCADE CONSTRAINTS", strings.ToUpper(table))
	_, err := m.db.Exec(query)
	if err != nil {
		// 表不存在时不报错
		if strings.Contains(err.Error(), "ORA-00942") {
			return nil
		}
		return fmt.Errorf("failed to drop table: %w", err)
	}
	return nil
}

// TruncateTable 清空表数据
func (m *Manager) TruncateTable(database, table string) error {
	query := fmt.Sprintf("TRUNCATE TABLE \"%s\"", strings.ToUpper(table))
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to truncate table: %w", err)
	}
	return nil
}

// DisableForeignKeyCheck 禁用外键检查
func (m *Manager) DisableForeignKeyCheck() error {
	// Oracle 没有全局禁用外键检查的命令
	// 需要在每个表上禁用约束
	return nil
}

// EnableForeignKeyCheck 启用外键检查
func (m *Manager) EnableForeignKeyCheck() error {
	// Oracle 没有全局启用外键检查的命令
	return nil
}

// DisableKeys 禁用索引（Oracle 不支持 DISABLE KEYS）
func (m *Manager) DisableKeys(database, table string) error {
	// Oracle 不支持 DISABLE KEYS，返回 nil
	return nil
}

// EnableKeys 启用索引
func (m *Manager) EnableKeys(database, table string) error {
	// Oracle 不支持 ENABLE KEYS
	return nil
}

// SetUniqueChecks 设置唯一检查（Oracle 没有此设置）
func (m *Manager) SetUniqueChecks(enable bool) error {
	// Oracle 没有全局的 unique_checks 设置
	return nil
}
