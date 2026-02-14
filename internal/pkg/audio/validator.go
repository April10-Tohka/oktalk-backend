// Package audio 提供音频验证功能
package audio

import (
	"fmt"
)

// AudioValidator 音频验证器
type AudioValidator struct {
	maxFileSize    int64
	maxDuration    int
	allowedFormats []string
}

// NewAudioValidator 创建音频验证器
func NewAudioValidator() *AudioValidator {
	return &AudioValidator{
		maxFileSize:    10 * 1024 * 1024, // 10MB
		maxDuration:    300,               // 5 分钟
		allowedFormats: []string{"wav", "mp3", "pcm", "m4a"},
	}
}

// Validate 验证音频文件
func (v *AudioValidator) Validate(data []byte, format string) *AudioValidationResult {
	result := &AudioValidationResult{IsValid: true}

	// 检查文件大小
	if int64(len(data)) > v.maxFileSize {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("file size exceeds %d bytes", v.maxFileSize))
	}

	// 检查格式
	formatValid := false
	for _, f := range v.allowedFormats {
		if f == format {
			formatValid = true
			break
		}
	}
	if !formatValid {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported format: %s", format))
	}

	// TODO: 检查音频时长
	// TODO: 检查音频质量

	return result
}

// ValidateFormat 验证格式
func (v *AudioValidator) ValidateFormat(format string) bool {
	for _, f := range v.allowedFormats {
		if f == format {
			return true
		}
	}
	return false
}

// ValidateSize 验证大小
func (v *AudioValidator) ValidateSize(size int64) bool {
	return size <= v.maxFileSize
}

// SetMaxFileSize 设置最大文件大小
func (v *AudioValidator) SetMaxFileSize(size int64) {
	v.maxFileSize = size
}

// SetMaxDuration 设置最大时长
func (v *AudioValidator) SetMaxDuration(duration int) {
	v.maxDuration = duration
}

// AudioValidationResult 音频验证结果
type AudioValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors"`
}
