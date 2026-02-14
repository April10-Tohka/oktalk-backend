// Package aliyun 提供阿里云 OSS 对象存储的具体实现
// 基于阿里云 OSS Go SDK V2 (github.com/aliyun/alibabacloud-oss-go-sdk-v2)
// 此包内的 SDK 细节对外部（Service 层）不可见，
// 仅通过 AliyunOSSAdapter 实现 domain.OSSProvider 接口对外暴露
package aliyun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"

	"pronunciation-correction-system/internal/config"
)

// ===================== 客户端定义 =====================

// internalClient 阿里云 OSS 内部 SDK 客户端
// 封装了与阿里云 OSS API 的所有通信细节
type internalClient struct {
	client    *oss.Client // 阿里云 OSS SDK 客户端
	bucket    string      // Bucket 名称
	endpoint  string      // OSS Endpoint
	region    string      // 地域
	cdnDomain string      // CDN 加速域名（可选）
}

// newInternalClient 根据配置创建内部客户端
// 使用 StaticCredentialsProvider 和 LoadDefaultConfig 初始化 SDK 客户端
func newInternalClient(cfg config.AliyunOSSConfig) (*internalClient, error) {
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

	log.Printf("[AliyunOSS] Client initialized, bucket=%s, region=%s, endpoint=%s",
		cfg.Bucket, cfg.Region, cfg.Endpoint)

	return &internalClient{
		client:    client,
		bucket:    cfg.Bucket,
		endpoint:  cfg.Endpoint,
		region:    cfg.Region,
		cdnDomain: cfg.CDNDomain,
	}, nil
}

// ===================== 上传操作 =====================

// putObject 上传对象
func (c *internalClient) putObject(ctx context.Context, objectKey string, reader io.Reader, contentType string) error {
	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(objectKey),
		Body:   reader,
	}

	// 设置 Content-Type（如果提供）
	if contentType != "" {
		req.ContentType = oss.Ptr(contentType)
	}

	_, err := c.client.PutObject(ctx, req)
	if err != nil {
		return fmt.Errorf("oss put object failed: %w", err)
	}

	return nil
}

// ===================== 下载操作 =====================

// getObject 下载对象
func (c *internalClient) getObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(objectKey),
	}

	result, err := c.client.GetObject(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("oss get object failed: %w", err)
	}

	return result.Body, nil
}

// ===================== 删除操作 =====================

// deleteObject 删除单个对象
func (c *internalClient) deleteObject(ctx context.Context, objectKey string) error {
	req := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(objectKey),
	}

	_, err := c.client.DeleteObject(ctx, req)
	if err != nil {
		return fmt.Errorf("oss delete object failed: %w", err)
	}

	return nil
}

// deleteMultipleObjects 批量删除对象
func (c *internalClient) deleteMultipleObjects(ctx context.Context, objectKeys []string) error {
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
		Bucket:  oss.Ptr(c.bucket),
		Objects: objects,
		Quiet:   true, // 静默模式，只返回删除失败的对象
	}

	_, err := c.client.DeleteMultipleObjects(ctx, req)
	if err != nil {
		return fmt.Errorf("oss delete multiple objects failed: %w", err)
	}

	return nil
}

// ===================== 查询操作 =====================

// headObject 获取对象元信息
func (c *internalClient) headObject(ctx context.Context, objectKey string) (*oss.HeadObjectResult, error) {
	req := &oss.HeadObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(objectKey),
	}

	result, err := c.client.HeadObject(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("oss head object failed: %w", err)
	}

	return result, nil
}

// isObjectExist 检查对象是否存在
// 使用 SDK 内置的 IsObjectExist 方法
func (c *internalClient) isObjectExist(ctx context.Context, objectKey string) (bool, error) {
	exists, err := c.client.IsObjectExist(ctx, c.bucket, objectKey)
	if err != nil {
		return false, fmt.Errorf("oss check object exist failed: %w", err)
	}
	return exists, nil
}

// ===================== URL 生成 =====================

// getObjectURL 获取对象的公开访问 URL
// 如果配置了 CDN 域名，优先使用 CDN；否则使用 OSS 默认域名
func (c *internalClient) getObjectURL(objectKey string) string {
	if c.cdnDomain != "" {
		return fmt.Sprintf("https://%s/%s", c.cdnDomain, objectKey)
	}
	// 标准 OSS URL: https://{bucket}.{endpoint}/{key}
	return fmt.Sprintf("https://%s.%s/%s", c.bucket, c.endpoint, objectKey)
}

// getSignedURL 获取预签名 URL（临时访问）
// 使用 Presign 方法生成带签名的 GET 请求 URL
func (c *internalClient) getSignedURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	// Presign 需要传入一个请求对象，这里使用 GetObjectRequest 生成 GET 签名 URL
	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(objectKey),
	}

	result, err := c.client.Presign(ctx, req, func(opts *oss.PresignOptions) {
		opts.Expires = time.Duration(expireSeconds) * time.Second
	})
	if err != nil {
		return "", fmt.Errorf("oss presign failed: %w", err)
	}

	return result.URL, nil
}

// ===================== 工具方法 =====================

// isNotFoundError 判断是否为 404 错误（对象不存在）
func isNotFoundError(err error) bool {
	var serviceErr *oss.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.StatusCode == 404
	}
	return false
}

// close 关闭客户端
func (c *internalClient) close() error {
	log.Println("[AliyunOSS] Client closed")
	return nil
}
