package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// AliyunTTSAdapter 阿里云 CosyVoice TTS 适配器
// 实现 domain.TTSProvider 接口，将领域层调用转换为阿里云 WebSocket API 调用
type AliyunTTSAdapter struct {
	client   *internalClient
	apiKey   string   // DashScope API Key
	wsURL    string   // WebSocket 地址
	model    string   // 模型名称
	defaults wsParams // 默认合成参数
}

// 编译时检查：确保 AliyunTTSAdapter 实现了 domain.TTSProvider 接口
var _ domain.TTSProvider = (*AliyunTTSAdapter)(nil)

// NewAliyunTTSAdapter 创建阿里云 CosyVoice TTS 适配器
func NewAliyunTTSAdapter(cfg config.AliyunTTSConfig) *AliyunTTSAdapter {
	return &AliyunTTSAdapter{
		client: newInternalClient(cfg),
		apiKey: cfg.APIKey,
		wsURL:  cfg.Endpoint,
		model:  cfg.Model,
		defaults: wsParams{
			TextType:   defaultTextType,
			Voice:      cfg.DefaultOptions.Voice,
			Format:     cfg.DefaultOptions.Format,
			SampleRate: cfg.DefaultOptions.SampleRate,
			Volume:     cfg.DefaultOptions.Volume,
			Rate:       cfg.DefaultOptions.Rate,
			Pitch:      cfg.DefaultOptions.Pitch,
		},
	}
}

// Synthesize 合成语音（同步方式，返回完整音频数据）
// 将单段文本发送给 CosyVoice，等待合成完成后返回完整音频
func (a *AliyunTTSAdapter) Synthesize(ctx context.Context, text string, options *domain.SynthesizeOptions) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	// 建立 WebSocket 连接
	conn, err := a.connectWebSocket(ctx)
	if err != nil {
		return nil, fmt.Errorf("aliyun tts connect websocket failed: %w", err)
	}
	defer conn.Close()

	// 生成任务ID
	taskID := uuid.New().String()
	// 发送run-task指令
	err = a.sendRunTask(ctx, conn, taskID)
	if err != nil {
		return nil, fmt.Errorf("send run-task failed: %w", err)
	}
	// 等待task-started事件
	err = a.waitTaskStarted(ctx, conn, taskID)
	if err != nil {
		return nil, fmt.Errorf("wait task-started failed: %w", err)
	}
	errCh := make(chan error, 1)
	resultCh := make(chan []byte, 1)
	// 启动结果接收器
	a.startResultReceiver(ctx, conn, taskID, resultCh, errCh)
	// 发送待合成的文本
	err = a.sendContinueTask(ctx, conn, taskID, text)
	if err != nil {
		return nil, fmt.Errorf("send continue-task failed: %w", err)
	}
	// 延迟发送finish-task
	time.Sleep(500 * time.Millisecond)
	// 发送finish-task指令
	err = a.sendFinishTask(ctx, conn, taskID)
	if err != nil {
		return nil, fmt.Errorf("send finish-task failed: %w", err)
	}
	// 等待合成任务完成或失败
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("synthesize audio canceled: %w", ctx.Err())
	case audioData := <-resultCh:
		return audioData, nil
	case err := <-errCh:
		return nil, fmt.Errorf("synthesize audio failed: %w", err)
	}

}

