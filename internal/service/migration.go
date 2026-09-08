package service

import (
	"context"
	"database/sql"
	"fmt"
	"goswitch/internal/config"
	"goswitch/internal/core"
	"goswitch/pkg/util"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Version 版本号
const Version = "1.2.0"

// MigrationService 迁移服务
type MigrationService struct {
	config     *config.Config
	srcDB      *sql.DB
	dstDB      *sql.DB
	srcMeta    core.MetadataProvider
	srcReader  core.DataReader
	dstMeta    core.MetadataProvider  // 目标端元数据提供者
	dstFactory core.Factory
	dstMgr     core.TableManager
}

// NewMigrationService 创建迁移服务
func NewMigrationService(cfg *config.Config, srcDB, dstDB *sql.DB) (*MigrationService, error) {
	// 获取方言工厂
	srcFactory, err := core.GetFactory(cfg.Source.Type)
	if err != nil {
		return nil, err
	}
	dstFactory, err := core.GetFactory(cfg.Target.Type)
	if err != nil {
		return nil, err
	}

	return &MigrationService{
		config:     cfg,
		srcDB:      srcDB,
		dstDB:      dstDB,
		srcMeta:    srcFactory.MetadataProvider(srcDB),
		srcReader:  srcFactory.DataReader(srcDB),
		dstMeta:    dstFactory.MetadataProvider(dstDB),  // 目标端元数据提供者
		dstFactory: dstFactory,
		dstMgr:     dstFactory.TableManager(dstDB),
	}, nil
}

// Run 执行迁移
func (s *MigrationService) Run(ctx context.Context) error {
	startTime := time.Now()

	// 1. 显示版本信息
	s.printVersion()

	// 2. 显示配置信息
	s.printConfig()

	// 3. 获取源端表列表
	tables, err := s.srcMeta.ListTables(s.config.Source.Database)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	// 4. 过滤表
	filteredTables := s.filterTables(tables)
	fmt.Printf("\n待迁移表数量: %d\n", len(filteredTables))

	if len(filteredTables) == 0 {
		fmt.Println("没有需要迁移的表")
		return nil
	}

	// 5. 显示表信息预览
	s.printTablePreview(filteredTables)

	// 6. 并行迁移
	parallel := s.config.Target.Parallel
	if parallel > len(filteredTables) {
		parallel = len(filteredTables)
	}
	fmt.Printf("\n并行度: %d\n", parallel)

	var totalRows int64
	var successCount, failCount int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel) // 并发信号量

	for i, srcTable := range filteredTables {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(idx int, table string) {
			defer wg.Done()

			// 获取信号量（控制并发数）
			sem <- struct{}{}
			defer func() { <-sem }()

			// 计算目标表名
			dstTable := s.getDstTableName(table)

			mu.Lock()
			fmt.Printf("\n[%d/%d] ", idx+1, len(filteredTables))
			mu.Unlock()

			// 为每个表创建独立的 Writer
			dstWriter := s.dstFactory.DataWriter(s.dstDB)

			// 执行单表迁移
			task := &TableTask{
				ctx:        ctx,
				config:     s.config,
				srcDB:      s.srcMeta,
				srcReader:  s.srcReader,
				dstMeta:    s.dstMeta,
				dstManager: s.dstMgr,
				dstWriter:  dstWriter,
				pipeline:   s.config.Target.Pipeline,
				showProgress: parallel == 1, // 只有串行模式才显示进度条
			}

			rows, err := task.Execute(table, dstTable)
			if err != nil {
				mu.Lock()
				fmt.Printf("  ✗ 迁移失败: %v\n", err)
				mu.Unlock()
				atomic.AddInt64(&failCount, 1)
				return
			}

			atomic.AddInt64(&totalRows, rows)
			atomic.AddInt64(&successCount, 1)
		}(i, srcTable)
	}

	wg.Wait()

	// 4. 打印总结
	elapsed := time.Since(startTime)
	finalSuccess := atomic.LoadInt64(&successCount)
	finalFail := atomic.LoadInt64(&failCount)
	finalRows := atomic.LoadInt64(&totalRows)

	s.printSummary(elapsed, finalSuccess, finalFail, finalRows, len(filteredTables), parallel)

	return nil
}

