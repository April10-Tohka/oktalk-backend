package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/infrastructure/vad/vadpb"
	"pronunciation-correction-system/internal/pkg/uuid"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// ===================== Text Frame 类型常量 =====================

const (
	MsgTypeLLMToken  = "llm_token" // LLM 流式 token（后端 → App）
	MsgTypeTurnEnd   = "turn_end"  // 本轮 AI 回复结束（后端 → App）
	MsgTypeError     = "error"     // 错误通知（后端 → App）
	MsgTypeListening = "listening" // VAD 检测到用户开始说话（后端 → App）
	MsgTypeThinking  = "thinking"  // VAD 检测到用户说话结束，后端开始处理（后端 → App）
)

// ===================== 后端 → App（Text Frame）=====================

// OutgoingMessage 后端推给 App 的 Text Frame 结构
type OutgoingMessage struct {
	// Type 消息类型（见上方常量）
	Type string `json:"type"`

	// Text 文本内容
	// - llm_token: LLM 生成的增量 token
	Text string `json:"text,omitempty"`

	// Code 错误码（仅 error 时使用）
	Code string `json:"code,omitempty"`

	// Message 错误信息（仅 error 时使用）
	Message string `json:"message,omitempty"`
}

// ===================== Binary Frame =====================
// Binary Frame 携带 PCM 裸音频数据，无任何包装：
// - App → 后端：用户录音 PCM，16kHz 单声道 16bit
// - 后端 → App：TTS 合成音频（格式由配置决定，默认 PCM）

// ===================== WebSocket 内部消息类型 =====================

// wsMessage 内部 WebSocket 消息封装
// 用于 writerGoroutine 串行化所有写操作
type wsMessage struct {
	// messageType websocket.TextMessage 或 websocket.BinaryMessage
	messageType int

	// data 消息内容（JSON 或二进制音频）
	data []byte
}

// ===================== Session 结构体 =====================

