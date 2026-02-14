// Package audio 提供音频处理工具
package audio

import (
	"fmt"
)

// Converter 音频格式转换器
type Converter struct{}

// NewConverter 创建转换器
func NewConverter() *Converter {
	return &Converter{}
}

// Convert 转换音频格式
func (c *Converter) Convert(data []byte, fromFormat, toFormat string) ([]byte, error) {
	// TODO: 实现音频格式转换
	// 可以使用 ffmpeg 或其他音频处理库
	return nil, fmt.Errorf("not implemented")
}

// ToPCM 转换为 PCM 格式
func (c *Converter) ToPCM(data []byte, format string) ([]byte, error) {
	return c.Convert(data, format, "pcm")
}

// ToWAV 转换为 WAV 格式
func (c *Converter) ToWAV(data []byte, format string) ([]byte, error) {
	return c.Convert(data, format, "wav")
}

// ToMP3 转换为 MP3 格式
func (c *Converter) ToMP3(data []byte, format string) ([]byte, error) {
	return c.Convert(data, format, "mp3")
}

// SupportedFormats 支持的格式
var SupportedFormats = []string{"wav", "mp3", "pcm", "m4a", "ogg", "flac"}

// IsSupportedFormat 检查是否为支持的格式
func IsSupportedFormat(format string) bool {
	for _, f := range SupportedFormats {
		if f == format {
			return true
		}
	}
	return false
}
