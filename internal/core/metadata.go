package core

// MetadataProvider 元数据查询接口
type MetadataProvider interface {
	// ListTables 查询指定数据库的表列表
	ListTables(database string) ([]string, error)

	// GetColumns 查询表的列信息
	GetColumns(database, table string) ([]ColumnMeta, error)

	// GetPrimaryKeys 查询表的主键列
	GetPrimaryKeys(database, table string) ([]string, error)

	// GenerateCreateTableDDL 生成建表 DDL
	GenerateCreateTableDDL(database, table string, columns []ColumnMeta, primaryKeys []string) (string, error)
}

// ColumnMeta 列元数据
type ColumnMeta struct {
	Name         string  // 列名
	DataType     string  // 数据类型 (如 varchar, int, datetime)
	Length       int64   // 长度
	Precision    int     // 精度
	Scale        int     // 小数位数
	Nullable     bool    // 是否可空
	DefaultValue *string // 默认值
	Comment      string  // 注释
	AutoIncr     bool    // 是否自增
}
