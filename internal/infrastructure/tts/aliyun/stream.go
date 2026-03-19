package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"pronunciation-correction-system/internal/domain"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ===================== ConnectTTS 流式长连接实现 =====================

// ConnectTTS 建立流式 TTS 长连接（用于 Free Talk 模式）
// 返回 TTSStreamer，支持多轮文本推送和音频接收
func (a *AliyunTTSAdapter) ConnectTTS(ctx context.Context) (*websocket.Conn, error) {
	conn, err := a.client.connectWebSocket(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect TTS websocket failed: %w", err)
	}
	return conn, nil
}

// connectTTS 内部实现
func (c *internalClient) connectTTS(ctx context.Context, options *domain.SynthesizeOptions) (domain.TTSStreamer, error) {
	params := c.mergeParams(options)

	// 建立 WebSocket 连接
	conn, err := c.connectWebSocket(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect TTS websocket failed: %w", err)
	}

	streamCtx, streamCancel := context.WithCancel(ctx)

	streamer := &ttsStreamerImpl{
		conn:    conn,
		model:   c.model,
		params:  params,
		audioCh: make(chan []byte, 128),
		ctx:     streamCtx,
		cancel:  streamCancel,
	}

	// 启动后台接收 goroutine
	go streamer.receiveLoop()

	slog.Info("[AliyunTTS] ConnectTTS established",
		"voice", params.Voice,
		"format", params.Format,
		"sample_rate", params.SampleRate,
	)

	return streamer, nil
}

// ===================== TTSStreamer 实现 =====================

// ttsStreamerImpl 流式 TTS 连接实现
type ttsStreamerImpl struct {
	conn   *websocket.Conn
	model  string
	params wsParams

	// 音频输出通道（整个连接生命周期共享）
	audioCh chan []byte

	// 写入保护（防止并发写入 WebSocket）
	writeMu sync.Mutex

	// 每轮任务状态
	taskMu      sync.Mutex
	taskID      string
	taskStarted chan struct{}
	taskDoneCh  chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
}

// RunTask 启动新的合成任务（发送 run-task 指令）
func (s *ttsStreamerImpl) RunTask(ctx context.Context) error {
	s.taskMu.Lock()
	s.taskID = uuid.New().String()
	s.taskStarted = make(chan struct{})
	s.taskDoneCh = make(chan struct{})
	taskID := s.taskID
	startedCh := s.taskStarted
	s.taskMu.Unlock()

	// 构建 run-task 事件
	event := wsEvent{
		Header: wsHeader{
			Action:    actionRunTask,
			TaskID:    taskID,
			Streaming: streamingDuplex,
		},
		Payload: wsPayload{
			TaskGroup:  taskGroupAudio,
			Task:       taskTTS,
			Function:   functionSpeechSynth,
			Model:      s.model,
			Parameters: s.params,
			Input:      wsInput{},
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal run-task failed: %w", err)
	}

	s.writeMu.Lock()
	err = s.conn.WriteMessage(websocket.TextMessage, data)
	s.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("send run-task failed: %w", err)
	}

	slog.Info("[AliyunTTS] Sent run-task (stream)",
		"task_id", taskID,
		"model", s.model,
		"voice", s.params.Voice,
	)

	// 等待 task-started
	select {
	case <-startedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-time.After(defaultTaskStartTimeout):
		return fmt.Errorf("wait task-started timeout, task_id=%s", taskID)
	}
}

