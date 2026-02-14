// Package tasks 定义音频上传任务
package tasks

import (
	"context"
)

// AudioUploadTask 音频上传任务
type AudioUploadTask struct {
	AudioData []byte `json:"audio_data"`
	Filename  string `json:"filename"`
	UserID    string `json:"user_id"`
}

// AudioUploadHandler 音频上传处理器
type AudioUploadHandler struct {
	// TODO: 添加服务依赖
	// ossClient  oss.Client
	// audioCache *cache.AudioCache
}

// NewAudioUploadHandler 创建音频上传处理器
func NewAudioUploadHandler() *AudioUploadHandler {
	return &AudioUploadHandler{}
}

// Handle 处理音频上传任务
func (h *AudioUploadHandler) Handle(ctx context.Context, task *AudioUploadTask) error {
	// TODO: 实现音频上传逻辑
	// 1. 验证音频格式和大小
	// 2. 生成唯一文件名
	// 3. 上传到 OSS
	// 4. 返回 CDN URL
	// 5. 更新缓存
	
	return nil
}

// AudioUploadResult 音频上传结果
type AudioUploadResult struct {
	FileKey string `json:"file_key"`
	URL     string `json:"url"`
	CDNUrl  string `json:"cdn_url"`
	Size    int64  `json:"size"`
}