// filterTables 过滤表列表
func (s *MigrationService) filterTables(tables []string) []string {
	var result []string

	// 处理包含列表
	if s.config.Source.Includes != "" {
		includeMap := make(map[string]bool)
		for _, name := range strings.Split(s.config.Source.Includes, ",") {
			includeMap[strings.TrimSpace(name)] = true
		}

		for _, table := range tables {
			if includeMap[table] {
				result = append(result, table)
			}
		}
		return result
	}

	// 处理排除列表
	excludeMap := make(map[string]bool)
	if s.config.Source.Excludes != "" {
		for _, name := range strings.Split(s.config.Source.Excludes, ",") {
			excludeMap[strings.TrimSpace(name)] = true
		}
	}

	for _, table := range tables {
		if !excludeMap[table] {
			result = append(result, table)
		}
	}

	return result
}

// getDstTableName 获取目标表名
func (s *MigrationService) getDstTableName(srcTable string) string {
	// 应用正则映射
	dstTable := util.ApplyRegexMapper(srcTable, s.config.Source.TableMapper)

	// 应用大小写转换
	switch strings.ToUpper(s.config.Target.TableNameCase) {
	case "UPPER":
		dstTable = strings.ToUpper(dstTable)
	case "LOWER":
		dstTable = strings.ToLower(dstTable)
	}

	return dstTable
}

// printVersion 打印版本信息
func (s *MigrationService) printVersion() {
	termInfo := util.GetTerminalInfo()
	width := termInfo.Width
	if width < 60 {
		width = 60
	}
	if width > 100 {
		width = 100
	}

	util.PrintBox("", []string{
		fmt.Sprintf("goswitch v%s", Version),
	}, width)
}

// printConfig 打印配置信息
func (s *MigrationService) printConfig() {
	termInfo := util.GetTerminalInfo()
	width := termInfo.Width
	if width < 60 {
		width = 60
	}
	if width > 100 {
		width = 100
	}
	innerWidth := width - 4

	// 构建配置内容
	content := []string{
		"源端配置:",
		fmt.Sprintf("  类型:     %s", s.config.Source.Type),
		fmt.Sprintf("  地址:     %s:%d", s.config.Source.Host, s.config.Source.Port),
		fmt.Sprintf("  数据库:   %s", s.config.Source.Database),
		fmt.Sprintf("  用户名:   %s", s.config.Source.Username),
		fmt.Sprintf("  批次大小: %d", s.config.Source.FetchSize),
		"",
		"目标端配置:",
		fmt.Sprintf("  类型:     %s", s.config.Target.Type),
		fmt.Sprintf("  地址:     %s:%d", s.config.Target.Host, s.config.Target.Port),
		fmt.Sprintf("  数据库:   %s", s.config.Target.Database),
		fmt.Sprintf("  用户名:   %s", s.config.Target.Username),
		fmt.Sprintf("  批次大小: %d", s.config.Target.BatchSize),
		fmt.Sprintf("  并行度:   %d", s.config.Target.Parallel),
		fmt.Sprintf("  事务间隔: %d", s.config.Target.CommitInterval),
		fmt.Sprintf("  流水线:   %v", s.config.Target.Pipeline),
		"",
		"过滤规则:",
	}

	// 添加过滤规则
	if s.config.Source.Includes != "" {
		content = append(content, fmt.Sprintf("  包含:     %s", s.config.Source.Includes))
	} else {
		content = append(content, "  包含:     全部")
	}
	if s.config.Source.Excludes != "" {
		content = append(content, fmt.Sprintf("  排除:     %s", s.config.Source.Excludes))
	} else {
		content = append(content, "  排除:     无")
	}

	// 添加转换规则
	content = append(content, "", "转换规则:")
	if len(s.config.Source.TableMapper) > 0 {
		for _, mapper := range s.config.Source.TableMapper {
			content = append(content, fmt.Sprintf("  映射:     %s → %s", mapper.FromPattern, mapper.ToValue))
		}
	} else {
		content = append(content, "  映射:     无")
	}
	if s.config.Target.TableNameCase != "" {
		content = append(content, fmt.Sprintf("  大小写:   %s", s.config.Target.TableNameCase))
	} else {
		content = append(content, "  大小写:   NONE")
	}

	// 截断过长的行
	for i, line := range content {
		if len(line) > innerWidth {
			content[i] = util.TruncateString(line, innerWidth)
		}
	}

	util.PrintBox("配置信息", content, width)

	// 记录日志
	util.LogInfo("配置加载完成: %s -> %s", s.config.Source.Type, s.config.Target.Type)
}

