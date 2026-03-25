package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// AliyunASRAdapter 阿里云 FUN-ASR 适配器
// 实现 domain.ASRProvider 接口，将领域层调用转换为阿里云 WebSocket API 调用
type AliyunASRAdapter struct {
	apiKey string // DashScope API Key
	wsURL  string // WebSocket 地址
	model  string // 模型名称
	client *internalClient
}

// 编译时检查：确保 AliyunASRAdapter 实现了 domain.ASRProvider 接口
var _ domain.ASRProvider = (*AliyunASRAdapter)(nil)

// NewAliyunASRAdapter 创建阿里云 FUN-ASR 适配器
func NewAliyunASRAdapter(cfg config.AliyunASRConfig) *AliyunASRAdapter {
	wsURL := cfg.Endpoint
	if wsURL == "" {
		wsURL = defaultWSURL
	}

	model := cfg.Model
	if model == "" {
		model = modelFunASR
	}

	logger.Info("[AliyunASR] Adapter initialized", "model", model, "endpoint", wsURL)
	return &AliyunASRAdapter{
		client: newInternalClient(cfg),
		apiKey: cfg.APIKey,
		wsURL:  wsURL,
		model:  model,
	}
}

// RecognizeAudio 同步识别音频数据
// 发送完整音频，等待识别完成后返回汇总结果
func (a *AliyunASRAdapter) RecognizeAudio(ctx context.Context, audioData []byte) (string, error) {
	// 连接WebSocket服务
	conn, err := a.connectWebSocket(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	// 生成任务ID
	taskID := uuid.New().String()
	// 发送run-task指令
	err = a.sendRunTask(ctx, conn, taskID)
	if err != nil {
		return "", fmt.Errorf("send run-task failed: %w", err)
	}
	// 等待task-started事件
	err = a.waitTaskStarted(ctx, conn, taskID)
	if err != nil {
		return "", fmt.Errorf("wait task-started failed: %w", err)
	}
	errCh := make(chan error, 1)
	resultCh := make(chan string, 1)
	// 启动结果接收器
	a.startResultReceiver(ctx, conn, resultCh, errCh)
	// 发送待识别的音频流
	err = a.sendAudioData(ctx, conn, audioData)
	if err != nil {
		return "", fmt.Errorf("send audio data failed: %w", err)
	}
	// 发送finish-task指令
	err = a.sendFinishTask(ctx, conn, taskID)
	if err != nil {
		return "", fmt.Errorf("send finish-task failed: %w", err)
	}
	// 等待识别任务完成或失败
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("recognize audio canceled: %w", ctx.Err())
	case text := <-resultCh:
		return text, nil
	case err := <-errCh:
		return "", fmt.Errorf("recognize audio failed: %w", err)
	}

}

// RecognizeAudioStream 流式识别音频数据
// 发送音频的同时实时返回中间结果和最终结果
func (a *AliyunASRAdapter) RecognizeAudioStream(ctx context.Context, audioData []byte, format string, sampleRate int) (<-chan *domain.ASRStreamEvent, error) {
	return a.client.recognizeStream(ctx, audioData, format, sampleRate)
}

// Close 关闭客户端，释放资源
func (a *AliyunASRAdapter) Close() error {
	return a.client.close()
}

// ======
// connectWebSocket 建立 WebSocket 连接
func (a *AliyunASRAdapter) connectWebSocket(ctx context.Context) (*websocket.Conn, error) {
	header := make(http.Header)
	header.Set("Authorization", fmt.Sprintf("bearer %s", a.apiKey))

	// 使用带超时的 dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: defaultConnectTimeout,
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("connect websocket canceled: %w", ctx.Err())
	default:
	}
	conn, _, err := dialer.DialContext(ctx, a.wsURL, header)
	if err != nil {
		logger.ErrorContext(ctx, "[AliyunASR] WebSocket connect failed: %v", err)
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	logger.InfoContext(ctx, "[AliyunASR] WebSocket connected to ", "url", a.wsURL)
	return conn, nil
}

