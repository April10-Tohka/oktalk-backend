package freetalk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/model"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ===================== 状态机 =====================

type sessionState int

const (
	stateIdle       sessionState = iota // 接收并转发用户音频
	stateAISpeaking                     // AI 回复中，丢弃用户音频
)

// ===================== Session 结构体 =====================

// Session 管理单个 Free Talk WebSocket 会话的完整生命周期
type Session struct {
	appConn *websocket.Conn

	// 注入的 domain 接口（由 Handler 创建后传入）
	asrProvider domain.ASRProvider
	llmProvider domain.LLMProvider
	ttsProvider domain.TTSProvider

	// ASR 长连接（会话级，整个 free talk 期间不断开）
	audioSender domain.AudioSender

	// TTS 长连接（会话级）
	ttsStreamer domain.TTSStreamer

	// 状态机
	state   sessionState
	stateMu sync.Mutex

	// 所有 goroutine 通过此 channel 向 App 写数据，串行化写操作
	writeChan chan wsMessage

	// 会话信息
	conversationID string
	userID         string
	repos          *db.Repositories
	cfg            *config.FreeTalkConfig

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSession 创建一个 Free Talk 会话
func NewSession(
	appConn *websocket.Conn,
	asrProvider domain.ASRProvider,
	llmProvider domain.LLMProvider,
	ttsProvider domain.TTSProvider,
	repos *db.Repositories,
	cfg *config.FreeTalkConfig,
	conversationID string,
	userID string,
) *Session {
	return &Session{
		appConn:        appConn,
		asrProvider:    asrProvider,
		llmProvider:    llmProvider,
		ttsProvider:    ttsProvider,
		repos:          repos,
		cfg:            cfg,
		conversationID: conversationID,
		userID:         userID,
		writeChan:      make(chan wsMessage, 256),
	}
}

// ===================== 状态机方法 =====================

func (s *Session) setState(state sessionState) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.state = state
}

func (s *Session) getState() sessionState {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.state
}

// ===================== Run: 启动会话 =====================

// Run 启动会话，阻塞直到会话结束
// 内部启动以下 goroutine（全部用 go func() 直接启动，不使用 Worker Pool）：
//  1. writerGoroutine：唯一负责写 appConn 的 goroutine
//  2. appReaderGoroutine：持续读取 App 发来的帧
//  3. asrReaderGoroutine：持续读取 ASR 事件
//  4. ttsReaderGoroutine：持续读取 TTS 音频
func (s *Session) Run(ctx context.Context) error {
	// s.ctx, s.cancel = context.WithCancel(ctx)
	// defer s.cancel()

	// // 1. 建立 ASR 长连接
	// asrFormat := s.cfg.ASRFormat
	// if asrFormat == "" {
	// 	asrFormat = "pcm"
	// }
	// asrSampleRate := s.cfg.ASRSampleRate
	// if asrSampleRate == 0 {
	// 	asrSampleRate = 16000
	// }

	//  err := s.asrProvider.ConnectASR(s.ctx, asrFormat, asrSampleRate)
	// if err != nil {
	// 	return fmt.Errorf("connect ASR failed: %w", err)
	// }
	// s.audioSender = audioSender
	// defer func() {
	// 	if s.audioSender != nil {
	// 		_ = s.audioSender.Close()
	// 	}
	// }()

	// // 2. 建立 TTS 长连接
	// ttsOpts := &domain.SynthesizeOptions{
	// 	Voice:      s.cfg.TTSVoice,
	// 	Format:     s.cfg.TTSFormat,
	// 	SampleRate: s.cfg.TTSSampleRate,
	// }
	// ttsStreamer, err := s.ttsProvider.ConnectTTS(s.ctx, ttsOpts)
	// if err != nil {
	// 	return fmt.Errorf("connect TTS failed: %w", err)
	// }
	// s.ttsStreamer = ttsStreamer
	// defer func() {
	// 	if s.ttsStreamer != nil {
	// 		_ = s.ttsStreamer.Close()
	// 	}
	// }()

	// // 3. WaitGroup 用于等待所有 goroutine 结束
	// var wg sync.WaitGroup

	// // ① writerGoroutine（最先启动）
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	s.writerGoroutine()
	// }()

	// // ② appReaderGoroutine
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	s.appReaderGoroutine()
	// }()

	// // ③ asrReaderGoroutine
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	s.asrReaderGoroutine(asrEventCh)
	// }()

	// // ④ ttsReaderGoroutine
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	s.ttsReaderGoroutine()
	// }()

	// // 阻塞等待 ctx 取消
	// <-s.ctx.Done()

	// // 关闭 writeChan 以终止 writerGoroutine
	// close(s.writeChan)

	// // 等待所有 goroutine 退出
	// wg.Wait()

	// slog.Info("[FreeTalk] Session ended",
	// 	"conversation_id", s.conversationID,
	// 	"user_id", s.userID,
	// )

	return nil
}

