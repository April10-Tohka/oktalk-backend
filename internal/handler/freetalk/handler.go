package freetalk

import (
	"log/slog"
	"net/http"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ===================== WebSocket Upgrader =====================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  16 * 1024, // 16KB，适合 PCM 音频块
	WriteBufferSize: 16 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发阶段允许所有来源，生产环境应限制
	},
}

// ===================== Handler 结构体 =====================

// Handler Free Talk WebSocket 处理器
type Handler struct {
	asrProvider domain.ASRProvider
	llmProvider domain.LLMProvider
	ttsProvider domain.TTSProvider
	repos       *db.Repositories
	cfg         *config.FreeTalkConfig
}

// NewHandler 创建 Free Talk Handler
func NewHandler(
	asrProvider domain.ASRProvider,
	llmProvider domain.LLMProvider,
	ttsProvider domain.TTSProvider,
	repos *db.Repositories,
	cfg *config.FreeTalkConfig,
) *Handler {
	return &Handler{
		asrProvider: asrProvider,
		llmProvider: llmProvider,
		ttsProvider: ttsProvider,
		repos:       repos,
		cfg:         cfg,
	}
}

// ===================== WebSocket 入口 =====================

// HandleWebSocket GET /api/v1/chat/freetalk
// WebSocket 升级 + 鉴权 + 启动 Session
// 查询参数:
//   - conversation_id: 会话 ID（必传）
//
// 认证方式：通过 Auth 中间件注入的 user_id
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// 1. 获取认证信息（由 Auth 中间件注入）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未认证",
		})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户信息无效",
		})
		return
	}

	// 2. 获取 conversation_id
	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 conversation_id 参数",
		})
		return
	}

	// 3. 验证会话归属（确保 conversation 属于当前用户）
	conversation, err := h.repos.VoiceConversation.GetByID(c.Request.Context(), conversationID)
	if err != nil {
		slog.Error("[FreeTalk] Get conversation failed",
			"error", err,
			"conversation_id", conversationID,
		)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "会话不存在",
		})
		return
	}
	if conversation.UserID != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "无权访问此会话",
		})
		return
	}

	// 4. 升级为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("[FreeTalk] WebSocket upgrade failed",
			"error", err,
			"user_id", userIDStr,
			"conversation_id", conversationID,
		)
		// Upgrade 失败后 Gin 不需要再写响应
		return
	}
	defer conn.Close()

	slog.Info("[FreeTalk] WebSocket connected",
		"user_id", userIDStr,
		"conversation_id", conversationID,
	)

	// 5. 更新会话状态为 active
	_ = h.repos.VoiceConversation.UpdateStatus(c.Request.Context(), conversationID, "active")

	// 6. 创建并启动 Session
	session := NewSession(
		conn,
		h.asrProvider,
		h.llmProvider,
		h.ttsProvider,
		h.repos,
		h.cfg,
		conversationID,
		userIDStr,
	)

	if err := session.Run(c.Request.Context()); err != nil {
		slog.Error("[FreeTalk] Session error",
			"error", err,
			"user_id", userIDStr,
			"conversation_id", conversationID,
		)
	}

	// 7. 会话结束，更新状态
	_ = h.repos.VoiceConversation.UpdateStatus(c.Request.Context(), conversationID, "completed")

	slog.Info("[FreeTalk] WebSocket session ended",
		"user_id", userIDStr,
		"conversation_id", conversationID,
	)
}