// ===================== WebSocket 连接管理 =====================
// connectWebSocket 建立 WebSocket 连接
func (a *AliyunTTSAdapter) connectWebSocket(ctx context.Context) (*websocket.Conn, error) {
	header := make(http.Header)
	header.Set("Authorization", fmt.Sprintf("bearer %s", a.apiKey))
	header.Set("X-DashScope-DataInspection", "enable")

	dialer := websocket.Dialer{
		HandshakeTimeout: defaultConnectTimeout,
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	conn, _, err := dialer.DialContext(ctx, a.wsURL, header)
	if err != nil {
		logger.ErrorContext(ctx, "[AliyunTTS] WebSocket connect failed", "url", a.wsURL, "error", err)
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	logger.InfoContext(ctx, "[AliyunTTS] WebSocket connected", "url", a.wsURL)
	return conn, nil
}

// ===================== 发送指令 =====================

// sendRunTask 发送 run-task 指令，启动合成任务
func (a *AliyunTTSAdapter) sendRunTask(ctx context.Context, conn *websocket.Conn, taskID string) error {
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
			Model:      a.model,
			Parameters: a.defaults,
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
	err = conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		return fmt.Errorf("send run-task failed: %w", err)
	}

	logger.InfoContext(ctx, "[AliyunTTS] Sent run-task",
		"task_id", taskID,
		"model", a.model,
		"voice", a.defaults.Voice,
		"format", a.defaults.Format,
		"sample_rate", a.defaults.SampleRate,
		"volume", a.defaults.Volume,
		"rate", a.defaults.Rate,
		"pitch", a.defaults.Pitch,
	)
	return nil
}

// sendContinueTask 发送 continue-task 指令（推送文本）
func (a *AliyunTTSAdapter) sendContinueTask(ctx context.Context, conn *websocket.Conn, taskID string, text string) error {
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

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	err = conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		return fmt.Errorf("send continue-task failed: %w", err)
	}

	logger.InfoContext(ctx, "[AliyunTTS] Sent continue-task",
		"task_id", taskID,
		"text", text,
	)
	return nil
}

// sendFinishTask 发送 finish-task 指令
func (a *AliyunTTSAdapter) sendFinishTask(ctx context.Context, conn *websocket.Conn, taskID string) error {
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

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	err = conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		return fmt.Errorf("send finish-task failed: %w", err)
	}

	logger.InfoContext(ctx, "[AliyunTTS] Sent finish-task", "task_id", taskID)
	return nil
}

// waitTaskStarted 等待 task-started 事件
func (a *AliyunTTSAdapter) waitTaskStarted(ctx context.Context, conn *websocket.Conn, taskID string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTaskStartTimeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			// 如果是父ctx取消，直接返回错误
			if ctx.Err() != nil {
				return fmt.Errorf("wait task-started canceled: %w", ctx.Err())
			}
			// 如果是超时，返回超时错误
			return fmt.Errorf("wait task-started timeout after %v", defaultTaskStartTimeout)
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message while waiting task-started: %w", err)
		}

		var event wsEvent
		if err := json.Unmarshal(message, &event); err != nil {
			logger.ErrorContext(ctx, "[AliyunTTS] Parse event failed while waiting task-started: %v", err)
			continue
		}

		switch event.Header.Event {
		case eventTaskStarted:
			logger.InfoContext(ctx, "[AliyunTTS] Task started", "taskID", taskID)
			return nil
		case eventTaskFailed:
			errMsg := event.Header.ErrorMessage
			if errMsg == "" {
				errMsg = "unknown error"
			}
			return fmt.Errorf("task failed before starting: code=%s, message=%s",
				event.Header.ErrorCode, errMsg)
		default:
			logger.InfoContext(ctx, "[AliyunTTS] Unexpected event while waiting task-started", "event", event.Header.Event)
		}
	}
}

