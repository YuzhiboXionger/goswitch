package core

// DataReader 数据读取接口
type DataReader interface {
	// QueryData 全量查询表数据
	// 返回列名列表和数据行的 channel
	QueryData(database, table string, columns []string, fetchSize int) (<-chan DataBatch, error)
}

// DataBatch 数据批次
type DataBatch struct {
	Columns []string        // 列名
	Rows    [][]interface{} // 数据行，每行是一个 interface{} 切片
	Error   error           // 错误信息
}
