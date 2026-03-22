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

// PronunciationHandler 发音纠正 v2 HTTP
type PronunciationHandler struct {
	svc *service.PronunciationService
}

// NewPronunciationHandler 构造函数
func NewPronunciationHandler(svc *service.PronunciationService) *PronunciationHandler {
	return &PronunciationHandler{svc: svc}
}

// GetUnitList GET /api/v1/pronunciation/units
func (h *PronunciationHandler) GetUnitList(c *gin.Context) {
	typeFilter := c.Query("type")
	list := h.svc.GetUnitList(c.Request.Context(), typeFilter)
	OK(c, list)
}

// StartSession POST /api/v1/pronunciation/session/start
func (h *PronunciationHandler) StartSession(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	var body struct {
		UnitID string `json:"unit_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	resp, err := h.svc.StartSession(c.Request.Context(), &service.PronunciationStartSessionRequest{
		UserID: userID,
		UnitID: body.UnitID,
	})
	if err != nil {
		writePronunciationErr(c, err)
		return
	}
	OK(c, resp)
}

// Evaluate POST /api/v1/pronunciation/evaluate
func (h *PronunciationHandler) Evaluate(c *gin.Context) {
	userID, ok := pronunciationUserID(c)
	if !ok {
		return
	}
	type form struct {
		SessionID string                `form:"session_id" binding:"required"`
		ItemID    string                `form:"item_id" binding:"required"`
		AudioType string                `form:"audio_type"`
		AudioFile *multipart.FileHeader `form:"audio_file" binding:"required"`
	}
	var f form
	if err := c.ShouldBind(&f); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	itemID, err := strconv.Atoi(f.ItemID)
	if err != nil || itemID <= 0 {
		BadRequest(c, "item_id 无效")
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

	resp, err := h.svc.Evaluate(c.Request.Context(), &service.PronunciationEvaluateRequest{
		UserID:    userID,
		SessionID: f.SessionID,
		ItemID:    itemID,
		AudioData: audioData,
		AudioType: f.AudioType,
	})
	if err != nil {
		writePronunciationErr(c, err)
		return
	}
	OK(c, resp)
}

// Advance POST /api/v1/pronunciation/session/advance
func (h *PronunciationHandler) Advance(c *gin.Context) {
	userID, ok := pronunciationUserID(c)
	if !ok {
		return
	}
	var body struct {
		SessionID     string `json:"session_id" binding:"required"`
		CurrentItemID int    `json:"current_item_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	resp, err := h.svc.Advance(c.Request.Context(), &service.PronunciationAdvanceRequest{
		UserID:        userID,
		SessionID:     body.SessionID,
		CurrentItemID: body.CurrentItemID,
	})
	if err != nil {
		writePronunciationErr(c, err)
		return
	}
	OK(c, resp)
}

// GetSummary GET /api/v1/pronunciation/session/:session_id/summary
func (h *PronunciationHandler) GetSummary(c *gin.Context) {
	userID, ok := pronunciationUserID(c)
	if !ok {
		return
	}
	sid := c.Param("session_id")
	if sid == "" {
		BadRequest(c, "session_id 无效")
		return
	}
	resp, err := h.svc.GetSummary(c.Request.Context(), userID, sid)
	if err != nil {
		writePronunciationErr(c, err)
		return
	}
	OK(c, resp)
}

func pronunciationUserID(c *gin.Context) (string, bool) {
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

func writePronunciationErr(c *gin.Context, err error) {
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
