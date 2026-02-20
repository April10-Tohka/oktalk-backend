package aliyun

import (
	"context"
	"fmt"

	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/pkg/logger"
)

// eventHandler 事件处理器
// 负责将 WebSocket 原始事件转换为领域层事件
type eventHandler struct {
	taskID string
}

// newEventHandler 创建事件处理器
func newEventHandler(taskID string) *eventHandler {
	return &eventHandler{taskID: taskID}
}

// handleEvent 处理 WebSocket 事件
// 返回值:
//   - *domain.ASRStreamEvent: 领域层事件（可能为 nil，表示无需向上层推送）
//   - bool: 是否为终止事件（task-finished 或 task-failed）
//   - error: 处理过程中的错误
func (h *eventHandler) handleEvent(ctx context.Context, event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	switch event.Header.Event {
	case eventTaskStarted:
		return h.onTaskStarted(ctx, event)

	case eventResultGenerate:
		return h.onResultGenerated(ctx, event)

	case eventTaskFinished:
		return h.onTaskFinished(ctx, event)

	case eventTaskFailed:
		return h.onTaskFailed(ctx, event)

	default:
		logger.ErrorContext(ctx, "[AliyunASR] Unknown event: %s, taskID=%s", event.Header.Event, h.taskID)
		return nil, false, nil
	}
}

// onTaskStarted 处理 task-started 事件
func (h *eventHandler) onTaskStarted(ctx context.Context, event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	logger.InfoContext(ctx, "[AliyunASR] Task started, taskID=%s", h.taskID)
	// task-started 是内部状态事件，不需要向上层推送
	return nil, false, nil
}

// onResultGenerated 处理 result-generated 事件
// 将识别结果转换为领域层的流式事件
func (h *eventHandler) onResultGenerated(ctx context.Context, event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	sentence := event.Payload.Output.Sentence

	logger.InfoContext(ctx, "[AliyunASR] Result generated",
		"sentence", logger.Any("sentence", sentence),
		"taskID", h.taskID)
	eventType := "partial"
	// 判断是中间结果还是最终结果
	// 当 SentenceEnd 为 true 时，表示该句子已识别完成（最终结果）
	if sentence.SentenceEnd {
		eventType = "final"
	}

	var endTime int64
	if sentence.EndTime != nil {
		endTime = *sentence.EndTime
	}

	// 转换单词级结果
	words := make([]domain.ASRWordResult, len(sentence.Words))
	for i, w := range sentence.Words {
		var wEndTime int64
		if w.EndTime != nil {
			wEndTime = *w.EndTime
		}
		words[i] = domain.ASRWordResult{
			Text:        w.Text,
			BeginTime:   w.BeginTime,
			EndTime:     wEndTime,
			Punctuation: w.Punctuation,
		}
	}

	streamEvent := &domain.ASRStreamEvent{
		Type:      eventType,
		Text:      sentence.Text,
		BeginTime: sentence.BeginTime,
		EndTime:   endTime,
		Words:     words,
	}

	// 如果有 usage 信息，添加时长
	if event.Payload.Usage != nil {
		streamEvent.Duration = event.Payload.Usage.Duration
	}

	if eventType == "final" {
		logger.InfoContext(ctx, "[AliyunASR] Final result: text=%q, taskID=%s", sentence.Text, h.taskID)
	}

	return streamEvent, false, nil
}

// onTaskFinished 处理 task-finished 事件
func (h *eventHandler) onTaskFinished(ctx context.Context, event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	logger.InfoContext(ctx, "[AliyunASR] Task finished, taskID=%s", h.taskID)
	return nil, true, nil
}

// onTaskFailed 处理 task-failed 事件
func (h *eventHandler) onTaskFailed(ctx context.Context, event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	errMsg := event.Header.ErrorMessage
	errCode := event.Header.ErrorCode
	if errMsg == "" {
		errMsg = "unknown error"
	}

	logger.ErrorContext(ctx, "[AliyunASR] Task failed: code=%s, message=%s, taskID=%s", errCode, errMsg, h.taskID)

	streamEvent := &domain.ASRStreamEvent{
		Type:  "error",
		Error: fmt.Errorf("asr task failed: code=%s, message=%s", errCode, errMsg),
	}

	return streamEvent, true, fmt.Errorf("asr task failed: code=%s, message=%s", errCode, errMsg)
}
