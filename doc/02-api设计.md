# 【阶段 4：API 接口设计】

---

## 一、API 设计总体规范

### 1.1 通用说明

- **API 前缀**：`/api/v1`
- **通用响应结构**：所有接口返回统一格式
- **认证方式**：请求头 `Authorization: Bearer {token}`
- **Content-Type**：`application/json`
- **请求超时**：30 秒
- **分页规范**：`page`（页码，从 1 开始）、`page_size`（每页条数，默认 20）

### 1.2 统一响应结构

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 状态码：200 成功，400 参数错误，401 未认证，403 禁止，500 服务器错误 |
| `message` | string | 结果说明 |
| `data` | object / array / null | 业务数据 |

### 1.3 分页响应结构（列表接口通用）

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 100,
      "total_pages": 5
    }
  }
}
```

---

## 二、AI 语音对话 API（Chat 模块）

### 2.1 功能说明

用户通过语音与 AI 进行对话，后端完成三个阶段的处理：
1. **ASR**：语音识别用户的语音为文本
2. **LLM**：调用通义千问生成 AI 回复
3. **TTS**：合成 AI 回复为语音

---

### 2.2 接口清单

#### **2.2.0 同步语音对话（MVP）**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/chat/MVP` |
| **功能说明** | 用户上传语音文件，后端同步执行 ASR + LLM + TTS，并直接返回 TTS 生成的音频流（用于快速体验核心对话能力的 MVP 接口） |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `audio_file` | file | ✓ | 用户语音文件（WAV / MP3，最大 10MB） |
| `audio_type` | string | ✓ | 音频格式：`wav` / `mp3` |
| `conversation_type` | string | ✓ | 对话类型：`free_talk` / `question_answer` |
| `difficulty_level` | string | ✗ | 难度等级：`beginner` / `intermediate` / `advanced` |

**响应说明**：

- 成功：`HTTP 200`，`Content-Type: audio/mpeg`，响应体为 TTS 生成的音频二进制流，不使用通用 JSON 包装。
- 失败：返回通用 JSON 错误结构（`code` ≠ 200，`message` 描述错误原因）。

> 说明：该接口用于“先跑通一条完整链路（Fun-ASR → Qwen → CosyVoice）”的 MVP，后续可逐步替换为异步任务型接口（见 2.2.1+）。

#### **2.2.1 提交语音对话请求**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/chat/submit` |
| **功能说明** | 用户上传语音文件，后端异步处理 ASR + LLM + TTS，返回任务 ID 供轮询 |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `audio_file` | file | ✓ | 语音文件（WAV / MP3，最大 10MB） |
| `audio_type` | string | ✓ | 音频格式：`wav` / `mp3` |
| `session_id` | string | ✓ | 会话 ID，用于管理对话历史 |
| `user_language` | string | ✗ | 用户语言，默认 `zh_CN`（中文）；支持 `en_US`（英文） |
| `topic_id` | string | ✗ | 话题 ID（可选），用于对话上下文约束 |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "task_id": "chat_20240115_abc123def456",
    "session_id": "sess_20240115_xyz789",
    "status": "pending",
    "message": "语音对话任务已提交，请轮询查询结果"
  }
}
```

**返回字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `task_id` | string | 任务 ID，用于查询处理结果 |
| `session_id` | string | 会话 ID，确保对话连贯性 |
| `status` | string | 任务状态：`pending` / `processing` / `success` / `failed` |
| `message` | string | 提示信息 |

---

#### **2.2.2 查询语音对话处理结果**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/chat/result/{task_id}` |
| **功能说明** | 客户端轮询查询语音对话的处理结果 |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `task_id` | string | 任务 ID（从提交接口返回） |

**返回结构示例（处理中）**：

```json
{
  "code": 200,
  "message": "processing",
  "data": {
    "task_id": "chat_20240115_abc123def456",
    "status": "processing",
    "progress": 50,
    "current_stage": "generating_response"
  }
}
```

**返回结构示例（处理完成）**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "task_id": "chat_20240115_abc123def456",
    "status": "success",
    "session_id": "sess_20240115_xyz789",
    "user_input": {
      "text": "你好，今天天气如何",
      "duration_ms": 2500
    },
    "ai_response": {
      "text": "你好！今天天气晴朗，温度约 15 度，适合外出。",
      "audio_url": "https://oss.example.com/audio/chat_20240115_abc123def456.mp3",
      "duration_ms": 4200
    },
    "created_at": "2024-01-15T10:30:45Z",
    "feedback_url": "/api/v1/chat/feedback"
  }
}
```

**返回字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `task_id` | string | 任务 ID |
| `status` | string | 任务状态 |
| `progress` | int | 进度百分比（仅处理中返回） |
| `current_stage` | string | 当前阶段：`asr` / `llm` / `tts` / `completed` |
| `user_input.text` | string | 用户语音识别后的文本 |
| `user_input.duration_ms` | int | 用户语音时长（毫秒） |
| `ai_response.text` | string | AI 生成的回复文本 |
| `ai_response.audio_url` | string | AI 回复的语音 URL |
| `ai_response.duration_ms` | int | AI 语音时长（毫秒） |
| `created_at` | string | 任务创建时间（ISO 8601） |

**返回结构示例（处理失败）**：

```json
{
  "code": 200,
  "message": "failed",
  "data": {
    "task_id": "chat_20240115_abc123def456",
    "status": "failed",
    "error_stage": "asr",
    "error_message": "语音识别失败，请检查音频质量"
  }
}
```

---

#### **2.2.3 获取对话历史**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/chat/history/{session_id}` |
| **功能说明** | 获取指定会话的对话历史记录 |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `session_id` | string | 会话 ID |