// ===================== ① writerGoroutine =====================

// writerGoroutine 唯一负责写 appConn 的 goroutine
// 从 writeChan 取消息 → appConn.WriteMessage
// writeChan 关闭或 ctx 取消时退出
func (s *Session) writerGoroutine() {
	for msg := range s.writeChan {
		if err := s.appConn.WriteMessage(msg.messageType, msg.data); err != nil {
			slog.Error("[FreeTalk] Write to app failed",
				"error", err,
				"conversation_id", s.conversationID,
			)
			s.cancel()
			return
		}
	}
}

// ===================== ② appReaderGoroutine =====================

// appReaderGoroutine 持续读取 App 发来的帧
// Text Frame → 解析 IncomingMessage（start/stop 控制指令）
// Binary Frame（PCM）→ 根据状态转发到 ASR 或丢弃
func (s *Session) appReaderGoroutine() {
	defer s.cancel()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		messageType, data, err := s.appConn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Info("[FreeTalk] App disconnected normally",
					"conversation_id", s.conversationID,
				)
				return
			}
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			slog.Error("[FreeTalk] Read from app failed",
				"error", err,
				"conversation_id", s.conversationID,
			)
			return
		}

		switch messageType {
		case websocket.TextMessage:
			s.handleTextFrame(data)

		case websocket.BinaryMessage:
			s.handleBinaryFrame(data)

		default:
			slog.Warn("[FreeTalk] Unexpected message type",
				"type", messageType,
				"conversation_id", s.conversationID,
			)
		}
	}
}

// handleTextFrame 处理 App 发来的文本帧
func (s *Session) handleTextFrame(data []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Warn("[FreeTalk] Parse text frame failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
		return
	}

	switch msg.Type {
	case "stop":
		slog.Info("[FreeTalk] Received stop command",
			"conversation_id", s.conversationID,
		)
		s.cancel()

	default:
		slog.Warn("[FreeTalk] Unknown text frame type",
			"type", msg.Type,
			"conversation_id", s.conversationID,
		)
	}
}

// handleBinaryFrame 处理 App 发来的二进制帧（PCM 音频）
func (s *Session) handleBinaryFrame(data []byte) {
	currentState := s.getState()

	switch currentState {
	case stateIdle:
		// 转发音频到 ASR
		if s.audioSender != nil {
			if err := s.audioSender.SendAudio(data); err != nil {
				slog.Error("[FreeTalk] Send audio to ASR failed",
					"error", err,
					"conversation_id", s.conversationID,
				)
			}
		}

	case stateAISpeaking:
		// AI 正在说话，丢弃用户音频
	}
}

// ===================== ③ asrReaderGoroutine =====================

// asrReaderGoroutine 持续读取 ASR 事件
// sentence_end=true → 切换状态为 AI 说话，启动 handleTurn goroutine
func (s *Session) asrReaderGoroutine(asrEventCh <-chan *domain.ASRStreamEvent) {
	for {
		select {
		case <-s.ctx.Done():
			return

		case event, ok := <-asrEventCh:
			if !ok {
				slog.Info("[FreeTalk] ASR event channel closed",
					"conversation_id", s.conversationID,
				)
				return
			}

			if event.Error != nil {
				slog.Error("[FreeTalk] ASR error",
					"error", event.Error,
					"conversation_id", s.conversationID,
				)
				s.sendError("asr_error", event.Error.Error())
				continue
			}

			switch event.Type {
			case "partial":
				// 推送中间识别结果给 App（可选，用于实时展示）
				s.sendTextMessage(MsgTypeASRPartial, event.Text)

			case "final":
				// 推送最终识别文本给 App
				s.sendTextMessage(MsgTypeASRText, event.Text)

				// 跳过空文本
				if strings.TrimSpace(event.Text) == "" {
					continue
				}

				// 切换为 AI 说话状态
				s.setState(stateAISpeaking)

				// 启动 handleTurn goroutine 处理本轮对话
				go s.handleTurn(event.Text)
			}
		}
	}
}

// ===================== ④ ttsReaderGoroutine =====================

// ttsReaderGoroutine 持续读取 TTS 音频并推送给 App
func (s *Session) ttsReaderGoroutine() {
	audioCh := s.ttsStreamer.AudioChan()
	for {
		select {
		case <-s.ctx.Done():
			return

		case audioData, ok := <-audioCh:
			if !ok {
				slog.Info("[FreeTalk] TTS audio channel closed",
					"conversation_id", s.conversationID,
				)
				return
			}

			// 推送 TTS 音频给 App（Binary Frame）
			select {
			case s.writeChan <- wsMessage{
				messageType: websocket.BinaryMessage,
				data:        audioData,
			}:
			case <-s.ctx.Done():
				return
			}
		}
	}
}

