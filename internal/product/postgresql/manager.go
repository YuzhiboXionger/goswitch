package postgresql

import (
	"database/sql"
	"fmt"
)

// Manager PostgreSQL 表管理器
type Manager struct {
	db *sql.DB
}

// TableExists 判断表是否存在
func (m *Manager) TableExists(database, table string) (bool, error) {
	query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1"
	var count int
	err := m.db.QueryRow(query, table).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
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
	query := fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE", table)
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}
	return nil
}

// TruncateTable 清空表数据
func (m *Manager) TruncateTable(database, table string) error {
	query := fmt.Sprintf("TRUNCATE TABLE \"%s\" CASCADE", table)
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to truncate table: %w", err)
	}
	return nil
}

// DisableForeignKeyCheck 禁用外键检查
func (m *Manager) DisableForeignKeyCheck() error {
	// PostgreSQL 没有全局禁用外键检查的命令
	// 需要在每个表上使用 TRUNCATE ... CASCADE 或者在事务中禁用触发器
	return nil
}

// EnableForeignKeyCheck 启用外键检查
func (m *Manager) EnableForeignKeyCheck() error {
	// PostgreSQL 没有全局启用外键检查的命令
	return nil
}

// DisableKeys 禁用索引（PostgreSQL 不支持 DISABLE KEYS）
func (m *Manager) DisableKeys(database, table string) error {
	// PostgreSQL 不支持 DISABLE KEYS，但可以删除索引后重建
	// 这里返回 nil，不执行任何操作
	return nil
}

// EnableKeys 启用索引
func (m *Manager) EnableKeys(database, table string) error {
	// PostgreSQL 不支持 ENABLE KEYS
	return nil
}

// SetUniqueChecks 设置唯一检查
func (m *Manager) SetUniqueChecks(enable bool) error {
	// PostgreSQL 没有全局的 unique_checks 设置
	// 可以通过 session_replication_role 来控制，但需要超级用户权限
	return nil
}