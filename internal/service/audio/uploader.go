// Package audio 提供音频上传服务
package audio

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"

	"pronunciation-correction-system/internal/domain"
)

// Uploader 音频上传器
type Uploader struct {
	oss domain.OSSProvider // OSS 服务（通过 domain 接口）
}

// NewUploader 创建音频上传器
func NewUploader(oss domain.OSSProvider) *Uploader {
	return &Uploader{
		oss: oss,
	}
}

// Upload 上传音频到 CDN
func (u *Uploader) Upload(ctx context.Context, data []byte, filename string) (*UploadResult, error) {
	// 1. 生成唯一文件名
	key := generateAudioKey(filename)

	// 2. 上传到 OSS（通过 domain.OSSProvider 接口）
	result, err := u.oss.UploadFile(ctx, key, bytes.NewReader(data), "audio/mpeg")
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio: %w", err)
	}

	return &UploadResult{
		FileKey: key,
		URL:     result,
		CDNUrl:  result,
		Size:    int64(len(data)),
		Format:  path.Ext(filename),
	}, nil
}

// UploadFromURL 从 URL 上传音频
func (u *Uploader) UploadFromURL(ctx context.Context, sourceURL, filename string) (*UploadResult, error) {
	// TODO: 从 URL 下载后上传
	_ = ctx
	_ = sourceURL
	_ = filename
	return nil, nil
}

// Delete 删除音频文件
func (u *Uploader) Delete(ctx context.Context, fileKey string) error {
	return u.oss.DeleteFile(ctx, fileKey)
}

// UploadResult 上传结果
type UploadResult struct {
	FileKey string `json:"file_key"`
	URL     string `json:"url"`
	CDNUrl  string `json:"cdn_url"`
	Size    int64  `json:"size"`
	Format  string `json:"format"`
}

// generateAudioKey 生成音频文件的存储路径
func generateAudioKey(filename string) string {
	date := time.Now().Format("2006/01/02")
	id := uuid.New().String()
	ext := path.Ext(filename)
	return fmt.Sprintf("audio/%s/%s%s", date, id, ext)
}
