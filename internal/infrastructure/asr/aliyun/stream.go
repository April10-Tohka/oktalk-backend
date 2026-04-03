package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"pronunciation-correction-system/internal/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ===================== ConnectASR 流式长连接实现 =====================

// ConnectASR 建立流式 ASR 长连接（用于 Free Talk 模式）
// 返回 AudioSender 用于持续推送实时音频，eventCh 接收识别结果（含 VAD 断句）
func (a *AliyunASRAdapter) ConnectASR(ctx context.Context) (*websocket.Conn, error) {
	// 连接WebSocket服务
	conn, err := a.connectWebSocket(ctx)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// connectASR 内部实现：建立 WebSocket 连接并启动接收 goroutine
func (c *internalClient) connectASR(ctx context.Context, audioChan <-chan []byte, llmInputChan chan<- string, ttsNewTurnChan chan<- struct{}) error {
	// 1. 建立 WebSocket 连接
	conn, err := c.connectWebSocket(ctx)
	if err != nil {
		return fmt.Errorf("connect websocket failed: %w", err)
	}

	// 2. 生成任务 ID
	taskID := uuid.New().String()
	handler := newEventHandler(taskID)

	// 3. 发送 run-task 指令
	if err := c.sendRunTask(ctx,
		conn,
		taskID,
		WithHeartbeat(true),
		WithMultiThresholdModeEnabled(true),
	); err != nil {
		conn.Close()
		return fmt.Errorf("send run-task failed: %w", err)
	}

	// 4. 等待 task-started 事件
	if err := c.waitTaskStarted(ctx, conn, handler); err != nil {
		conn.Close()
		return fmt.Errorf("task start failed: %w", err)
	}

	// 5. 启动并发 goroutine：发送音频
	go func(ctx context.Context, conn *websocket.Conn, audioChan <-chan []byte) {
		for audio := range audioChan {
			if err := conn.WriteMessage(websocket.BinaryMessage, audio); err != nil {
				logger.ErrorContext(ctx, "[AliyunASR] Send audio failed: %v", err)
			}
		}
	}(ctx, conn, audioChan)

	// 6. 启动并发 goroutine：接收结果
	go func(ctx context.Context, conn *websocket.Conn, handler *eventHandler, llmInputChan chan<- string, ttsNewTurnChan chan<- struct{}) {
		// 设置websocket读取超时时间
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, message, err := conn.ReadMessage()
			if err != nil {
				// 1. 检查是否为超时错误
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					//  读取超时，可以重试或继续下一轮循环
					logger.WarnContext(ctx, "[AliyunASR] Read message timeout, retry...", "task_id", handler.taskID)
					continue
				}
				// 2. 检查是否为正常关闭
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.InfoContext(ctx, "[AliyunASR] Connection closed normally", "task_id", handler.taskID)
					return
				}
				// 3. 检查是否为意外关闭
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.ErrorContext(ctx, "[AliyunASR] Connection closed unexpectedly", "task_id", handler.taskID)
					return
				}
				// 4. 其他错误
				logger.ErrorContext(ctx, "[AliyunASR] Read message failed ", "task_id", handler.taskID, "error", err)
				return
			}
			// 重置websocket读取超时时间
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			// 解析事件
			var event wsEvent
			if err := json.Unmarshal(message, &event); err != nil {
				logger.ErrorContext(ctx, "[AliyunASR] Parse event failed: %v, taskID=%s", err, handler.taskID)
				continue
			}
			// 处理事件
			switch event.Header.Event {
			case eventResultGenerate:
				sentence := event.Payload.Output.Sentence
				logger.InfoContext(ctx, "[AliyunASR] Result generated", "task_id", handler.taskID)
				if sentence.SentenceEnd {
					logger.InfoContext(ctx, "[AliyunASR] Sentence end", "sentence", logger.Any("sentence", sentence), "task_id", handler.taskID)
					llmInputChan <- sentence.Text
					ttsNewTurnChan <- struct{}{}
				}
			case eventTaskFailed:
				errMsg := event.Header.ErrorMessage
				errCode := event.Header.ErrorCode
				if errMsg == "" {
					errMsg = "unknown error"
				}
				logger.ErrorContext(ctx, "[AliyunASR] Task failed", "errCode", errCode, "errMsg", errMsg, "task_id", handler.taskID)
				close(llmInputChan)
				close(ttsNewTurnChan)
				return
			default:
				logger.WarnContext(ctx, "[AliyunASR] Unknown event", "event", event.Header.Event, "task_id", handler.taskID)
				continue
			}
		}
	}(ctx, conn, handler, llmInputChan, ttsNewTurnChan)

	slog.Info("[AliyunASR] ConnectASR established",
		"task_id", taskID,
	)

	return nil
}

// ===================== AudioSender 实现 =====================

// audioSenderImpl 音频发送器实现
// 用于 Free Talk 模式下持续向 FunASR 发送实时音频数据
type audioSenderImpl struct {
	conn   *websocket.Conn
	taskID string
	client *internalClient
	ctx    context.Context
}

// SendAudio 发送一块音频数据到 ASR
// gorilla/websocket 支持一个并发 reader 和一个并发 writer，
// 此方法由 appReaderGoroutine 调用，与 receiveResults goroutine 不冲突
func (s *audioSenderImpl) SendAudio(data []byte) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}

	if err := s.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("send audio to ASR failed: %w", err)
	}
	return nil
}

// Close 发送 finish-task 并关闭 WebSocket 连接
func (s *audioSenderImpl) Close() error {
	// 发送 finish-task 通知 ASR 服务端音频结束
	if err := s.client.sendFinishTask(s.ctx, s.conn, s.taskID); err != nil {
		slog.Warn("[AliyunASR] Send finish-task on close failed",
			"error", err,
			"task_id", s.taskID,
		)
	}

	// 关闭 WebSocket 连接
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("close ASR websocket failed: %w", err)
	}

	slog.Info("[AliyunASR] AudioSender closed", "task_id", s.taskID)
	return nil
}
