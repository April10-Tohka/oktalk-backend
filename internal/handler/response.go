package handler

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PageData 分页响应数据
type PageData struct {
	Items      interface{} `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// OK 成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// OKPage 分页成功响应
func OKPage(c *gin.Context, items interface{}, page, pageSize int, total int64) {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	OK(c, PageData{
		Items: items,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Fail 通用失败响应
func Fail(c *gin.Context, httpCode, bizCode int, message string) {
	c.JSON(httpCode, Response{
		Code:    bizCode,
		Message: message,
		Data:    nil,
	})
}

// BadRequest 参数错误 400
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, 400, message)
}

// Unauthorized 未认证 401
func Unauthorized(c *gin.Context) {
	Fail(c, http.StatusUnauthorized, 401, "unauthorized")
}

// Forbidden 禁止访问 403
func Forbidden(c *gin.Context) {
	Fail(c, http.StatusForbidden, 403, "forbidden")
}

// NotFound 资源不存在 404
func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, 404, message)
}

// InternalError 服务器错误 500
func InternalError(c *gin.Context, message string) {
	Fail(c, http.StatusInternalServerError, 500, message)
}
