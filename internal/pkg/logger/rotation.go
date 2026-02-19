package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"pronunciation-correction-system/internal/config"
)

const dateLayout = "2006-01-02"

// RotatingWriter 按日期轮转的日志文件 Writer
// 每天自动创建新文件，旧文件按日期命名归档
//
// 文件命名示例:
//
//	logs/app.log             ← 当前日志
//	logs/app-2024-01-14.log   ← 前一天归档
//	logs/app-2024-01-14-1.log ← 同一天超过大小后的归档
type RotatingWriter struct {
	mu           sync.Mutex
	file         *os.File
	basePath     string
	dir          string
	baseName     string
	ext          string
	curDate      string
	currentIndex int
	maxSizeBytes int64
	maxBackups   int
	maxAgeDays   int
	compress     bool
}

// NewRotatingWriter 创建日志轮转 Writer
// 自动创建目录（如果不存在）
func NewRotatingWriter(cfg config.FileConfig) (*RotatingWriter, error) {
	if strings.TrimSpace(cfg.Filename) == "" {
		return nil, fmt.Errorf("log filename is empty")
	}

	dir := filepath.Dir(cfg.Filename)
	base := filepath.Base(cfg.Filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory failed: %w", err)
	}

	rw := &RotatingWriter{
		basePath:   cfg.Filename,
		dir:        dir,
		baseName:   name,
		ext:        ext,
		curDate:    time.Now().Format(dateLayout),
		maxBackups: cfg.MaxBackups,
		maxAgeDays: cfg.MaxAge,
		compress:   cfg.Compress,
	}

	if cfg.MaxSize > 0 {
		rw.maxSizeBytes = int64(cfg.MaxSize) * 1024 * 1024
	}

	if err := rw.openFile(); err != nil {
		return nil, fmt.Errorf("open log file failed: %w", err)
	}

	_ = rw.cleanupOldArchives()
	return rw, nil
}