**查询参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `page` | int | ✗ | 页码，默认 1 |
| `page_size` | int | ✗ | 每页条数，默认 20 |
| `order` | string | ✗ | 排序：`asc`（升序，默认）/ `desc`（降序） |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "sess_20240115_xyz789",
    "items": [
      {
        "turn": 1,
        "user_text": "你好",
        "user_audio_url": "https://oss.example.com/audio/user_1.mp3",
        "ai_text": "你好，很高兴认识你",
        "ai_audio_url": "https://oss.example.com/audio/ai_1.mp3",
        "created_at": "2024-01-15T10:30:45Z"
      },
      {
        "turn": 2,
        "user_text": "今天天气如何",
        "user_audio_url": "https://oss.example.com/audio/user_2.mp3",
        "ai_text": "今天天气晴朗，温度约 15 度",
        "ai_audio_url": "https://oss.example.com/audio/ai_2.mp3",
        "created_at": "2024-01-15T10:32:10Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 2,
      "total_pages": 1
    }
  }
}
```

**返回字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `session_id` | string | 会话 ID |
| `turn` | int | 对话轮次 |
| `user_text` | string | 用户文本 |
| `user_audio_url` | string | 用户语音 URL |
| `ai_text` | string | AI 回复文本 |
| `ai_audio_url` | string | AI 语音 URL |
| `created_at` | string | 创建时间 |

---

#### **2.2.4 删除对话会话**

| 项目 | 内容 |
|------|------|
| **接口路径** | `DELETE /api/v1/chat/session/{session_id}` |
| **功能说明** | 删除指定会话及其所有对话记录 |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `session_id` | string | 会话 ID |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "sess_20240115_xyz789",
    "deleted_records": 5,
    "message": "会话已删除"
  }
}
```

---

#### **2.2.5 获取会话列表**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/chat/sessions` |
| **功能说明** | 获取当前用户的所有会话列表 |

**查询参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `page` | int | ✗ | 页码，默认 1 |
| `page_size` | int | ✗ | 每页条数，默认 20 |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "session_id": "sess_20240115_xyz789",
        "created_at": "2024-01-15T10:30:00Z",
        "last_message": "今天天气如何",
        "message_count": 5,
        "last_interaction_at": "2024-01-15T10:32:10Z"
      },
      {
        "session_id": "sess_20240114_abc123",
        "created_at": "2024-01-14T14:00:00Z",
        "last_message": "再见",
        "message_count": 12,
        "last_interaction_at": "2024-01-14T15:45:30Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 2,
      "total_pages": 1
    }
  }
}
```

---

#### **2.2.6 对话反馈提交**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/chat/feedback` |
| **功能说明** | 用户对 AI 回复的反馈（用于模型优化） |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `task_id` | string | ✓ | 任务 ID |
| `session_id` | string | ✓ | 会话 ID |
| `turn` | int | ✓ | 对话轮次 |
| `rating` | int | ✓ | 评分：1（很差） ~ 5（很好） |
| `comment` | string | ✗ | 评论 |
| `helpful` | bool | ✗ | 是否有帮助 |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "feedback_id": "feedback_20240115_def789",
    "message": "感谢您的反馈"
  }
}
```

---

## 三、AI 发音纠正 API（Evaluate 模块）

### 3.1 功能说明

用户上传朗读音频，后端调用科大讯飞语音评测 API 进行评分，返回发音错误、改进建议和示例音频。

---

### 3.2 接口清单

#### **3.2.0 同步发音评测（MVP）**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/evaluate/MVP` |
| **功能说明** | 用户上传朗读语音，后端同步调用讯飞语音评测 + LLM 生成反馈 + TTS 合成反馈语音，并直接返回反馈语音的音频流（分级反馈 MVP 接口） |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `audio_file` | file | ✓ | 用户朗读语音（WAV / MP3，最大 10MB） |
| `audio_type` | string | ✓ | 音频格式：`wav` / `mp3` |
| `text_id` | string | ✓ | 朗读文本 ID（对应数据库中的句子/段落） |

**响应说明**：

