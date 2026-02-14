// Package handler 提供 HTTP 请求处理
// 负责接收请求、解析参数、调用服务层、返回响应
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheck 健康检查端点
// GET /health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Service is running",
	})
}

// ReadinessCheck 就绪检查端点
// GET /ready
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// TODO: 检查数据库、Redis 等依赖服务的连接状态
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"message": "Service is ready to accept requests",
	})
}

// LivenessCheck 存活检查端点
// GET /live
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "alive",
		"message": "Service is alive",
	})
}
