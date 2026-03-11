package aliyun

import (
	"context"
	"fmt"
	"log/slog"

	"pronunciation-correction-system/internal/domain"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ===================== ConnectASR 流式长连接实现 =====================

// ConnectASR 建立流式 ASR 长连接（用于 Free Talk 模式）
// 返回 AudioSender 用于持续推送实时音频，eventCh 接收识别结果（含 VAD 断句）
func (a *AliyunASRAdapter) ConnectASR(ctx context.Context, format string, sampleRate int) (domain.AudioSender, <-chan *domain.ASRStreamEvent, error) {
	return a.client.connectASR(ctx, format, sampleRate)
}

// connectASR 内部实现：建立 WebSocket 连接并启动接收 goroutine
func (c *internalClient) connectASR(ctx context.Context, format string, sampleRate int) (domain.AudioSender, <-chan *domain.ASRStreamEvent, error) {
	// 1. 建立 WebSocket 连接
	conn, err := c.connectWebSocket(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("connect websocket failed: %w", err)
	}

	// 2. 生成任务 ID
	taskID := uuid.New().String()
	handler := newEventHandler(taskID)

	// 3. 发送 run-task 指令
	if err := c.sendRunTask(ctx, conn, taskID, format, sampleRate); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("send run-task failed: %w", err)
	}

	// 4. 等待 task-started 事件
	if err := c.waitTaskStarted(ctx, conn, handler); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("task start failed: %w", err)
	}

	// 5. 创建输出通道
	eventCh := make(chan *domain.ASRStreamEvent, 64)

	// 6. 启动后台接收 goroutine（持续运行直到连接关闭或 ctx 取消）
	go c.receiveResults(ctx, conn, handler, eventCh)

	// 7. 创建 AudioSender
	sender := &audioSenderImpl{
		conn:   conn,
		taskID: taskID,
		client: c,
		ctx:    ctx,
	}

	slog.Info("[AliyunASR] ConnectASR established",
		"task_id", taskID,
		"format", format,
		"sample_rate", sampleRate,
	)

	return sender, eventCh, nil
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