- 成功：`HTTP 200`，`Content-Type: audio/mpeg`，响应体为反馈语音的音频二进制流，其中内容已根据评测分数自动生成 S/A/B/C 分级反馈文案。
- 失败：返回通用 JSON 错误结构（`code` ≠ 200，`message` 描述错误原因）。

> 说明：该接口聚焦“听得见的分级反馈体验”，后续可引入异步评测与结果查询接口（见 3.2.1+）。

#### **3.2.1 提交发音评测请求**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/evaluate/submit` |
| **功能说明** | 用户上传朗读音频，后端异步调用讯飞评测 API，返回任务 ID |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `audio_file` | file | ✓ | 用户朗读语音（WAV / MP3，最大 10MB） |
| `audio_type` | string | ✓ | 音频格式：`wav` / `mp3` |
| `text_id` | string | ✓ | 朗读文本 ID（对应数据库中的句子/段落） |
| `reference_text` | string | ✗ | 朗读文本（若不提供则从数据库查询） |
| `language` | string | ✗ | 语言：`zh_CN`（中文，默认）/ `en_US`（英文） |
| `assessment_type` | string | ✗ | 评测类型：`sentence`（句子，默认）/ `word`（词语）/ `paragraph`（段落） |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "eval_id": "eval_20240115_xyz789",
    "text_id": "text_12345",
    "status": "pending",
    "message": "发音评测任务已提交，请轮询查询结果"
  }
}
```

**返回字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `eval_id` | string | 评测 ID，用于查询结果 |
| `text_id` | string | 朗读文本 ID |
| `status` | string | 任务状态：`pending` / `processing` / `success` / `failed` |
| `message` | string | 提示信息 |

---

#### **3.2.2 查询发音评测结果**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/evaluate/result/{eval_id}` |
| **功能说明** | 客户端轮询查询发音评测的结果 |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `eval_id` | string | 评测 ID |

**返回结构示例（处理中）**：

```json
{
  "code": 200,
  "message": "processing",
  "data": {
    "eval_id": "eval_20240115_xyz789",
    "status": "processing",
    "progress": 75,
    "message": "正在分析音素..."
  }
}
```

**返回结构示例（处理完成）**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "eval_id": "eval_20240115_xyz789",
    "status": "success",
    "text_id": "text_12345",
    "reference_text": "你好，很高兴认识你",
    "overall_score": 82.5,
    "scores": {
      "pronunciation": 85.0,
      "fluency": 80.0,
      "integrity": 85.0
    },
    "duration_ms": 3500,
    "phonemes": [
      {
        "phoneme": "n",
        "text": "你",
        "score": 90.0,
        "start_time_ms": 0,
        "end_time_ms": 600
      },
      {
        "phoneme": "h",
        "text": "好",
        "score": 88.0,
        "start_time_ms": 600,
        "end_time_ms": 1200
      },
      {
        "phoneme": "h",
        "text": "很",
        "score": 75.0,
        "start_time_ms": 1800,
        "end_time_ms": 2400,
        "error_type": "mispronunciation",
        "suggestion": "舌头放低，气流均匀"
      }
    ],
    "detailed_feedback": {
      "strengths": ["语调自然", "节奏均衡"],
      "improvements": ["第三个字 '很' 发音不准确，应该发音为 'hěn'"],
      "suggestions": ["多练习鼻音的发音", "注意语调的起伏"]
    },
    "reference_audio": "https://oss.example.com/audio/reference_text_12345.mp3",
    "created_at": "2024-01-15T10:30:45Z"
  }
}
```

**返回字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `eval_id` | string | 评测 ID |
| `status` | string | 任务状态 |
| `text_id` | string | 朗读文本 ID |
| `reference_text` | string | 标准文本 |
| `overall_score` | float | 总体得分（0-100） |
| `pronunciation` | float | 发音得分 |
| `fluency` | float | 流利度得分 |
| `integrity` | float | 完整性得分（是否读漏字） |
| `duration_ms` | int | 用户朗读时长 |
| `phonemes[]` | array | 音素级详细分析 |
| `phonemes[].phoneme` | string | 音素（拼音） |
| `phonemes[].text` | string | 对应的汉字 |
| `phonemes[].score` | float | 该音素的得分 |
| `phonemes[].error_type` | string | 错误类型：`mispronunciation`（发音错误）/ `omission`（遗漏）/ `addition`（多读） |
| `phonemes[].suggestion` | string | 改进建议 |
| `detailed_feedback.strengths[]` | array | 优点 |
| `detailed_feedback.improvements[]` | array | 需要改进的点 |
| `detailed_feedback.suggestions[]` | array | 具体建议 |
| `reference_audio` | string | 标准发音音频 URL |
| `created_at` | string | 创建时间 |

**返回结构示例（处理失败）**：

```json
{
  "code": 200,
  "message": "failed",
  "data": {
    "eval_id": "eval_20240115_xyz789",
    "status": "failed",
    "error_message": "音频质量过低，无法进行评测"
  }
}
```

---

#### **3.2.3 获取评测历史**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/evaluate/history` |
| **功能说明** | 获取当前用户的评测历史列表 |

