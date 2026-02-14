# OKTalk AI 发音纠正功能 PRD v2.0

## 文档版本信息
- **版本号**: v2.0
- **更新日期**: 2026-01-21
- **负责人**: 产品 & 技术团队
- **更新说明**: 完善分级反馈机制和示范音频生成策略

---

## 1. 功能概述

### 1.1 功能定位
AI 发音纠正是 OKTalk 的核心功能之一，旨在为 6-12 岁儿童提供**智能化、个性化、鼓励性**的英语发音评测与反馈服务。

### 1.2 核心价值
- 为儿童提供**即时、量化的发音评分**
- 通过 AI 生成**个性化、鼓励性的语音反馈**
- 根据评分等级提供**针对性的发音示范**
- 帮助儿童建立**正向的学习反馈循环**

---

## 2. 用户交互流程

### 2.1 完整流程图

```
用户进入纠音页面
   ↓
选择练习内容（单词/短语/句子）
   ↓
查看目标文本："I like apples"
   ↓
点击 [开始录音] 按钮
   ↓
录音中（显示波形动画）
   ↓
点击 [停止录音] 按钮
   ↓
显示 "AI 正在评分中..." 加载动画
   ↓
显示评分结果页面
   ├─ 综合评分卡片
   ├─ 各维度得分（准确度/流利度/完整度）
   ├─ AI 语音反馈（自动播放）
   ├─ [可选] 标准示范音频
   └─ [再试一次] / [下一题] 按钮
```

### 2.2 关键交互细节

| 交互节点 | 用户操作 | 系统反馈 |
|---------|---------|---------|
| 选择内容 | 点击单词/句子卡片 | 高亮选中，显示目标文本 |
| 开始录音 | 点击麦克风按钮 | 按钮变红，显示录音波形 |
| 停止录音 | 再次点击按钮 | 上传音频，显示加载动画 |
| 评分完成 | 无需操作 | 显示评分卡片 + 自动播放反馈 |
| 听示范音频 | 点击 🔊 图标 | 播放标准发音示范 |
| 再试一次 | 点击按钮 | 返回录音页面 |

---

## 3. 后端处理流程

### 3.1 技术架构流程图

```
音频文件上传 (WAV, ≤60秒)
   ↓
【步骤1】科大讯飞语音评测 API
   ├─ 输入：音频 + 目标文本
   ├─ 输出：评分数据（整体分 + 音素级详情）
   └─ 耗时：~1-2秒
   ↓
【步骤2】评分数据分析与决策
   ├─ 解析评分结果
   ├─ 识别问题单词/音素
   └─ 确定反馈级别（S/A/B/C）
   ↓
【步骤3】生成反馈文本（LLM）
   ├─ 根据反馈级别选择提示模板
   ├─ 输入：评分数据 + 问题诊断
   ├─ 调用：通义千问 Qwen3-Max
   ├─ 输出：个性化反馈文本
   └─ 耗时：~1秒
   ↓
【步骤4】生成反馈语音（TTS）
   ├─ 输入：反馈文本
   ├─ 调用：阿里云 CosyVoice
   ├─ 输出：反馈语音 MP3
   └─ 耗时：~1-2秒
   ↓
【步骤5】[条件触发] 生成示范音频（TTS）
   ├─ 触发条件：整体得分 < 90
   ├─ 输入：问题单词 或 完整句子
   ├─ 调用：阿里云 CosyVoice
   ├─ 输出：标准示范音频 MP3
   └─ 耗时：~1秒
   ↓
【步骤6】上传音频到 CDN
   ├─ 上传反馈语音到阿里云 OSS
   ├─ [可选] 上传示范音频到 OSS
   └─ 生成公网 CDN URL
   ↓
【步骤7】保存评测记录到数据库
   ├─ 表：pronunciation_evaluations
   └─ 记录所有评分数据和反馈内容
   ↓
返回完整响应给前端
```

### 3.2 总耗时估算
- 最快场景（90+ 分）：~3-4 秒
- 一般场景（需示范音频）：~5-6 秒
- 最慢场景（整句示范）：~6-7 秒

---

## 4. 评分维度与数据结构

### 4.1 科大讯飞 API 返回的评分数据