// startResultReceiver 启动一个异步goroutine
func (a *AliyunASRAdapter) startResultReceiver(ctx context.Context, conn *websocket.Conn, resultCh chan string, errCh chan error) {
	go func() {
		var builder strings.Builder
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, message, err := conn.ReadMessage()
				if err != nil {
					logger.ErrorContext(ctx, "[AliyunASR] Read message failed", "error", err)
					errCh <- fmt.Errorf("read message failed: %v", err)
					return
				}
				logger.InfoContext(ctx, "[AliyunASR] Received message")
				var event wsEvent
				if err := json.Unmarshal(message, &event); err != nil {
					logger.ErrorContext(ctx, "[AliyunASR] Parse event failed", "error", err)
					continue
				}
				switch event.Header.Event {
				case eventResultGenerate:
					logger.InfoContext(ctx, "[AliyunASR] Result generated", "text", event.Payload.Output.Sentence.Text)
					sentence := event.Payload.Output.Sentence
					if sentence.SentenceEnd {
						logger.InfoContext(ctx, "[AliyunASR] Sentence end", "text", sentence.Text)
						builder.WriteString(sentence.Text)
					}
				case eventTaskFinished:
					logger.InfoContext(ctx, "[AliyunASR] Task finished")
					resultCh <- builder.String()
					return
				case eventTaskFailed:
					logger.ErrorContext(ctx, "[AliyunASR] Task failed", "error", event.Header.ErrorMessage)
					errCh <- fmt.Errorf("task failed: %v", event.Header.ErrorMessage)
					return
				default:
					logger.InfoContext(ctx, "[AliyunASR] Unexpected event", "event", event.Header.Event)
					continue
				}
			}
		}
	}()
}

// sendRunTask 发送 run-task 指令，启动识别任务
func (a *AliyunASRAdapter) sendRunTask(ctx context.Context, conn *websocket.Conn, taskID string) error {
	heartbeat := true
	params := wsParams{
		Format:     "pcm", // 默认格式
		SampleRate: 16000, // 默认采样率
		Heartbeat:  &heartbeat,
	}

	event := wsEvent{
		Header: wsHeader{
			Action:    actionRunTask,
			TaskID:    taskID,
			Streaming: streamingDuplex,
		},
		Payload: wsPayload{
			TaskGroup:  taskGroupAudio,
			Task:       taskASR,
			Function:   functionRecognize,
			Model:      a.model,
			Parameters: params,
			Input:      wsInput{},
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal run-task failed: %w", err)
	}
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return fmt.Errorf("send run-task canceled: %w", ctx.Err())
	default:
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("send run-task failed: %w", err)
	}

	logger.InfoContext(ctx, "[AliyunASR] Sent run-task", "taskID", taskID, "model", a.model, "format", params.Format, "sampleRate", params.SampleRate)
	return nil
}

// sendFinishTask 发送 finish-task 指令，通知服务端音频发送完毕
func (a *AliyunASRAdapter) sendFinishTask(ctx context.Context, conn *websocket.Conn, taskID string) error {
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
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return fmt.Errorf("send finish-task canceled: %w", ctx.Err())
	default:
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("send finish-task failed: %w", err)
	}

	logger.InfoContext(ctx, "[AliyunASR] Sent finish-task", "taskID", taskID)
	return nil
}

// waitTaskStarted 等待 task-started 事件
func (a *AliyunASRAdapter) waitTaskStarted(ctx context.Context, conn *websocket.Conn, taskID string) error {
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
			logger.ErrorContext(ctx, "[AliyunASR] Parse event failed while waiting task-started: %v", err)
			continue
		}

		switch event.Header.Event {
		case eventTaskStarted:
			logger.InfoContext(ctx, "[AliyunASR] Task started", "taskID", taskID)
			return nil
		case eventTaskFailed:
			errMsg := event.Header.ErrorMessage
			if errMsg == "" {
				errMsg = "unknown error"
			}
			return fmt.Errorf("task failed before starting: code=%s, message=%s",
				event.Header.ErrorCode, errMsg)
		default:
			logger.InfoContext(ctx, "[AliyunASR] Unexpected event while waiting task-started", "event", event.Header.Event)
		}
	}
}

// sendAudio 发送音频数据
func (a *AliyunASRAdapter) sendAudioData(
	ctx context.Context,
	conn *websocket.Conn,
	audioData []byte,
) error {
	// 分片发送音频数据
	chunkSize := defaultSendChunkSize
	// 每次发送间隔，模拟实时音频流速率
	interval := time.Duration(defaultSendInterval) * time.Millisecond

	for offset := 0; offset < len(audioData); offset += chunkSize {
		select {
		case <-ctx.Done():
			return fmt.Errorf("audio sending canceled: %w", ctx.Err())
		default:
		}

		end := offset + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("audio sending canceled: %w", ctx.Err())
		default:
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, audioData[offset:end]); err != nil {
			return fmt.Errorf("send audio chunk failed: %w", err)
		}

		// 模拟实时音频流速率
		if offset+chunkSize < len(audioData) {
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return fmt.Errorf("audio sending canceled: %w", ctx.Err())
			}
		}
	}
	logger.InfoContext(ctx, "[AliyunASR] Audio data sent successfully", "total", len(audioData))
	return nil
}