// ===================== handleTurn: 处理一轮对话 =====================

// handleTurn 处理一轮对话：LLM 流式生成 + TTS 流式合成
// 由 asrReaderGoroutine 在检测到 sentence_end 时启动
func (s *Session) handleTurn(userText string) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		slog.Error("[FreeTalk] handleTurn panic recovered",
	// 			"error", fmt.Sprintf("%v", r),
	// 			"conversation_id", s.conversationID,
	// 		)
	// 	}
	// }()

	// slog.Info("[FreeTalk] New turn started",
	// 	"user_text", userText,
	// 	"conversation_id", s.conversationID,
	// )

	// // 1. 启动 TTS 新任务
	// if err := s.ttsStreamer.RunTask(s.ctx); err != nil {
	// 	slog.Error("[FreeTalk] TTS RunTask failed",
	// 		"error", err,
	// 		"conversation_id", s.conversationID,
	// 	)
	// 	s.sendError("tts_error", "TTS 任务启动失败")
	// 	s.setState(stateIdle)
	// 	s.sendTextMessage(MsgTypeTurnEnd, "")
	// 	return
	// }

	// // 2. 构建 LLM 消息历史
	// messages := s.buildLLMMessages(userText)

	// // 3. 流式调用 LLM
	// var aiText strings.Builder
	// var accumulatedText strings.Builder
	// textFlushLen := s.cfg.TextFlushLen
	// if textFlushLen <= 0 {
	// 	textFlushLen = 20 // 默认积累 20 字符后推送 TTS
	// }

	// err := s.llmProvider.ChatStream(s.ctx, messages, func(token string) {
	// 	// 推送每个 token 给 App
	// 	s.sendTextMessage(MsgTypeLLMToken, token)

	// 	aiText.WriteString(token)
	// 	accumulatedText.WriteString(token)

	// 	// 积累足够文本后推送到 TTS
	// 	if accumulatedText.Len() >= textFlushLen {
	// 		if feedErr := s.ttsStreamer.FeedText(s.ctx, accumulatedText.String()); feedErr != nil {
	// 			slog.Error("[FreeTalk] TTS FeedText failed",
	// 				"error", feedErr,
	// 				"conversation_id", s.conversationID,
	// 			)
	// 		}
	// 		accumulatedText.Reset()
	// 	}
	// })

	// if err != nil {
	// 	slog.Error("[FreeTalk] LLM ChatStream failed",
	// 		"error", err,
	// 		"conversation_id", s.conversationID,
	// 	)
	// 	s.sendError("llm_error", "AI 回复生成失败")
	// 	// 仍然需要结束 TTS 任务
	// 	_ = s.ttsStreamer.FinishTask(s.ctx)
	// 	s.setState(stateIdle)
	// 	s.sendTextMessage(MsgTypeTurnEnd, "")
	// 	return
	// }

	// // 4. 推送剩余文本到 TTS
	// if accumulatedText.Len() > 0 {
	// 	if err := s.ttsStreamer.FeedText(s.ctx, accumulatedText.String()); err != nil {
	// 		slog.Error("[FreeTalk] TTS FeedText remaining failed",
	// 			"error", err,
	// 			"conversation_id", s.conversationID,
	// 		)
	// 	}
	// }

	// // 5. 通知 TTS 文本发送完毕
	// if err := s.ttsStreamer.FinishTask(s.ctx); err != nil {
	// 	slog.Error("[FreeTalk] TTS FinishTask failed",
	// 		"error", err,
	// 		"conversation_id", s.conversationID,
	// 	)
	// }

	// // 6. 等待 TTS 播放完成
	// select {
	// case <-s.ttsStreamer.TaskDone():
	// 	slog.Info("[FreeTalk] TTS task done",
	// 		"conversation_id", s.conversationID,
	// 	)
	// case <-s.ctx.Done():
	// 	return
	// case <-time.After(120 * time.Second):
	// 	slog.Warn("[FreeTalk] Wait TTS task done timeout",
	// 		"conversation_id", s.conversationID,
	// 	)
	// }

	// // 7. 发送 turn_end 给 App
	// s.sendTextMessage(MsgTypeTurnEnd, "")

	// // 8. 切换回 idle 状态
	// s.setState(stateIdle)

	// // 9. 异步保存对话消息到数据库
	// go s.saveMessages(userText, aiText.String())

	// slog.Info("[FreeTalk] Turn completed",
	// 	"user_text_len", len(userText),
	// 	"ai_text_len", aiText.Len(),
	// 	"conversation_id", s.conversationID,
	// )
}