```json
{
  "overall_score": 78,      // 综合得分 (0-100)
  "fluency_score": 82,      // 流利度
  "accuracy_score": 75,     // 准确度
  "integrity_score": 95,    // 完整度
  "words": [
    {
      "text": "I",
      "score": 95,
      "begin_time": 0,
      "end_time": 200
    },
    {
      "text": "like",
      "score": 88,
      "begin_time": 200,
      "end_time": 600
    },
    {
      "text": "apples",
      "score": 68,          // 问题单词
      "begin_time": 600,
      "end_time": 1200,
      "phonemes": [
        {"phoneme": "æ", "score": 60},  // 问题音素
        {"phoneme": "p", "score": 75},
        {"phoneme": "l", "score": 70},
        {"phoneme": "z", "score": 65}
      ]
    }
  ]
}
```

### 4.2 系统处理后的分析结果

```json
{
  "analysis": {
    "feedback_level": "B",           // S/A/B/C 级别
    "problem_words": ["apples"],     // 得分 < 70 的单词
    "lowest_score_word": {
      "word": "apples",
      "score": 68,
      "problem_phonemes": ["æ", "z"]
    },
    "need_demo_audio": true,         // 是否需要示范音频
    "demo_type": "word"              // word / sentence
  }
}
```

---

## 5. 分级反馈机制（核心设计）

### 5.1 反馈级别定义

| 级别 | 分数范围 | 称号 | 反馈策略 |
|------|---------|------|---------|
| S 级 | 90-100 | Perfect | 纯鼓励 |
| A 级 | 70-89 | Good | 鼓励 + 诊断 |
| B 级 | 50-69 | Try Again | 诊断 + 示范 |
| C 级 | 0-49 | Keep Practicing | 完整示范 |

### 5.2 分级反馈详细规则

#### S 级反馈（90-100 分）
**策略**: 纯鼓励，强化正向反馈

**反馈文本示例**:
- "Perfect! Your pronunciation is excellent!"
- "Amazing job! You sound just like a native speaker!"
- "Wonderful! Keep up the great work!"

**是否提供示范音频**: ❌ 否

**LLM 提示模板**:
```
用户朗读了 "{target_text}"，得分 {score} 分。
请生成一句简短的鼓励性反馈（20 词以内，英文）。
要求：热情、积极、适合儿童。
```

---

#### A 级反馈（70-89 分）
**策略**: 鼓励 + 诊断，指出可改进的地方

**反馈文本示例**:
- "Very good! But 'apples' can be even better."
- "Great job! Try to pronounce 'like' more clearly."
- "Well done! The word 'apples' needs a little practice."

**是否提供示范音频**: ✅ 是（提供问题单词的标准发音）

**LLM 提示模板**:
```
用户朗读了 "{target_text}"，得分 {score} 分。
问题单词是 "{problem_word}"（得分 {word_score} 分）。
请生成反馈（30 词以内，英文）：
1. 先鼓励
2. 指出问题单词
3. 语气温和
```

**示范音频内容**: 仅朗读问题单词（如 "apples"）

---

#### B 级反馈（50-69 分）
**策略**: 诊断 + 示范，明确引导练习

**反馈文本示例**:
- "Good try! Let's practice 'apples' together. Listen to how I say it."
- "You're doing well! The word 'apples' is tricky. Let me show you."
- "Nice effort! Let's work on 'apples'. Hear how I pronounce it."

**是否提供示范音频**: ✅ 是（提供问题单词的标准发音）

**LLM 提示模板**:
```
用户朗读了 "{target_text}"，得分 {score} 分。
问题单词是 "{problem_word}"（得分 {word_score} 分）。
请生成反馈（40 词以内，英文）：
1. 温和鼓励
2. 说明需要练习的单词
3. 引导用户听示范
4. 不要打击自信心
```

**示范音频内容**: 仅朗读问题单词（如 "apples"）

---

#### C 级反馈（0-49 分）
**策略**: 完整示范，提供整句标准朗读

**反馈文本示例**:
- "Let's read together: I like apples. Listen and repeat after me."
- "Good try! Let's practice the whole sentence. Listen carefully: I like apples."
- "Keep going! Let me read it for you first: I like apples. Now you try!"

**是否提供示范音频**: ✅ 是（提供整句的标准朗读）

