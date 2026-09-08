package util

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// ProgressBar 实时进度条
type ProgressBar struct {
	total       int64
	current     int64
	startTime   time.Time
	lastUpdate  time.Time
	lastRows    int64
	speed       float64 // 行/秒
	tableName   string
}

// NewProgressBar 创建进度条
func NewProgressBar(total int64, tableName string) *ProgressBar {
	return &ProgressBar{
		total:      total,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
		tableName:  tableName,
	}
}

// Update 更新进度
func (pb *ProgressBar) Update(rows int64) {
	atomic.AddInt64(&pb.current, rows)

	now := time.Now()
	elapsed := now.Sub(pb.lastUpdate)

	// 每500ms更新一次显示
	if elapsed >= 500*time.Millisecond {
		pb.updateSpeed()
		pb.print()
		pb.lastUpdate = now
		pb.lastRows = atomic.LoadInt64(&pb.current)
	}
}

// SetTotal 设置总数
func (pb *ProgressBar) SetTotal(total int64) {
	pb.total = total
}

// updateSpeed 计算速度
func (pb *ProgressBar) updateSpeed() {
	current := atomic.LoadInt64(&pb.current)
	elapsed := time.Since(pb.startTime).Seconds()

	if elapsed > 0 {
		pb.speed = float64(current) / elapsed
	}
}

// print 打印进度条
func (pb *ProgressBar) print() {
	current := atomic.LoadInt64(&pb.current)
	elapsed := time.Since(pb.startTime)

	// 计算百分比
	var percent float64
	if pb.total > 0 {
		percent = float64(current) / float64(pb.total) * 100
	}

	// 生成进度条
	barWidth := 30
	filled := int(percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// 计算预计剩余时间
	var eta string
	if pb.speed > 0 && pb.total > 0 {
		remaining := float64(pb.total-current) / pb.speed
		eta = FormatDuration(time.Duration(remaining * float64(time.Second)))
	} else {
		eta = "计算中..."
	}

	// 格式化速度
	speedStr := FormatSpeed(pb.speed)

	// 格式化已用时间
	elapsedStr := FormatDuration(elapsed)

	// 打印进度条（使用 \r 回到行首）
	fmt.Printf("\r  [%s] %.1f%% | %s | 速度: %s | 已用: %s | 剩余: %s    ",
		bar, percent, formatRowCount(current), speedStr, elapsedStr, eta)
}

// PrintDone 打印完成信息
func (pb *ProgressBar) PrintDone() {
	current := atomic.LoadInt64(&pb.current)
	elapsed := time.Since(pb.startTime)

	// 清除进度条行
	fmt.Printf("\r%s\r", strings.Repeat(" ", 100))

	// 打印完成信息
	fmt.Printf("  ✓ 迁移完成: %s\n", pb.tableName)
	fmt.Printf("    总行数:   %s\n", formatRowCount(current))
	fmt.Printf("    总耗时:   %s\n", FormatDuration(elapsed))

	if elapsed.Seconds() > 0 {
		speed := float64(current) / elapsed.Seconds()
		fmt.Printf("    平均速度: %s\n", FormatSpeed(speed))
	}

	// 估算数据量（假设每行平均100字节）
	dataSize := current * 100
	fmt.Printf("    数据量:   ~%s\n", FormatDataSize(dataSize))
}

// FormatSpeed 格式化速度
func FormatSpeed(rowsPerSec float64) string {
	if rowsPerSec < 1 {
		return fmt.Sprintf("%.2f 行/秒", rowsPerSec)
	}
	if rowsPerSec < 1000 {
		return fmt.Sprintf("%.1f 行/秒", rowsPerSec)
	}
	if rowsPerSec < 1000000 {
		return fmt.Sprintf("%.1fK 行/秒", rowsPerSec/1000)
	}
	return fmt.Sprintf("%.1fM 行/秒", rowsPerSec/1000000)
}

// FormatDataSize 格式化数据大小
func FormatDataSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// formatRowCount 格式化行数
func formatRowCount(count int64) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	if count < 1000000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(count)/1000000)
}