// FeedText 向当前任务推送文本片段（发送 continue-task 指令）
func (s *ttsStreamerImpl) FeedText(ctx context.Context, text string) error {
	s.taskMu.Lock()
	taskID := s.taskID
	s.taskMu.Unlock()

	if taskID == "" {
		return fmt.Errorf("no active task, call RunTask first")
	}

	event := wsEvent{
		Header: wsHeader{
			Action:    actionContinueTask,
			TaskID:    taskID,
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

	s.writeMu.Lock()
	err = s.conn.WriteMessage(websocket.TextMessage, data)
	s.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("send continue-task failed: %w", err)
	}

	slog.Debug("[AliyunTTS] Sent continue-task (stream)",
		"task_id", taskID,
		"text_len", len(text),
	)
	return nil
}

// FinishTask 通知当前任务文本发送完毕（发送 finish-task 指令）
func (s *ttsStreamerImpl) FinishTask(ctx context.Context) error {
	s.taskMu.Lock()
	taskID := s.taskID
	s.taskMu.Unlock()

	if taskID == "" {
		return fmt.Errorf("no active task")
	}

	event := wsEvent{
		Header: wsHeader{
			Action:    actionFinishTask,
			TaskID:    taskID,
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

	s.writeMu.Lock()
	err = s.conn.WriteMessage(websocket.TextMessage, data)
	s.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("send finish-task failed: %w", err)
	}

	slog.Info("[AliyunTTS] Sent finish-task (stream)", "task_id", taskID)
	return nil
}

// AudioChan 返回接收合成音频数据的通道
func (s *ttsStreamerImpl) AudioChan() <-chan []byte {
	return s.audioCh
}

// TaskDone 返回当前任务完成信号通道
func (s *ttsStreamerImpl) TaskDone() <-chan struct{} {
	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	if s.taskDoneCh == nil {
		// 返回一个已关闭的 channel（防止死锁）
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return s.taskDoneCh
}

// Close 关闭 WebSocket 连接并释放资源
func (s *ttsStreamerImpl) Close() error {
	s.cancel()
	close(s.audioCh)

	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			return fmt.Errorf("close TTS websocket failed: %w", err)
		}
	}

	slog.Info("[AliyunTTS] TTSStreamer closed")
	return nil
}

// ===================== 后台接收 =====================

// receiveLoop 持续接收 CosyVoice WebSocket 消息
// 处理二进制音频块和文本事件（task-started, task-finished, task-failed）
func (s *ttsStreamerImpl) receiveLoop() {
	for {
		select {
		case <-s.ctx.Done():
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
			select {
			case <-s.ctx.Done():
				return // 正常退出
			default:
			}
			slog.Error("[AliyunTTS] Stream receiveLoop read error",
				"error", err,
			)
			return
		}

		// 处理二进制消息（音频数据块）
		if messageType == websocket.BinaryMessage {
			select {
			case s.audioCh <- message:
			case <-s.ctx.Done():
				return
			}
			continue
		}

		// 处理文本消息（事件）
		var event wsEvent
		if err := json.Unmarshal(message, &event); err != nil {
			slog.Warn("[AliyunTTS] Stream parse event failed", "error", err)
			continue
		}

		switch event.Header.Event {
		case eventTaskStarted:
			s.taskMu.Lock()
			ch := s.taskStarted
			s.taskMu.Unlock()
			if ch != nil {
				close(ch)
			}
			slog.Info("[AliyunTTS] Stream task started", "task_id", event.Header.TaskID)

		case eventResultGenerated:
			// result-generated 表示合成进度，可忽略

		case eventTaskFinished:
			slog.Info("[AliyunTTS] Stream task finished", "task_id", event.Header.TaskID)
			s.taskMu.Lock()
			ch := s.taskDoneCh
			s.taskMu.Unlock()
			if ch != nil {
				close(ch)
			}

		case eventTaskFailed:
			errMsg := event.Header.ErrorMessage
			if errMsg == "" {
				errMsg = "unknown error"
			}
			slog.Error("[AliyunTTS] Stream task failed",
				"error_code", event.Header.ErrorCode,
				"error_message", errMsg,
				"task_id", event.Header.TaskID,
			)
			// 关闭 taskDoneCh 以解除等待
			s.taskMu.Lock()
			ch := s.taskDoneCh
			s.taskMu.Unlock()
			if ch != nil {
				close(ch)
			}

		default:
			slog.Warn("[AliyunTTS] Stream unknown event",
				"event", event.Header.Event,
				"task_id", event.Header.TaskID,
			)
		}
	}
}
