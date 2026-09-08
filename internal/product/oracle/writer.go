package oracle

import (
	"database/sql"
	"fmt"
	"strings"
)

// Writer Oracle 数据写入器
type Writer struct {
	db             *sql.DB
	database       string
	table          string
	columns        []string
	colClause      string // 预构建的列名子句
	tx             *sql.Tx
	batchCount     int // 当前事务内已写入的 batch 数
	commitInterval int // 每 N 个 batch 提交一次事务
}

// SetCommitInterval 设置事务提交间隔（每 N 个 batch 提交一次）
func (w *Writer) SetCommitInterval(interval int) {
	w.commitInterval = interval
}

// Prepare 写入前准备
func (w *Writer) Prepare(database, table string, columns []string) error {
	w.database = database
	w.table = table
	w.columns = columns
	w.batchCount = 0

	// 预构建列名子句（Oracle 使用双引号，大写）
	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = fmt.Sprintf("\"%s\"", strings.ToUpper(col))
	}
	w.colClause = strings.Join(quotedColumns, ", ")

	// 开启事务
	if err := w.beginTx(); err != nil {
		return err
	}

	return nil
}

// beginTx 开启新事务
func (w *Writer) beginTx() error {
	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	w.tx = tx
	w.batchCount = 0
	return nil
}

// commitAndRestart 提交当前事务并开启新事务
func (w *Writer) commitAndRestart() error {
	if w.tx != nil {
		if err := w.tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}
	return w.beginTx()
}

// Write 批量写入数据（多行 VALUES 一次插入）
func (w *Writer) Write(records [][]interface{}) error {
	if len(records) == 0 {
		return nil
	}

	// 达到提交间隔，提交当前事务并开启新事务
	if w.commitInterval > 0 && w.batchCount >= w.commitInterval {
		if err := w.commitAndRestart(); err != nil {
			return err
		}
	}

	colCount := len(w.columns)
	// Oracle 使用 :1, :2, :3 作为占位符
	// Oracle 对绑定变量有限制，分批处理
	maxRowsPerInsert := 10000 / colCount
	if maxRowsPerInsert < 1 {
		maxRowsPerInsert = 1
	}

	// 分批插入
	for start := 0; start < len(records); start += maxRowsPerInsert {
		end := start + maxRowsPerInsert
		if end > len(records) {
			end = len(records)
		}
		batch := records[start:end]
		rowCount := len(batch)

		// 构建批量 INSERT 语句
		// Oracle 不支持多行 VALUES，使用 UNION ALL 方式
		if rowCount == 1 {
			// 单行插入
			placeholders := make([]string, colCount)
			args := make([]interface{}, colCount)
			for j := 0; j < colCount; j++ {
				placeholders[j] = fmt.Sprintf(":%d", j+1)
				args[j] = batch[0][j]
			}

			sql := fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s)",
				strings.ToUpper(w.table), w.colClause, strings.Join(placeholders, ", "))

			if _, err := w.tx.Exec(sql, args...); err != nil {
				return fmt.Errorf("failed to execute insert: %w", err)
			}
		} else {
			// 多行插入使用 INSERT ALL
			var sb strings.Builder
			sb.WriteString("INSERT ALL\n")

			args := make([]interface{}, 0, rowCount*colCount)
			argIndex := 1

			for _, row := range batch {
				sb.WriteString(fmt.Sprintf("INTO \"%s\" (%s) VALUES (", strings.ToUpper(w.table), w.colClause))
				placeholders := make([]string, colCount)
				for j := 0; j < colCount; j++ {
					placeholders[j] = fmt.Sprintf(":%d", argIndex)
					args = append(args, row[j])
					argIndex++
				}
				sb.WriteString(strings.Join(placeholders, ", "))
				sb.WriteString(")\n")
			}
			sb.WriteString("SELECT * FROM dual")

			if _, err := w.tx.Exec(sb.String(), args...); err != nil {
				return fmt.Errorf("failed to execute batch insert: %w", err)
			}
		}
	}

	w.batchCount++

	return nil
}

// Finish 完成写入
func (w *Writer) Finish() error {
	// 提交事务
	if w.tx != nil {
		if err := w.tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}
