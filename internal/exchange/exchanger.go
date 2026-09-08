package exchange

import (
	"context"
	"fmt"
	"goswitch/internal/core"
	"sync"
	"sync/atomic"
)

// ProgressFunc 进度回调函数类型
type ProgressFunc func(rows int64)

// Exchanger 数据交换器
type Exchanger struct {
	ctx          context.Context
	progressFunc ProgressFunc
}

// NewExchanger 创建交换器
func NewExchanger(ctx context.Context) *Exchanger {
	return &Exchanger{ctx: ctx}
}

// SetProgressFunc 设置进度回调函数
func (e *Exchanger) SetProgressFunc(fn ProgressFunc) {
	e.progressFunc = fn
}

// Exchange 执行单表数据交换（串行模式，保留兼容）
func (e *Exchanger) Exchange(
	srcDatabase, srcTable string,
	dstDatabase, dstTable string,
	columns []string,
	reader core.DataReader,
	writer core.DataWriter,
	batchSize int,
) (int64, error) {

	// 准备写入器（写入目标端）
	if err := writer.Prepare(dstDatabase, dstTable, columns); err != nil {
		return 0, fmt.Errorf("failed to prepare writer: %w", err)
	}
	defer writer.Finish()

	// 启动读取（从源端读取）
	batchCh, err := reader.QueryData(srcDatabase, srcTable, columns, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to start reader: %w", err)
	}

	// 消费数据并写入
	var totalRows int64
	for batch := range batchCh {
		// 检查错误
		if batch.Error != nil {
			return totalRows, fmt.Errorf("read error: %w", batch.Error)
		}

		// 检查上下文取消
		select {
		case <-e.ctx.Done():
			return totalRows, e.ctx.Err()
		default:
		}

		// 写入数据
		if err := writer.Write(batch.Rows); err != nil {
			return totalRows, fmt.Errorf("write error: %w", err)
		}

		rows := int64(len(batch.Rows))
		totalRows += rows

		// 调用进度回调
		if e.progressFunc != nil {
			e.progressFunc(rows)
		}
	}

	return totalRows, nil
}

// ExchangePipeline 执行单表数据交换（流水线并行模式）
//
// 流水线原理：
//
//	读取协程 ──> [Buffered Channel] ──> 写入协程（主 goroutine）
//	    ↓                                    ↓
//	持续读取                            持续写入
//	不等待写入完成                      不等待读取完成
//
// 优势：读取和写入并行执行，消除等待时间，预计提升 30-50%
func (e *Exchanger) ExchangePipeline(
	srcDatabase, srcTable string,
	dstDatabase, dstTable string,
	columns []string,
	reader core.DataReader,
	writer core.DataWriter,
	batchSize int,
) (int64, error) {

	// 准备写入器（写入目标端）
	if err := writer.Prepare(dstDatabase, dstTable, columns); err != nil {
		return 0, fmt.Errorf("failed to prepare writer: %w", err)
	}
	defer writer.Finish()

	// 创建带缓冲的 channel，实现双缓冲
	// 容量为 2：一个正在写入，一个正在读取，还有一个等待被写入
	batchCh := make(chan core.DataBatch, 2)

	// 错误通道，用于从读取协程传递错误
	errCh := make(chan error, 1)

	// 完成信号，用于通知读取协程写入已完成（发生错误时）
	done := make(chan struct{})
	defer close(done)

	// 读取协程
	var readCount int64
	go func() {
		defer close(batchCh)

		// 启动读取
		srcBatchCh, err := reader.QueryData(srcDatabase, srcTable, columns, batchSize)
		if err != nil {
			errCh <- fmt.Errorf("failed to start reader: %w", err)
			return
		}

		// 持续读取，不等待写入完成
		for batch := range srcBatchCh {
			// 检查是否需要停止（写入端发生错误）
			select {
			case <-done:
				return
			default:
			}

			// 检查上下文取消
			select {
			case <-e.ctx.Done():
				// 发送取消信号到 channel
				batchCh <- core.DataBatch{Error: e.ctx.Err()}
				return
			case <-done:
				return
			case batchCh <- batch:
				// 成功发送到 channel
				if batch.Error == nil {
					atomic.AddInt64(&readCount, int64(len(batch.Rows)))
				}
			}
		}
	}()

	// 写入协程（主 goroutine）
	var totalRows int64
	for batch := range batchCh {
		// 检查批次错误
		if batch.Error != nil {
			return totalRows, fmt.Errorf("read error: %w", batch.Error)
		}

		// 检查读取协程是否发生错误
		select {
		case err := <-errCh:
			return totalRows, err
		default:
		}

		// 写入数据
		if err := writer.Write(batch.Rows); err != nil {
			return totalRows, fmt.Errorf("write error: %w", err)
		}

		rows := int64(len(batch.Rows))
		totalRows += rows

		// 调用进度回调
		if e.progressFunc != nil {
			e.progressFunc(rows)
		}
	}

	// 检查读取协程是否发生错误
	select {
	case err := <-errCh:
		return totalRows, err
	default:
	}

	// 检查上下文取消
	select {
	case <-e.ctx.Done():
		return totalRows, e.ctx.Err()
	default:
	}

	return totalRows, nil
}