**查询参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `text_id` | string | ✗ | 按文本 ID 筛选 |
| `date_from` | string | ✗ | 开始日期（YYYY-MM-DD） |
| `date_to` | string | ✗ | 结束日期（YYYY-MM-DD） |
| `page` | int | ✗ | 页码，默认 1 |
| `page_size` | int | ✗ | 每页条数，默认 20 |
| `order_by` | string | ✗ | 排序字段：`created_at`（默认）/ `score` |
| `order` | string | ✗ | 排序方向：`desc`（默认）/ `asc` |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "eval_id": "eval_20240115_xyz789",
        "text_id": "text_12345",
        "reference_text": "你好，很高兴认识你",
        "overall_score": 82.5,
        "scores": {
          "pronunciation": 85.0,
          "fluency": 80.0,
          "integrity": 85.0
        },
        "created_at": "2024-01-15T10:30:45Z",
        "status": "success"
      },
      {
        "eval_id": "eval_20240114_abc123",
        "text_id": "text_12346",
        "reference_text": "今天天气很好",
        "overall_score": 88.0,
        "scores": {
          "pronunciation": 88.0,
          "fluency": 88.0,
          "integrity": 88.0
        },
        "created_at": "2024-01-14T14:00:45Z",
        "status": "success"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 25,
      "total_pages": 2
    }
  }
}
```

---

#### **3.2.4 获取评测详情**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/evaluate/{eval_id}/detail` |
| **功能说明** | 获取某次评测的完整详情（带详细音素分析） |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `eval_id` | string | 评测 ID |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "eval_id": "eval_20240115_xyz789",
    "status": "success",
    "text_id": "text_12345",
    "reference_text": "你好，很高兴认识你",
    "overall_score": 82.5,
    "scores": {
      "pronunciation": 85.0,
      "fluency": 80.0,
      "integrity": 85.0
    },
    "duration_ms": 3500,
    "phonemes": [
      {
        "phoneme": "n",
        "text": "你",
        "score": 90.0,
        "start_time_ms": 0,
        "end_time_ms": 600
      },
      {
        "phoneme": "h",
        "text": "很",
        "score": 75.0,
        "start_time_ms": 1800,
        "end_time_ms": 2400,
        "error_type": "mispronunciation",
        "suggestion": "舌头放低，气流均匀"
      }
    ],
    "detailed_feedback": {
      "strengths": ["语调自然", "节奏均衡"],
      "improvements": ["第三个字 '很' 发音不准确"],
      "suggestions": ["多练习鼻音的发音", "注意语调的起伏"]
    },
    "reference_audio": "https://oss.example.com/audio/reference_text_12345.mp3",
    "user_audio": "https://oss.example.com/audio/eval_20240115_xyz789.mp3",
    "created_at": "2024-01-15T10:30:45Z"
  }
}
```

---

#### **3.2.5 删除评测记录**

| 项目 | 内容 |
|------|------|
| **接口路径** | `DELETE /api/v1/evaluate/{eval_id}` |
| **功能说明** | 删除指定的评测记录 |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `eval_id` | string | 评测 ID |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "eval_id": "eval_20240115_xyz789",
    "message": "评测记录已删除"
  }
}
```

---

