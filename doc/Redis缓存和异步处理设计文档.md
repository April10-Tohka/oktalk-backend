# OKTalk 缓存和异步处理设计文档

## 📋 目录

1. [Redis 缓存设计](#redis-缓存设计)
2. [异步任务处理架构](#异步任务处理架构)
3. [Key 设计规范](#key-设计规范)
4. [数据一致性策略](#数据一致性策略)
5. [实现细节](#实现细节)

---

## 🎯 核心目标

1. **减少外部服务调用**：缓存 LLM/TTS 结果，避免重复计费
2. **提升响应速度**：缓存热点数据（示范音频、用户信息）
3. **异步处理长耗时任务**：评测、对话、报告生成
4. **保证数据一致性**：Redis 与 DB 的同步策略
5. **防止任务丢失**：Redis 持久化 + 定时回补

---

## 📦 Redis 缓存设计

### 1. 核心业务结果缓存

#### 1.1 AI 语音对话结果（Chat）

**Key 设计**：`chat:result:{task_id}`

**数据结构**：String（JSON）

**内容**：
```json
{
  "task_id": "chat_xxx",
  "status": "completed",
  "user_input": {
    "text": "Hello",
    "audio_url": "..."
  },
  "ai_response": {
    "text": "Hi! How are you?",
    "audio_url": "..."
  },
  "conversation_id": "conv_xxx",
  "created_at": "2024-02-20T10:30:00Z"
}
```

**TTL**：7 天（用户可能多次查看）

**写入时机**：任务完成时

**读取场景**：用户查询对话结果、前端轮询

---

#### 1.2 AI 发音评测结果（Evaluate）

**Key 设计**：`evaluate:result:{eval_id}`

**数据结构**：String（JSON）

**内容**：
```json
{
  "eval_id": "eval_xxx",
  "status": "completed",
  "overall_score": 78.0,
  "feedback_level": "B",
  "feedback_text": "...",
  "feedback_audio_url": "...",
  "demo_audio": {
    "type": "word",
    "text": "apples",
    "audio_url": "..."
  },
  "word_details": [...],
  "created_at": "2024-02-20T10:30:00Z"
}
```

**TTL**：30 天（学习记录，保留时间长）

**写入时机**：评测完成时

**读取场景**：用户查看评测详情、前端轮询

---

#### 1.3 智能学习报告结果（Report）

**Key 设计**：`report:result:{report_id}`

**数据结构**：String（JSON）

**内容**：完整的 ReportMVPResponse

**TTL**：90 天（周报/月报，长期保留）

**写入时机**：报告生成完成时

**读取场景**：用户查看报告、家长查看孩子学习情况

---

### 2. 历史列表缓存

#### 2.1 对话历史列表

**Key 设计**：`chat:history:{user_id}:{page}`

**数据结构**：String（JSON 数组）

**内容**：
```json
{
  "items": [
    {"conversation_id": "conv_001", "topic": "Greetings", "created_at": "..."},
    {"conversation_id": "conv_002", "topic": "Daily", "created_at": "..."}
  ],
  "total": 25,
  "page": 1
}
```

**TTL**：1 小时（列表可能频繁更新）

**写入时机**：首次查询时写入

**失效时机**：新对话创建时删除

---

#### 2.2 评测历史列表

**Key 设计**：`evaluate:history:{user_id}:{page}`

**数据结构**：String（JSON 数组）

**TTL**：1 小时

**失效时机**：新评测完成时删除

---

#### 2.3 报告历史列表

**Key 设计**：`report:history:{user_id}:{page}`

**数据结构**：String（JSON 数组）

**TTL**：6 小时（报告生成频率低）

**失效时机**：新报告生成时删除

---

### 3. 示范音频缓存（全局共享）⭐

#### 3.1 单词示范音频

**Key 设计**：`demo:audio:word:{word}:{voice}`

**数据结构**：String（URL）

**示例**：
```
Key: demo:audio:word:apples:longanyang
Value: https://oss.example.com/demo/apples_longanyang.mp3
```

**TTL**：永久（或 365 天）

**写入时机**：首次生成时写入

**优势**：所有用户共享，大幅减少 TTS 调用

---

#### 3.2 句子示范音频

**Key 设计**：`demo:audio:sentence:{text_hash}:{voice}`

**数据结构**：String（URL）

**示例**：
```
Key: demo:audio:sentence:md5("I like apples"):longanyang
Value: https://oss.example.com/demo/sentence_abc123.mp3
```

**TTL**：永久（或 365 天）

**写入时机**：首次生成时写入

**说明**：使用 MD5 避免 Key 过长

---

### 4. LLM 缓存（减少重复调用）⭐

#### 4.1 反馈文本缓存（Evaluate）

**Key 设计**：`llm:feedback:{level}:{score}:{problem_word}`

**数据结构**：String（文本）

**示例**：
```
Key: llm:feedback:B:68:apples
Value: "Good try! Let's practice 'apples' together. Listen to how I say it."
```

**TTL**：30 天

**写入时机**：LLM 生成后

**命中条件**：相同级别 + 相似分数（±5）+ 相同问题单词

**说明**：
- S 级：不需要缓存（模板固定）
- A/B 级：`{level}:{score_range}:{word}`
- C 级：`{level}:{score_range}:{sentence_hash}`

---

#### 4.2 报告文本缓存（Report）

**Key 设计**：`llm:report:{user_id}:{period}:{stats_hash}`

**数据结构**：String（JSON）

**示例**：
```
Key: llm:report:user_123:2024W08:md5(统计数据)
Value: {"period_summary": "...", "highlights": [...], ...}
```

**TTL**：7 天

**写入时机**：报告生成后

**命中条件**：相同用户 + 相同周期 + 相同统计数据

**说明**：stats_hash 包含所有统计维度（对话数、评测数、分数等）

---

### 5. TTS 缓存（基于文本内容）⭐

**Key 设计**：`tts:audio:{text_hash}:{voice}:{options_hash}`

**数据结构**：String（OSS URL）

**示例**：
```
Key: tts:audio:md5("Perfect! Your pronunciation is excellent!"):longanyang:default
Value: https://oss.example.com/tts/perfect_xxx.mp3
```

**TTL**：30 天

**写入时机**：TTS 合成后上传 OSS 完成

**命中条件**：相同文本 + 相同音色 + 相同合成参数

**优势**：
- 避免相同文本重复合成
- 多用户共享
- 大幅降低 TTS 成本

---

### 6. 会话完整记录缓存

**Key 设计**：`chat:session:{conversation_id}`

**数据结构**：List（JSON 数组）

**内容**：
```json
[
  {"seq": 1, "sender": "user", "text": "Hello", "audio_url": "..."},
  {"seq": 2, "sender": "ai", "text": "Hi!", "audio_url": "..."},
  ...
]
```

**TTL**：24 小时（对话可能持续进行）

**写入时机**：每条消息保存后追加

**读取场景**：用户查看对话历史、继续对话时需要上下文

---

### 7. 用户信息缓存（低优先级）

**Key 设计**：`user:info:{user_id}`

**数据结构**：String（JSON）

**TTL**：1 小时

**失效时机**：用户信息更新时删除

---

### 8. 学习文本内容缓存（低优先级）

**Key 设计**：`text:content:{text_id}`

**数据结构**：String（JSON）

**内容**：
```json
{
  "text_id": "text_001",
  "content": "I like apples",
  "difficulty": "beginner",
  "category": "food",
  "reference_audio_url": "..."
}
```

**TTL**：永久（内容不变）

---

## ⚙️ 异步任务处理架构

### 1. 架构设计图

```
┌─────────────┐
│  HTTP API   │
└──────┬──────┘
       │ 提交任务
       ↓
┌──────────────────────────────────────┐
│  Task Manager (任务管理器)            │
│  - 生成 task_id                       │
│  - 写入 Redis (task:meta:{id})       │
│  - 发送到对应 Channel                 │
└──────────────────────────────────────┘
       │
       ├──→ Chat Channel    ──→ Chat Worker Pool    (10 workers)
       ├──→ Eval Channel    ──→ Eval Worker Pool    (15 workers)
       └──→ Report Channel  ──→ Report Worker Pool  (5 workers)

每个 Worker:
1. 从 Channel 接收任务
2. 执行业务逻辑（ASR/LLM/TTS 等）
3. 更新 Redis 任务状态
4. 写入 Redis 结果缓存
5. 写入 DB（持久化）
```

---

### 2. Redis 任务元数据设计

#### 2.1 任务元数据（轻量级，快速查询）

**Key 设计**：`task:meta:{task_id}`

**数据结构**：Hash

**字段**：
```
type        → chat / evaluate / report
status      → pending / processing / success / failed
result_key  → 结果缓存 Key（如 chat:result:xxx）
error       → 错误信息（失败时）
created_at  → 创建时间戳
updated_at  → 更新时间戳
user_id     → 用户 ID（用于回补）
```

**TTL**：24 小时

**示例**：
```
task:meta:chat_abc123
  type: chat
  status: success
  result_key: chat:result:chat_abc123
  created_at: 1708502400
  updated_at: 1708502430
  user_id: user_123
```

---

#### 2.2 Pending 任务索引（用于回补）

**Key 设计**：`task:pending:{type}`

**数据结构**：ZSet（按时间排序）

**内容**：
```
member: task_id
score: 创建时间戳
```

**示例**：
```
task:pending:chat
  chat_abc123: 1708502400
  chat_def456: 1708502410
```

**用途**：服务重启后扫描所有 Pending 任务回补到 Channel

---

### 3. Worker Pool 设计

```go
// 任务结构
type Task struct {
    ID        string
    Type      string // chat / evaluate / report
    UserID    string
    Payload   interface{}
    CreatedAt time.Time
}

// Worker Pool
type WorkerPool struct {
    taskChannel chan *Task
    workerCount int
    wg          sync.WaitGroup
}

// 启动 Worker Pool
func (p *WorkerPool) Start() {
    for i := 0; i < p.workerCount; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

// Worker 逻辑
func (p *WorkerPool) worker(id int) {
    defer p.wg.Done()
    for task := range p.taskChannel {
        // 1. 更新状态为 processing
        updateTaskStatus(task.ID, "processing")
        
        // 2. 执行业务逻辑
        result, err := processTask(task)
        
        // 3. 写入结果
        if err != nil {
            updateTaskStatus(task.ID, "failed", err.Error())
        } else {
            saveResult(task.ID, result)         // 写入 Redis
            saveToDB(task.ID, result)           // 写入 DB
            updateTaskStatus(task.ID, "success")
        }
        
        // 4. 从 Pending 索引移除
        removePendingTask(task.Type, task.ID)
    }
}
```

---

### 4. 任务回补机制（防止丢失）⭐

```go
// 定期扫描 Pending 任务（每 30 秒）
func StartTaskRecovery() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        recoverPendingTasks("chat")
        recoverPendingTasks("evaluate")
        recoverPendingTasks("report")
    }
}

// 回补某类型的 Pending 任务
func recoverPendingTasks(taskType string) {
    // 使用 ZSCAN 而非 KEYS（避免阻塞）
    cursor := uint64(0)
    for {
        keys, cursor, err := redis.ZScan(
            ctx, 
            fmt.Sprintf("task:pending:%s", taskType), 
            cursor, 
            "*", 
            100,
        ).Result()
        
        if err != nil {
            log.Error("scan failed", zap.Error(err))
            return
        }
        
        // 处理每个 task_id
        for i := 0; i < len(keys); i += 2 {
            taskID := keys[i]
            
            // 检查任务是否仍在 pending（超时 5 分钟）
            meta := getTaskMeta(taskID)
            if meta.Status == "pending" && 
               time.Now().Unix() - meta.CreatedAt > 300 {
                // 重新入队
                resubmitTask(taskID, taskType)
            }
        }
        
        if cursor == 0 {
            break
        }
    }
}
```

---


## 🔑 Key 设计规范

### 1. 命名规范

```
格式: {模块}:{资源}:{标识}:{子标识}

示例:
chat:result:chat_abc123
evaluate:history:user_123:page_1
demo:audio:word:apples:longanyang
task:meta:eval_xyz789
```

### 2. Key 长度控制

- 最长不超过 200 字符
- 句子/长文本使用 MD5 哈希
- 避免包含空格和特殊字符

### 3. TTL 策略表

| 数据类型 | TTL | 原因 |
|---------|-----|------|
| 任务元数据 | 24h | 轮询完即可删除 |
| 对话结果 | 7d | 短期查看频率高 |
| 评测结果 | 30d | 学习记录，中期保留 |
| 报告结果 | 90d | 长期报告，低频访问 |
| 历史列表 | 1h | 易失效，快速刷新 |
| 示范音频 | 365d | 全局共享，长期有效 |
| LLM/TTS 缓存 | 30d | 复用率高 |
| 会话记录 | 24h | 对话中使用 |

---

## 🔄 数据一致性策略

### 1. Cache-Aside 模式（主模式）

```
读流程:
1. 查询 Redis
2. 命中 → 返回
3. 未命中 → 查询 DB → 写入 Redis → 返回

写流程:
1. 写入 DB
2. 删除 Redis 缓存（或更新）
```

### 2. 写入顺序（异步任务完成时）

```
1. 写入 Redis 结果缓存（chat:result:xxx）
2. 更新 Redis 任务状态（task:meta:xxx）
3. 写入 DB（持久化）
4. 删除相关列表缓存（history）
```

### 3. 缓存失效策略

**主动失效**：
- 新对话/评测/报告创建 → 删除历史列表缓存
- 用户信息更新 → 删除用户信息缓存

**被动失效**：
- 依赖 TTL 自动过期

---

## 📊 缓存命中率优化

### 1. LLM 缓存优化

```go
// 生成 LLM 缓存 Key
func getLLMCacheKey(level string, score float64, word string) string {
    // 分数区间化（±5 分）
    scoreRange := int(math.Round(score/5) * 5)
    
    if level == "S" {
        return "" // S 级不缓存，使用模板
    } else if level == "C" {
        return fmt.Sprintf("llm:feedback:%s:%d:%s", 
            level, scoreRange, md5(sentence))
    } else {
        return fmt.Sprintf("llm:feedback:%s:%d:%s", 
            level, scoreRange, word)
    }
}
```

### 2. TTS 缓存优化

```go
// 生成 TTS 缓存 Key
func getTTSCacheKey(text string, voice string, options *SynthesizeOptions) string {
    // 标准化文本（去除空格、标点，转小写）
    normalizedText := normalizeText(text)
    textHash := md5(normalizedText)
    
    // options 哈希（只包含影响结果的参数）
    optionsHash := md5(fmt.Sprintf("%s_%d_%f", 
        options.Format, options.SampleRate, options.Rate))
    
    return fmt.Sprintf("tts:audio:%s:%s:%s", textHash, voice, optionsHash)
}
```

---

## 🎯 实现优先级

### P0（MVP 必须）
1. ✅ 任务元数据缓存（task:meta）
2. ✅ 三大功能结果缓存（result）
3. ✅ 异步 Worker Pool
4. ✅ 任务回补机制

### P1（优化）
1. ⬜ 示范音频缓存（demo:audio）
2. ⬜ TTS 缓存
3. ⬜ 历史列表缓存

### P2（进阶）
1. ⬜ LLM 缓存
2. ⬜ 会话记录缓存
3. ⬜ 用户信息缓存

---


## ✅ 总结

### 缓存策略
```
1. 结果缓存（3 个功能）→ 减少重复计算
2. 示范音频缓存（全局）→ 减少 TTS 调用
3. LLM/TTS 缓存（内容）→ 避免重复生成
4. 历史列表缓存（分页）→ 减少 DB 查询
5. 会话记录缓存（实时）→ 加速对话上下文
```

### 异步架构
```
1. 三个独立 Channel + Worker Pool
2. Redis 任务元数据（轻量级）
3. Pending 任务索引（ZSet）
4. 定时回补机制（ZSCAN）
5. 双写策略（Redis + DB）
```

### 数据流向
```
提交任务 → Channel → Worker → 
Redis 结果 → 更新状态 → 
DB 持久化 → 删除列表缓存 → 
前端轮询 → 返回结果
```

---
