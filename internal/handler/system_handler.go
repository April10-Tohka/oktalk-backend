// Package handler 提供系统状态和资源 HTTP 处理器
package handler

import "github.com/gin-gonic/gin"

// SystemHandler 系统状态 / 学习资源处理器
type SystemHandler struct{}

// NewSystemHandler 创建 SystemHandler
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// GetSystemStatus GET /api/v1/system/status
// 获取系统各服务健康状态
func (h *SystemHandler) GetSystemStatus(c *gin.Context) {
	// TODO: Step2 检查数据库、Redis 连接状态
	OK(c, map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
		"services": map[string]string{
			"database": "healthy",
			"redis":    "healthy",
			"tts":      "healthy",
			"asr":      "healthy",
			"llm":      "healthy",
		},
	})
}

// GetLearningTexts GET /api/v1/resources/texts
// 获取学习资源文本列表
func (h *SystemHandler) GetLearningTexts(c *gin.Context) {
	// TODO: Step2 从数据库查询学习资源
	// 查询参数: difficulty, category, page, page_size
	OKPage(c, []interface{}{}, 1, 20, 0)
}