#### **3.2.6 获取发音提示音频**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/evaluate/reference-audio/{text_id}` |
| **功能说明** | 获取指定文本的标准发音音频（用于对比学习） |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `text_id` | string | 文本 ID |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "text_id": "text_12345",
    "reference_text": "你好，很高兴认识你",
    "audio_url": "https://oss.example.com/audio/reference_text_12345.mp3",
    "duration_ms": 3200
  }
}
```

---

## 四、智能学习报告 API（Report 模块）

### 4.1 功能说明

基于用户的学习数据（对话、评测记录），生成阶段性智能学习报告，包括学习进度、发音改善、学习建议等。

---

### 4.2 接口清单

#### **4.2.0 同步生成学习报告（MVP）**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/report/MVP` |
| **功能说明** | 基于当前用户的历史对话与发音评测数据，同步生成一份学习报告，并直接返回报告摘要和完整内容（报告生成 MVP 接口） |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `report_type` | string | ✓ | 报告类型：`weekly` / `monthly` / `custom` |
| `start_date` | string | ✗ | 开始日期（YYYY-MM-DD，若不填则自动推算本期起始） |
| `end_date` | string | ✗ | 结束日期（YYYY-MM-DD，若不填则为今天） |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "report_id": "report_20240115_abc123",
    "report_type": "weekly",
    "period_start_date": "2024-01-08",
    "period_end_date": "2024-01-14",
    "total_conversations": 12,
    "total_evaluations": 8,
    "average_evaluation_score": 78.5,
    "improvement_rate": 12.3,
    "strengths": ["流利度提升明显"],
    "weaknesses": ["准确度仍需加强"],
    "recommendations": "本周可以多练习包含 /th/ 音的句子。",
    "report_content": "完整的报告内容（Markdown 或纯文本）"
  }
}
```

> 说明：该接口直接完成“查询数据 → 统计分析 → LLM 生成报告”的同步流程，用于快速验证学习报告功能；复杂场景可使用 4.2.1 的异步生成方式。

#### **4.2.1 生成学习报告**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/report/generate` |
| **功能说明** | 触发学习报告的生成，后端异步分析数据并用 LLM 生成报告 |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `report_type` | string | ✓ | 报告类型：`daily`（日报，默认）/ `weekly`（周报）/ `monthly`（月报）/ `custom`（自定义） |
| `start_date` | string | ✗ | 开始日期（YYYY-MM-DD，若不填则自动计算） |
| `end_date` | string | ✗ | 结束日期（YYYY-MM-DD，若不填则为今天） |
| `include_evaluations` | bool | ✗ | 是否包含发音评测分析，默认 true |
| `include_chat_stats` | bool | ✗ | 是否包含对话统计，默认 true |
| `custom_prompt` | string | ✗ | 自定义生成提示语（用于定制化报告） |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "report_id": "report_20240115_abc123",
    "report_type": "daily",
    "status": "generating",
    "message": "学习报告正在生成，请稍候",
    "estimated_time_seconds": 30
  }
}
```

**返回字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `report_id` | string | 报告 ID |
| `report_type` | string | 报告类型 |
| `status` | string | 报告生成状态：`generating` / `success` / `failed` |
| `estimated_time_seconds` | int | 预计生成时间（秒） |

---

#### **4.2.2 查询报告生成进度**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/report/{report_id}/status` |
| **功能说明** | 查询报告生成的进度和状态 |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `report_id` | string | 报告 ID |

**返回结构示例（生成中）**：

```json
{
  "code": 200,
  "message": "processing",
  "data": {
    "report_id": "report_20240115_abc123",
    "status": "generating",
    "progress": 65,
    "current_stage": "analyzing_evals",
    "message": "正在分析发音评测数据..."
  }
}
```

**返回结构示例（生成完成）**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "report_id": "report_20240115_abc123",
    "status": "success",
    "report_type": "daily",
    "created_at": "2024-01-15T10:30:00Z",
    "start_date": "2024-01-15",
    "end_date": "2024-01-15",
    "message": "报告生成完成"
  }
}
```

---

#### **4.2.3 获取报告详情**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/report/{report_id}` |
| **功能说明** | 获取完整的学习报告详情 |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `report_id` | string | 报告 ID |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "report_id": "report_20240115_abc123",
    "report_type": "daily",
    "user_id": 10086,
    "created_at": "2024-01-15T10:30:00Z",
    "start_date": "2024-01-15",
    "end_date": "2024-01-15",
    "summary": {
      "total_study_time_minutes": 45,
      "total_interactions": 12,
      "evaluation_count": 5,
      "chat_count": 7
    },
    "pronunciation_analysis": {
      "average_score": 82.5,
      "improved_phonemes": ["n", "h", "s"],
      "problematic_phonemes": ["x", "q", "z"],
      "trend": "improving",
      "improvement_percentage": 5.2,
      "details": {
        "best_performance": {
          "phoneme": "n",
          "score": 90.0
        },
        "needs_improvement": {
          "phoneme": "x",
          "score": 72.0,
          "suggestion": "注意舌位，应该在硬腭前部"
        }
      }
    },
    "chat_statistics": {
      "total_sessions": 2,
      "total_turns": 7,
      "average_response_length": 15,
      "language_used": "Chinese",
      "topics": ["weather", "greetings", "daily_life"]
    },
    "learning_insights": {
      "strengths": ["发音基础扎实", "学习态度积极", "对话流利度提升"],
      "areas_for_improvement": ["舌尖音需要加强练习", "语调起伏仍需改善"],
      "recommendations": [
        "建议每天重点练习舌尖音（z、c、s）15 分钟",
        "多听标准发音示范，进行对比学习",
        "坚持每日对话练习，目标 20 分钟"
      ]
    },
    "milestone_achievement": {
      "daily_target": "学习 30 分钟",
      "achieved": true,
      "actual_time": 45,
      "completion_percentage": 150
    },
    "ai_generated_report": {
      "title": "2024 年 1 月 15 日学习报告",
      "content": "亲爱的学习者，今天你的学习表现很棒！...",
      "sections": [
        {
          "title": "学习总结",
          "content": "今日完成了 5 次发音评测和 7 轮对话，总学习时长 45 分钟..."
        },
        {
          "title": "发音进步",
          "content": "你的整体发音得分为 82.5 分，相比昨天提升了 5.2%..."
        },
        {
          "title": "学习建议",
          "content": "建议在接下来的学习中重点关注舌尖音的练习..."
        }
      ]
    },
    "next_goals": [
      "明日目标：完成 8 次发音评测",
      "本周目标：掌握所有舌尖音发音",
      "本月目标：发音平均分达到 85 分以上"
    ]
  }
}
```

**返回字段说明**：

| 一级字段 | 二级字段 | 类型 | 说明 |
|---------|---------|------|------|
| `report_id` | - | string | 报告 ID |
| `report_type` | - | string | 报告类型 |
| `created_at` | - | string | 创建时间 |
| `start_date` | - | string | 报告覆盖的开始日期 |
| `end_date` | - | string | 报告覆盖的结束日期 |
| `summary.total_study_time_minutes` | - | int | 总学习时间（分钟） |
| `summary.evaluation_count` | - | int | 发音评测次数 |
| `summary.chat_count` | - | int | 对话交互次数 |
| `pronunciation_analysis.average_score` | - | float | 平均发音得分 |
| `pronunciation_analysis.improved_phonemes[]` | - | array | 进步的音素 |
| `pronunciation_analysis.problematic_phonemes[]` | - | array | 需要改善的音素 |
| `pronunciation_analysis.trend` | - | string | 趋势：`improving` / `stable` / `declining` |
| `pronunciation_analysis.improvement_percentage` | - | float | 相比前一报告的改善百分比 |
| `chat_statistics.total_sessions` | - | int | 总会话数 |
| `chat_statistics.total_turns` | - | int | 总对话轮次 |
| `learning_insights.strengths[]` | - | array | 优点 |
| `learning_insights.areas_for_improvement[]` | - | array | 需要改善的方面 |
| `learning_insights.recommendations[]` | - | array | 具体建议 |
| `ai_generated_report.content` | - | string | AI 生成的报告主文本 |
| `ai_generated_report.sections[]` | - | array | 报告分章节 |

---

#### **4.2.4 获取报告列表**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/report/list` |
| **功能说明** | 获取当前用户的所有报告列表 |