// printSummary 打印迁移汇总统计
func (s *MigrationService) printSummary(elapsed time.Duration, success, fail, totalRows int64, tableCount, parallel int) {
	termInfo := util.GetTerminalInfo()
	width := termInfo.Width
	if width < 60 {
		width = 60
	}
	if width > 100 {
		width = 100
	}

	// 构建内容
	content := []string{
		"基本信息:",
		fmt.Sprintf("  总耗时:     %s", util.FormatDuration(elapsed)),
		fmt.Sprintf("  总表数:     %d", tableCount),
		fmt.Sprintf("  成功:       %d", success),
	}
	if fail > 0 {
		content = append(content, fmt.Sprintf("  失败:       %d", fail))
	}
	content = append(content, fmt.Sprintf("  并行度:     %d", parallel), "")

	// 性能统计
	content = append(content, "性能统计:")
	content = append(content, fmt.Sprintf("  总行数:     %s", formatLargeNumber(totalRows)))

	if elapsed.Seconds() > 0 {
		rowsPerSec := float64(totalRows) / elapsed.Seconds()
		content = append(content, fmt.Sprintf("  平均速度:   %s", util.FormatSpeed(rowsPerSec)))

		// 估算每行数据大小（假设平均100字节）
		bytesPerRow := int64(100)
		totalBytes := totalRows * bytesPerRow
		throughput := float64(totalBytes) / elapsed.Seconds()
		content = append(content, fmt.Sprintf("  吞吐量:     %s", formatThroughput(throughput)))
	}

	// 估算数据量
	dataSize := totalRows * 100 // 假设每行平均100字节
	content = append(content, fmt.Sprintf("  数据量:     ~%s", util.FormatDataSize(dataSize)), "")

	// 资源使用统计
	content = append(content, "资源使用:")
	content = append(content, fmt.Sprintf("  批次大小:   %d", s.config.Source.FetchSize))
	content = append(content, fmt.Sprintf("  写入批次:   %d", s.config.Target.BatchSize))
	content = append(content, fmt.Sprintf("  事务间隔:   %d", s.config.Target.CommitInterval))
	content = append(content, fmt.Sprintf("  流水线:     %v", s.config.Target.Pipeline))

	// 获取内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	content = append(content, fmt.Sprintf("  内存使用:   %s", util.FormatDataSize(int64(memStats.Alloc))))
	content = append(content, fmt.Sprintf("  GC次数:     %d", memStats.NumGC))

	util.PrintBox("迁移完成汇总", content, width)

	// 记录日志
	util.LogInfo("迁移完成: %d 行, 耗时 %s, 成功 %d, 失败 %d",
		totalRows, util.FormatDuration(elapsed), success, fail)
}

// formatLargeNumber 格式化大数字
func formatLargeNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(n)/1000000000)
}

// formatThroughput 格式化吞吐量
func formatThroughput(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.1f B/s", bytesPerSec)
	}
	if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	}
	if bytesPerSec < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB/s", bytesPerSec/(1024*1024*1024))
}

