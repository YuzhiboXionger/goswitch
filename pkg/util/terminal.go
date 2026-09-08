package util

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// TerminalInfo 终端信息
type TerminalInfo struct {
	Width      int  // 终端宽度
	Height     int  // 终端高度
	IsTerminal bool // 是否是终端（非重定向）
	IsWindows  bool // 是否是 Windows系统
	SupportANSI bool // 是否支持 ANSI 转义序列
}

// GetTerminalInfo 获取终端信息
func GetTerminalInfo() *TerminalInfo {
	info := &TerminalInfo{
		IsWindows: runtime.GOOS == "windows",
	}

	// 获取终端尺寸
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// 无法获取终端大小，使用默认值
		info.Width = 80
		info.Height = 24
		info.IsTerminal = false
	} else {
		info.Width = width
		info.Height = height
		info.IsTerminal = true
	}

	// 检测 ANSI 支持
	info.SupportANSI = detectANSISupport()

	return info
}

// detectANSISupport 检测是否支持 ANSI 转义序列
func detectANSISupport() bool {
	// Windows 10+ 支持 ANSI
	if runtime.GOOS == "windows" {
		// 检查 TERM 环境变量
		if os.Getenv("TERM") != "" {
			return true
		}
		// 检查 ANSICON 环境变量
		if os.Getenv("ANSICON") != "" {
			return true
		}
		// 检查 WT_SESSION 环境变量 (Windows Terminal)
		if os.Getenv("WT_SESSION") != "" {
			return true
		}
		// Windows 10 默认支持，但需要检查
		return false
	}

	// Unix/Linux/Mac 通常支持
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}
	return true
}

// TruncateString 截断字符串到指定宽度
func TruncateString(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	return s[:maxWidth-3] + "..."
}

// PadRight 右填充字符串到指定宽度
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// PadLeft 左填充字符串到指定宽度
func PadLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// Center 居中字符串到指定宽度
func Center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	left := (width - len(s)) / 2
	right := width - len(s) - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// PrintBox 打印边框盒子
func PrintBox(title string, content []string, width int) {
	if width < 40 {
		width = 40
	}
	if width > 120 {
		width = 120
	}

	innerWidth := width - 4

	fmt.Printf("╔%s╗\n", strings.Repeat("═", width-2))
	if title != "" {
		fmt.Printf("║ %s ║\n", Center(title, innerWidth))
		fmt.Printf("╠%s╣\n", strings.Repeat("═", width-2))
	}

	for _, line := range content {
		// 处理中文字符宽度
		displayWidth := getDisplayWidth(line)
		if displayWidth > innerWidth {
			line = TruncateString(line, innerWidth)
		}
		padding := innerWidth - getDisplayWidth(line)
		if padding < 0 {
			padding = 0
		}
		fmt.Printf("║ %s%s ║\n", line, strings.Repeat(" ", padding))
	}

	fmt.Printf("╚%s╝\n", strings.Repeat("═", width-2))
}

// getDisplayWidth 获取字符串显示宽度（考虑中文字符）
func getDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		if isChinese(r) {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// isChinese 判断是否是中文字符
func isChinese(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK 统一汉字
		(r >= 0x3400 && r <= 0x4DBF) || // CJK 扩展 A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK 扩展 B
		(r >= 0x2A700 && r <= 0x2B73F) || // CJK 扩展 C
		(r >= 0x2B740 && r <= 0x2B81F) || // CJK 扩展 D
		(r >= 0xF900 && r <= 0xFAFF) || // CJK 兼容汉字
		(r >= 0x2F800 && r <= 0x2FA1F) // CJK 兼容汉字补充
}

// PrintSeparator 打印分隔线
func PrintSeparator(width int, char rune) {
	if width < 40 {
		width = 40
	}
	fmt.Println(strings.Repeat(string(char), width))
}
