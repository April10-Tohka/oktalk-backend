package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/domain"
)

// ===================== 默认配置 =====================

const (
	// defaultWSURL 默认 WebSocket 地址（北京地域）
	defaultWSURL = "wss://dashscope.aliyuncs.com/api-ws/v1/inference/"

	// defaultConnectTimeout WebSocket 连接超时
	defaultConnectTimeout = 10 * time.Second

	// defaultTaskStartTimeout 等待 task-started 事件的超时
	defaultTaskStartTimeout = 10 * time.Second
)

// ===================== 客户端定义 =====================

// internalClient 阿里云 FUN-ASR 内部 WebSocket 客户端
// 封装了与 DashScope WebSocket API 的所有通信细节
type internalClient struct {
	apiKey string // DashScope API Key
	wsURL  string // WebSocket 地址
	model  string // 模型名称
}

// newInternalClient 根据配置创建内部客户端
func newInternalClient(cfg config.AliyunASRConfig) *internalClient {
	wsURL := cfg.Endpoint
	if wsURL == "" {
		wsURL = defaultWSURL
	}

	model := cfg.Model
	if model == "" {
		model = modelFunASR
	}

	log.Printf("[AliyunASR] Client initialized, model=%s, endpoint=%s", model, wsURL)

	return &internalClient{
		apiKey: cfg.APIKey,
		wsURL:  wsURL,
		model:  model,
	}
}

// ===================== 核心方法 =====================

// recognizeAudio 同步识别音频数据
// 发送完整音频，等待所有结果返回后汇总
func (c *internalClient) recognizeAudio(ctx context.Context, audioData []byte, format string, sampleRate int) (*domain.ASRResult, error) {
	start := time.Now()

	// 通过流式接口获取所有事件
	eventCh, err := c.recognizeStream(ctx, audioData, format, sampleRate)
	if err != nil {
		return nil, err
	}

	// 收集所有 final 结果，拼接为完整文本
	var (
		fullText string
		allWords []domain.ASRWordResult
		duration int
	)

	for event := range eventCh {
		if event.Error != nil {
			return nil, event.Error
		}
		if event.Type == "final" {
			fullText += event.Text
			allWords = append(allWords, event.Words...)
			if event.Duration > 0 {
				duration = event.Duration
			}
		}
	}

	elapsed := time.Since(start)
	log.Printf("[AliyunASR] Recognition completed: textLen=%d, duration=%ds, elapsed=%v",
		len(fullText), duration, elapsed)

	return &domain.ASRResult{
		Text:     fullText,
		Duration: duration,
		Words:    allWords,
	}, nil
}

// recognizeStream 流式识别音频数据
// 启动 WebSocket 会话，并发发送音频和接收结果
func (c *internalClient) recognizeStream(ctx context.Context, audioData []byte, format string, sampleRate int) (<-chan *domain.ASRStreamEvent, error) {
	// 1. 建立 WebSocket 连接
	conn, err := c.connectWebSocket(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect websocket: %w", err)
	}

	// 2. 生成任务 ID
	taskID := uuid.New().String()
	handler := newEventHandler(taskID)

	// 3. 发送 run-task 指令
	if err := c.sendRunTask(conn, taskID, format, sampleRate); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send run-task: %w", err)
	}

	// 4. 等待 task-started 事件
	if err := c.waitTaskStarted(ctx, conn, handler); err != nil {
		conn.Close()
		return nil, fmt.Errorf("task start failed: %w", err)
	}

	// 5. 创建输出通道
	eventCh := make(chan *domain.ASRStreamEvent, 64)

	// 6. 启动并发 goroutine：发送音频 + 接收结果
	go c.runSession(ctx, conn, handler, taskID, audioData, eventCh)

	return eventCh, nil
}

// ===================== WebSocket 连接管理 =====================

// connectWebSocket 建立 WebSocket 连接
func (c *internalClient) connectWebSocket(ctx context.Context) (*websocket.Conn, error) {
	header := make(http.Header)
	header.Set("Authorization", fmt.Sprintf("bearer %s", c.apiKey))

	// 使用带超时的 dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: defaultConnectTimeout,
	}

	conn, _, err := dialer.DialContext(ctx, c.wsURL, header)
	if err != nil {
		log.Printf("[AliyunASR] WebSocket connect failed: %v", err)
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	log.Printf("[AliyunASR] WebSocket connected to %s", c.wsURL)
	return conn, nil
}

// ===================== 发送指令 =====================

// sendRunTask 发送 run-task 指令，启动识别任务
func (c *internalClient) sendRunTask(conn *websocket.Conn, taskID, format string, sampleRate int) error {
	event := wsEvent{
		Header: wsHeader{
			Action:    actionRunTask,
			TaskID:    taskID,
			Streaming: streamingDuplex,
		},
		Payload: wsPayload{
			TaskGroup: taskGroupAudio,
			Task:      taskASR,
			Function:  functionRecognize,
			Model:     c.model,
			Parameters: wsParams{
				Format:     format,
				SampleRate: sampleRate,
			},
			Input: wsInput{},
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal run-task failed: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("send run-task failed: %w", err)
	}

	log.Printf("[AliyunASR] Sent run-task: taskID=%s, model=%s, format=%s, sampleRate=%d",
		taskID, c.model, format, sampleRate)
	return nil
}

// sendFinishTask 发送 finish-task 指令，通知服务端音频发送完毕
func (c *internalClient) sendFinishTask(conn *websocket.Conn, taskID string) error {
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

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("send finish-task failed: %w", err)
	}

	log.Printf("[AliyunASR] Sent finish-task: taskID=%s", taskID)
	return nil
}

