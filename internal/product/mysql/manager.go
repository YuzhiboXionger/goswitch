package mysql

import (
	"database/sql"
	"fmt"
)

// Manager MySQL 表管理器
type Manager struct {
	db *sql.DB
}

// TableExists 判断表是否存在
func (m *Manager) TableExists(database, table string) (bool, error) {
	query := "SELECT 1 FROM information_schema.tables WHERE table_schema = ? AND table_name = ? LIMIT 1"
	var dummy int
	err := m.db.QueryRow(query, database, table).Scan(&dummy)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	return true, nil
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
	query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", database, table)
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}
	return nil
}

// DisableForeignKeyCheck 禁用外键检查
func (m *Manager) DisableForeignKeyCheck() error {
	_, err := m.db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	return err
}

// EnableForeignKeyCheck 启用外键检查
func (m *Manager) EnableForeignKeyCheck() error {
	_, err := m.db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	return err
}

// TruncateTable 清空表数据
func (m *Manager) TruncateTable(database, table string) error {
	query := fmt.Sprintf("TRUNCATE TABLE `%s`.`%s`", database, table)
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to truncate table: %w", err)
	}
	return nil
}

// DisableKeys 禁用非唯一索引（批量写入前调用）
func (m *Manager) DisableKeys(database, table string) error {
	query := fmt.Sprintf("ALTER TABLE `%s`.`%s` DISABLE KEYS", database, table)
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to disable keys: %w", err)
	}
	return nil
}

// EnableKeys 启用非唯一索引（批量写入后调用）
func (m *Manager) EnableKeys(database, table string) error {
	query := fmt.Sprintf("ALTER TABLE `%s`.`%s` ENABLE KEYS", database, table)
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to enable keys: %w", err)
	}
	return nil
}

// SetUniqueChecks 设置唯一检查
func (m *Manager) SetUniqueChecks(enable bool) error {
	val := 1
	if !enable {
		val = 0
	}
	_, err := m.db.Exec(fmt.Sprintf("SET SESSION unique_checks = %d", val))
	return err
}