**查询参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `report_type` | string | ✗ | 按报告类型筛选：`daily` / `weekly` / `monthly` |
| `date_from` | string | ✗ | 开始日期（YYYY-MM-DD） |
| `date_to` | string | ✗ | 结束日期（YYYY-MM-DD） |
| `page` | int | ✗ | 页码，默认 1 |
| `page_size` | int | ✗ | 每页条数，默认 10 |
| `order_by` | string | ✗ | 排序字段：`created_at`（默认）/ `score` |
| `order` | string | ✗ | 排序方向：`desc`（默认）/ `asc` |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "report_id": "report_20240115_abc123",
        "report_type": "daily",
        "created_at": "2024-01-15T10:30:00Z",
        "start_date": "2024-01-15",
        "end_date": "2024-01-15",
        "summary": {
          "total_study_time_minutes": 45,
          "evaluation_count": 5,
          "chat_count": 7
        },
        "pronunciation_analysis": {
          "average_score": 82.5,
          "trend": "improving"
        }
      },
      {
        "report_id": "report_20240114_xyz789",
        "report_type": "daily",
        "created_at": "2024-01-14T10:30:00Z",
        "start_date": "2024-01-14",
        "end_date": "2024-01-14",
        "summary": {
          "total_study_time_minutes": 35,
          "evaluation_count": 4,
          "chat_count": 5
        },
        "pronunciation_analysis": {
          "average_score": 78.3,
          "trend": "stable"
        }
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 10,
      "total": 30,
      "total_pages": 3
    }
  }
}
```

---

#### **4.2.5 删除报告**

| 项目 | 内容 |
|------|------|
| **接口路径** | `DELETE /api/v1/report/{report_id}` |
| **功能说明** | 删除指定的学习报告 |

**路径参数**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `report_id` | string | 报告 ID |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "report_id": "report_20240115_abc123",
    "message": "报告已删除"
  }
}
```

---


#### **4.2.6 获取学习统计面板**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/report/dashboard` |
| **功能说明** | 获取学习总体统计数据（用于展示仪表板） |

