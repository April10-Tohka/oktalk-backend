// Package handler 提供 Handler 聚合结构
// 用于依赖注入时统一传递所有 Handler
package handler

import "pronunciation-correction-system/internal/handler/freetalk"

// Handlers 所有 HTTP Handler 的聚合
type Handlers struct {
	Auth     *AuthHandler
	User     *UserHandler
	Chat     *ChatHandler
	Evaluate *EvaluateHandler
	Report   *ReportHandler
	System   *SystemHandler
	FreeTalk *freetalk.Handler
	Scene         *SceneHandler
	Pronunciation *PronunciationHandler
}