// Session 管理单个 Free Talk WebSocket 会话的完整生命周期
type Session struct {
	appConn *websocket.Conn

	// 注入的 domain 接口（由 Handler 创建后传入）
	asrProvider domain.ASRProvider
	llmProvider domain.LLMProvider
	ttsProvider domain.TTSProvider

	// VAD gRPC 客户端
	vadClient vadpb.VADServiceClient

	// 会话信息
	conversationID string
	userID         string

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

func NewSession(
	appConn *websocket.Conn,
	asrProvider domain.ASRProvider,
	llmProvider domain.LLMProvider,
	ttsProvider domain.TTSProvider,
	conversationID string,
	userID string,
	vadClient vadpb.VADServiceClient,
) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		appConn:        appConn,
		asrProvider:    asrProvider,
		llmProvider:    llmProvider,
		ttsProvider:    ttsProvider,
		conversationID: conversationID,
		userID:         userID,
		vadClient:      vadClient,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// buildUserProfile 从 Session 信息构建用户画像
// P0 阶段返回默认值；P2 阶段可改为从 Redis 加载持久化画像
func (s *Session) buildUserProfile() UserProfile {
	return UserProfile{
		// TODO P2: 从用户服务加载真实姓名和年龄
		Name:     "",
		AgeGroup: "6-12", // 默认全段，后续细化
	}
}

func (s *Session) Run() error {
	rawAudioChan := make(chan []byte, 1024)           // App音频 → vadSendGoroutine
	asrAudioChan := make(chan []byte, 8)              // vadRecvGoroutine → asrGoroutine（完整句子PCM）
	llmInputChan := make(chan string, 1024)           // ASR goroutine → LLM goroutine（触发LLM，携带识别文本）
	llmOutputChan := make(chan domain.LLMChunk, 1024) // LLM goroutine → TTS goroutine（流式token投喂TTS）
	writeChan := make(chan wsMessage, 1024)           // 所有goroutine → Writer goroutine（统一写App WebSocket）
	ttsNewTurnChan := make(chan struct{}, 1024)       // LLM goroutine → TTS goroutine（触发新一轮TTS任务）

	// 建立 VAD gRPC 双向流
	vadStream, err := s.vadClient.StreamingVAD(s.ctx)
	if err != nil {
		slog.Error("[FreeTalk] Connect VAD gRPC stream failed",
			"error", err,
		)
		return err
	}

	go s.readerGoroutine(rawAudioChan)
	go s.writerGoroutine(writeChan)
	go s.vadSendGoroutine(vadStream, rawAudioChan)
	go s.vadRecvGoroutine(vadStream, asrAudioChan, writeChan)
	go s.asrGoroutine(asrAudioChan, llmInputChan)
	go s.llmGoroutine(llmInputChan, llmOutputChan, ttsNewTurnChan, writeChan)
	go s.ttsGoroutine(llmOutputChan, ttsNewTurnChan, writeChan)

	<-s.ctx.Done()
	slog.Info("[FreeTalk] Session ending, closing channels")
	// 统一关闭所有 channel，让各 goroutine 的 range 循环自然退出
	// 顺序：按数据流方向，从源头到末端
	close(rawAudioChan)
	close(asrAudioChan)
	close(llmInputChan)
	close(llmOutputChan)
	close(ttsNewTurnChan)
	close(writeChan)
	return nil
}

// ===================== ① writerGoroutine =====================

// writerGoroutine 唯一负责写 appConn 的 goroutine
// 从 writeChan 取消息 → appConn.WriteMessage
// writeChan 关闭或 ctx 取消时退出
func (s *Session) writerGoroutine(writeChan <-chan wsMessage) {
	for msg := range writeChan {
		if err := s.appConn.WriteMessage(msg.messageType, msg.data); err != nil {
			slog.Error("[FreeTalk] Write to app failed",
				"error", err,
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
func (s *Session) readerGoroutine(rawAudioChan chan<- []byte) {
	defer s.cancel()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		slog.Debug("[FreeTalk-Reader] Waiting for app message")
		messageType, data, err := s.appConn.ReadMessage()
		if websocket.IsCloseError(err,
			websocket.CloseNormalClosure,    // 1000
			websocket.CloseGoingAway,        // 1001
			websocket.CloseNoStatusReceived, // 1005  ← 重要
			websocket.CloseAbnormalClosure,  // 1006
		) {
			slog.Info("[FreeTalk-Reader] App closed the connection normally")
			return
		}
		switch messageType {
		case websocket.TextMessage:
			slog.Warn("[FreeTalk-Reader] Unexpected text frame from app, ignoring",
				"data", string(data),
			)

		case websocket.BinaryMessage:
			select {
			case <-s.ctx.Done():
				return
			case rawAudioChan <- data:
			}

		default:
			slog.Warn("[FreeTalk-Reader] Unexpected message type",
				"type", messageType,
				"conversation_id", s.conversationID,
			)
		}
	}
}

// ===================== ② VAD Goroutines =====================

// vadSendGoroutine 从 rawAudioChan 读取原始 PCM 帧并转发给 VAD gRPC 服务
func (s *Session) vadSendGoroutine(stream vadpb.VADService_StreamingVADClient, rawAudioChan <-chan []byte) {
	defer stream.CloseSend()
	for {
		select {
		case <-s.ctx.Done():
			return
		case audio, ok := <-rawAudioChan:
			if !ok {
				return
			}
			if err := stream.Send(&vadpb.AudioChunk{
				PcmData:    audio,
				SampleRate: 16000,
			}); err != nil {
				slog.Error("[FreeTalk-VADSend] VAD stream send failed",
					"error", err,
					"conversation_id", s.conversationID,
				)
				s.cancel()
				return
			}
		}
	}
}

// vadRecvGoroutine 从 VAD gRPC 服务接收事件，分发到 writeChan / asrAudioChan
func (s *Session) vadRecvGoroutine(stream vadpb.VADService_StreamingVADClient, asrAudioChan chan<- []byte, writeChan chan<- wsMessage) {
	for {
		event, err := stream.Recv()
		if err != nil {
			if err == io.EOF || s.ctx.Err() != nil {
				// 正常退出：流关闭或 ctx 已取消
				return
			}
			slog.Error("[FreeTalk-VADRecv] VAD stream recv failed",
				"error", err,
				"conversation_id", s.conversationID,
			)
			s.cancel()
			return
		}

		switch event.Type {
		case vadpb.VADEvent_SPEECH_START:
			slog.Debug("[FreeTalk-VADRecv] User started speaking")
			// 通知 App 前端：用户开始说话
			msg := OutgoingMessage{Type: MsgTypeListening}
			data, _ := json.Marshal(msg)
			select {
			case writeChan <- wsMessage{messageType: websocket.TextMessage, data: data}:
			case <-s.ctx.Done():
				return
			}

		case vadpb.VADEvent_SPEECH_END:
			slog.Debug("[FreeTalk-VADRecv] User stopped speaking")
			// 通知 App：后端已收到语音，开始 ASR→LLM→TTS 处理
			thinkingMsg := OutgoingMessage{Type: MsgTypeThinking}
			thinkingData, _ := json.Marshal(thinkingMsg)
			select {
			case writeChan <- wsMessage{messageType: websocket.TextMessage, data: thinkingData}:
			case <-s.ctx.Done():
				return
			}
			// 把完整句子 PCM 送给 ASR goroutine
			if len(event.AudioData) > 0 {
				select {
				case asrAudioChan <- event.AudioData:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

// ===================== ③ asrGoroutine =====================
func (s *Session) asrGoroutine(audioChan <-chan []byte, llmInputChan chan<- string) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case audio, ok := <-audioChan:
			if !ok {
				// audioChan 只有在Session结束时才会被关闭，正常情况下不应该发生
				return
			}
			result, err := s.asrProvider.RecognizeAudio(s.ctx, audio)
			if err != nil {
				slog.Error("[FreeTalk-ASR] ASR recognize audio failed",
					"error", err,
					"conversation_id", s.conversationID,
				)
				s.cancel()
				return
			}
			llmInputChan <- result
		}
	}
}

func (s *Session) llmGoroutine(
	llmInputChan <-chan string,
	llmOutputChan chan<- domain.LLMChunk,
	ttsNewTurnChan chan<- struct{},
	writeChan chan<- wsMessage,
) {
	profile := s.buildUserProfile()

	memory, err := NewConversationMemory(s.ctx, s.llmProvider, profile)
	if err != nil {
		slog.Error("[FreeTalk-LLM] Init conversation memory failed",
			"error", err,
			"conversation_id", s.conversationID,
		)
		s.cancel()
		return
	}

	// tokenChan 传递两种信号：
	//   普通字符串 → token，追加进 builder
	//   "__END__"  → LLM 本轮输出结束，切分 goroutine 负责 flush 残留并发 IsDone:true
	tokenChan := make(chan string, 100)

	// ── 句子切分 goroutine ────────────────────────────────────────
	// 唯一写 llmOutputChan 的地方，保证 IsDone:true 一定在所有句子之后
	go func() {
		var builder strings.Builder
		lastFlush := time.Now()

		flush := func() {
			text := strings.TrimSpace(builder.String())
			if text == "" {
				return
			}
			llmOutputChan <- domain.LLMChunk{Text: text, IsDone: false}
			builder.Reset()
			lastFlush = time.Now()
		}

		isSentenceEnd := func(s string) bool {
			s = strings.TrimSpace(s)
			return strings.HasSuffix(s, ".") ||
				strings.HasSuffix(s, "!") ||
				strings.HasSuffix(s, "?") ||
				strings.HasSuffix(s, "。") ||
				strings.HasSuffix(s, "！") ||
				strings.HasSuffix(s, "？")
		}

		// 时间兜底 ticker
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case token := <-tokenChan:

				// 空字符串 = LLM 本轮结束信号
				if token == "__END__" {
					// 先把残留内容 flush 出去
					flush()
					// 再发 IsDone:true，保证顺序
					llmOutputChan <- domain.LLMChunk{Text: "", IsDone: true}
					continue
				}

				builder.WriteString(token)
				current := builder.String()

				// 标点触发
				if isSentenceEnd(current) {
					flush()
					continue
				}

				// 长度兜底
				if builder.Len() > 60 {
					flush()
				}

			case <-ticker.C:
				// 时间兜底：800ms 没有新 token 就 flush
				if builder.Len() > 0 && time.Since(lastFlush) > 800*time.Millisecond {
					flush()
				}

			case <-s.ctx.Done():
				return
			}
		}
	}()

	// ── LLM 流式读取 goroutine ────────────────────────────────────
	// 只负责调用 LLM 并把 token/信号 投进 tokenChan
	// 不直接写 llmOutputChan
	go func() {

		for {
			select {
			case <-s.ctx.Done():
				return
			case input, ok := <-llmInputChan:
				if !ok {
					return
				}

				// ① 记录本轮用户输入（在调用 LLM 之前）
				memory.OnUserInput(input)

				// ② 取最新 convID（可能已被 rebuild 更新）
				convID := memory.ConvID()

				stream := s.llmProvider.ConversationChatStream(s.ctx, convID, input)
				firstToken := true
				for stream.Next() {
					event := stream.Current()
					switch event.Type {

					case "response.output_text.delta":
						slog.Info("[FreeTalk-LLM] LLM delta", "delta", event.Delta)
						if firstToken {
							firstToken = false
							select {
							case ttsNewTurnChan <- struct{}{}: // 触发 TTS 建立连接
							case <-s.ctx.Done():
								return
							}
						}
						// ③ 追加 token 到记忆（用于生成摘要）
						memory.OnAssistantToken(event.Delta)

						// 发 token 给切分 goroutine
						tokenChan <- event.Delta
						// 同时转发给 App 显示实时字幕
						msg := OutgoingMessage{Type: MsgTypeLLMToken, Text: event.Delta}
						data, _ := json.Marshal(msg)
						writeChan <- wsMessage{messageType: websocket.TextMessage, data: data}

					case "response.output_text.done":
						slog.Info("[FreeTalk-LLM] LLM done")
						// 发__END__通知切分 goroutine：本轮结束，flush 并发 IsDone
						tokenChan <- "__END__"

						// ④ 本轮完整回复已收到，提交记忆并触发重建检查
						// 注意：必须在 __END__ 之后调用，确保 currentAssistant 已完整
						// 使用 background context 避免 session ctx 取消时中断重建
						memory.OnTurnComplete(context.Background())
					}
				}

				if stream.Err() != nil {
					slog.Error("[FreeTalk-LLM] LLM stream failed",
						"error", stream.Err(),
					)
					s.cancel()
					return
				}
			}
		}
	}()
}

func (s *Session) ttsGoroutine(llmOutputChan <-chan domain.LLMChunk, ttsNewTurnChan <-chan struct{}, writeChan chan<- wsMessage) {

	// 启动一个goroutine，持续获取ttsNewTurnChan中的数据。
	// 每当收到一个新事件，代表这是新的一轮tts合成音频的任务。需要发送run-task指令
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ttsNewTurnChan:
			//  建立websocket连接。需要使用ttsProvider.ConnectTTS()方法。
			ttsConn, err := s.ttsProvider.ConnectTTS(s.ctx)
			if err != nil {
				slog.Error("[FreeTalk-TTS] Connect TTS failed",
					"error", err,
					"conversation_id", s.conversationID,
				)
				return
			}

			// 生成任务ID
			taskID := uuid.New()

			// 发送run-task指令
			runTaskCmd := map[string]interface{}{
				"header": map[string]interface{}{
					"action":    "run-task",
					"task_id":   taskID,
					"streaming": "duplex",
				},
				"payload": map[string]interface{}{
					"task_group": "audio",
					"task":       "tts",
					"function":   "SpeechSynthesizer",
					"model":      "cosyvoice-clone-v1",
					"parameters": map[string]interface{}{
						"text_type":   "PlainText",
						"voice":       "longwan",
						"format":      "pcm",
						"sample_rate": 16000,
						"volume":      50,
						"rate":        1,
						"pitch":       1,
						// 如果enable_ssml设为true，只允许发送一次continue-task指令，否则会报错“Text request limit violated, expected 1.”
						"enable_ssml": false,
					},
					"input": map[string]interface{}{},
				},
			}

			runTaskJSON, _ := json.Marshal(runTaskCmd)
			err = ttsConn.WriteMessage(websocket.TextMessage, runTaskJSON)
			if err != nil {
				slog.Error("[FreeTalk-TTS] Send run task failed",
					"error", err,
				)
				return
			}
			// 创建一个通道，用于接收task-started事件
			taskStartedChan := make(chan struct{}, 1)
			// 启动一个goroutine异步接收WebSocket消息
			go func() {
				const pcmFlushThreshold = 8 * 1024 // 8KB，约 0.25 秒的 16kHz 单声道 PCM
				var pcmBuffer []byte               // 新增 PCM 缓冲
				for {
					messageType, message, err := ttsConn.ReadMessage()
					if err != nil {
						slog.Error("[FreeTalk-TTS] Read tts message failed",
							"error", err,
						)
						return
					}
					if messageType == websocket.BinaryMessage {
						// 攒进缓冲区
						pcmBuffer = append(pcmBuffer, message...)
						// 够阈值就推给 App
						if len(pcmBuffer) >= pcmFlushThreshold {
							writeChan <- wsMessage{
								messageType: websocket.BinaryMessage,
								data:        pcmBuffer,
							}
							pcmBuffer = nil
						}
						continue
					} else if messageType == websocket.TextMessage {
						var event wsEvent
						if err := json.Unmarshal(message, &event); err != nil {
							slog.Error("[FreeTalk-TTS] Parse event failed",
								"error", err,
							)
						}
						if event.Header.Event == "task-started" {
							taskStartedChan <- struct{}{}
						} else if event.Header.Event == "task-failed" {
							slog.Error("[FreeTalk-TTS] Task failed",
								"error", event.Header.ErrorMessage,
							)
							return
						} else if event.Header.Event == "task-finished" {
							slog.Info("[FreeTalk-TTS] Task finished",
								"task_id", event.Header.TaskID,
							)
							// flush 剩余 PCM
							if len(pcmBuffer) > 0 {
								writeChan <- wsMessage{
									messageType: websocket.BinaryMessage,
									data:        pcmBuffer,
								}
								pcmBuffer = nil
							}
							// 通知 App：本轮 AI 音频全部发送完毕
							turnEndMsg := OutgoingMessage{Type: MsgTypeTurnEnd}
							turnEndData, _ := json.Marshal(turnEndMsg)
							writeChan <- wsMessage{messageType: websocket.TextMessage, data: turnEndData}
							return
						}
					}
				}
			}()
			select {
			case <-s.ctx.Done():
				return
			case <-taskStartedChan:
				// 收到 task-started 事件，继续往下走
				slog.Info("[FreeTalk-TTS] Task started, begin feeding tokens",
					"task_id", taskID,
				)
				break
			}
			// 持续获取llmOutputChan中的数据,并不断发送待合成的文本
			done := false
			for !done {
				select {
				case <-s.ctx.Done():
					return
				case chunk, ok := <-llmOutputChan:
					if !ok {
						// llmOutputChan 只有在 Session 结束时才会被关闭，此时应该退出循环
						done = true
						break
					}
					// 发送continue-task指令
					continueTaskCmd := map[string]interface{}{
						"header": map[string]interface{}{
							"action":    "continue-task",
							"task_id":   taskID,
							"streaming": "duplex",
						},
						"payload": map[string]interface{}{
							"input": map[string]interface{}{
								"text": chunk.Text,
							},
						},
					}

					continueTaskJSON, _ := json.Marshal(continueTaskCmd)
					err = ttsConn.WriteMessage(websocket.TextMessage, continueTaskJSON)
					if err != nil {
						slog.Error("[FreeTalk-TTS] Send continue task failed",
							"error", err,
						)
						return
					}

					if chunk.IsDone {
						// 发送finish-task指令
						finishTaskCmd := map[string]interface{}{
							"header": map[string]interface{}{
								"action":    "finish-task",
								"task_id":   taskID,
								"streaming": "duplex",
							},
							"payload": map[string]interface{}{
								"input": map[string]interface{}{},
							},
						}

						finishTaskJSON, _ := json.Marshal(finishTaskCmd)

						err = ttsConn.WriteMessage(websocket.TextMessage, finishTaskJSON)
						if err != nil {
							slog.Error("[FreeTalk-TTS] Send finish task failed",
								"error", err,
							)
						}
						done = true // 退出内层循环，继续外层
					}
				}
			}
			ttsConn.Close() // 确保连接被关闭
		}
	}
}

// wsEvent WebSocket 事件（请求和响应的统一结构）
type wsEvent struct {
	Header  wsHeader  `json:"header"`
	Payload wsPayload `json:"payload"`
}

// wsHeader 事件头部
type wsHeader struct {
	// Action 请求动作: "run-task", "continue-task", "finish-task"
	Action string `json:"action,omitempty"`

	// TaskID 任务唯一标识
	TaskID string `json:"task_id"`

	// Streaming 流式模式: "duplex"（双工）
	Streaming string `json:"streaming,omitempty"`

	// Event 响应事件类型: "task-started", "result-generated", "task-finished", "task-failed"
	Event string `json:"event,omitempty"`

	// ErrorCode 错误码（仅 task-failed 时有值）
	ErrorCode string `json:"error_code,omitempty"`

	// ErrorMessage 错误信息（仅 task-failed 时有值）
	ErrorMessage string `json:"error_message,omitempty"`

	// Attributes 附加属性
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// wsPayload 事件负载
type wsPayload struct {
	// 请求字段
	TaskGroup  string   `json:"task_group,omitempty"`
	Task       string   `json:"task,omitempty"`
	Function   string   `json:"function,omitempty"`
	Model      string   `json:"model,omitempty"`
	Parameters wsParams `json:"parameters,omitempty"`
	Input      wsInput  `json:"input"`

	// 响应字段
	Output wsOutput `json:"output,omitempty"`
	Usage  *wsUsage `json:"usage,omitempty"`
}

// wsParams TTS 合成参数
type wsParams struct {
	// TextType 文本类型: "PlainText"
	TextType string `json:"text_type,omitempty"`

	// Voice 音色: "longanyang", "longxiaochun" 等
	Voice string `json:"voice,omitempty"`

	// Format 音频格式: "mp3", "wav", "pcm"
	Format string `json:"format,omitempty"`

	// SampleRate 采样率: 8000, 16000, 22050, 24000, 48000
	SampleRate int `json:"sample_rate,omitempty"`

	// Volume 音量: 0-100
	Volume int `json:"volume,omitempty"`

	// Rate 语速: 0.5-2.0
	Rate float64 `json:"rate,omitempty"`

	// Pitch 音调: 0.5-2.0
	Pitch float64 `json:"pitch,omitempty"`

	// EnableSSML 是否启用 SSML（启用后只允许发送一次 continue-task）
	EnableSSML bool `json:"enable_ssml,omitempty"`
}

// wsInput 输入内容
type wsInput struct {
	// Text 待合成文本（用于 continue-task 指令）
	Text string `json:"text,omitempty"`
}

// wsOutput 输出内容（用于 result-generated 事件）
type wsOutput struct {
	// 部分 TTS 事件可能在此返回额外信息
}

// wsUsage 计费信息
type wsUsage struct {
	// Characters 已消耗字符数
	Characters int `json:"characters,omitempty"`

	// Duration 音频时长（秒）
	Duration int `json:"duration,omitempty"`
}
