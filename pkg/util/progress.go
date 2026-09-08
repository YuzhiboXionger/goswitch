package util

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Progress 进度跟踪器
type Progress struct {
	total     int64
	current   int64
	startTime time.Time
	mu        sync.RWMutex
}

// NewProgress 创建进度跟踪器
func NewProgress(total int64) *Progress {
	return &Progress{
		total:     total,
		startTime: time.Now(),
	}
}

// Increment 增加进度
func (p *Progress) Increment() {
	atomic.AddInt64(&p.current, 1)
}

// SetTotal 设置总数
func (p *Progress) SetTotal(total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
}

// Print 打印进度
func (p *Progress) Print(tableName string) {
	current := atomic.LoadInt64(&p.current)
	elapsed := time.Since(p.startTime)

	if p.total > 0 {
		percent := float64(current) / float64(p.total) * 100
		fmt.Printf("\r[%s] %s: %d/%d (%.1f%%) - %s",
			elapsed.Truncate(time.Second), tableName, current, p.total, percent,
			FormatDuration(elapsed))
	} else {
		fmt.Printf("\r[%s] %s: %d rows - %s",
			elapsed.Truncate(time.Second), tableName, current,
			FormatDuration(elapsed))
	}
}

// PrintDone 打印完成信息
func (p *Progress) PrintDone(tableName string) {
	current := atomic.LoadInt64(&p.current)
	elapsed := time.Since(p.startTime)
	fmt.Printf("\r[%s] %s: %d rows ✓ - %s\n",
		elapsed.Truncate(time.Second), tableName, current,
		FormatDuration(elapsed))
}

// FormatDuration 格式化时间
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm%ds", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}
