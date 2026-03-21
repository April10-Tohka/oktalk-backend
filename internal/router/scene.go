package router

import (
	"github.com/gin-gonic/gin"

	"pronunciation-correction-system/internal/handler"
)

func setupSceneRoutes(rg *gin.RouterGroup, h *handler.SceneHandler) {
	if h == nil {
		return
	}
	g := rg.Group("/scene")
	{
		g.GET("/list", h.GetSceneList)
		g.POST("/session/start", h.StartSession)
		g.POST("/session/next", h.SubmitAnswer)
		g.GET("/session/:session_id/summary", h.GetSummary)
		g.GET("/session/:session_id/history", h.GetHistory)
	}
}