**查询参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `days` | int | ✗ | 统计近 N 天的数据，默认 7 天 |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "period": "last_7_days",
    "statistics": {
      "total_study_time_minutes": 280,
      "average_daily_time_minutes": 40,
      "total_evaluations": 35,
      "total_chat_turns": 50,
      "overall_pronunciation_score": 82.5,
      "score_improvement": 5.2,
      "days_with_study": 6,
      "consecutive_study_days": 4
    },
    "daily_breakdown": [
      {
        "date": "2024-01-15",
        "study_time_minutes": 45,
        "evaluation_count": 5,
        "chat_count": 7,
        "average_score": 82.5
      },
      {
        "date": "2024-01-14",
        "study_time_minutes": 35,
        "evaluation_count": 4,
        "chat_count": 5,
        "average_score": 78.3
      }
    ],
    "phoneme_performance": [
      {
        "phoneme": "n",
        "average_score": 90.0,
        "practice_count": 25
      },
      {
        "phoneme": "x",
        "average_score": 72.0,
        "practice_count": 10
      }
    ],
    "learning_goal": {
      "target_study_time_minutes": 300,
      "current_progress": 280,
      "completion_percentage": 93.3,
      "days_remaining": 2
    }
  }
}
```

---

## 五、通用接口（非模块特定）

### 5.1 用户认证相关

#### **5.1.1 用户登录**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/auth/login` |
| **功能说明** | 用户账户密码登录 |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `email` | string | ✓ | 用户邮箱 |
| `password` | string | ✓ | 用户密码 |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "user_id": 10086,
    "email": "user@example.com",
    "username": "张三",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 604800,
    "refresh_token": "refresh_token_xyz789...",
    "avatar_url": "https://oss.example.com/avatar/10086.jpg"
  }
}
```

---

#### **5.1.2 用户注册**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/auth/register` |
| **功能说明** | 新用户注册 |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `email` | string | ✓ | 用户邮箱 |
| `password` | string | ✓ | 用户密码（至少 8 位） |
| `username` | string | ✓ | 用户名 |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "user_id": 10087,
    "email": "newuser@example.com",
    "username": "李四",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 604800
  }
}
```

---

#### **5.1.3 用户登出**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/auth/logout` |
| **功能说明** | 用户登出（删除 Token） |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "登出成功"
  }
}
```

---

#### **5.1.4 刷新 Token**

| 项目 | 内容 |
|------|------|
| **接口路径** | `POST /api/v1/auth/refresh` |
| **功能说明** | 使用 Refresh Token 获取新的访问 Token |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `refresh_token` | string | ✓ | Refresh Token |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 604800
  }
}
```

---

### 5.2 用户信息相关

#### **5.2.1 获取用户信息**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/user/profile` |
| **功能说明** | 获取当前登录用户的信息 |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "user_id": 10086,
    "email": "user@example.com",
    "username": "张三",
    "avatar_url": "https://oss.example.com/avatar/10086.jpg",
    "created_at": "2024-01-01T00:00:00Z",
    "native_language": "zh_CN",
    "learning_language": "en_US",
    "bio": "Hello, I'm learning English.",
    "learning_goal": "流利英文对话",
    "daily_target_minutes": 30
  }
}
```

---

#### **5.2.2 更新用户信息**

| 项目 | 内容 |
|------|------|
| **接口路径** | `PUT /api/v1/user/profile` |
| **功能说明** | 更新当前用户的个人信息 |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `username` | string | ✗ | 用户名 |
| `avatar_file` | file | ✗ | 头像文件 |
| `bio` | string | ✗ | 个人签名 |
| `native_language` | string | ✗ | 母语 |
| `learning_language` | string | ✗ | 学习目标语言 |
| `daily_target_minutes` | int | ✗ | 每日学习目标（分钟） |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "user_id": 10086,
    "message": "用户信息已更新"
  }
}
```

---

### 5.3 系统信息相关

#### **5.3.1 获取系统状态**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/system/status` |
| **功能说明** | 获取后端系统的健康状态（不需要 Token） |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:45Z",
    "version": "1.0.0",
    "services": {
      "database": "connected",
      "redis": "connected",
      "tts": "healthy",
      "asr": "healthy",
      "llm": "healthy",
      "evaluation": "healthy"
    }
  }
}
```

---

#### **5.3.2 获取学习资源列表**

| 项目 | 内容 |
|------|------|
| **接口路径** | `GET /api/v1/resources/texts` |
| **功能说明** | 获取可学习的文本资源（句子、段落等） |

**查询参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `category` | string | ✗ | 分类：`greetings` / `daily` / `business` 等 |
| `level` | string | ✗ | 难度级别：`beginner` / `intermediate` / `advanced` |
| `language` | string | ✗ | 语言：`zh_CN` / `en_US` |
| `page` | int | ✗ | 页码，默认 1 |
| `page_size` | int | ✗ | 每页条数，默认 20 |

**返回结构示例**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "text_id": "text_12345",
        "content": "你好，很高兴认识你",
        "category": "greetings",
        "level": "beginner",
        "language": "zh_CN",
        "phonetic": "nǐ hǎo, hěn gāoxìng rènshi nǐ",
        "english_translation": "Hello, nice to meet you",
        "reference_audio_url": "https://oss.example.com/audio/text_12345.mp3"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 150,
      "total_pages": 8
    }
  }
}
```

---

## 六、错误处理规范

### 6.1 错误响应格式

所有错误均按以下格式返回：

```json
{
  "code": 400,
  "message": "error_description",
  "data": null
}
```

### 6.2 常见错误码表