// printTablePreview 打印表信息预览
func (s *MigrationService) printTablePreview(tables []string) {
	termInfo := util.GetTerminalInfo()
	width := termInfo.Width
	if width < 60 {
		width = 60
	}
	if width > 100 {
		width = 100
	}
	innerWidth := width - 4

	// 构建内容
	content := []string{
		fmt.Sprintf("%-4s %-30s %-6s %-18s", "序号", "表名", "列数", "数据量"),
		strings.Repeat("─", innerWidth),
	}

	// 当表大于3张时只显示前3张
	previewCount := 3
	if len(tables) < previewCount {
		previewCount = len(tables)
	}

	for i := 0; i < previewCount; i++ {
		table := tables[i]
		dstTable := s.getDstTableName(table)

		// 获取列信息
		columns, err := s.srcMeta.GetColumns(s.config.Source.Database, table)
		if err != nil {
			content = append(content, fmt.Sprintf("%-4d %-30s %-6s %-18s", i+1, table, "错误", "无法获取"))
			continue
		}

		// 获取数据量（使用 COUNT(*)）
		var rowCount int64
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)
		err = s.srcDB.QueryRow(countQuery).Scan(&rowCount)
		if err != nil {
			// 尝试使用 PostgreSQL 语法
			countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table)
			err = s.srcDB.QueryRow(countQuery).Scan(&rowCount)
			if err != nil {
				rowCount = -1
			}
		}

		// 格式化数据量
		rowCountStr := formatRowCount(rowCount)

		// 如果有映射，显示映射后的表名
		if dstTable != table {
			content = append(content, fmt.Sprintf("%-4d %-30s %-6d %-18s", i+1, fmt.Sprintf("%s → %s", table, dstTable), len(columns), rowCountStr))
		} else {
			content = append(content, fmt.Sprintf("%-4d %-30s %-6d %-18s", i+1, table, len(columns), rowCountStr))
		}

		// 添加列信息
		content = append(content, "")
		content = append(content, "列信息:")
		content = append(content, fmt.Sprintf("  %-4s %-20s %-15s %-8s %-8s", "序号", "列名", "类型", "可空", "自增"))
		content = append(content, "  "+strings.Repeat("─", innerWidth-2))

		// 最多显示 10列
		displayCount := 10
		if len(columns) < displayCount {
			displayCount = len(columns)
		}

		for j, col := range columns {
			if j >= displayCount {
				content = append(content, fmt.Sprintf("  ... 还有 %d 列未显示 ...", len(columns)-displayCount))
				break
			}

			nullableStr := "否"
			if col.Nullable {
				nullableStr = "是"
			}

			autoIncrStr := "否"
			if col.AutoIncr {
				autoIncrStr = "是"
			}

			// 格式化类型
			typeStr := col.DataType
			if col.Length > 0 {
				typeStr = fmt.Sprintf("%s(%d)", col.DataType, col.Length)
			}

			content = append(content, fmt.Sprintf("  %-4d %-20s %-15s %-8s %-8s", j+1, col.Name, typeStr, nullableStr, autoIncrStr))
		}

		if i < previewCount-1 {
			content = append(content, "")
			content = append(content, strings.Repeat("─", innerWidth))
		}
	}

	if len(tables) > previewCount {
		content = append(content, "")
		content = append(content, fmt.Sprintf("... 还有 %d 个表未显示 ...", len(tables)-previewCount))
	}

	// 截断过长的行
	for i, line := range content {
		if len(line) > innerWidth {
			content[i] = util.TruncateString(line, innerWidth)
		}
	}

	util.PrintBox("表信息预览", content, width)

	// 记录日志
	util.LogInfo("待迁移表数量: %d", len(tables))
}

// formatRowCount 格式化行数
func formatRowCount(count int64) string {
	if count < 0 {
		return "未知"
	}
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	if count < 1000000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(count)/1000000)
}
