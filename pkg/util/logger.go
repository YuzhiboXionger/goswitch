package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String 返回日志级别字符串
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志记录器
type Logger struct {
	level      LogLevel
	fileLogger *log.Logger
	console    io.Writer
	logFile    *os.File
	mu         sync.Mutex
}

// NewLogger 创建日志记录器
func NewLogger(level LogLevel, logFilePath string) (*Logger, error) {
	logger := &Logger{
		level:   level,
		console: os.Stdout,
	}

	// 如果指定了日志文件，创建文件日志
	if logFilePath != "" {
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.logFile = file
		logger.fileLogger = log.New(file, "", 0)
	}

	return logger, nil
}

// Close 关闭日志文件
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// log 写入日志
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 格式化消息
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 写入日志文件
	if l.fileLogger != nil {
		l.fileLogger.Printf("[%s] [%s] %s", timestamp, level, msg)
	}
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// ConsoleInfo 输出到终端（同时记录日志）
func (l *Logger) ConsoleInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(l.console, msg)
	l.Info(msg)
}

// ConsoleError 输出到终端（同时记录日志）
func (l *Logger) ConsoleError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(l.console, msg)
	l.Error(msg)
}

// 全局日志实例
var globalLogger *Logger

// InitLogger 初始化全局日志
func InitLogger(level LogLevel, logFilePath string) error {
	var err error
	globalLogger, err = NewLogger(level, logFilePath)
	return err
}

// GetLogger 获取全局日志
func GetLogger() *Logger {
	if globalLogger == nil {
		// 创建一个默认的日志记录器（只输出到终端）
		globalLogger, _ = NewLogger(INFO, "")
	}
	return globalLogger
}

// CloseLogger 关闭全局日志
func CloseLogger() {
	if globalLogger != nil {
		globalLogger.Close()
	}
}

// 便捷函数
func LogDebug(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

func LogInfo(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

func LogWarn(format string, args ...interface{}) {
	GetLogger().Warn(format, args...)
}

func LogError(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

func LogConsole(format string, args ...interface{}) {
	GetLogger().ConsoleInfo(format, args...)
}
