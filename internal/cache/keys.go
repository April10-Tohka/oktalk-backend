// Package cache 提供 Redis 缓存层封装
package cache

import (
	"crypto/md5"
	"fmt"
)

// ===================== Redis Key 常量 =====================
// 格式规范: {模块}:{资源}:{标识}
// Key 最大长度不超过 200 字符

// 任务类
const (
	KeyTaskMeta    = "task:meta:%s"    // task:meta:{task_id}
	KeyTaskPending = "task:pending:%s" // task:pending:{type}
)

// 结果类
const (
	KeyChatResult   = "chat:result:%s"     // chat:result:{conversation_id}
	KeyEvalResult   = "evaluate:result:%s" // evaluate:result:{eval_id}
	KeyReportResult = "report:result:%s"   // report:result:{report_id}
)

// 历史列表类
const (
	KeyChatHistory   = "chat:history:%s:%d"     // chat:history:{user_id}:{page}
	KeyEvalHistory   = "evaluate:history:%s:%d" // evaluate:history:{user_id}:{page}
	KeyReportHistory = "report:history:%s:%d"   // report:history:{user_id}:{page}
)

// 示范音频类（全局共享）
const (
	KeyDemoAudioWord     = "demo:audio:word:%s:%s"     // demo:audio:word:{word}:{voice}
	KeyDemoAudioSentence = "demo:audio:sentence:%s:%s" // demo:audio:sentence:{hash}:{voice}
)

// LLM 缓存类
const (
	KeyLLMFeedback = "llm:feedback:%s:%d:%s" // llm:feedback:{level}:{score}:{word}
	KeyLLMReport   = "llm:report:%s:%s:%s"   // llm:report:{user_id}:{period}:{stats_hash}
)

// TTS 缓存类
const (
	KeyTTSAudio = "tts:audio:%s:%s:%s" // tts:audio:{text_hash}:{voice}:{options_hash}
)

// 会话记录类
const (
	KeyChatSession = "chat:session:%s" // chat:session:{conversation_id}
)

// 用户信息类
const (
	KeyUserInfo = "user:info:%s" // user:info:{user_id}
)

// 学习文本类
const (
	KeyTextContent = "text:content:%s" // text:content:{text_id}
)

// ===================== 工具函数 =====================

// MD5Hash 对文本取 MD5 摘要，用于长文本 Key 参数
func MD5Hash(text string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(text)))
}