// Write 实现 io.Writer 接口
// 每次写入前检查日期和大小，必要时进行轮转
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	today := time.Now().Format(dateLayout)
	if today != rw.curDate {
		if err := rw.rotateByDate(today); err != nil {
			fmt.Fprintf(os.Stderr, "[logger] rotation failed: %v\n", err)
		}
	}

	if rw.maxSizeBytes > 0 && rw.file != nil {
		if info, statErr := rw.file.Stat(); statErr == nil {
			if info.Size()+int64(len(p)) > rw.maxSizeBytes {
				if err := rw.rotateBySize(); err != nil {
					fmt.Fprintf(os.Stderr, "[logger] size rotation failed: %v\n", err)
				}
			}
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

// rotateByDate 执行按日期轮转
func (rw *RotatingWriter) rotateByDate(newDate string) error {
	if err := rw.rotateCurrentFile(rw.curDate, rw.currentIndex); err != nil {
		return err
	}
	rw.curDate = newDate
	rw.currentIndex = 0
	if err := rw.openFile(); err != nil {
		return err
	}
	return rw.cleanupOldArchives()
}

// rotateBySize 执行按大小轮转
func (rw *RotatingWriter) rotateBySize() error {
	rw.currentIndex++
	if err := rw.rotateCurrentFile(rw.curDate, rw.currentIndex); err != nil {
		return err
	}
	if err := rw.openFile(); err != nil {
		return err
	}
	return rw.cleanupOldArchives()
}

// rotateCurrentFile 将当前文件归档为带日期的文件
func (rw *RotatingWriter) rotateCurrentFile(date string, index int) error {
	if rw.file != nil {
		_ = rw.file.Close()
		rw.file = nil
	}

	if _, err := os.Stat(rw.basePath); err != nil {
		return nil
	}

	archivePath, err := rw.nextArchivePath(date, index)
	if err != nil {
		return err
	}

	if err := os.Rename(rw.basePath, archivePath); err != nil {
		return fmt.Errorf("rename log file failed: %w", err)
	}

	if rw.compress {
		if _, err := compressFile(archivePath); err != nil {
			fmt.Fprintf(os.Stderr, "[logger] compress log failed: %v\n", err)
		}
	}

	return nil
}

// nextArchivePath 生成不冲突的归档文件路径
func (rw *RotatingWriter) nextArchivePath(date string, index int) (string, error) {
	if date == "" {
		return "", fmt.Errorf("empty date for archive")
	}

	for i := index; ; i++ {
		var name string
		if i == 0 {
			name = fmt.Sprintf("%s-%s%s", rw.baseName, date, rw.ext)
		} else {
			name = fmt.Sprintf("%s-%s-%d%s", rw.baseName, date, i, rw.ext)
		}
		path := filepath.Join(rw.dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		}
	}
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

type archiveFile struct {
	path    string
	date    time.Time
	index   int
	modTime time.Time
}

// cleanupOldArchives 清理超期或多余的归档文件
func (rw *RotatingWriter) cleanupOldArchives() error {
	entries, err := os.ReadDir(rw.dir)
	if err != nil {
		return err
	}

	archives := make([]archiveFile, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		date, index, ok := rw.parseArchiveInfo(entry.Name())
		if !ok {
			continue
		}
		archives = append(archives, archiveFile{
			path:    filepath.Join(rw.dir, entry.Name()),
			date:    date,
			index:   index,
			modTime: info.ModTime(),
		})
	}

	if rw.maxAgeDays > 0 {
		threshold := time.Now().AddDate(0, 0, -rw.maxAgeDays)
		remaining := make([]archiveFile, 0, len(archives))
		for _, file := range archives {
			if file.date.Before(threshold) {
				_ = os.Remove(file.path)
				continue
			}
			remaining = append(remaining, file)
		}
		archives = remaining
	}

	if rw.maxBackups > 0 && len(archives) > rw.maxBackups {
		sort.Slice(archives, func(i, j int) bool {
			if archives[i].date.Equal(archives[j].date) {
				if archives[i].index == archives[j].index {
					return archives[i].modTime.Before(archives[j].modTime)
				}
				return archives[i].index < archives[j].index
			}
			return archives[i].date.Before(archives[j].date)
		})

		removeCount := len(archives) - rw.maxBackups
		for i := 0; i < removeCount; i++ {
			_ = os.Remove(archives[i].path)
		}
	}

	return nil
}

// parseArchiveInfo 解析归档文件名中的日期与序号
func (rw *RotatingWriter) parseArchiveInfo(name string) (time.Time, int, bool) {
	if !strings.HasPrefix(name, rw.baseName+"-") {
		return time.Time{}, 0, false
	}

	trimmed := strings.TrimPrefix(name, rw.baseName+"-")
	if strings.HasSuffix(trimmed, rw.ext+".gz") {
		trimmed = strings.TrimSuffix(trimmed, rw.ext+".gz")
	} else if strings.HasSuffix(trimmed, rw.ext) {
		trimmed = strings.TrimSuffix(trimmed, rw.ext)
	} else {
		return time.Time{}, 0, false
	}

	parts := strings.Split(trimmed, "-")
	if len(parts) < 3 {
		return time.Time{}, 0, false
	}

	dateStr := strings.Join(parts[:3], "-")
	date, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return time.Time{}, 0, false
	}

	index := 0
	if len(parts) > 3 {
		value, err := strconv.Atoi(parts[3])
		if err != nil {
			return time.Time{}, 0, false
		}
		index = value
	}

	return date, index, true
}

// compressFile 压缩日志文件为 gzip
func compressFile(path string) (string, error) {
	source, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()

	targetPath := path + ".gz"
	target, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}

	gz := gzip.NewWriter(target)
	if _, err := io.Copy(gz, source); err != nil {
		_ = gz.Close()
		_ = target.Close()
		return "", err
	}
	if err := gz.Close(); err != nil {
		_ = target.Close()
		return "", err
	}
	if err := target.Close(); err != nil {
		return "", err
	}

	if err := os.Remove(path); err != nil {
		return "", err
	}
	return targetPath, nil
}
