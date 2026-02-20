package logger

import (
	"fmt"
	"runtime"
	"strings"
)

// captureStack 捕获当前调用栈信息
// skip: 跳过的帧数（用于跳过 logger 内部调用）
// maxFrames: 最多捕获的帧数
// 返回格式化的调用栈字符串
func captureStack(skip, max int) string {
	pcs := make([]uintptr, max)
	n := runtime.Callers(skip, pcs)

	frames := runtime.CallersFrames(pcs[:n])
	var sb strings.Builder

	level := 0
	for {
		frame, more := frames.Next()
		if !more {
			break
		}

		// 过滤 gin / runtime
		if strings.Contains(frame.File, "/gin-gonic/") {
			continue
		}
		if strings.Contains(frame.File, "/runtime/") {
			continue
		}

		sb.WriteString(fmt.Sprintf(
			"[%d] %s\n    %s:%d\n",
			level,
			frame.Function,
			frame.File,
			frame.Line,
		))
		level++
	}

	return sb.String()
}

// isInternalFrame 判断是否为内部帧（应跳过的帧）
func isInternalFrame(funcName string) bool {
	// 跳过 Go runtime 帧
	if strings.HasPrefix(funcName, "runtime.") {
		return true
	}
	// 跳过 slog 内部帧
	if strings.Contains(funcName, "log/slog") {
		return true
	}
	// 跳过我们的 logger 包内部帧
	if strings.Contains(funcName, "pkg/logger") {
		return true
	}
	return false
}

// shortFuncName 将完整函数名缩短为可读形式
// 例如: "pronunciation-correction-system/internal/service/evaluation.(*ServiceImpl).Evaluate"
// 缩短为: "evaluation.(*ServiceImpl).Evaluate"
func shortFuncName(fullName string) string {
	// 找最后一个 "/" 后面的部分
	if idx := strings.LastIndex(fullName, "/"); idx >= 0 {
		return fullName[idx+1:]
	}
	return fullName
}
