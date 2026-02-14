package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RotatingWriter 按日期轮转的日志文件 Writer
// 每天自动创建新文件，旧文件按日期命名归档
//
// 文件命名示例:
//
//	logs/app.log           ← 当前日志
//	logs/app-2024-01-14.log ← 前一天归档
//	logs/app-2024-01-13.log ← 更早归档
type RotatingWriter struct {
	mu       sync.Mutex
	file     *os.File  // 当前打开的文件
	basePath string    // 基础路径，如 "logs/app.log"
	dir      string    // 目录部分
	baseName string    // 不含扩展名的文件名
	ext      string    // 扩展名（含点号）
	curDate  string    // 当前文件对应的日期 "2006-01-02"
}

// NewRotatingWriter 创建日志轮转 Writer
// filePath: 日志文件路径，如 "logs/app.log"
// 自动创建目录（如果不存在）
func NewRotatingWriter(filePath string) (*RotatingWriter, error) {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// 创建日志目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory failed: %w", err)
	}

	rw := &RotatingWriter{
		basePath: filePath,
		dir:      dir,
		baseName: name,
		ext:      ext,
		curDate:  time.Now().Format("2006-01-02"),
	}

	// 打开（或创建）当前日志文件
	if err := rw.openFile(); err != nil {
		return nil, fmt.Errorf("open log file failed: %w", err)
	}

	return rw, nil
}

// Write 实现 io.Writer 接口
// 每次写入前检查日期，必要时进行轮转
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// 检查是否需要轮转
	today := time.Now().Format("2006-01-02")
	if today != rw.curDate {
		if err := rw.rotate(today); err != nil {
			// 轮转失败时仍尝试写入当前文件
			fmt.Fprintf(os.Stderr, "[logger] rotation failed: %v\n", err)
		}
	}

	if rw.file == nil {
		return 0, fmt.Errorf("log file is not open")
	}

	return rw.file.Write(p)
}

// Close 关闭当前日志文件
func (rw *RotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		err := rw.file.Close()
		rw.file = nil
		return err
	}
	return nil
}

// rotate 执行日志轮转
// 1. 关闭当前文件
// 2. 将当前文件重命名为带日期的归档文件
// 3. 创建新的当前文件
func (rw *RotatingWriter) rotate(newDate string) error {
	// 关闭当前文件
	if rw.file != nil {
		_ = rw.file.Close()
		rw.file = nil
	}

	// 将当前文件重命名为归档文件
	// app.log → app-2024-01-14.log（使用旧日期）
	archiveName := fmt.Sprintf("%s-%s%s", rw.baseName, rw.curDate, rw.ext)
	archivePath := filepath.Join(rw.dir, archiveName)

	// 检查当前文件是否存在
	if _, err := os.Stat(rw.basePath); err == nil {
		if err := os.Rename(rw.basePath, archivePath); err != nil {
			return fmt.Errorf("rename log file failed: %w", err)
		}
	}

	// 更新日期
	rw.curDate = newDate

	// 打开新文件
	return rw.openFile()
}

// openFile 打开（或创建）当前日志文件
func (rw *RotatingWriter) openFile() error {
	f, err := os.OpenFile(rw.basePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	rw.file = f
	return nil
}
