package aliyun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
)

// AliyunOSSAdapter 阿里云 OSS 适配器
// 实现 domain.OSSProvider 接口，将领域层调用转换为阿里云 OSS SDK 调用
type AliyunOSSAdapter struct {
	client *internalClient
}

// 编译时检查：确保 AliyunOSSAdapter 实现了 domain.OSSProvider 接口
var _ domain.OSSProvider = (*AliyunOSSAdapter)(nil)

// NewAliyunOSSAdapter 创建阿里云 OSS 适配器
// 返回 error 是因为需要初始化 SDK 客户端（可能失败）
func NewAliyunOSSAdapter(cfg config.AliyunOSSConfig) (*AliyunOSSAdapter, error) {
	client, err := newInternalClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create oss client failed: %w", err)
	}

	return &AliyunOSSAdapter{
		client: client,
	}, nil
}

// ===================== 上传方法 =====================

// UploadFile 上传文件
// 实现 domain.OSSProvider.UploadFile
func (a *AliyunOSSAdapter) UploadFile(ctx context.Context, objectKey string, reader io.Reader, contentType string) (string, error) {
	// 自动推断 Content-Type
	if contentType == "" {
		contentType = guessContentType(objectKey)
	}

	// 上传到 OSS
	if err := a.client.putObject(ctx, objectKey, reader, contentType); err != nil {
		log.Printf("[AliyunOSS] Upload failed: key=%s, error=%v", objectKey, err)
		return "", fmt.Errorf("upload file failed: %w", err)
	}

	// 获取访问 URL
	url := a.client.getObjectURL(objectKey)

	log.Printf("[AliyunOSS] File uploaded: key=%s, url=%s", objectKey, url)
	return url, nil
}

// UploadAudio 上传音频文件（便捷方法）
// 实现 domain.OSSProvider.UploadAudio
func (a *AliyunOSSAdapter) UploadAudio(ctx context.Context, objectKey string, audioData []byte) (string, error) {
	// 根据扩展名确定 Content-Type
	contentType := "audio/mpeg" // 默认 mp3
	if strings.HasSuffix(strings.ToLower(objectKey), ".wav") {
		contentType = "audio/wav"
	} else if strings.HasSuffix(strings.ToLower(objectKey), ".ogg") {
		contentType = "audio/ogg"
	} else if strings.HasSuffix(strings.ToLower(objectKey), ".m4a") {
		contentType = "audio/mp4"
	}

	reader := bytes.NewReader(audioData)
	return a.UploadFile(ctx, objectKey, reader, contentType)
}

// UploadBytes 上传字节数据
// 实现 domain.OSSProvider.UploadBytes
func (a *AliyunOSSAdapter) UploadBytes(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)
	return a.UploadFile(ctx, objectKey, reader, contentType)
}

// ===================== URL 方法 =====================

// GetPublicURL 获取公开访问 URL（不发起网络请求）
// 实现 domain.OSSProvider.GetPublicURL
func (a *AliyunOSSAdapter) GetPublicURL(objectKey string) string {
	return a.client.getObjectURL(objectKey)
}

// GetSignedURL 获取签名 URL（临时访问）
// 实现 domain.OSSProvider.GetSignedURL
func (a *AliyunOSSAdapter) GetSignedURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	url, err := a.client.getSignedURL(ctx, objectKey, expireSeconds)
	if err != nil {
		log.Printf("[AliyunOSS] Get signed URL failed: key=%s, error=%v", objectKey, err)
		return "", fmt.Errorf("get signed url failed: %w", err)
	}

	log.Printf("[AliyunOSS] Signed URL generated: key=%s, expires=%ds", objectKey, expireSeconds)
	return url, nil
}

// ===================== 删除方法 =====================

// DeleteFile 删除单个文件
// 实现 domain.OSSProvider.DeleteFile
func (a *AliyunOSSAdapter) DeleteFile(ctx context.Context, objectKey string) error {
	if err := a.client.deleteObject(ctx, objectKey); err != nil {
		log.Printf("[AliyunOSS] Delete failed: key=%s, error=%v", objectKey, err)
		return fmt.Errorf("delete file failed: %w", err)
	}

	log.Printf("[AliyunOSS] File deleted: key=%s", objectKey)
	return nil
}

// DeleteFiles 批量删除文件
// 实现 domain.OSSProvider.DeleteFiles
func (a *AliyunOSSAdapter) DeleteFiles(ctx context.Context, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	if err := a.client.deleteMultipleObjects(ctx, objectKeys); err != nil {
		log.Printf("[AliyunOSS] Batch delete failed: count=%d, error=%v", len(objectKeys), err)
		return fmt.Errorf("batch delete files failed: %w", err)
	}

	log.Printf("[AliyunOSS] Files batch deleted: count=%d", len(objectKeys))
	return nil
}

// ===================== 查询方法 =====================

// FileExists 检查文件是否存在
// 实现 domain.OSSProvider.FileExists
func (a *AliyunOSSAdapter) FileExists(ctx context.Context, objectKey string) (bool, error) {
	exists, err := a.client.isObjectExist(ctx, objectKey)
	if err != nil {
		log.Printf("[AliyunOSS] Check file exists failed: key=%s, error=%v", objectKey, err)
		return false, fmt.Errorf("check file exists failed: %w", err)
	}
	return exists, nil
}

// GetFileInfo 获取文件元信息
// 实现 domain.OSSProvider.GetFileInfo
func (a *AliyunOSSAdapter) GetFileInfo(ctx context.Context, objectKey string) (*domain.FileInfo, error) {
	result, err := a.client.headObject(ctx, objectKey)
	if err != nil {
		// 如果是 404 错误，返回更友好的错误信息
		if isNotFoundError(err) {
			return nil, fmt.Errorf("file not found: %s", objectKey)
		}
		log.Printf("[AliyunOSS] Get file info failed: key=%s, error=%v", objectKey, err)
		return nil, fmt.Errorf("get file info failed: %w", err)
	}

	info := &domain.FileInfo{
		Key:  objectKey,
		Size: result.ContentLength,
	}

	// 安全地解引用指针字段
	if result.ContentType != nil {
		info.ContentType = *result.ContentType
	}
	if result.ETag != nil {
		info.ETag = *result.ETag
	}
	if result.LastModified != nil {
		info.LastModified = *result.LastModified
	} else {
		info.LastModified = time.Time{}
	}

	return info, nil
}

// ===================== 生命周期 =====================

// Close 关闭客户端
// 实现 domain.OSSProvider.Close
func (a *AliyunOSSAdapter) Close() error {
	return a.client.close()
}

// ===================== 工具方法 =====================

// guessContentType 根据文件扩展名推断 Content-Type
func guessContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	contentTypes := map[string]string{
		// 音频格式
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".m4a":  "audio/mp4",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		// 图片格式
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		// 文档格式
		".pdf":  "application/pdf",
		".json": "application/json",
		".txt":  "text/plain",
		".html": "text/html",
		".xml":  "application/xml",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}

	return "application/octet-stream"
}
