package aliyun

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"path/filepath"
	"strings"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// AliyunOSSAdapter 阿里云 OSS 适配器
// 实现 domain.OSSProvider 接口，将领域层调用转换为阿里云 OSS SDK 调用
type AliyunOSSAdapter struct {
	client    *oss.Client // 阿里云 OSS SDK 客户端
	bucket    string      // Bucket 名称
	endpoint  string      // OSS Endpoint
	region    string      // 地域
	cdnDomain string      // CDN 加速域名（可选）
}

// 编译时检查：确保 AliyunOSSAdapter 实现了 domain.OSSProvider 接口
var _ domain.OSSProvider = (*AliyunOSSAdapter)(nil)

// NewAliyunOSSAdapter 创建阿里云 OSS 适配器
// 返回 error 是因为需要初始化 SDK 客户端（可能失败）
func NewAliyunOSSAdapter(cfg config.AliyunOSSConfig) (*AliyunOSSAdapter, error) {
	// 创建凭证提供者
	credProvider := credentials.NewStaticCredentialsProvider(
		cfg.AccessKeyID,
		cfg.AccessKeySecret,
	)

	// 加载默认配置并覆盖关键参数
	ossCfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credProvider).
		WithRegion(cfg.Region)

	// 如果配置了自定义 Endpoint，则设置
	if cfg.Endpoint != "" {
		ossCfg = ossCfg.WithEndpoint(cfg.Endpoint)
	}

	// 创建 OSS 客户端
	client := oss.NewClient(ossCfg)

	logger.Info("AliyunOSS Client initialized",
		"bucket", cfg.Bucket,
		"region", cfg.Region,
		"endpoint", cfg.Endpoint,
	)

	return &AliyunOSSAdapter{
		client:    client,
		bucket:    cfg.Bucket,
		endpoint:  cfg.Endpoint,
		region:    cfg.Region,
		cdnDomain: cfg.CDNDomain,
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
	if err := a.putObject(ctx, objectKey, reader, contentType); err != nil {
		logger.ErrorContext(ctx, "upload file failed", "key", objectKey, "error", err)
		return "", fmt.Errorf("upload file failed: %w", err)
	}

	// 获取访问 URL
	url := a.getObjectURL(objectKey)

	logger.InfoContext(ctx, "file uploaded", "key", objectKey, "url", url)
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
	request := &oss.PutObjectRequest{
		Bucket:      oss.Ptr(a.bucket),
		Key:         oss.Ptr(objectKey),
		Body:        reader,
		ContentType: oss.Ptr(contentType),
	}
	_, err := a.client.PutObject(ctx, request)
	if err != nil {
		return "", fmt.Errorf("upload audio failed: %w", err)
	}
	return a.getObjectURL(objectKey), nil
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
	return a.getObjectURL(objectKey)
}

// GetSignedURL 获取签名 URL（临时访问）
// 实现 domain.OSSProvider.GetSignedURL
func (a *AliyunOSSAdapter) GetSignedURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	url, err := a.getSignedURL(ctx, objectKey, expireSeconds)
	if err != nil {
		logger.ErrorContext(ctx, "get signed url failed", "key", objectKey, "error", err)
		return "", fmt.Errorf("get signed url failed: %w", err)
	}

	logger.InfoContext(ctx, "signed url generated", "key", objectKey, "expires", expireSeconds)
	return url, nil
}

// ===================== 删除方法 =====================

// DeleteFile 删除单个文件
// 实现 domain.OSSProvider.DeleteFile
func (a *AliyunOSSAdapter) DeleteFile(ctx context.Context, objectKey string) error {
	if err := a.deleteObject(ctx, objectKey); err != nil {
		logger.ErrorContext(ctx, "delete file failed", "key", objectKey, "error", err)
		return fmt.Errorf("delete file failed: %w", err)
	}

	logger.InfoContext(ctx, "file deleted", "key", objectKey)
	return nil
}

// DeleteFiles 批量删除文件
// 实现 domain.OSSProvider.DeleteFiles
func (a *AliyunOSSAdapter) DeleteFiles(ctx context.Context, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	if err := a.deleteMultipleObjects(ctx, objectKeys); err != nil {
		logger.ErrorContext(ctx, "batch delete files failed", "count", len(objectKeys), "error", err)
		return fmt.Errorf("batch delete files failed: %w", err)
	}

	logger.InfoContext(ctx, "files batch deleted", "count", len(objectKeys))
	return nil
}

// ===================== 查询方法 =====================

// FileExists 检查文件是否存在
// 实现 domain.OSSProvider.FileExists
func (a *AliyunOSSAdapter) FileExists(ctx context.Context, objectKey string) (bool, error) {
	exists, err := a.isObjectExist(ctx, objectKey)
	if err != nil {
		logger.ErrorContext(ctx, "check file exists failed", "key", objectKey, "error", err)
		return false, fmt.Errorf("check file exists failed: %w", err)
	}
	return exists, nil
}

