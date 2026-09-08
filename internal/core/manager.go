package core

// TableManager 表管理接口
type TableManager interface {
	// TableExists 判断表是否存在
	TableExists(database, table string) (bool, error)

	// CreateTable 创建表
	CreateTable(database, table, ddl string) error

	// DropTable 删除表
	DropTable(database, table string) error

	// TruncateTable 清空表数据
	TruncateTable(database, table string) error
}
