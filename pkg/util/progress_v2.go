package util

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// ProgressBarV2 终端兼容的进度条
type ProgressBarV2 struct {
	total        int64
	current      int64
	startTime    time.Time
	lastUpdate   time.Time
	lastRows     int64
	speed        float64
	tableName    string
	termInfo     *TerminalInfo
	barWidth     int
	isFinished   bool
}

// NewProgressBarV2 创建进度条
func NewProgressBarV2(total int64, tableName string) *ProgressBarV2 {
	termInfo := GetTerminalInfo()

	// 计算进度条宽度（终端宽度减去其他内容的宽度）
	barWidth := termInfo.Width - 50 // 为其他信息留出空间
	if barWidth < 20 {
		barWidth = 20
	}
	if barWidth > 60 {
		barWidth = 60
	}

	return &ProgressBarV2{
		total:      total,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
		tableName:  tableName,
		termInfo:   termInfo,
		barWidth:   barWidth,
	}
}

// Update 更新进度
func (pb *ProgressBarV2) Update(rows int64) {
	if !pb.termInfo.IsTerminal {
		// 非终端模式，不显示进度条
		return
	}

	atomic.AddInt64(&pb.current, rows)

	now := time.Now()
	elapsed := now.Sub(pb.lastUpdate)

	// 每 500ms 更新一次显示
	if elapsed >= 500*time.Millisecond {
		pb.updateSpeed()
		pb.print()
		pb.lastUpdate = now
		pb.lastRows = atomic.LoadInt64(&pb.current)
	}
}

// SetTotal 设置总数
func (pb *ProgressBarV2) SetTotal(total int64) {
	pb.total = total
}

// updateSpeed 计算速度
func (pb *ProgressBarV2) updateSpeed() {
	current := atomic.LoadInt64(&pb.current)
	elapsed := time.Since(pb.startTime).Seconds()

	if elapsed > 0 {
		pb.speed = float64(current) / elapsed
	}
}

// print 打印进度条
func (pb *ProgressBarV2) print() {
	if pb.isFinished {
		return
	}

	current := atomic.LoadInt64(&pb.current)
	elapsed := time.Since(pb.startTime)

	// 计算百分比
	var percent float64
	if pb.total > 0 {
		percent = float64(current) / float64(pb.total) * 100
	}

	// 生成进度条
	filled := int(percent / 100 * float64(pb.barWidth))
	if filled > pb.barWidth {
		filled = pb.barWidth
	}

	var bar string
	if pb.termInfo.SupportANSI {
		// 使用 ANSI 字符
		bar = strings.Repeat("█", filled) + strings.Repeat("░", pb.barWidth-filled)
	} else {
		// 使用 ASCII 字符
		bar = strings.Repeat("#", filled) + strings.Repeat("-", pb.barWidth-filled)
	}

	// 计算预计剩余时间
	var eta string
	if pb.speed > 0 && pb.total > 0 {
		remaining := float64(pb.total-current) / pb.speed
		eta = FormatDuration(time.Duration(remaining * float64(time.Second)))
	} else {
		eta = "--:--"
	}

	// 格式化速度
	speedStr := FormatSpeed(pb.speed)

	// 格式化已用时间
	elapsedStr := FormatDuration(elapsed)

	// 构建进度行
	progressLine := fmt.Sprintf("  [%s] %.1f%% | %s | %s | %s | ETA: %s",
		bar, percent, formatLargeNumber(current), speedStr, elapsedStr, eta)

	// 使用 \r 回到行首并打印
	fmt.Printf("\r%s\r%s", strings.Repeat(" ", pb.termInfo.Width), progressLine)
}

// PrintDone 打印完成信息
func (pb *ProgressBarV2) PrintDone() {
	current := atomic.LoadInt64(&pb.current)
	elapsed := time.Since(pb.startTime)
	pb.isFinished = true

	// 清除进度条行
	if pb.termInfo.IsTerminal {
		fmt.Printf("\r%s\r", strings.Repeat(" ", pb.termInfo.Width))
	}

	// 打印完成信息
	fmt.Printf("  ✓ %s 迁移完成\n", pb.tableName)
	fmt.Printf("    总行数:   %s\n", formatLargeNumber(current))
	fmt.Printf("    总耗时:   %s\n", FormatDuration(elapsed))

	if elapsed.Seconds() > 0 {
		speed := float64(current) / elapsed.Seconds()
		fmt.Printf("    平均速度: %s\n", FormatSpeed(speed))
	}

	// 估算数据量
	dataSize := current * 100
	fmt.Printf("    数据量:   ~%s\n", FormatDataSize(dataSize))
}

// PrintDoneSimple 打印简单完成信息（并行模式）
func (pb *ProgressBarV2) PrintDoneSimple() {
	current := atomic.LoadInt64(&pb.current)
	pb.isFinished = true

	fmt.Printf("  ✓ %s: %s 行\n", pb.tableName, formatLargeNumber(current))
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