// GetFileInfo 获取文件元信息
// 实现 domain.OSSProvider.GetFileInfo
func (a *AliyunOSSAdapter) GetFileInfo(ctx context.Context, objectKey string) (*domain.FileInfo, error) {
	result, err := a.headObject(ctx, objectKey)
	if err != nil {
		// 如果是 404 错误，返回更友好的错误信息
		if isNotFoundError(err) {
			return nil, fmt.Errorf("file not found: %s", objectKey)
		}
		logger.ErrorContext(ctx, "get file info failed", "key", objectKey, "error", err)
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

	logger.InfoContext(ctx, "file info retrieved", "key", objectKey, "size", info.Size, "contentType", info.ContentType, "etag", info.ETag, "lastModified", info.LastModified)
	return info, nil
}

// ===================== 生命周期 =====================

// Close 关闭客户端
// 实现 domain.OSSProvider.Close
func (a *AliyunOSSAdapter) Close() error {
	return nil
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

// ===================== 上传操作 =====================

// putObject 上传对象
func (a *AliyunOSSAdapter) putObject(ctx context.Context, objectKey string, reader io.Reader, contentType string) error {
	req := &oss.PutObjectRequest{
		Bucket:      oss.Ptr(a.bucket),
		Key:         oss.Ptr(objectKey),
		Body:        reader,
		ContentType: oss.Ptr(contentType),
	}

	_, err := a.client.PutObject(ctx, req)
	if err != nil {
		return fmt.Errorf("oss put object failed: %w", err)
	}

	return nil
}

// ===================== 下载操作 =====================

// getObject 下载对象
func (a *AliyunOSSAdapter) getObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(a.bucket),
		Key:    oss.Ptr(objectKey),
	}

	result, err := a.client.GetObject(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("oss get object failed: %w", err)
	}

	return result.Body, nil
}

// ===================== 删除操作 =====================

// deleteObject 删除单个对象
func (a *AliyunOSSAdapter) deleteObject(ctx context.Context, objectKey string) error {
	req := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(a.bucket),
		Key:    oss.Ptr(objectKey),
	}

	_, err := a.client.DeleteObject(ctx, req)
	if err != nil {
		return fmt.Errorf("oss delete object failed: %w", err)
	}

	return nil
}

// deleteMultipleObjects 批量删除对象
func (a *AliyunOSSAdapter) deleteMultipleObjects(ctx context.Context, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	// 构建 DeleteObject 列表
	objects := make([]oss.DeleteObject, len(objectKeys))
	for i, key := range objectKeys {
		objects[i] = oss.DeleteObject{
			Key: oss.Ptr(key),
		}
	}

	req := &oss.DeleteMultipleObjectsRequest{
		Bucket:  oss.Ptr(a.bucket),
		Objects: objects,
		Quiet:   true, // 静默模式，只返回删除失败的对象
	}

	_, err := a.client.DeleteMultipleObjects(ctx, req)
	if err != nil {
		return fmt.Errorf("oss delete multiple objects failed: %w", err)
	}

	return nil
}

// ===================== 查询操作 =====================

// headObject 获取对象元信息
func (a *AliyunOSSAdapter) headObject(ctx context.Context, objectKey string) (*oss.HeadObjectResult, error) {
	req := &oss.HeadObjectRequest{
		Bucket: oss.Ptr(a.bucket),
		Key:    oss.Ptr(objectKey),
	}

	result, err := a.client.HeadObject(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("oss head object failed: %w", err)
	}

	return result, nil
}

// isObjectExist 检查对象是否存在
// 使用 SDK 内置的 IsObjectExist 方法
func (a *AliyunOSSAdapter) isObjectExist(ctx context.Context, objectKey string) (bool, error) {
	exists, err := a.client.IsObjectExist(ctx, a.bucket, objectKey)
	if err != nil {
		return false, fmt.Errorf("oss check object exist failed: %w", err)
	}
	return exists, nil
}

// ===================== URL 生成 =====================

// getObjectURL 获取对象的公开访问 URL
// 如果配置了 CDN 域名，优先使用 CDN；否则使用 OSS 默认域名
func (a *AliyunOSSAdapter) getObjectURL(objectKey string) string {
	if a.cdnDomain != "" {
		return fmt.Sprintf("https://%s/%s", a.cdnDomain, objectKey)
	}
	// 标准 OSS URL: https://{bucket}.{endpoint}/{key}
	return fmt.Sprintf("https://%s.%s/%s", a.bucket, a.endpoint, objectKey)
}

// getSignedURL 获取预签名 URL（临时访问）
// 使用 Presign 方法生成带签名的 GET 请求 URL
func (a *AliyunOSSAdapter) getSignedURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	// Presign 需要传入一个请求对象，这里使用 GetObjectRequest 生成 GET 签名 URL
	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(a.bucket),
		Key:    oss.Ptr(objectKey),
	}

	result, err := a.client.Presign(ctx, req, func(opts *oss.PresignOptions) {
		opts.Expires = time.Duration(expireSeconds) * time.Second
	})
	if err != nil {
		return "", fmt.Errorf("oss presign failed: %w", err)
	}

	return result.URL, nil
}
