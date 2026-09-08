package mysql

import (
	"database/sql"
	"fmt"
	"strings"
)

// Writer MySQL 数据写入器
type Writer struct {
	db            *sql.DB
	database      string
	table         string
	columns       []string
	colClause     string // 预构建的列名子句
	tx            *sql.Tx
	batchCount    int // 当前事务内已写入的 batch 数
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

	// 预构建列名子句
	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = fmt.Sprintf("`%s`", col)
	}
	w.colClause = strings.Join(quotedColumns, ", ")

	// 先在连接级别设置 SQL 模式
	_, err := w.db.Exec("SET SESSION sql_mode='NO_AUTO_VALUE_ON_ZERO'")
	if err != nil {
		return fmt.Errorf("failed to set sql_mode: %w", err)
	}

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
	// MySQL 限制预编译语句最多 65535 个占位符
	maxRowsPerInsert := 65535 / colCount
	if maxRowsPerInsert < 1 {
		maxRowsPerInsert = 1
	}

	// 分批插入，每批不超过 maxRowsPerInsert
	for start := 0; start < len(records); start += maxRowsPerInsert {
		end := start + maxRowsPerInsert
		if end > len(records) {
			end = len(records)
		}
		batch := records[start:end]
		rowCount := len(batch)

		// 构建批量 INSERT 语句: INSERT INTO t (cols) VALUES (?,?,?),(?,?,?),...
		oneRow := "(" + strings.Repeat("?,", colCount-1) + "?)"
		allRows := strings.Repeat(oneRow+",", rowCount-1) + oneRow

		sql := fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES %s",
			w.database, w.table, w.colClause, allRows)

		// 展开所有参数
		args := make([]interface{}, 0, rowCount*colCount)
		for _, row := range batch {
			args = append(args, row...)
		}

		if _, err := w.tx.Exec(sql, args...); err != nil {
			return fmt.Errorf("failed to execute batch insert: %w", err)
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
