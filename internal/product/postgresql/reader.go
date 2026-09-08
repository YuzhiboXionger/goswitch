package postgresql

import (
	"database/sql"
	"fmt"
	"goswitch/internal/core"
	"strings"
)

// Reader PostgreSQL 数据读取器
type Reader struct {
	db *sql.DB
}

// QueryData 全量查询表数据
func (r *Reader) QueryData(database, table string, columns []string, fetchSize int) (<-chan core.DataBatch, error) {
	// 构建查询 SQL
	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = fmt.Sprintf("\"%s\"", col)
	}

	query := fmt.Sprintf("SELECT %s FROM \"%s\"",
		strings.Join(quotedColumns, ", "), table)

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}

	ch := make(chan core.DataBatch, 10)

	go func() {
		defer close(ch)
		defer rows.Close()

		for {
			batch := core.DataBatch{
				Columns: columns,
			}

			// 读取一个批次
			count := 0
			for count < fetchSize && rows.Next() {
				// 创建扫描目标
				values := make([]interface{}, len(columns))
				ptrs := make([]interface{}, len(columns))
				for i := range values {
					ptrs[i] = &values[i]
				}

				if err := rows.Scan(ptrs...); err != nil {
					ch <- core.DataBatch{Error: fmt.Errorf("failed to scan row: %w", err)}
					return
				}

				// 转换 []byte 为 string
				for i, v := range values {
					if b, ok := v.([]byte); ok {
						values[i] = string(b)
					}
				}

				batch.Rows = append(batch.Rows, values)
				count++
			}

			// 检查是否有数据
			if count == 0 {
				break
			}

			// 发送批次
			ch <- batch

			// 检查是否还有更多数据
			if count < fetchSize {
				break
			}
		}

		// 检查错误
		if err := rows.Err(); err != nil {
			ch <- core.DataBatch{Error: fmt.Errorf("rows iteration error: %w", err)}
		}
	}()

	return ch, nil
}