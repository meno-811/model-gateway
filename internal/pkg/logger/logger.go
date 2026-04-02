package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// Logger 日志结构体（不使用接口）
type Logger struct {
	logger *slog.Logger
}

// 全局单例
var (
	globalLogger *Logger
	mu           sync.Mutex
)

// Config 配置
type Config struct {
	Level  Level
	Format string // "text" 或 "json"
	Color  bool   // 是否启用颜色
}

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

// ========== 初始化方法 ==========

// Init 初始化全局日志器（只需调用一次）
func Init(cfg Config) {
	mu.Lock()
	defer mu.Unlock()

	if globalLogger != nil {
		return // 避免重复初始化
	}

	globalLogger = New(cfg)
}

// New 创建新的日志器实例
func New(cfg Config) *Logger {
	var level slog.Level
	switch cfg.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// 文本格式 + 颜色 => 使用自定义彩色处理器
	if cfg.Format == "text" && cfg.Color {
		handler := newColoredTextHandler(level)
		return &Logger{logger: slog.New(handler)}
	}

	// 标准处理器（json或普通text）
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{logger: slog.New(handler)}
}

// Get 获取全局日志器（懒加载，线程安全）
// 如果未初始化，会使用默认配置（Info级别、文本格式、启用颜色）
func Get() *Logger {
	if globalLogger == nil {
		mu.Lock()
		if globalLogger == nil {
			globalLogger = New(Config{
				Level:  LevelInfo,
				Format: "text",
				Color:  true,
			})
		}
		mu.Unlock()
	}
	return globalLogger
}

// ========== 包级别简便函数（直接调用） ==========

func Debug(msg string) {
	Get().logger.Debug(msg)
}

func Info(msg string) {
	Get().logger.Info(msg)
}

func Warn(msg string) {
	Get().logger.Warn(msg)
}

func Error(msg string) {
	Get().logger.Error(msg)
}

// 带格式化参数的简便函数
func Debugf(format string, args ...any) {
	Get().logger.Debug(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...any) {
	Get().logger.Info(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...any) {
	Get().logger.Warn(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) {
	Get().logger.Error(fmt.Sprintf(format, args...))
}

// ========== Logger 实例方法 ==========

func (l *Logger) Debug(msg string) {
	l.logger.Debug(msg)
}

func (l *Logger) Info(msg string) {
	l.logger.Info(msg)
}

func (l *Logger) Warn(msg string) {
	l.logger.Warn(msg)
}

func (l *Logger) Error(msg string) {
	l.logger.Error(msg)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...any) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

// ========== 彩色文本处理器 ==========

type coloredTextHandler struct {
	level  slog.Level
	mu     sync.Mutex
	writer *os.File
}

func newColoredTextHandler(level slog.Level) *coloredTextHandler {
	return &coloredTextHandler{
		level:  level,
		writer: os.Stdout,
	}
}

func (h *coloredTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *coloredTextHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 时间格式：15:04:05.000
	timeStr := r.Time.Format("15:04:05.000")

	// 级别和颜色
	var levelStr, color string
	switch r.Level {
	case slog.LevelDebug:
		color = colorBlue
		levelStr = "DEBUG"
	case slog.LevelInfo:
		color = colorGreen
		levelStr = "INFO"
	case slog.LevelWarn:
		color = colorYellow
		levelStr = "WARN"
	case slog.LevelError:
		color = colorRed
		levelStr = "ERROR"
	default:
		levelStr = r.Level.String()
	}

	// 构建输出: 时间 [级别] 消息
	output := fmt.Sprintf("%s %s[%s]%s %s\n",
		timeStr,
		color,
		levelStr,
		colorReset,
		r.Message,
	)

	_, err := h.writer.WriteString(output)
	return err
}

func (h *coloredTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 简化：不处理属性，直接返回自身
	return h
}

func (h *coloredTextHandler) WithGroup(name string) slog.Handler {
	// 简化：不处理分组，直接返回自身
	return h
}
