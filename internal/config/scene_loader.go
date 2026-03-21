package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SceneLoader 从配置文件目录加载所有场景（启动时一次性读入内存）
type SceneLoader struct {
	byID     map[string]*SceneConfig
	order    []string // scene id 顺序，与按文件名字母序一致
	fileName map[string]string
}

// NewSceneLoader 遍历目录下所有 .json，解析为 SceneConfig
func NewSceneLoader(scenesDir string) (*SceneLoader, error) {
	entries, err := os.ReadDir(scenesDir)
	if err != nil {
		return nil, fmt.Errorf("read scenes dir: %w", err)
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

	l := &SceneLoader{
		byID:     make(map[string]*SceneConfig),
		fileName: make(map[string]string),
	}

	for _, fname := range jsonFiles {
		path := filepath.Join(scenesDir, fname)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read scene file %s: %w", path, err)
		}
		var cfg SceneConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse scene file %s: %w", path, err)
		}
		if cfg.ID == "" {
			return nil, fmt.Errorf("scene file %s: missing id", path)
		}
		if _, dup := l.byID[cfg.ID]; dup {
			return nil, fmt.Errorf("duplicate scene id %q", cfg.ID)
		}
		l.byID[cfg.ID] = &cfg
		l.order = append(l.order, cfg.ID)
		l.fileName[cfg.ID] = fname
	}

	return l, nil
}

// GetAll 返回所有场景列表（按文件名字母顺序）
func (l *SceneLoader) GetAll() []*SceneConfig {
	out := make([]*SceneConfig, 0, len(l.order))
	for _, id := range l.order {
		out = append(out, l.byID[id])
	}
	return out
}

// GetByID 根据 scene_id 获取单个场景配置
func (l *SceneLoader) GetByID(id string) (*SceneConfig, bool) {
	c, ok := l.byID[id]
	return c, ok
}

// GetStep 获取指定场景的指定步骤
func (l *SceneLoader) GetStep(sceneID string, stepID int) (*SceneStep, bool) {
	sc, ok := l.byID[sceneID]
	if !ok {
		return nil, false
	}
	for i := range sc.Steps {
		if sc.Steps[i].ID == stepID {
			return &sc.Steps[i], true
		}
	}
	return nil, false
}
