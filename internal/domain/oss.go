// Package domain 定义核心业务接口
package domain

import (
	"context"
	"io"
	"time"
)

// OSSProvider 对象存储服务接口（业务层抽象）
// 接口方法只使用 Go 原生类型，严禁出现任何第三方 SDK 结构体
type OSSProvider interface {
	// UploadFile 上传文件（支持任意文件类型）
	// 参数:
	//   - ctx: 上下文，支持超时和取消
	//   - objectKey: 对象存储路径，如 "feedback/eval_123.mp3"
	//   - reader: 文件数据流
	//   - contentType: MIME 类型，如 "audio/mpeg"；为空时根据扩展名自动推断
	// 返回:
	//   - string: 文件访问 URL
	//   - error: 错误信息
	UploadFile(ctx context.Context, objectKey string, reader io.Reader, contentType string) (string, error)

	// UploadAudio 上传音频文件（便捷方法）
	// 自动设置 Content-Type 为音频类型
	UploadAudio(ctx context.Context, objectKey string, audioData []byte) (string, error)

	// UploadBytes 上传字节数据
	UploadBytes(ctx context.Context, objectKey string, data []byte, contentType string) (string, error)

	// GetPublicURL 获取公开访问 URL（不发起网络请求，仅拼接 URL）
	GetPublicURL(objectKey string) string

	// GetSignedURL 获取签名 URL（临时访问，带过期时间）
	// expireSeconds: 过期秒数
	GetSignedURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error)

	// DeleteFile 删除单个文件
	DeleteFile(ctx context.Context, objectKey string) error

	// DeleteFiles 批量删除文件
	DeleteFiles(ctx context.Context, objectKeys []string) error

	// FileExists 检查文件是否存在
	FileExists(ctx context.Context, objectKey string) (bool, error)

	// GetFileInfo 获取文件元信息
	GetFileInfo(ctx context.Context, objectKey string) (*FileInfo, error)

	// Close 关闭客户端，释放资源
	Close() error
}

// FileInfo 文件元信息
type FileInfo struct {
	Key          string    `json:"key"`           // 对象键
	Size         int64     `json:"size"`          // 文件大小（字节）
	ContentType  string    `json:"content_type"`  // 内容类型
	LastModified time.Time `json:"last_modified"` // 最后修改时间
	ETag         string    `json:"etag"`          // ETag
}