// ===================== 辅助方法 =====================

// buildLLMMessages 构建 LLM 对话消息列表
func (s *Session) buildLLMMessages(userText string) []domain.ChatMessage {
	messages := make([]domain.ChatMessage, 0, 32)

	// 系统提示词
	systemPrompt := s.cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a friendly English tutor for children. " +
			"Speak in simple, encouraging English. " +
			"Help the child practice spoken English through natural conversation. " +
			"Keep responses short and age-appropriate."
	}
	messages = append(messages, domain.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// 从数据库加载历史对话（最近 N 轮）
	maxHistory := s.cfg.MaxTurnHistory
	if maxHistory <= 0 {
		maxHistory = 10
	}

	if s.repos != nil && s.conversationID != "" {
		historyMsgs, err := s.repos.ConversationMessage.GetByConversationID(s.ctx, s.conversationID)
		if err != nil {
			slog.Warn("[FreeTalk] Load conversation history failed",
				"error", err,
				"conversation_id", s.conversationID,
			)
		} else {
			// 取最近 maxHistory*2 条消息（每轮含 user + ai）
			startIdx := 0
			if len(historyMsgs) > maxHistory*2 {
				startIdx = len(historyMsgs) - maxHistory*2
			}
			for _, msg := range historyMsgs[startIdx:] {
				role := "user"
				if msg.SenderType == "ai" {
					role = "assistant"
				}
				messages = append(messages, domain.ChatMessage{
					Role:    role,
					Content: msg.MessageText,
				})
			}
		}
	}

	// 当前用户输入
	messages = append(messages, domain.ChatMessage{
		Role:    "user",
		Content: userText,
	})

	return messages
}

// sendTextMessage 发送文本消息给 App
func (s *Session) sendTextMessage(msgType, text string) {
	msg := OutgoingMessage{
		Type: msgType,
		Text: text,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("[FreeTalk] Marshal outgoing message failed",
			"error", err,
			"type", msgType,
		)
		return
	}

	select {
	case s.writeChan <- wsMessage{
		messageType: websocket.TextMessage,
		data:        data,
	}:
	case <-s.ctx.Done():
	}
}

// sendError 发送错误消息给 App
func (s *Session) sendError(code, message string) {
	msg := OutgoingMessage{
		Type:    MsgTypeError,
		Code:    code,
		Message: message,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case s.writeChan <- wsMessage{
		messageType: websocket.TextMessage,
		data:        data,
	}:
	case <-s.ctx.Done():
	}
}

// saveMessages 异步保存对话消息到数据库（在独立 goroutine 中执行）
func (s *Session) saveMessages(userText, aiText string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[FreeTalk] saveMessages panic recovered",
				"error", fmt.Sprintf("%v", r),
				"conversation_id", s.conversationID,
			)
		}
	}()

	if s.repos == nil || s.conversationID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取当前最大轮次
	maxTurn, err := s.repos.ConversationMessage.GetMaxTurnID(ctx, s.conversationID)
	if err != nil {
		slog.Error("[FreeTalk] Get max turn ID failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
		maxTurn = 0
	}
	nextTurn := maxTurn + 1

	// 获取下一个序号
	nextSeq, err := s.repos.ConversationMessage.GetNextSequenceNumber(ctx, s.conversationID)
	if err != nil {
		slog.Error("[FreeTalk] Get next sequence number failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
		nextSeq = 1
	}

	// 创建用户消息
	userMsg := &model.ConversationMessage{
		ID:             uuid.New().String(),
		ConversationID: s.conversationID,
		SenderType:     "user",
		TurnID:         nextTurn,
		MessageText:    userText,
		SequenceNumber: nextSeq,
	}
	if err := s.repos.ConversationMessage.Create(ctx, userMsg); err != nil {
		slog.Error("[FreeTalk] Save user message failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
	}

	// 创建 AI 消息
	aiMsg := &model.ConversationMessage{
		ID:             uuid.New().String(),
		ConversationID: s.conversationID,
		SenderType:     "ai",
		TurnID:         nextTurn,
		MessageText:    aiText,
		SequenceNumber: nextSeq + 1,
	}
	if err := s.repos.ConversationMessage.Create(ctx, aiMsg); err != nil {
		slog.Error("[FreeTalk] Save AI message failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
	}

	// 更新对话消息数
	_ = s.repos.VoiceConversation.IncrementMessageCount(ctx, s.conversationID)
	_ = s.repos.VoiceConversation.IncrementMessageCount(ctx, s.conversationID)

	slog.Info("[FreeTalk] Messages saved",
		"conversation_id", s.conversationID,
		"turn", nextTurn,
	)
}
