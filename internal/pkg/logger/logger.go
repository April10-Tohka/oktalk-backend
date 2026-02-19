// Package logger 提供基于 Go 标准库 log/slog 的结构化日志系统
//
// 核心特性：
//   - TraceID 自动注入：通过 context 传递，每条日志自动带上请求的 TraceID
//   - 双输出格式：控制台（彩色文本）+ 文件（JSON 结构化）
//   - 自动调用栈：Error 级别日志自动附加调用栈信息
//   - 日志轮转：按日期自动切割日志文件
//
// 使用示例：
//
//	// 初始化（通常在 main.go 或 app.go 中调用）
//	logger.Init(cfg.Log)
//	defer logger.Sync()
//
//	// 普通日志（不带 context）
//	slog.Info("server started", "port", 8080)
//
//	// 带 TraceID 的日志（在 HTTP 请求链路中使用）
//	slog.InfoContext(ctx, "request processed", "user_id", "123")
//
//	// Error 自动附加调用栈
//	slog.ErrorContext(ctx, "operation failed", "error", err)
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"pronunciation-correction-system/internal/config"
)

// 全局日志轮转 Writer（用于 Sync/Close）
var globalRotator *RotatingWriter

// Init 初始化全局日志系统
//
// 参数：
//   - cfg: 日志配置
//
// 行为：
//   - 控制台输出：文本格式（可选彩色）
//   - 文件输出：JSON 格式（结构化）
//   - 自动注入 TraceID 和 Error 调用栈
//   - 设置为 slog 全局默认 Logger
func Init(cfg config.LogConfig) (*slog.Logger, error) {
	slogLevel := parseLevel(cfg.Level)

	handlers := make([]slog.Handler, 0, 2)
	if cfg.Console.Enabled {
		handlers = append(handlers, newConsoleHandler(slogLevel, cfg.Console.Colorful))
	}

	if cfg.File.Enabled {
		rotator, err := NewRotatingWriter(cfg.File)
		if err != nil {
			return nil, err
		}
		globalRotator = rotator
		handlers = append(handlers, newFileHandler(slogLevel, rotator))
	}

	if len(handlers) == 0 {
		handlers = append(handlers, slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
			Level: slogLevel,
		}))
	}

	handler := newMultiHandler(handlers...)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slogLevel)

	slog.Info("logger initialized",
		"level", cfg.Level,
		"console_enabled", cfg.Console.Enabled,
		"file_enabled", cfg.File.Enabled,
		"file", cfg.File.Filename,
	)

	return logger, nil
}

// Sync 同步日志缓冲并关闭文件
// 应在程序退出前调用（通常使用 defer）
func Sync() error {
	if globalRotator != nil {
		return globalRotator.Close()
	}
	return nil
}

// ===================== 便捷函数 =====================

// L 获取带 TraceID 的 Logger
// 适用于在一个函数中多次记录日志的场景
//
// 用法：
//
//	log := logger.L(ctx)
//	log.Info("step 1 done")
//	log.Info("step 2 done")
func L(ctx context.Context) *slog.Logger {
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		return slog.Default().With("trace_id", traceID)
	}
	return slog.Default()
}

// InfoContext 带 context 的 Info 日志（便捷函数）
func InfoContext(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}

// ErrorContext 带 context 的 Error 日志（便捷函数）
func ErrorContext(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, args...)
}

// WarnContext 带 context 的 Warn 日志（便捷函数）
func WarnContext(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, args...)
}

// DebugContext 带 context 的 Debug 日志（便捷函数）
func DebugContext(ctx context.Context, msg string, args ...any) {
	slog.DebugContext(ctx, msg, args...)
}

// ===================== 内部工具 =====================

// parseLevel 将字符串日志级别转换为 slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// initDefault 确保在包初始化时有一个基本的 Logger（即使未调用 Init）
func init() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler))
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

// Err 创建一个 error 类型的日志属性（便捷函数）
// 用法: slog.ErrorContext(ctx, "failed", logger.Err(err))
func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", err.Error())
}

// SetLogOutput 设置标准 log 包的输出到 slog（兼容旧代码）
// Go 1.22+ 的 slog 内置了此功能：log 包的输出会自动路由到 slog
func SetLogOutput() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	// Go 1.22 中 log 包默认通过 slog 的 default handler 输出
	// 无需额外设置
}

// SetLevel 动态调整日志级别
// 注意：需要重新初始化 handler，当前仅在启动时生效
func SetLevel(level string) {
	slogLevel := parseLevel(level)
	// 由于 slog 不支持动态修改 handler 的 level，需要 leveler
	// 简单方案：重新初始化
	_ = slogLevel

	// 提示使用者重新调用 Init
	slog.Warn("dynamic level change not supported, please call logger.Init() again",
		"requested_level", level)
}

// InitForTest 用于测试环境的简化初始化
// 输出到 os.Stderr，不写文件
func InitForTest() {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler))
}
