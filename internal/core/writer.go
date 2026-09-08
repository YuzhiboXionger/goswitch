package core

// DataWriter 数据写入接口
type DataWriter interface {
	// Prepare 写入前准备（设置表名、列名等）
	Prepare(database, table string, columns []string) error

	// Write 批量写入数据
	Write(records [][]interface{}) error

	// Finish 完成写入（提交事务等）
	Finish() error
}
