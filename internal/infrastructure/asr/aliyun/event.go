package aliyun

import (
	"fmt"
	"log"

	"pronunciation-correction-system/internal/domain"
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
func (h *eventHandler) handleEvent(event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	switch event.Header.Event {
	case eventTaskStarted:
		return h.onTaskStarted(event)

	case eventResultGenerate:
		return h.onResultGenerated(event)

	case eventTaskFinished:
		return h.onTaskFinished(event)

	case eventTaskFailed:
		return h.onTaskFailed(event)

	default:
		log.Printf("[AliyunASR] Unknown event: %s, taskID=%s", event.Header.Event, h.taskID)
		return nil, false, nil
	}
}

// onTaskStarted 处理 task-started 事件
func (h *eventHandler) onTaskStarted(event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	log.Printf("[AliyunASR] Task started, taskID=%s", h.taskID)
	// task-started 是内部状态事件，不需要向上层推送
	return nil, false, nil
}

// onResultGenerated 处理 result-generated 事件
// 将识别结果转换为领域层的流式事件
func (h *eventHandler) onResultGenerated(event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	sentence := event.Payload.Output.Sentence

	// 判断是中间结果还是最终结果
	// 当 EndTime 不为 nil 时，表示该句子已识别完成（最终结果）
	eventType := "partial"
	var endTime int64
	if sentence.EndTime != nil {
		eventType = "final"
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
		log.Printf("[AliyunASR] Final result: text=%q, taskID=%s", sentence.Text, h.taskID)
	}

	return streamEvent, false, nil
}

// onTaskFinished 处理 task-finished 事件
func (h *eventHandler) onTaskFinished(event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	log.Printf("[AliyunASR] Task finished, taskID=%s", h.taskID)
	return nil, true, nil
}

// onTaskFailed 处理 task-failed 事件
func (h *eventHandler) onTaskFailed(event *wsEvent) (*domain.ASRStreamEvent, bool, error) {
	errMsg := event.Header.ErrorMessage
	errCode := event.Header.ErrorCode
	if errMsg == "" {
		errMsg = "unknown error"
	}

	log.Printf("[AliyunASR] Task failed: code=%s, message=%s, taskID=%s", errCode, errMsg, h.taskID)

	streamEvent := &domain.ASRStreamEvent{
		Type:  "error",
		Error: fmt.Errorf("asr task failed: code=%s, message=%s", errCode, errMsg),
	}

	return streamEvent, true, fmt.Errorf("asr task failed: code=%s, message=%s", errCode, errMsg)
}
