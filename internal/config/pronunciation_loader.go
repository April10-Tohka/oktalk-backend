package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PronunciationLoader 启动时加载所有 Unit 配置到内存
type PronunciationLoader struct {
	byID  map[string]*PronunciationUnitConfig
	order []string
}

// NewPronunciationLoader 遍历目录下所有 .json，解析为 PronunciationUnitConfig
func NewPronunciationLoader(unitsDir string) (*PronunciationLoader, error) {
	entries, err := os.ReadDir(unitsDir)
	if err != nil {
		return nil, fmt.Errorf("read pronunciation units dir: %w", err)
	}

	var jsonFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			jsonFiles = append(jsonFiles, name)
		}
	}
	sort.Strings(jsonFiles)

	l := &PronunciationLoader{
		byID: make(map[string]*PronunciationUnitConfig),
	}

	for _, fname := range jsonFiles {
		path := filepath.Join(unitsDir, fname)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read unit file %s: %w", path, err)
		}
		var cfg PronunciationUnitConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse unit file %s: %w", path, err)
		}
		if cfg.ID == "" {
			return nil, fmt.Errorf("unit file %s: missing id", path)
		}
		if _, dup := l.byID[cfg.ID]; dup {
			return nil, fmt.Errorf("duplicate pronunciation unit id %q", cfg.ID)
		}
		l.byID[cfg.ID] = &cfg
		l.order = append(l.order, cfg.ID)
	}

	return l, nil
}

// GetAll 返回所有 Unit（按文件名字母顺序）
func (l *PronunciationLoader) GetAll() []*PronunciationUnitConfig {
	out := make([]*PronunciationUnitConfig, 0, len(l.order))
	for _, id := range l.order {
		out = append(out, l.byID[id])
	}
	return out
}

// GetByType 按 type 筛选（如 "word" / "sentence"）
func (l *PronunciationLoader) GetByType(t string) []*PronunciationUnitConfig {
	if t == "" {
		return l.GetAll()
	}
	var out []*PronunciationUnitConfig
	for _, id := range l.order {
		c := l.byID[id]
		if c.Type == t {
			out = append(out, c)
		}
	}
	return out
}

// GetByID 根据 id 获取 Unit
func (l *PronunciationLoader) GetByID(id string) (*PronunciationUnitConfig, bool) {
	c, ok := l.byID[id]
	return c, ok
}

// GetItem 获取某 Unit 下的某条 item（按 item.id 匹配）
func (l *PronunciationLoader) GetItem(unitID string, itemID int) (*PronunciationItem, bool) {
	u, ok := l.byID[unitID]
	if !ok {
		return nil, false
	}
	for i := range u.Items {
		if u.Items[i].ID == itemID {
			return &u.Items[i], true
		}
	}
	return nil, false
}