**LLM 提示模板**:
```
用户朗读了 "{target_text}"，得分 {score} 分。
多个单词发音需要改进。
请生成反馈（40 词以内，英文）：
1. 积极鼓励（不要说"不好"）
2. 引导用户听整句示范
3. 鼓励再次尝试
4. 语气要温暖、耐心
```

**示范音频内容**: 朗读完整目标句子（如 "I like apples"）

---

## 6. API 接口定义

### 6.1 请求格式

**接口**: `POST /api/v1/voice/evaluate`

**Content-Type**: `multipart/form-data`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|-------|------|------|------|
| audio | file | 是 | 录音文件（WAV 格式） |
| user_id | string | 是 | 用户 ID |
| text | string | 是 | 目标朗读文本 |
| level | string | 否 | 难度级别（word/sentence/paragraph）默认 sentence |

### 6.2 响应格式

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "evaluation_id": "eval_67890",
    "target_text": "I like apples",
    "recognized_text": "I like apples",
    "scores": {
      "overall": 78,
      "accuracy": 75,
      "fluency": 82,
      "integrity": 95,
      "feedback_level": "B",
      "pronunciation_details": [
        {
          "word": "I",
          "score": 95
        },
        {
          "word": "like",
          "score": 88
        },
        {
          "word": "apples",
          "score": 68,
          "is_problem": true,
          "phonemes": [
            {"phoneme": "æ", "score": 60},
            {"phoneme": "p", "score": 75},
            {"phoneme": "l", "score": 70},
            {"phoneme": "z", "score": 65}
          ]
        }
      ]
    },
    "feedback": {
      "text": "Good try! Let's practice 'apples' together. Listen to how I say it.",
      "audio_url": "https://cdn.oktalk.com/feedback/eval_67890.mp3",
      "duration": 3.5
    },
    "demo_audio": {
      "type": "word",
      "content": "apples",
      "audio_url": "https://cdn.oktalk.com/demo/word_apples.mp3",
      "duration": 1.2
    },
    "timestamp": "2026-01-21T10:35:00Z"
  }
}
```

### 6.3 响应字段说明

| 字段 | 说明 |
|------|------|
| `feedback_level` | S/A/B/C 级别 |
| `is_problem` | 标记问题单词 |
| `demo_audio` | 示范音频对象（90+ 分时为 null） |
| `demo_audio.type` | word（单词示范）或 sentence（整句示范） |

---

## 7. 关键需求与约束

### 7.1 功能性需求

| 需求 | 说明 |
|------|------|
| FR-01 | 支持单词、短语、句子三种评测粒度 |
| FR-02 | 提供 4 个维度的评分（整体/准确度/流利度/完整度） |
| FR-03 | 支持音素级别的发音诊断 |
| FR-04 | 根据评分自动生成分级反馈 |
| FR-05 | 为 70 分以下的评测提供示范音频 |
| FR-06 | 所有反馈语言必须是鼓励性的 |

### 7.2 非功能性需求

| 需求 | 指标 |
|------|------|
| NFR-01 | 单次评测响应时间 ≤ 7 秒 |
| NFR-02 | 评测准确率 ≥ 85% |
| NFR-03 | 支持并发评测 ≥ 100 QPS |
| NFR-04 | 音频文件大小 ≤ 10 MB |
| NFR-05 | 音频时长 ≤ 60 秒 |

### 7.3 儿童友好性要求

| 要求 | 实施细节 |
|------|---------|
| 反馈语言简单 | 词汇量限制在小学 6 年级水平 |
| 避免负面词汇 | 禁用 "wrong", "bad", "terrible" 等词 |
| 鼓励为主 | 即使低分也要先鼓励再指出问题 |
| 语气温和 | 使用 "Let's practice" 而非 "You need to" |
| 可视化友好 | 使用星级、徽章等正向激励元素 |

---

## 8. 数据库设计要点

### 8.1 需要存储的关键数据

```sql
-- pronunciation_evaluations 表需要存储：
- evaluation_id          # 评测唯一 ID
- user_id               # 用户 ID
- target_text           # 目标文本
- recognized_text       # 识别文本
- overall_score         # 综合得分
- accuracy_score        # 准确度
- fluency_score         # 流利度
- integrity_score       # 完整度
- feedback_level        # S/A/B/C 级别
- pronunciation_details # JSON 格式音素级详情
- feedback_text         # 反馈文本
- feedback_audio_url    # 反馈音频 URL
- demo_audio_url        # 示范音频 URL（可为 null）
- demo_type             # word / sentence / null
- created_at            # 创建时间
```

### 8.2 索引设计

- 主键：`evaluation_id`
- 索引：`user_id`, `feedback_level`, `created_at`
- 用于查询：用户历史评测记录、不同级别的评测统计

---

## 9. 前端展示建议

### 9.1 评分结果页面布局

```
┌─────────────────────────────────┐
│      🎉 评分结果                 │
├─────────────────────────────────┤
│                                 │
│        ⭐⭐⭐                      │
│       综合得分: 78 分             │
│       级别: B (Good Try!)        │
│                                 │
├─────────────────────────────────┤
│  准确度: 75  流利度: 82  完整度: 95│
├─────────────────────────────────┤
│  🔊 AI 反馈 (自动播放)           │
│  "Good try! Let's practice      │
│   'apples' together..."         │
├─────────────────────────────────┤
│  📢 标准示范                     │
│  apples [▶️播放]                 │
├─────────────────────────────────┤
│  单词详情:                       │
│  ✅ I (95分)                    │
│  ✅ like (88分)                 │
│  ⚠️ apples (68分) ← 需要练习    │
├─────────────────────────────────┤
│  [再试一次]  [下一题]            │
└─────────────────────────────────┘
```

### 9.2 不同级别的视觉反馈

| 级别 | 颜色主题 | 表情符号 | 特效 |
|------|---------|---------|------|
| S 级 | 金色 | 🎉🏆⭐ | 烟花动画 |
| A 级 | 蓝色 | 👍😊💪 | 点赞动画 |
| B 级 | 橙色 | 🤔📖✨ | 加油动画 |
| C 级 | 绿色 | 🌱🎯💡 | 成长动画 |

---

## 10. 成功指标（KPI）

### 10.1 核心指标

| 指标 | 目标值 |
|------|--------|
| 评测完成率 | ≥ 90% |
| 平均评测时长 | ≤ 6 秒 |
| 用户重复评测率 | ≥ 60% |
| 评分准确性用户满意度 | ≥ 85% |

### 10.2 学习效果指标

| 指标 | 目标值 |
|------|--------|
| 同一内容二次评测进步率 | ≥ 70% 用户有提升 |
| 周留存率 | ≥ 40% |
| 用户 NPS（净推荐值） | ≥ 50 |

---

## 11. 风险与应对

### 11.1 技术风险

| 风险 | 概率 | 影响 | 应对措施 |
|------|------|------|---------|
| 科大讯飞 API 不稳定 | 中 | 高 | 实现重试机制，准备备用 API |
| TTS 生成速度慢 | 中 | 中 | 异步生成 + 缓存常用示范音频 |
| 并发量过大 | 低 | 高 | 实施限流 + 水平扩展 |

### 11.2 产品风险

| 风险 | 概率 | 影响 | 应对措施 |
|------|------|------|---------|
| 反馈不够鼓励 | 中 | 高 | 定期人工审核 LLM 输出质量 |
| 评分不准确 | 中 | 高 | 收集用户反馈，调优评测模型 |
| 儿童不理解反馈 | 中 | 中 | A/B 测试不同的反馈话术 |

---

## 12. 后续优化方向

### 12.1 V1.0（当前版本）
- ✅ 基础评测功能
- ✅ 分级反馈系统
- ✅ 选择性示范音频

### 12.2 V1.5（3 个月后）
- 🔄 增加重点音素专项训练
- 🔄 支持中文反馈（家长查看）
- 🔄 增加进步曲线图表

### 12.3 V2.0（6 个月后）
- 🚀 AI 虚拟老师实时对话纠音
- 🚀 支持情景对话评测
- 🚀 多维度能力雷达图

---

## 13. 附录

### 13.1 参考资料
- 科大讯飞语音评测 API 文档
- 阿里云 CosyVoice TTS 文档
- 通义千问 Prompt Engineering 最佳实践

### 13.2 相关文档
- 《OKTalk 整体 PRD》
- 《OKTalk API 接口文档》
- 《OKTalk 数据库设计文档》

---

**文档结束**