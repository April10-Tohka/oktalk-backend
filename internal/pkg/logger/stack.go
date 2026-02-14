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
func captureStack(skip, maxFrames int) string {
	pcs := make([]uintptr, maxFrames+skip)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return ""
	}

	frames := runtime.CallersFrames(pcs[:n])
	var buf strings.Builder
	count := 0

	for {
		frame, more := frames.Next()

		// 跳过 runtime、slog 和 logger 内部的帧
		if isInternalFrame(frame.Function) {
			if !more {
				break
			}
			continue
		}

		// 格式化输出：函数名 + 文件:行号
		funcName := shortFuncName(frame.Function)
		buf.WriteString(fmt.Sprintf("  %s\n    %s:%d\n", funcName, frame.File, frame.Line))

		count++
		if count >= maxFrames {
			break
		}
		if !more {
			break
		}
	}

	return strings.TrimRight(buf.String(), "\n")
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
