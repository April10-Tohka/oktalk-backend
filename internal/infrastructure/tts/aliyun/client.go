// Package aliyun 提供阿里云 CosyVoice TTS 的具体实现
// 此包内的所有类型对外部（Service 层）不可见，
// 仅通过 AliyunTTSAdapter 实现 domain.TTSProvider 接口对外暴露
package aliyun

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"
)

// ===================== 默认配置 =====================

const (
	// defaultConnectTimeout WebSocket 连接超时
	defaultConnectTimeout = 10 * time.Second

	// defaultTaskStartTimeout 等待 task-started 事件的超时
	defaultTaskStartTimeout = 15 * time.Second

	// defaultTaskFinishTimeout 等待 task-finished 事件的超时
	defaultTaskFinishTimeout = 120 * time.Second
)

// ===================== 客户端定义 =====================

// internalClient 阿里云 CosyVoice TTS 内部 WebSocket 客户端
// 封装了与 DashScope WebSocket API 的所有通信细节
type internalClient struct {
	apiKey   string   // DashScope API Key
	wsURL    string   // WebSocket 地址
	model    string   // 模型名称
	defaults wsParams // 默认合成参数
}

// newInternalClient 根据配置创建内部客户端
func newInternalClient(cfg config.AliyunTTSConfig) *internalClient {
	// 确定 WebSocket 地址
	wsURL := cfg.Endpoint
	if wsURL == "" {
		switch cfg.Region {
		case "singapore":
			wsURL = wsURLSingapore
		default: // beijing 或未指定
			wsURL = wsURLBeijing
		}
	}

	// 确定模型
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	// 构建默认参数
	defaults := buildDefaultParams(cfg.DefaultOptions)

	logger.Info("[AliyunTTS] Client initialized",
		"model", model,
		"endpoint", wsURL,
		"voice", defaults.Voice,
		"format", defaults.Format,
	)

	return &internalClient{
		apiKey:   cfg.APIKey,
		wsURL:    wsURL,
		model:    model,
		defaults: defaults,
	}
}

// buildDefaultParams 从配置构建默认 WebSocket 参数
func buildDefaultParams(opts config.TTSDefaultOptions) wsParams {
	p := wsParams{
		TextType:   defaultTextType,
		Voice:      opts.Voice,
		Format:     opts.Format,
		SampleRate: opts.SampleRate,
		Volume:     opts.Volume,
		Rate:       opts.Rate,
		Pitch:      opts.Pitch,
	}

	// 填充未设置的默认值
	if p.Voice == "" {
		p.Voice = "longanyang"
	}
	if p.Format == "" {
		p.Format = "mp3"
	}
	if p.SampleRate == 0 {
		p.SampleRate = 22050
	}
	if p.Volume == 0 {
		p.Volume = 50
	}
	if p.Rate == 0 {
		p.Rate = 1.0
	}
	if p.Pitch == 0 {
		p.Pitch = 1.0
	}
	return p
}

// mergeParams 合并用户选项与默认参数
func (c *internalClient) mergeParams(opts *domain.SynthesizeOptions) wsParams {
	if opts == nil {
		return c.defaults
	}

	p := c.defaults // 以默认值为基础
	if opts.Voice != "" {
		p.Voice = opts.Voice
	}
	if opts.Format != "" {
		p.Format = opts.Format
	}
	if opts.SampleRate != 0 {
		p.SampleRate = opts.SampleRate
	}
	if opts.Volume != 0 {
		p.Volume = opts.Volume
	}
	if opts.Rate != 0 {
		p.Rate = opts.Rate
	}
	if opts.Pitch != 0 {
		p.Pitch = opts.Pitch
	}
	return p
}

// ===================== 核心合成方法 =====================