// ===================== 等待事件 =====================

// waitTaskStarted 等待 task-started 事件
// 在收到 task-started 之前，不能发送音频数据
func (c *internalClient) waitTaskStarted(ctx context.Context, conn *websocket.Conn, handler *eventHandler) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTaskStartTimeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("wait task-started timeout after %v", defaultTaskStartTimeout)
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message while waiting task-started: %w", err)
		}

		var event wsEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("[AliyunASR] Parse event failed while waiting task-started: %v", err)
			continue
		}

		switch event.Header.Event {
		case eventTaskStarted:
			log.Printf("[AliyunASR] Task started, taskID=%s", handler.taskID)
			return nil
		case eventTaskFailed:
			errMsg := event.Header.ErrorMessage
			if errMsg == "" {
				errMsg = "unknown error"
			}
			return fmt.Errorf("task failed before starting: code=%s, message=%s",
				event.Header.ErrorCode, errMsg)
		default:
			log.Printf("[AliyunASR] Unexpected event while waiting task-started: %s", event.Header.Event)
		}
	}
}

// ===================== 会话管理 =====================

// runSession 运行一次完整的 ASR 会话
// 并发执行：发送音频数据 + 接收识别结果
func (c *internalClient) runSession(
	ctx context.Context,
	conn *websocket.Conn,
	handler *eventHandler,
	taskID string,
	audioData []byte,
	eventCh chan<- *domain.ASRStreamEvent,
) {
	defer close(eventCh)
	defer conn.Close()

	var wg sync.WaitGroup

	// 用于通知发送方停止
	sendCtx, sendCancel := context.WithCancel(ctx)
	defer sendCancel()

	// 用于通知接收方停止
	receiveDone := make(chan struct{})

	// 启动接收 goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(receiveDone)
		c.receiveResults(ctx, conn, handler, eventCh)
	}()

	// 启动发送 goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.sendAudioAndFinish(sendCtx, conn, taskID, audioData)
	}()

	// 等待接收完成（task-finished/task-failed），则可以取消发送
	select {
	case <-receiveDone:
		sendCancel()
	case <-ctx.Done():
		sendCancel()
	}

	wg.Wait()
}

// ===================== 发送音频 =====================

// sendAudioAndFinish 发送音频数据并发送 finish-task
func (c *internalClient) sendAudioAndFinish(
	ctx context.Context,
	conn *websocket.Conn,
	taskID string,
	audioData []byte,
) {
	// 分片发送音频数据
	chunkSize := defaultSendChunkSize
	interval := time.Duration(defaultSendInterval) * time.Millisecond

	for offset := 0; offset < len(audioData); offset += chunkSize {
		select {
		case <-ctx.Done():
			log.Printf("[AliyunASR] Audio sending canceled, taskID=%s", taskID)
			return
		default:
		}

		end := offset + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}

		if err := conn.WriteMessage(websocket.BinaryMessage, audioData[offset:end]); err != nil {
			log.Printf("[AliyunASR] Send audio chunk failed: %v, taskID=%s", err, taskID)
			return
		}

		// 模拟实时音频流速率
		if offset+chunkSize < len(audioData) {
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return
			}
		}
	}

	log.Printf("[AliyunASR] Audio data sent, total=%d bytes, taskID=%s", len(audioData), taskID)

	// 发送 finish-task 指令
	if err := c.sendFinishTask(conn, taskID); err != nil {
		log.Printf("[AliyunASR] Send finish-task failed: %v, taskID=%s", err, taskID)
	}
}

// ===================== 接收结果 =====================

// receiveResults 持续接收 WebSocket 消息并处理事件
// 直到收到 task-finished 或 task-failed 事件
func (c *internalClient) receiveResults(
	ctx context.Context,
	conn *websocket.Conn,
	handler *eventHandler,
	eventCh chan<- *domain.ASRStreamEvent,
) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AliyunASR] Receive canceled, taskID=%s", handler.taskID)
			eventCh <- &domain.ASRStreamEvent{
				Type:  "error",
				Error: ctx.Err(),
			}
			return
		default:
		}

		// 设置读取超时，防止永久阻塞
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		_, message, err := conn.ReadMessage()
		if err != nil {
			// 检查是否为正常关闭
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			log.Printf("[AliyunASR] Read message failed: %v, taskID=%s", err, handler.taskID)
			eventCh <- &domain.ASRStreamEvent{
				Type:  "error",
				Error: fmt.Errorf("read websocket message failed: %w", err),
			}
			return
		}

		// 解析事件
		var event wsEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("[AliyunASR] Parse event failed: %v, taskID=%s", err, handler.taskID)
			continue
		}

		// 处理事件
		streamEvent, isTerminal, _ := handler.handleEvent(&event)

		// 推送领域事件到通道
		if streamEvent != nil {
			select {
			case eventCh <- streamEvent:
			case <-ctx.Done():
				return
			}
		}

		// 终止事件：task-finished 或 task-failed
		if isTerminal {
			return
		}
	}
}

// close 关闭客户端
func (c *internalClient) close() error {
	log.Println("[AliyunASR] Client closed")
	return nil
}
