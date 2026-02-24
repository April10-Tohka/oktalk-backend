package handler

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/worker"
)

// ===================== TaskHandler =====================

// TaskHandler 任务查询处理器
type TaskHandler struct {
	manager *worker.Manager
}

// NewTaskHandler 创建 TaskHandler
func NewTaskHandler(manager *worker.Manager) *TaskHandler {
	return &TaskHandler{manager: manager}
}

// QueryTask GET /api/v1/task/:task_id
// 查询异步任务状态和结果
func (h *TaskHandler) QueryTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		BadRequest(c, "task_id is required")
		return
	}

	result, err := h.manager.QueryTask(c.Request.Context(), taskID)
	if err != nil {
		NotFound(c, err.Error())
		return
	}

	OK(c, result)
}