| 错误码 | HTTP 状态码 | 说明 | 示例消息 |
|--------|-----------|------|---------|
| 200 | 200 | 请求成功 | success |
| 400 | 400 | 请求参数错误 | invalid_parameter |
| 401 | 401 | 未认证或 Token 过期 | unauthorized |
| 403 | 403 | 禁止访问 | forbidden |
| 404 | 404 | 资源不存在 | resource_not_found |
| 409 | 409 | 资源冲突（如邮箱已存在） | resource_conflict |
| 429 | 429 | 请求过于频繁 | rate_limit_exceeded |
| 500 | 500 | 服务器内部错误 | internal_server_error |
| 503 | 503 | 服务不可用 | service_unavailable |

### 6.3 常见错误响应示例

**参数验证错误**：

```json
{
  "code": 400,
  "message": "参数验证失败",
  "data": {
    "field": "email",
    "error": "邮箱格式不正确"
  }
}
```

**Token 过期**：

```json
{
  "code": 401,
  "message": "Token 已过期，请重新登录",
  "data": null
}
```

**资源不存在**：

```json
{
  "code": 404,
  "message": "评测记录不存在",
  "data": {
    "eval_id": "eval_20240115_xyz789"
  }
}
```

---

## 七、接口摘要表

### 7.1 Chat 模块接口摘要

| 编号 | 接口路径 | 方法 | 功能说明 |
|------|---------|------|---------|
| C-0 | `/api/v1/chat/MVP` | POST | 同步语音对话（ASR + LLM + TTS，返回音频流） |
| C-1 | `/api/v1/chat/submit` | POST | 提交语音对话请求 |
| C-2 | `/api/v1/chat/result/{task_id}` | GET | 查询语音对话处理结果 |
| C-3 | `/api/v1/chat/history/{session_id}` | GET | 获取对话历史 |
| C-4 | `/api/v1/chat/session/{session_id}` | DELETE | 删除对话会话 |
| C-5 | `/api/v1/chat/sessions` | GET | 获取会话列表 |
| C-6 | `/api/v1/chat/feedback` | POST | 对话反馈提交 |

---

### 7.2 Evaluate 模块接口摘要

| 编号 | 接口路径 | 方法 | 功能说明 |
|------|---------|------|---------|
| E-0 | `/api/v1/evaluate/MVP` | POST | 同步发音评测（评测 + 分级反馈 + TTS，返回音频流） |
| E-1 | `/api/v1/evaluate/submit` | POST | 提交发音评测请求 |
| E-2 | `/api/v1/evaluate/result/{eval_id}` | GET | 查询发音评测结果 |
| E-3 | `/api/v1/evaluate/history` | GET | 获取评测历史 |
| E-4 | `/api/v1/evaluate/{eval_id}/detail` | GET | 获取评测详情 |
| E-5 | `/api/v1/evaluate/{eval_id}` | DELETE | 删除评测记录 |
| E-6 | `/api/v1/evaluate/reference-audio/{text_id}` | GET | 获取发音提示音频 |

---

### 7.3 Report 模块接口摘要

| 编号 | 接口路径 | 方法 | 功能说明 |
|------|---------|------|---------|
| R-0 | `/api/v1/report/MVP` | POST | 同步生成学习报告（统计 + LLM，直接返回报告内容） |
| R-1 | `/api/v1/report/generate` | POST | 生成学习报告 |
| R-2 | `/api/v1/report/{report_id}/status` | GET | 查询报告生成进度 |
| R-3 | `/api/v1/report/{report_id}` | GET | 获取报告详情 |
| R-4 | `/api/v1/report/list` | GET | 获取报告列表 |
| R-5 | `/api/v1/report/{report_id}` | DELETE | 删除报告 |
| R-6 | `/api/v1/report/dashboard` | GET | 获取学习统计面板 |

---

### 7.4 通用接口摘要

| 编号 | 接口路径 | 方法 | 功能说明 |
|------|---------|------|---------|
| A-1 | `/api/v1/auth/login` | POST | 用户登录 |
| A-2 | `/api/v1/auth/register` | POST | 用户注册 |
| A-3 | `/api/v1/auth/logout` | POST | 用户登出 |
| A-4 | `/api/v1/auth/refresh` | POST | 刷新 Token |
| U-1 | `/api/v1/user/profile` | GET | 获取用户信息 |
| U-2 | `/api/v1/user/profile` | PUT | 更新用户信息 |
| S-1 | `/api/v1/system/status` | GET | 获取系统状态 |
| S-2 | `/api/v1/resources/texts` | GET | 获取学习资源列表 |

---

## 八、总结

本阶段已完成了 **28 个 RESTful API 接口设计**，包括：

✅ **AI 语音对话模块** - 6 个接口
✅ **AI 发音纠正模块** - 6 个接口
✅ **智能学习报告模块** - 6 个接口
✅ **用户认证与管理** - 4 个接口
✅ **系统与资源** - 2 个接口

**设计特点**：
- 遵循 RESTful 风格，路径清晰易懂
- 统一的响应结构，便于前端处理
- 完整的错误处理规范
- 支持异步长操作的任务轮询
- 完全贴合真实业务场景

---
