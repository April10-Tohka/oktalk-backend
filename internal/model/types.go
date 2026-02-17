// Package model 定义数据库 JSON 字段的自定义类型
// 这些类型实现了 sql.Scanner 和 driver.Valuer 接口，
// 用于 GORM 自动序列化/反序列化 JSON 字段
package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// ========== StringArray ==========

// StringArray 字符串数组（用于 JSON 列的序列化/反序列化）
// 使用场景：problem_words, strengths, weaknesses, most_practiced_topics 等
type StringArray []string

// Scan 实现 sql.Scanner 接口，从数据库读取 JSON 数据
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("StringArray.Scan: failed to convert value to []byte")
	}
	return json.Unmarshal(bytes, sa)
}

// Value 实现 driver.Valuer 接口，写入数据库时序列化为 JSON
func (sa StringArray) Value() (driver.Value, error) {
	if sa == nil {
		return nil, nil
	}
	return json.Marshal(sa)
}

// ========== StringMap ==========

// StringMap 字符串键值对（用于 JSON 列的序列化/反序列化）
// 使用场景：problem_word_audio_urls ({"apple": "url1", "orange": "url2"})
type StringMap map[string]string

// Scan 实现 sql.Scanner 接口，从数据库读取 JSON 数据
func (sm *StringMap) Scan(value interface{}) error {
	if value == nil {
		*sm = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("StringMap.Scan: failed to convert value to []byte")
	}
	return json.Unmarshal(bytes, sm)
}

// Value 实现 driver.Valuer 接口，写入数据库时序列化为 JSON
func (sm StringMap) Value() (driver.Value, error) {
	if sm == nil {
		return nil, nil
	}
	return json.Marshal(sm)
}