// startResultReceiver 启动结果接收器
func (a *AliyunTTSAdapter) startResultReceiver(ctx context.Context, conn *websocket.Conn, taskID string, resultCh chan []byte, errCh chan error) {
	go func() {
		defer close(resultCh)
		defer close(errCh)
		audioBuffer := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					logger.InfoContext(ctx, "[AliyunTTS] WebSocket closed", "task_id", taskID)
					errCh <- fmt.Errorf("websocket closed: %w", err)
					return
				}
				logger.ErrorContext(ctx, "[AliyunTTS] Read message failed", "task_id", taskID, "error", err)
				errCh <- fmt.Errorf("read message failed: %w", err)
				return
			}
			// 处理二进制消息（音频数据块）
			if messageType == websocket.BinaryMessage {
				audioBuffer = append(audioBuffer, message...)

				logger.InfoContext(ctx, "[AliyunTTS] Received audio chunk",
					"bytes", len(message),
					"task_id", taskID,
				)
				continue
			}
			// 处理文本消息（事件）
			var event wsEvent
			if err := json.Unmarshal(message, &event); err != nil {
				logger.ErrorContext(ctx, "[AliyunTTS] Parse event failed",
					"error", err,
					"task_id", taskID,
				)
				continue
			}
			switch event.Header.Event {
			case eventResultGenerated:
				logger.DebugContext(ctx, "[AliyunTTS] Result generated",
					"task_id", taskID,
				)

			case eventTaskFinished:
				logger.InfoContext(ctx, "[AliyunTTS] Task finished", "task_id", taskID)
				if event.Payload.Usage != nil {
					logger.InfoContext(ctx, "[AliyunTTS] Usage info",
						"characters", event.Payload.Usage.Characters,
						"duration", event.Payload.Usage.Duration,
						"task_id", taskID,
					)
				}
				resultCh <- audioBuffer
				return

			case eventTaskFailed:
				errMsg := event.Header.ErrorMessage
				if errMsg == "" {
					errMsg = "unknown error"
				}
				logger.ErrorContext(ctx, "[AliyunTTS] Task failed",
					"error_code", event.Header.ErrorCode,
					"error_message", errMsg,
					"task_id", taskID,
				)
				errCh <- fmt.Errorf("task failed: code=%s, message=%s",
					event.Header.ErrorCode, errMsg)
				return

			default:
				logger.ErrorContext(ctx, "[AliyunTTS] Unknown event",
					"event", event.Header.Event,
					"task_id", taskID,
				)
			}
		}
	}()
}

// SynthesizeToFile 合成语音并保存到本地文件
func (a *AliyunTTSAdapter) SynthesizeToFile(ctx context.Context, text string, outputPath string, options *domain.SynthesizeOptions) error {
	if text == "" {
		return fmt.Errorf("text cannot be empty")
	}
	if outputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	// 先合成获取音频数据
	audioData, err := a.Synthesize(ctx, text, options)
	if err != nil {
		return err
	}

	// 写入文件
	if err := os.WriteFile(outputPath, audioData, 0644); err != nil {
		return fmt.Errorf("write audio file failed: %w", err)
	}

	slog.Info("[AliyunTTS] Audio saved to file",
		"path", outputPath,
		"bytes", len(audioData),
	)

	return nil
}

// SynthesizeStream 流式合成语音（实时返回音频片段）
// 音频块通过 audioChan 实时推送，合成完成后关闭 channel
func (a *AliyunTTSAdapter) SynthesizeStream(ctx context.Context, text string, options *domain.SynthesizeOptions, audioChan chan<- []byte) error {
	if text == "" {
		close(audioChan)
		return fmt.Errorf("text cannot be empty")
	}

	err := a.client.synthesizeStream(ctx, []string{text}, options, audioChan)
	if err != nil {
		return fmt.Errorf("aliyun tts stream failed: %w", err)
	}

	return nil
}

// SynthesizeMultiple 批量合成（多段文本拼接为一个音频）
// 通过 WebSocket 的 continue-task 机制连续发送多段文本
func (a *AliyunTTSAdapter) SynthesizeMultiple(ctx context.Context, texts []string, options *domain.SynthesizeOptions) ([]byte, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	// 过滤空文本
	validTexts := make([]string, 0, len(texts))
	for _, t := range texts {
		if t != "" {
			validTexts = append(validTexts, t)
		}
	}
	if len(validTexts) == 0 {
		return nil, fmt.Errorf("all texts are empty")
	}

	audioData, err := a.client.synthesize(ctx, validTexts, options)
	if err != nil {
		return nil, fmt.Errorf("aliyun tts synthesize multiple failed: %w", err)
	}

	return audioData, nil
}

// Close 关闭客户端，释放资源
func (a *AliyunTTSAdapter) Close() error {
	return a.client.close()
}
