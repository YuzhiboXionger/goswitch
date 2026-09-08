package service

import (
	"context"
	"fmt"
	"goswitch/internal/config"
	"goswitch/internal/core"
	"goswitch/internal/exchange"
	"goswitch/pkg/util"
)

// TableTask 单表迁移任务
type TableTask struct {
	ctx        context.Context
	config     *config.Config
	srcDB      core.MetadataProvider
	srcReader  core.DataReader
	dstMeta    core.MetadataProvider  // 目标端元数据提供者（用于生成目标端 DDL）
	dstManager core.TableManager
	dstWriter  core.DataWriter
	pipeline   bool // 是否启用流水线模式
	showProgress bool // 是否显示进度条
}

// Execute 执行单表迁移
func (t *TableTask) Execute(srcTable, dstTable string) (int64, error) {
	fmt.Printf("\n开始迁移: %s -> %s\n", srcTable, dstTable)

	// 1. 获取源表列信息
	columns, err := t.srcDB.GetColumns(t.config.Source.Database, srcTable)
	if err != nil {
		return 0, fmt.Errorf("failed to get columns: %w", err)
	}

	// 提取列名
	columnNames := make([]string, len(columns))
	for i, col := range columns {
		columnNames[i] = col.Name
	}

	// 2. 获取主键
	primaryKeys, err := t.srcDB.GetPrimaryKeys(t.config.Source.Database, srcTable)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary keys: %w", err)
	}

	// 3. 处理目标表
	if t.config.Target.DropTarget {
		// 禁用外键检查
		if manager, ok := t.dstManager.(interface{ DisableForeignKeyCheck() error }); ok {
			manager.DisableForeignKeyCheck()
			defer manager.(interface{ EnableForeignKeyCheck() error }).EnableForeignKeyCheck()
		}

		// 删除目标表
		if err := t.dstManager.DropTable(t.config.Target.Database, dstTable); err != nil {
			return 0, fmt.Errorf("failed to drop table: %w", err)
		}
		fmt.Printf("  ✓ 删除目标表: %s\n", dstTable)

		// 生成并创建表
		ddl, err := t.dstMeta.GenerateCreateTableDDL(t.config.Target.Database, dstTable, columns, primaryKeys)
		if err != nil {
			return 0, fmt.Errorf("failed to generate DDL: %w", err)
		}

		if err := t.dstManager.CreateTable(t.config.Target.Database, dstTable, ddl); err != nil {
			return 0, fmt.Errorf("failed to create table: %w", err)
		}
		fmt.Printf("  ✓ 创建目标表: %s\n", dstTable)
	} else {
		// 检查目标表是否存在
		exists, err := t.dstManager.TableExists(t.config.Target.Database, dstTable)
		if err != nil {
			return 0, fmt.Errorf("failed to check table existence: %w", err)
		}

		if !exists {
			// 表不存在，创建
			ddl, err := t.dstMeta.GenerateCreateTableDDL(t.config.Target.Database, dstTable, columns, primaryKeys)
			if err != nil {
				return 0, fmt.Errorf("failed to generate DDL: %w", err)
			}

			if err := t.dstManager.CreateTable(t.config.Target.Database, dstTable, ddl); err != nil {
				return 0, fmt.Errorf("failed to create table: %w", err)
			}
			fmt.Printf("  ✓ 创建目标表: %s\n", dstTable)
		} else {
			// 清空目标表
			if err := t.dstManager.TruncateTable(t.config.Target.Database, dstTable); err != nil {
				return 0, fmt.Errorf("failed to truncate table: %w", err)
			}
			fmt.Printf("  ✓ 清空目标表: %s\n", dstTable)
		}
	}

	// 4. 设置 Writer 提交间隔
	if setter, ok := t.dstWriter.(interface{ SetCommitInterval(int) }); ok {
		setter.SetCommitInterval(t.config.Target.CommitInterval)
	}

	// 5. 禁用索引和唯一检查（加速批量写入）
	if m, ok := t.dstManager.(interface {
		DisableKeys(string, string) error
		EnableKeys(string, string) error
		SetUniqueChecks(bool) error
	}); ok {
		m.SetUniqueChecks(false)
		m.DisableKeys(t.config.Target.Database, dstTable)
		fmt.Printf("  ✓ 禁用索引: %s\n", dstTable)
		defer func() {
			m.SetUniqueChecks(true)
			m.EnableKeys(t.config.Target.Database, dstTable)
		}()
	}

	// 6. 创建进度条
	var progressBar *util.ProgressBarV2
	if t.showProgress {
		progressBar = util.NewProgressBarV2(0, srcTable)
	}

	// 7. 执行数据交换（从源端读取，写入目标端）
	exchanger := exchange.NewExchanger(t.ctx)

	// 设置进度回调
	if progressBar != nil {
		exchanger.SetProgressFunc(func(rows int64) {
			progressBar.Update(rows)
		})
	}

	var totalRows int64
	if t.pipeline {
		// 流水线并行模式：读写并行，消除等待时间
		fmt.Printf("  ℹ 使用流水线并行模式\n")
		totalRows, err = exchanger.ExchangePipeline(
			t.config.Source.Database,  // 源端数据库（读取）
			srcTable,                  // 源端表名（读取）
			t.config.Target.Database,  // 目标端数据库（写入）
			dstTable,                  // 目标端表名（写入）
			columnNames,
			t.srcReader,
			t.dstWriter,
			t.config.Source.FetchSize,
		)
	} else {
		// 传统串行模式
		totalRows, err = exchanger.Exchange(
			t.config.Source.Database,  // 源端数据库（读取）
			srcTable,                  // 源端表名（读取）
			t.config.Target.Database,  // 目标端数据库（写入）
			dstTable,                  // 目标端表名（写入）
			columnNames,
			t.srcReader,
			t.dstWriter,
			t.config.Source.FetchSize,
		)
	}

	if err != nil {
		return 0, err
	}

	// 8. 打印完成信息
	if progressBar != nil {
		progressBar.SetTotal(totalRows)
		progressBar.PrintDone()
	} else {
		// 并行模式下只打印简单完成信息
		fmt.Printf("  ✓ %s: %d 行\n", srcTable, totalRows)
	}

	// 记录日志
	util.LogInfo("表 %s 迁移完成: %d 行", srcTable, totalRows)

	return totalRows, nil
}
