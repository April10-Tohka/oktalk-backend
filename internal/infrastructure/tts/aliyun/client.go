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

	// 等待 task-started
	if err := session.waitTaskStarted(ctx); err != nil {
		return nil, err
	}

	// 发送所有文本（continue-task）
	for _, text := range texts {
		if err := session.sendContinueTask(text); err != nil {
			return nil, err
		}
	}

	// 发送 finish-task
	if err := session.sendFinishTask(); err != nil {
		return nil, err
	}

	// 等待接收完成，收集所有音频数据
	audioData, err := session.waitAndCollect(ctx)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	logger.Info("[AliyunTTS] Synthesis completed",
		"texts_count", len(texts),
		"audio_bytes", len(audioData),
		"elapsed", elapsed.String(),
		"task_id", session.taskID,
	)

	return audioData, nil
}

// synthesizeStream 流式合成语音（实时推送音频块）
func (c *internalClient) synthesizeStream(ctx context.Context, texts []string, opts *domain.SynthesizeOptions, audioChan chan<- []byte) error {
	params := c.mergeParams(opts)

	session, err := c.newSession(ctx, params)
	if err != nil {
		return err
	}
	defer session.close()

	// 等待 task-started
	if err := session.waitTaskStarted(ctx); err != nil {
		return err
	}

	// 启动接收 goroutine（流式推送到 audioChan）
	receiveDone := make(chan error, 1)
	go func() {
		receiveDone <- session.receiveAndStream(ctx, audioChan)
	}()

	// 发送所有文本
	for _, text := range texts {
		if err := session.sendContinueTask(text); err != nil {
			return err
		}
	}

	// 发送 finish-task
	if err := session.sendFinishTask(); err != nil {
		return err
	}

	// 等待接收完成
	select {
	case err := <-receiveDone:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
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
	mu     sync.Mutex // 保护 conn 的并发写入

	// 同步信号
	taskStarted chan struct{}

	// 接收到的音频数据
	audioBuffer []byte
	audioMu     sync.Mutex
}

// newSession 创建新的合成会话（建立 WebSocket 连接 + 发送 run-task）
func (c *internalClient) newSession(ctx context.Context, params wsParams) (*synthesisSession, error) {
	// 1. 建立 WebSocket 连接
	conn, err := c.connectWebSocket(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect websocket failed: %w", err)
	}

	// 2. 生成任务 ID
	taskID := uuid.New().String()

	session := &synthesisSession{
		conn:        conn,
		taskID:      taskID,
		taskStarted: make(chan struct{}),
	}

	// 3. 发送 run-task 指令
	if err := session.sendRunTask(ctx, c.model, params); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send run-task failed: %w", err)
	}

	// 4. 启动后台接收消息 goroutine（用于 task-started 事件检测）
	go session.receiveLoop(ctx)

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
	s.mu.Lock()
	err = s.conn.WriteMessage(websocket.TextMessage, data)
	s.mu.Unlock()

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
func (s *synthesisSession) sendContinueTask(text string) error {
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

	s.mu.Lock()
	err = s.conn.WriteMessage(websocket.TextMessage, data)
	s.mu.Unlock()

	if err != nil {
		return fmt.Errorf("send continue-task failed: %w", err)
	}

	logger.Debug("[AliyunTTS] Sent continue-task",
		"task_id", s.taskID,
		"text_len", len(text),
	)
	return nil
}

// sendFinishTask 发送 finish-task 指令
func (s *synthesisSession) sendFinishTask() error {
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

	s.mu.Lock()
	err = s.conn.WriteMessage(websocket.TextMessage, data)
	s.mu.Unlock()

	if err != nil {
		return fmt.Errorf("send finish-task failed: %w", err)
	}

	logger.Debug("[AliyunTTS] Sent finish-task", "task_id", s.taskID)
	return nil
}

// ===================== 接收消息 =====================

// receiveLoop 持续接收 WebSocket 消息
// 处理二进制音频块和文本事件
func (s *synthesisSession) receiveLoop(ctx context.Context) {
	taskStartedNotified := false

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

			logger.Debug("[AliyunTTS] Received audio chunk",
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
			logger.Info("[AliyunTTS] Task started", "task_id", s.taskID)
			if !taskStartedNotified {
				close(s.taskStarted)
				taskStartedNotified = true
			}

		case eventResultGenerated:
			logger.Debug("[AliyunTTS] Result generated",
				"task_id", s.taskID,
			)
			if event.Payload.Usage != nil {
				logger.Debug("[AliyunTTS] Usage info",
					"characters", event.Payload.Usage.Characters,
					"duration", event.Payload.Usage.Duration,
					"task_id", s.taskID,
				)
			}

		case eventTaskFinished:
			logger.Info("[AliyunTTS] Task finished", "task_id", s.taskID)
			return

		case eventTaskFailed:
			errMsg := event.Header.ErrorMessage
			if errMsg == "" {
				errMsg = "unknown error"
			}
			logger.Error("[AliyunTTS] Task failed",
				"error_code", event.Header.ErrorCode,
				"error_message", errMsg,
				"task_id", s.taskID,
			)
			// 如果 task-started 还没通知，也要释放等待方
			if !taskStartedNotified {
				close(s.taskStarted)
				taskStartedNotified = true
			}
			return

		default:
			logger.Warn("[AliyunTTS] Unknown event",
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
	}
}

// waitAndCollect 等待所有音频接收完成，返回完整音频数据
func (s *synthesisSession) waitAndCollect(ctx context.Context) ([]byte, error) {
	// receiveLoop 会在 task-finished / task-failed 时退出
	// 我们通过定时检查 conn 的状态来判断是否完成
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.After(defaultTaskFinishTimeout)
	for {
		select {
		case <-deadline:
			return nil, fmt.Errorf("wait task-finished timeout after %v, task_id=%s",
				defaultTaskFinishTimeout, s.taskID)
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// 检查 receiveLoop 是否已经退出（通过尝试读取已缓存的数据判断）
			// 简单方案：等一小段时间后尝试 ping，如果连接已关闭说明完成
		}

		// 尝试 ping 来检测连接是否仍存活
		s.mu.Lock()
		err := s.conn.WriteMessage(websocket.PingMessage, nil)
		s.mu.Unlock()

		if err != nil {
			// 连接已关闭 = receiveLoop 已结束（task-finished/task-failed）
			s.audioMu.Lock()
			data := make([]byte, len(s.audioBuffer))
			copy(data, s.audioBuffer)
			s.audioMu.Unlock()
			return data, nil
		}
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
