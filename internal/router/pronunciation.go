package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

func setupPronunciationRoutes(rg *gin.RouterGroup, h *handler.PronunciationHandler) {
	if h == nil {
		return
	}
	g := rg.Group("/pronunciation")
	{
		g.GET("/units", h.GetUnitList)
		g.POST("/session/start", h.StartSession)
		g.POST("/evaluate", h.Evaluate)
		g.POST("/session/advance", h.Advance)
		g.GET("/session/:session_id/summary", h.GetSummary)
	}
}
