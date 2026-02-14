package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// ===================== 自定义多路 Handler =====================

// MultiHandler 多路输出 Handler
// 将日志同时发送到多个底层 Handler（如控制台 + 文件）
// 同时自动注入 TraceID 和 Error 级别调用栈
type MultiHandler struct {
	handlers []slog.Handler // 底层 Handler 列表
	level    slog.Level     // 最低日志级别
}

// 确保 MultiHandler 实现了 slog.Handler 接口
var _ slog.Handler = (*MultiHandler)(nil)

// newMultiHandler 创建多路输出 Handler
// level: 最低日志级别
// handlers: 底层 Handler 列表
func newMultiHandler(level slog.Level, handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{
		handlers: handlers,
		level:    level,
	}
}

// Enabled 判断指定级别的日志是否启用
// 只要有一个底层 Handler 启用了该级别，就返回 true
func (h *MultiHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle 处理日志记录
// 自动注入 TraceID（从 context）和调用栈（Error 级别）
func (h *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	// 克隆 record 以避免修改原始数据
	r := record.Clone()

	// 从 context 中提取 TraceID 并注入到日志属性
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}

	// Error 级别自动添加调用栈信息
	if r.Level >= slog.LevelError {
		stack := captureStack(2, 10) // skip=2 跳过 Handle 和 captureStack
		if stack != "" {
			r.AddAttrs(slog.String("stack", stack))
		}
	}

	// 发送到所有底层 Handler
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}

	return nil
}

// WithAttrs 返回一个携带额外属性的新 Handler
func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &MultiHandler{
		handlers: newHandlers,
		level:    h.level,
	}
}

// WithGroup 返回一个使用指定分组名的新 Handler
func (h *MultiHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &MultiHandler{
		handlers: newHandlers,
		level:    h.level,
	}
}

// ===================== Handler 工厂方法 =====================

// newConsoleHandler 创建控制台文本 Handler
// 使用彩色输出，时间格式为 "2006-01-02 15:04:05"
func newConsoleHandler(level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 自定义时间格式：简洁可读
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
			}
			return a
		},
	}

	// 使用彩色 Writer 包装 Stdout
	colorOut := &colorWriter{w: os.Stdout}
	return slog.NewTextHandler(colorOut, opts)
}

// newFileHandler 创建文件 JSON Handler
// 使用标准 JSON 格式输出，便于后期日志分析
func newFileHandler(level slog.Level, w io.Writer) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	return slog.NewJSONHandler(w, opts)
}

// ===================== 彩色输出 Writer =====================

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// colorWriter 彩色输出 Writer
// 将日志级别文本替换为带 ANSI 颜色码的版本
type colorWriter struct {
	w  io.Writer
	mu sync.Mutex
}

// Write 实现 io.Writer 接口
// 根据日志级别添加 ANSI 颜色
func (cw *colorWriter) Write(p []byte) (n int, err error) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	line := string(p)

	// 根据级别关键字添加颜色
	switch {
	case containsLevel(line, "ERROR"):
		line = colorize(line, "ERROR", colorRed)
	case containsLevel(line, "WARN"):
		line = colorize(line, "WARN", colorYellow)
	case containsLevel(line, "INFO"):
		line = colorize(line, "INFO", colorGreen)
	case containsLevel(line, "DEBUG"):
		line = colorize(line, "DEBUG", colorCyan)
	}

	_, err = cw.w.Write([]byte(line))
	return len(p), err
}

// containsLevel 检查行中是否包含指定级别
func containsLevel(line, level string) bool {
	// slog.TextHandler 输出格式: level=INFO
	target := "level=" + level
	for i := 0; i <= len(line)-len(target); i++ {
		if line[i:i+len(target)] == target {
			return true
		}
	}
	return false
}

// colorize 为指定级别文本添加颜色
func colorize(line, level, color string) string {
	target := "level=" + level
	colored := "level=" + color + level + colorReset
	// 手动替换第一个匹配
	for i := 0; i <= len(line)-len(target); i++ {
		if line[i:i+len(target)] == target {
			return line[:i] + colored + line[i+len(target):]
		}
	}
	return line
}