// synthesize 同步合成语音（返回完整音频数据）
// 发送一或多段文本，等待所有音频块返回后拼接
func (c *internalClient) synthesize(ctx context.Context, texts []string, opts *domain.SynthesizeOptions) ([]byte, error) {
	start := time.Now()
	params := c.mergeParams(opts)

	// 建立 WebSocket 连接 + 发送 run-task
	session, err := c.newSession(ctx, params)
	if err != nil {
		return nil, err
	}
	defer session.close()

	// 发送 run-task 指令
	if err := session.sendRunTask(ctx, c.model, params); err != nil {
		return nil, err
	}
	// 启动后台接收消息 goroutine
	go session.receiveLoop(ctx)

	// 等待 task-started
	if err := session.waitTaskStarted(ctx); err != nil {
		return nil, err
	}

	// 发送所有文本（continue-task）
	for _, text := range texts {
		if err := session.sendContinueTask(ctx, text); err != nil {
			return nil, err
		}
	}

	// 发送 finish-task
	if err := session.sendFinishTask(ctx); err != nil {
		return nil, err
	}

	// 等待接收完成，收集所有音频数据
	audioData, err := session.waitAndCollect(ctx)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	logger.InfoContext(ctx, "[AliyunTTS] Synthesis completed",
		"texts_count", len(texts),
		"audio_bytes", len(audioData),
		"elapsed", elapsed.String(),
		"task_id", session.taskID,
	)

	return audioData, nil
}

// synthesizeStream 流式合成语音（实时推送音频块）
func (c *internalClient) synthesizeStream(ctx context.Context, texts []string, opts *domain.SynthesizeOptions, audioChan chan<- []byte) error {
	return nil
	// params := c.mergeParams(opts)

	// session, err := c.newSession(ctx, params)
	// if err != nil {
	// 	return err
	// }
	// defer session.close()

	// // 等待 task-started
	// if err := session.waitTaskStarted(ctx); err != nil {
	// 	return err
	// }

	// // 启动接收 goroutine（流式推送到 audioChan）
	// receiveDone := make(chan error, 1)
	// go func() {
	// 	receiveDone <- session.receiveAndStream(ctx, audioChan)
	// }()

	// // 发送所有文本
	// for _, text := range texts {
	// 	if err := session.sendContinueTask(text); err != nil {
	// 		return err
	// 	}
	// }

	// // 发送 finish-task
	// if err := session.sendFinishTask(); err != nil {
	// 	return err
	// }

	// // 等待接收完成
	// select {
	// case err := <-receiveDone:
	// 	return err
	// case <-ctx.Done():
	// 	return ctx.Err()
	// }
}

// close 关闭客户端
func (c *internalClient) close() error {
	logger.Info("[AliyunTTS] Client closed")
	return nil
}

// ===================== 会话管理 =====================

// synthesisSession 单次 TTS 合成会话
// 封装一次 WebSocket 连接的完整生命周期
type synthesisSession struct {
	conn   *websocket.Conn
	taskID string

	// 同步信号
	taskStarted  chan struct{} // 任务启动信号
	taskFinished chan struct{} // 任务完成信号
	taskFailed   chan struct{} // 任务失败信号

	// 接收到的音频数据
	audioBuffer []byte
	audioMu     sync.Mutex
}

// newSession 创建新的合成会话（建立 WebSocket 连接）
func (c *internalClient) newSession(ctx context.Context, params wsParams) (*synthesisSession, error) {
	// 1. 建立 WebSocket 连接
	conn, err := c.connectWebSocket(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect websocket failed: %w", err)
	}

	// 2. 生成任务 ID
	taskID := uuid.New().String()

	session := &synthesisSession{
		conn:         conn,
		taskID:       taskID,
		taskStarted:  make(chan struct{}),
		taskFinished: make(chan struct{}),
		taskFailed:   make(chan struct{}),
	}

	return session, nil
}

// ===================== WebSocket 连接管理 =====================

