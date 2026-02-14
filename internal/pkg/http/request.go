// Package http 提供 HTTP 请求工具
package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) string {
	userID, _ := c.Get("user_id")
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

// GetRequestID 获取请求 ID
func GetRequestID(c *gin.Context) string {
	requestID, _ := c.Get("request_id")
	if id, ok := requestID.(string); ok {
		return id
	}
	return ""
}

// GetPage 获取分页参数
func GetPage(c *gin.Context) int {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

// GetPageSize 获取每页数量
func GetPageSize(c *gin.Context) int {
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 {
		return 10
	}
	if pageSize > 100 {
		return 100
	}
	return pageSize
}

// GetPagination 获取分页参数
func GetPagination(c *gin.Context) (page, pageSize int) {
	return GetPage(c), GetPageSize(c)
}

// QueryParams 查询参数辅助
type QueryParams struct {
	c *gin.Context
}

// NewQueryParams 创建查询参数辅助
func NewQueryParams(c *gin.Context) *QueryParams {
	return &QueryParams{c: c}
}

// String 获取字符串参数
func (q *QueryParams) String(key string, defaultValue string) string {
	return q.c.DefaultQuery(key, defaultValue)
}

// Int 获取整数参数
func (q *QueryParams) Int(key string, defaultValue int) int {
	value, err := strconv.Atoi(q.c.DefaultQuery(key, strconv.Itoa(defaultValue)))
	if err != nil {
		return defaultValue
	}
	return value
}

// Bool 获取布尔参数
func (q *QueryParams) Bool(key string, defaultValue bool) bool {
	value := q.c.DefaultQuery(key, "")
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1"
}

// Float64 获取浮点数参数
func (q *QueryParams) Float64(key string, defaultValue float64) float64 {
	value, err := strconv.ParseFloat(q.c.DefaultQuery(key, ""), 64)
	if err != nil {
		return defaultValue
	}
	return value
}
