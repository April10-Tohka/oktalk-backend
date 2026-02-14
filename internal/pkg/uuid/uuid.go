// Package uuid 提供 ID 生成工具
package uuid

import (
	"github.com/google/uuid"
)

// New 生成新的 UUID
func New() string {
	return uuid.New().String()
}

// NewWithoutDash 生成无连字符的 UUID
func NewWithoutDash() string {
	id := uuid.New()
	return id.String()[:8] + id.String()[9:13] + id.String()[14:18] + id.String()[19:23] + id.String()[24:]
}

// IsValid 验证 UUID 格式
func IsValid(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// Parse 解析 UUID
func Parse(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

// MustParse 解析 UUID（panic on error）
func MustParse(id string) uuid.UUID {
	return uuid.MustParse(id)
}