// connectWebSocket 建立 WebSocket 连接
func (c *internalClient) connectWebSocket(ctx context.Context) (*websocket.Conn, error) {
	header := make(http.Header)
	header.Set("Authorization", fmt.Sprintf("bearer %s", c.apiKey))
	header.Set("X-DashScope-DataInspection", "enable")

	dialer := websocket.Dialer{
		HandshakeTimeout: defaultConnectTimeout,
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	conn, _, err := dialer.DialContext(ctx, c.wsURL, header)
	if err != nil {
		logger.ErrorContext(ctx, "[AliyunTTS] WebSocket connect failed", "error", err)
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	logger.DebugContext(ctx, "[AliyunTTS] WebSocket connected", "url", c.wsURL)
	return conn, nil
}

// ===================== 发送指令 =====================

// sendRunTask 发送 run-task 指令，启动合成任务
func (s *synthesisSession) sendRunTask(ctx context.Context, model string, params wsParams) error {
	event := wsEvent{
		Header: wsHeader{
			Action:    actionRunTask,
			TaskID:    s.taskID,
			Streaming: streamingDuplex,
		},
		Payload: wsPayload{
			TaskGroup:  taskGroupAudio,
			Task:       taskTTS,
			Function:   functionSpeechSynth,
			Model:      model,
			Parameters: params,
			Input:      wsInput{},
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal run-task failed: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	err = s.conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		return fmt.Errorf("send run-task failed: %w", err)
	}

	logger.DebugContext(ctx, "[AliyunTTS] Sent run-task",
		"task_id", s.taskID,
		"model", model,
		"voice", params.Voice,
		"format", params.Format,
	)
	return nil
}

// sendContinueTask 发送 continue-task 指令（推送文本）
func (s *synthesisSession) sendContinueTask(ctx context.Context, text string) error {
	event := wsEvent{
		Header: wsHeader{
			Action:    actionContinueTask,
			TaskID:    s.taskID,
			Streaming: streamingDuplex,
		},
		Payload: wsPayload{
			Input: wsInput{Text: text},
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal continue-task failed: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	err = s.conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		return fmt.Errorf("send continue-task failed: %w", err)
	}

	logger.DebugContext(ctx, "[AliyunTTS] Sent continue-task",
		"task_id", s.taskID,
		"text_len", len(text),
	)
	return nil
}

// sendFinishTask 发送 finish-task 指令
func (s *synthesisSession) sendFinishTask(ctx context.Context) error {
	event := wsEvent{
		Header: wsHeader{
			Action:    actionFinishTask,
			TaskID:    s.taskID,
			Streaming: streamingDuplex,
		},
		Payload: wsPayload{
			Input: wsInput{},
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal finish-task failed: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	err = s.conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		return fmt.Errorf("send finish-task failed: %w", err)
	}

	logger.DebugContext(ctx, "[AliyunTTS] Sent finish-task", "task_id", s.taskID)
	return nil
}

// ===================== 接收消息 =====================

// receiveLoop 持续接收 WebSocket 消息
// 处理二进制音频块和文本事件
func (s *synthesisSession) receiveLoop(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 设置读取超时
		_ = s.conn.SetReadDeadline(time.Now().Add(defaultTaskFinishTimeout))

		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			// 超时或连接断开
			logger.Error("[AliyunTTS] Read message failed",
				"error", err,
				"task_id", s.taskID,
			)
			return
		}

		// 处理二进制消息（音频数据块）
		if messageType == websocket.BinaryMessage {
			s.audioMu.Lock()
			s.audioBuffer = append(s.audioBuffer, message...)
			s.audioMu.Unlock()

			logger.DebugContext(ctx, "[AliyunTTS] Received audio chunk",
				"bytes", len(message),
				"task_id", s.taskID,
			)
			continue
		}

		// 处理文本消息（事件）
		var event wsEvent
		if err := json.Unmarshal(message, &event); err != nil {
			logger.Warn("[AliyunTTS] Parse event failed",
				"error", err,
				"task_id", s.taskID,
			)
			continue
		}

		switch event.Header.Event {
		case eventTaskStarted:
			logger.InfoContext(ctx, "[AliyunTTS] Task started", "task_id", s.taskID)
			close(s.taskStarted)

		case eventResultGenerated:
			logger.DebugContext(ctx, "[AliyunTTS] Result generated",
				"task_id", s.taskID,
			)

		case eventTaskFinished:
			logger.InfoContext(ctx, "[AliyunTTS] Task finished", "task_id", s.taskID)
			if event.Payload.Usage != nil {
				logger.DebugContext(ctx, "[AliyunTTS] Usage info",
					"characters", event.Payload.Usage.Characters,
					"duration", event.Payload.Usage.Duration,
					"task_id", s.taskID,
				)
			}
			close(s.taskFinished)
			return

		case eventTaskFailed:
			errMsg := event.Header.ErrorMessage
			if errMsg == "" {
				errMsg = "unknown error"
			}
			logger.ErrorContext(ctx, "[AliyunTTS] Task failed",
				"error_code", event.Header.ErrorCode,
				"error_message", errMsg,
				"task_id", s.taskID,
			)
			close(s.taskFailed)
			return

		default:
			logger.WarnContext(ctx, "[AliyunTTS] Unknown event",
				"event", event.Header.Event,
				"task_id", s.taskID,
			)
		}
	}
}

// ===================== 同步等待 =====================

// waitTaskStarted 等待 task-started 事件
func (s *synthesisSession) waitTaskStarted(ctx context.Context) error {
	select {
	case <-s.taskStarted:
		return nil
	case <-time.After(defaultTaskStartTimeout):
		return fmt.Errorf("wait task-started timeout after %v, task_id=%s",
			defaultTaskStartTimeout, s.taskID)
	case <-ctx.Done():
		return ctx.Err()
	case <-s.taskFailed:
		return fmt.Errorf("task failed before started, task_id=%s", s.taskID)
	}
}

// waitAndCollect 等待所有音频接收完成，返回完整音频数据
func (s *synthesisSession) waitAndCollect(ctx context.Context) ([]byte, error) {
	// 等待 task-finished 或 task-failed 事件
	select {
	case <-s.taskFinished:
		// 任务正常完成，返回音频数据
		s.audioMu.Lock()
		data := make([]byte, len(s.audioBuffer))
		copy(data, s.audioBuffer)
		s.audioMu.Unlock()
		logger.DebugContext(ctx, "[AliyunTTS] Collected audio data",
			"bytes", len(data),
			"task_id", s.taskID,
		)
		return data, nil
	case <-s.taskFailed:
		return nil, fmt.Errorf("task failed, task_id=%s", s.taskID)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ===================== 流式接收 =====================

// receiveAndStream 流式接收音频并推送到 channel
func (s *synthesisSession) receiveAndStream(ctx context.Context, audioChan chan<- []byte) error {
	defer close(audioChan)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_ = s.conn.SetReadDeadline(time.Now().Add(defaultTaskFinishTimeout))

		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return nil
			}
			return fmt.Errorf("read message failed: %w", err)
		}

		// 推送二进制音频块
		if messageType == websocket.BinaryMessage {
			select {
			case audioChan <- message:
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		// 解析文本事件
		var event wsEvent
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		switch event.Header.Event {
		case eventTaskFinished:
			logger.Info("[AliyunTTS] Stream task finished", "task_id", s.taskID)
			return nil

		case eventTaskFailed:
			errMsg := event.Header.ErrorMessage
			if errMsg == "" {
				errMsg = "unknown error"
			}
			return fmt.Errorf("tts task failed: code=%s, message=%s",
				event.Header.ErrorCode, errMsg)

		case eventResultGenerated:
			// result-generated 在 TTS 中表示合成进度，可忽略
		}
	}
}

// ===================== 会话关闭 =====================

// close 关闭会话连接
func (s *synthesisSession) close() {
	if s.conn != nil {
		_ = s.conn.Close()
	}
}
