package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler/middleware"
	"pronunciation-correction-system/internal/service"
)

// SceneHandler 场景引导 HTTP 处理
type SceneHandler struct {
	sceneService *service.SceneService
}

// NewSceneHandler 构造函数
func NewSceneHandler(sceneService *service.SceneService) *SceneHandler {
	return &SceneHandler{sceneService: sceneService}
}

// GetSceneList GET /api/v1/scene/list
func (h *SceneHandler) GetSceneList(c *gin.Context) {
	list := h.sceneService.GetSceneList(c.Request.Context())
	OK(c, list)
}

// StartSession POST /api/v1/scene/session/start
func (h *SceneHandler) StartSession(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	var body struct {
		SceneID string `json:"scene_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	resp, err := h.sceneService.StartSession(c.Request.Context(), &service.SceneStartSessionRequest{
		UserID:  userID,
		SceneID: body.SceneID,
	})
	if err != nil {
		writeSceneErr(c, err)
		return
	}
	OK(c, resp)
}

// SubmitAnswer POST /api/v1/scene/session/next
func (h *SceneHandler) SubmitAnswer(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	type form struct {
		SessionID string                `form:"session_id" binding:"required"`
		StepID    string                `form:"step_id" binding:"required"`
		AudioType string                `form:"audio_type"`
		AudioFile *multipart.FileHeader `form:"audio_file" binding:"required"`
	}
	var f form
	if err := c.ShouldBind(&f); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	stepID, err := strconv.Atoi(f.StepID)
	if err != nil || stepID <= 0 {
		BadRequest(c, "step_id 无效")
		return
	}
	file, err := f.AudioFile.Open()
	if err != nil {
		InternalError(c, "读取音频失败")
		return
	}
	defer file.Close()
	audioData, err := io.ReadAll(file)
	if err != nil {
		InternalError(c, "读取音频失败")
		return
	}

	resp, err := h.sceneService.SubmitAnswer(c.Request.Context(), &service.SubmitAnswerRequest{
		UserID:    userID,
		SessionID: f.SessionID,
		StepID:    stepID,
		AudioData: audioData,
		AudioType: f.AudioType,
	})
	if err != nil {
		writeSceneErr(c, err)
		return
	}
	OK(c, resp)
}

// GetSummary GET /api/v1/scene/session/:session_id/summary
func (h *SceneHandler) GetSummary(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	sid := c.Param("session_id")
	if sid == "" {
		BadRequest(c, "session_id 无效")
		return
	}
	resp, err := h.sceneService.GetSummary(c.Request.Context(), userID, sid)
	if err != nil {
		writeSceneErr(c, err)
		return
	}
	OK(c, resp)
}

// GetHistory GET /api/v1/scene/session/:session_id/history
func (h *SceneHandler) GetHistory(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	sid := c.Param("session_id")
	if sid == "" {
		BadRequest(c, "session_id 无效")
		return
	}
	resp, err := h.sceneService.GetHistory(c.Request.Context(), userID, sid)
	if err != nil {
		writeSceneErr(c, err)
		return
	}
	OK(c, resp)
}

func getUserID(c *gin.Context) (string, bool) {
	v, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		Unauthorized(c)
		return "", false
	}
	s, _ := v.(string)
	if s == "" {
		Unauthorized(c)
		return "", false
	}
	return s, true
}

func writeSceneErr(c *gin.Context, err error) {
	var he *service.HTTPError
	if errors.As(err, &he) {
		switch he.Status {
		case http.StatusBadRequest:
			BadRequest(c, he.Message)
		case http.StatusForbidden:
			Forbidden(c)
		case http.StatusNotFound:
			NotFound(c, he.Message)
		default:
			if he.Status >= 400 && he.Status < 500 {
				Fail(c, he.Status, he.Status, he.Message)
			} else {
				InternalError(c, he.Message)
			}
		}
		return
	}
	InternalError(c, err.Error())
}