// ExchangePipelineV2 执行单表数据交换（增强版流水线并行）
//
// 相比 ExchangePipeline 的改进：
//   - 支持自定义缓冲大小
//   - 支持读写统计
//   - 更精细的错误处理
func (e *Exchanger) ExchangePipelineV2(
	srcDatabase, srcTable string,
	dstDatabase, dstTable string,
	columns []string,
	reader core.DataReader,
	writer core.DataWriter,
	batchSize int,
	bufferSize int, // 新增：自定义缓冲大小
) (int64, *PipelineStats, error) {

	// 参数校验
	if bufferSize < 1 {
		bufferSize = 2
	}
	if bufferSize > 10 {
		bufferSize = 10 // 限制最大缓冲，避免内存溢出
	}

	// 准备写入器
	if err := writer.Prepare(dstDatabase, dstTable, columns); err != nil {
		return 0, nil, fmt.Errorf("failed to prepare writer: %w", err)
	}
	defer writer.Finish()

	// 创建带缓冲的 channel
	batchCh := make(chan core.DataBatch, bufferSize)

	// 错误通道
	errCh := make(chan error, 1)

	// 完成信号
	done := make(chan struct{})
	defer close(done)

	// 统计信息
	stats := &PipelineStats{}

	// 读取协程
	go func() {
		defer close(batchCh)

		srcBatchCh, err := reader.QueryData(srcDatabase, srcTable, columns, batchSize)
		if err != nil {
			errCh <- fmt.Errorf("failed to start reader: %w", err)
			return
		}

		for batch := range srcBatchCh {
			select {
			case <-done:
				return
			default:
			}

			select {
			case <-e.ctx.Done():
				batchCh <- core.DataBatch{Error: e.ctx.Err()}
				return
			case <-done:
				return
			case batchCh <- batch:
				if batch.Error == nil {
					rows := int64(len(batch.Rows))
					atomic.AddInt64(&stats.RowsRead, rows)
					atomic.AddInt64(&stats.BatchesRead, 1)
				}
			}
		}
	}()

	// 写入协程
	var totalRows int64
	for batch := range batchCh {
		if batch.Error != nil {
			return totalRows, stats, fmt.Errorf("read error: %w", batch.Error)
		}

		select {
		case err := <-errCh:
			return totalRows, stats, err
		default:
		}

		if err := writer.Write(batch.Rows); err != nil {
			return totalRows, stats, fmt.Errorf("write error: %w", err)
		}

		rows := int64(len(batch.Rows))
		totalRows += rows
		atomic.AddInt64(&stats.RowsWritten, rows)
		atomic.AddInt64(&stats.BatchesWritten, 1)
	}

	select {
	case err := <-errCh:
		return totalRows, stats, err
	default:
	}

	select {
	case <-e.ctx.Done():
		return totalRows, stats, e.ctx.Err()
	default:
	}

	return totalRows, stats, nil
}

// PipelineStats 流水线统计信息
type PipelineStats struct {
	RowsRead       int64 // 已读取行数
	RowsWritten    int64 // 已写入行数
	BatchesRead    int64 // 已读取批次数
	BatchesWritten int64 // 已写入批次数
}

// String 格式化输出统计信息
func (s *PipelineStats) String() string {
	return fmt.Sprintf("read: %d rows/%d batches, written: %d rows/%d batches",
		s.RowsRead, s.BatchesRead, s.RowsWritten, s.BatchesWritten)
}

// ExchangeMultiPipeline 执行多表流水线并行（实验性功能）
//
// 原理：多个表的数据通过同一个 channel 流转，共享读写资源
func (e *Exchanger) ExchangeMultiPipeline(
	tables []TableTask,
	reader core.DataReader,
	writer core.DataWriter,
	batchSize int,
) (map[string]int64, error) {

	results := make(map[string]int64)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, len(tables))

	// 为每个表启动独立的流水线
	for _, table := range tables {
		wg.Add(1)
		go func(t TableTask) {
			defer wg.Done()

			totalRows, err := e.ExchangePipeline(
				t.SrcDatabase, t.SrcTable,
				t.DstDatabase, t.DstTable,
				t.Columns, reader, writer, batchSize,
			)

			mu.Lock()
			results[t.SrcTable] = totalRows
			mu.Unlock()

			if err != nil {
				errCh <- fmt.Errorf("table %s: %w", t.SrcTable, err)
			}
		}(table)
	}

	wg.Wait()
	close(errCh)

	// 收集错误
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("pipeline errors: %v", errs)
	}

	return results, nil
}

// TableTask 表任务定义
type TableTask struct {
	SrcDatabase string
	SrcTable    string
	DstDatabase string
	DstTable    string
	Columns     []string
}
