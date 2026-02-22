# 智能报告生成Report MVP 实现 

## 📱 前端 App 报告页面布局（关键参考）

### 顶部：周期信息卡片
```
┌─────────────────────────────────┐
│  📊 本周学习报告                 │
│  2024.02.12 - 2024.02.18        │
└─────────────────────────────────┘
```

### 区域 1：学习活跃度面板
```
┌─────────────────────────────────┐
│  🎯 学习活跃度                   │
├─────────────────────────────────┤
│  🗣️ 对话练习: 12 次             │
│  🎤 发音评测: 8 次               │
│  ⏱️ 学习时长: 45 分钟            │
│  📅 活跃天数: 5 天               │
└─────────────────────────────────┘
```

### 区域 2：能力雷达图
```
┌─────────────────────────────────┐
│  📈 能力雷达图                   │
├─────────────────────────────────┤
│        准确度                    │
│         75                       │
│        ╱  ╲                     │
│    流利度──完整度                │
│     82      88                   │
│                                 │
│  💬 整体表现很棒！              │
└─────────────────────────────────┘
```

### 区域 3：进步趋势面板
```
┌─────────────────────────────────┐
│  🚀 本周进步                     │
├─────────────────────────────────┤
│  ✨ 流利度提升 +8 分！           │
│  ✨ S/A 级评测增加 3 次          │
│  📊 综合分数: 75 → 82 (+7)      │
└─────────────────────────────────┘
```

### 区域 4：AI 鼓励卡片（展示给孩子）
```
┌─────────────────────────────────┐
│  🎉 "这周你更敢开口了，真棒！"   │
├─────────────────────────────────┤
│  🌟 本周亮点：                   │
│  • 流利度提升                    │
│  • 发音更清晰                    │
├─────────────────────────────────┤
│  🎯 小目标：                     │
│  每天跟读 5 分钟                 │
└─────────────────────────────────┘
```

### 区域 5：难词卡片
```
┌─────────────────────────────────┐
│  📚 需要加强的单词               │
├─────────────────────────────────┤
│  🔊 apples     [▶️播放]         │
│  🔊 beautiful  [▶️播放]         │
│  🔊 together   [▶️播放]         │
└─────────────────────────────────┘
```

### 区域 6：完整报告（可展开）
```
┌─────────────────────────────────┐
│  📄 详细报告                     │
│  [点击展开]                      │
├─────────────────────────────────┤
│  这周你完成了 12 次对话练习...   │
│  你的流利度进步很明显...         │
│  建议每天跟读 5 分钟...          │
│  （100-150 字）                 │
└─────────────────────────────────┘
```

---

## 🎯 功能说明

**接口**: `POST /api/v1/report/MVP`

**功能**: 统计学习数据 → LLM 生成报告 → 保存数据库 → 返回完整报告

**核心流程**:
```
查询数据 → 统计分析 → 计算进步 → LLM生成报告 → 保存数据库 → 返回JSON
```

---

## 📊 数据统计来源

### 来源表 1：voice_conversations（对话数据）
```
可统计字段：
- 对话次数: COUNT(id)
- 学习时长: SUM(duration_seconds) / 60（转分钟）
- 活跃天数: COUNT(DISTINCT DATE(created_at))
- 主题分布: GROUP BY topic
- 难度分布: GROUP BY difficulty_level
- 对话类型: GROUP BY conversation_type
```

### 来源表 2：pronunciation_evaluations（评测数据）
```
可统计字段：
- 评测次数: COUNT(id)
- 平均综合分: AVG(overall_score)
- 平均准确度: AVG(accuracy_score)
- 平均流利度: AVG(fluency_score)
- 平均完整度: AVG(integrity_score)
- S/A/B/C 级别数: COUNT(CASE WHEN feedback_level='S' ...)
- 问题单词: 聚合所有 problem_words（取高频 3-5 个）
```

### 综合统计指标
```
学习活跃度 = 对话次数 + 评测次数
能力雷达 = (平均准确度, 平均流利度, 平均完整度)
进步趋势 = 本期平均分 - 上期平均分
等级提升 = 本期 S/A 数 - 上期 S/A 数
```

---

## 📋 分步生成计划（严格按顺序）

### 🔹 Step 1：定义响应结构体

**目标文件**: `internal/service/report_service.go`

**需要定义的结构体**:

```go
// ReportMVPRequest MVP 报告生成请求
type ReportMVPRequest struct {
    ReportType string // weekly / monthly / custom
    StartDate  string // YYYY-MM-DD
    EndDate    string // YYYY-MM-DD
    UserID     string
}

// ReportMVPResponse MVP 报告响应（完全对应前端布局）
type ReportMVPResponse struct {
    // === 顶部：基本信息 ===
    ReportID        string    `json:"report_id"`
    ReportType      string    `json:"report_type"`      // weekly/monthly
    PeriodStartDate string    `json:"period_start_date"` // 2024-02-12
    PeriodEndDate   string    `json:"period_end_date"`   // 2024-02-18
    
    // === 区域 1：学习活跃度 ===
    ActivityStats *ActivityStats `json:"activity_stats"`
    
    // === 区域 2：能力雷达图 ===
    AbilityRadar *AbilityRadar `json:"ability_radar"`
    
    // === 区域 3：进步趋势 ===
    ProgressStats *ProgressStats `json:"progress_stats"`
    
    // === 区域 4：给孩子的鼓励卡片 ===
    KidFriendlyCard *KidFriendlyCard `json:"kid_friendly_card"`
    
    // === 区域 5：难词卡片 ===
    DifficultWords []DifficultWord `json:"difficult_words"` // 3-5 个高频问题单词
    
    // === 区域 6：完整报告 ===
    FullReport *FullReport `json:"full_report"`
}

// ActivityStats 学习活跃度统计（对应区域 1）
type ActivityStats struct {
    ConversationCount int `json:"conversation_count"` // 对话次数: 12
    EvaluationCount   int `json:"evaluation_count"`   // 评测次数: 8
    TotalMinutes      int `json:"total_minutes"`      // 学习时长: 45 分钟
    ActiveDays        int `json:"active_days"`        // 活跃天数: 5 天
}

// AbilityRadar 能力雷达图（对应区域 2）
type AbilityRadar struct {
    AccuracyScore  float64 `json:"accuracy_score"`  // 准确度: 75
    FluencyScore   float64 `json:"fluency_score"`   // 流利度: 82
    IntegrityScore float64 `json:"integrity_score"` // 完整度: 88
    Summary        string  `json:"summary"`         // 一句总结: "整体表现很棒！"
}

// ProgressStats 进步趋势（对应区域 3）
type ProgressStats struct {
    OverallScoreChange int      `json:"overall_score_change"` // 综合分变化: +7
    PreviousScore      float64  `json:"previous_score"`       // 上期分数: 75
    CurrentScore       float64  `json:"current_score"`        // 本期分数: 82
    Highlights         []string `json:"highlights"`           // 进步亮点: ["流利度提升 +8 分", "S/A 级增加 3 次"]
    LevelImprovement   string   `json:"level_improvement"`    // 等级变化: "S/A 级评测增加 3 次"
}

// KidFriendlyCard 给孩子的鼓励卡片（对应区域 4）
type KidFriendlyCard struct {
    EncouragementText string   `json:"encouragement_text"` // "这周你更敢开口了，真棒！"
    Highlights        []string `json:"highlights"`         // ["流利度提升", "发音更清晰"]
    SmallGoal         string   `json:"small_goal"`         // "每天跟读 5 分钟"
}

// DifficultWord 难词卡片（对应区域 5）
type DifficultWord struct {
    Word         string `json:"word"`          // "apples"
    Frequency    int    `json:"frequency"`     // 出现次数: 3
    DemoAudioURL string `json:"demo_audio_url"` // 示范音频 URL
}

// FullReport 完整报告（对应区域 6，给家长看）
type FullReport struct {
    PeriodSummary     string   `json:"period_summary"`      // 本周期小结（鼓励式）
    AbilityAnalysis   string   `json:"ability_analysis"`    // 能力表现分析
    ProgressHighlight string   `json:"progress_highlight"`  // 进步亮点
    ImprovementAreas  []string `json:"improvement_areas"`   // 需要改进的地方（1-2 个）
    Recommendations   []string `json:"recommendations"`     // 具体建议（1-2 条）
    FullText          string   `json:"full_text"`           // 完整报告文本（100-150 字）
}
```

**说明**:
- 每个结构体对应前端的一个区域，字段完全匹配
- `KidFriendlyCard` 是给孩子看的简短版本
- `FullReport` 是给家长看的完整版本
- `DifficultWords` 从 `problem_words` 聚合生成

---

### 🔹 Step 2：定义 LLM Prompt 模板

**目标文件**: `internal/infrastructure/llm/report_prompts.go`

```go
package llm

import "fmt"

// GenerateReportPrompt 生成学习报告的 LLM Prompt
func GenerateReportPrompt(stats *ReportStats) string {
    return fmt.Sprintf(`你是一位专业的儿童英语学习顾问，现在要为一个 6-12 岁的孩子生成本周学习报告。

【本周学习数据】
时间周期：%s 到 %s
对话次数：%d 次
评测次数：%d 次
学习时长：%d 分钟
活跃天数：%d 天

【能力评分（0-100）】
准确度：%.1f 分
流利度：%.1f 分
完整度：%.1f 分
综合分：%.1f 分

【评级分布】
S 级（优秀）：%d 次
A 级（良好）：%d 次
B 级（一般）：%d 次
C 级（需加强）：%d 次

【进步对比】
上周综合分：%.1f 分
本周综合分：%.1f 分
进步：%+.1f 分

【高频问题单词】
%s

---

请生成以下内容（JSON 格式）：

{
  "period_summary": "本周期小结（50字，鼓励式语言，适合孩子阅读）",
  "ability_analysis": "能力表现分析（40字，说明准确度/流利度/完整度表现）",
  "progress_highlight": "进步亮点（30字，强调提升点）",
  "improvement_areas": ["改进点1（温和）", "改进点2（可选）"],
  "recommendations": ["具体建议1（可执行）", "具体建议2（可选）"],
  "encouragement_text": "一句话鼓励（15字内，充满正能量）",
  "highlights": ["亮点1（4字）", "亮点2（4字）"],
  "small_goal": "小目标（10字内，简单明确）",
  "full_text": "完整报告正文（100-150字，口语化，鼓励语气）"
}

【关键要求】
1. 语气温暖、正向、鼓励
2. 不使用"差""不好"等负面词汇
3. 改进点要温和委婉
4. 建议要具体可执行（如"每天跟读 5 分钟"）
5. 适合 6-12 岁儿童理解

请只返回 JSON，不要有其他内容。`,
        stats.StartDate, stats.EndDate,
        stats.ConversationCount, stats.EvaluationCount,
        stats.TotalMinutes, stats.ActiveDays,
        stats.AvgAccuracy, stats.AvgFluency, stats.AvgIntegrity, stats.AvgOverall,
        stats.SCount, stats.ACount, stats.BCount, stats.CCount,
        stats.PreviousScore, stats.CurrentScore, stats.ScoreChange,
        formatProblemWords(stats.ProblemWords))
}

// formatProblemWords 格式化问题单词
func formatProblemWords(words map[string]int) string {
    if len(words) == 0 {
        return "无"
    }
    var result string
    for word, freq := range words {
        result += fmt.Sprintf("- %s (出现 %d 次)\n", word, freq)
    }
    return result
}

// ReportStats 报告统计数据（传给 LLM）
type ReportStats struct {
    StartDate         string
    EndDate           string
    ConversationCount int
    EvaluationCount   int
    TotalMinutes      int
    ActiveDays        int
    AvgAccuracy       float64
    AvgFluency        float64
    AvgIntegrity      float64
    AvgOverall        float64
    SCount            int
    ACount            int
    BCount            int
    CCount            int
    PreviousScore     float64
    CurrentScore      float64
    ScoreChange       float64
    ProblemWords      map[string]int // word -> frequency
}
```

**说明**:
- Prompt 包含完整的统计数据
- 要求 LLM 返回 JSON 格式（方便解析）
- 明确要求儿童友好的语气和用词

---

### 🔹 Step 3：实现 Handler 层

**目标文件**: `internal/handler/report_handler.go`

**需要实现的方法**: `ReportHandler.ReportMVP(c *gin.Context)`

```go
func (h *ReportHandler) ReportMVP(c *gin.Context) {
    // 1. 解析 JSON 请求
    var reqBody struct {
        ReportType string `json:"report_type" binding:"required"`
        StartDate  string `json:"start_date"` // 可选，默认根据 type 计算
        EndDate    string `json:"end_date"`   // 可选，默认今天
    }
    
    if err := c.ShouldBindJSON(&reqBody); err != nil {
        BadRequest(c, "invalid request")
        return
    }
    
    // 2. 计算日期范围（如果未提供）
    startDate := reqBody.StartDate
    endDate := reqBody.EndDate
    
    if endDate == "" {
        endDate = time.Now().Format("2006-01-02")
    }
    
    if startDate == "" {
        // 根据 report_type 计算
        endTime, _ := time.Parse("2006-01-02", endDate)
        switch reqBody.ReportType {
        case "weekly":
            startDate = endTime.AddDate(0, 0, -7).Format("2006-01-02")
        case "monthly":
            startDate = endTime.AddDate(0, -1, 0).Format("2006-01-02")
        default:
            BadRequest(c, "start_date is required for custom report")
            return
        }
    }
    
    // 3. 获取 user_id
    userID := c.GetString("user_id")
    if userID == "" {
        Unauthorized(c)
        return
    }
    
    // 4. 调用 Service
    req := &service.ReportMVPRequest{
        ReportType: reqBody.ReportType,
        StartDate:  startDate,
        EndDate:    endDate,
        UserID:     userID,
    }
    
    ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
    defer cancel()
    
    response, err := h.reportService.ReportMVP(ctx, req)
    if err != nil {
        h.logger.Error("ReportMVP failed", zap.Error(err))
        InternalError(c, err.Error())
        return
    }
    
    // 5. 返回 JSON
    OK(c, response)
}
```

---

### 🔹 Step 4：实现 Service 层（核心逻辑）

**目标文件**: `internal/service/report_service.go`

**需要实现的方法**: `reportServiceImpl.ReportMVP(ctx, req) (*ReportMVPResponse, error)`

**完整实现步骤**:

#### 步骤 1：统计对话数据
```go
// 1. 统计对话数据
convStats, err := s.conversationRepo.GetStatsByUserAndDateRange(ctx, 
    req.UserID, req.StartDate, req.EndDate)
if err != nil {
    s.logger.Error("query conversation stats failed", zap.Error(err))
    convStats = &ConversationStats{} // 使用空数据
}

// convStats 应包含：
// - TotalCount int
// - TotalDurationSeconds int
// - ActiveDays int
// - TopicDistribution map[string]int

totalConversations := convStats.TotalCount
totalMinutes := convStats.TotalDurationSeconds / 60
activeDays := convStats.ActiveDays
```

#### 步骤 2：统计评测数据
```go
// 2. 统计评测数据
evalStats, err := s.evaluationRepo.GetStatsByUserAndDateRange(ctx,
    req.UserID, req.StartDate, req.EndDate)
if err != nil {
    s.logger.Error("query evaluation stats failed", zap.Error(err))
    evalStats = &EvaluationStats{}
}

// evalStats 应包含：
// - TotalCount int
// - AvgOverallScore float64
// - AvgAccuracyScore float64
// - AvgFluencyScore float64
// - AvgIntegrityScore float64
// - SLevelCount int
// - ALevelCount int
// - BLevelCount int
// - CLevelCount int
// - ProblemWords map[string]int

totalEvaluations := evalStats.TotalCount
avgAccuracy := evalStats.AvgAccuracyScore
avgFluency := evalStats.AvgFluencyScore
avgIntegrity := evalStats.AvgIntegrityScore
avgOverall := evalStats.AvgOverallScore
```

#### 步骤 3：查询上期数据（计算进步）
```go
// 3. 查询上期数据（用于对比）
prevStartDate, prevEndDate := calculatePreviousPeriod(req.StartDate, req.EndDate)

prevEvalStats, err := s.evaluationRepo.GetStatsByUserAndDateRange(ctx,
    req.UserID, prevStartDate, prevEndDate)
if err != nil {
    prevEvalStats = &EvaluationStats{AvgOverallScore: 0}
}

previousScore := prevEvalStats.AvgOverallScore
scoreChange := avgOverall - previousScore
```

#### 步骤 4：聚合高频问题单词（Top 3-5）
```go
// 4. 聚合问题单词（取 Top 5）
problemWordsList := aggregateProblemWords(evalStats.ProblemWords, 5)

// problemWordsList 示例：
// [
//   {Word: "apples", Frequency: 5},
//   {Word: "beautiful", Frequency: 3},
//   {Word: "together", Frequency: 2},
// ]
```

#### 步骤 5：调用 LLM 生成报告
```go
// 5. 构建 LLM Prompt
reportStats := &llm.ReportStats{
    StartDate:         req.StartDate,
    EndDate:           req.EndDate,
    ConversationCount: totalConversations,
    EvaluationCount:   totalEvaluations,
    TotalMinutes:      totalMinutes,
    ActiveDays:        activeDays,
    AvgAccuracy:       avgAccuracy,
    AvgFluency:        avgFluency,
    AvgIntegrity:      avgIntegrity,
    AvgOverall:        avgOverall,
    SCount:            evalStats.SLevelCount,
    ACount:            evalStats.ALevelCount,
    BCount:            evalStats.BLevelCount,
    CCount:            evalStats.CLevelCount,
    PreviousScore:     previousScore,
    CurrentScore:      avgOverall,
    ScoreChange:       scoreChange,
    ProblemWords:      evalStats.ProblemWords,
}

llmPrompt := llm.GenerateReportPrompt(reportStats)

// 调用 LLM
llmResponse, err := s.llmProvider.GenerateReply(ctx, llmPrompt, nil)
if err != nil {
    s.logger.Error("LLM failed", zap.Error(err))
    // 降级：使用模板
    llmResponse = generateTemplateReport(reportStats)
}

// 解析 LLM 返回的 JSON
var llmData LLMReportData
if err := json.Unmarshal([]byte(llmResponse), &llmData); err != nil {
    s.logger.Error("parse LLM JSON failed", zap.Error(err))
    llmData = getDefaultReportData()
}
```

#### 步骤 6：为问题单词生成示范音频（TTS）
```go
// 6. 为问题单词生成 TTS 示范音频
difficultWords := make([]DifficultWord, 0, len(problemWordsList))

ttsOptions := &domain.SynthesizeOptions{
    Voice:      "longanyang",
    Format:     "mp3",
    SampleRate: 22050,
}

for _, pw := range problemWordsList {
    // TTS 合成单词
    audioData, err := s.ttsProvider.Synthesize(ctx, pw.Word, ttsOptions)
    if err != nil {
        s.logger.Warn("TTS for problem word failed", 
            zap.String("word", pw.Word), zap.Error(err))
        continue
    }
    
    // 上传到 OSS
    audioKey := fmt.Sprintf("report/words/%s_%s.mp3", req.UserID, pw.Word)
    audioURL, err := s.ossProvider.UploadAudio(ctx, audioKey, audioData)
    if err != nil {
        s.logger.Warn("upload word audio failed", zap.Error(err))
        continue
    }
    
    difficultWords = append(difficultWords, DifficultWord{
        Word:         pw.Word,
        Frequency:    pw.Frequency,
        DemoAudioURL: audioURL,
    })
}
```

#### 步骤 7：保存报告到数据库
```go
// 7. 保存到 learning_reports 表
reportID := uuid.New().String()

report := &model.LearningReport{
    ID:                       reportID,
    UserID:                   req.UserID,
    ReportType:               req.ReportType,
    PeriodStartDate:          parseDate(req.StartDate),
    PeriodEndDate:            parseDate(req.EndDate),
    TotalConversations:       totalConversations,
    TotalEvaluations:         totalEvaluations,
    TotalStudyMinutes:        totalMinutes,
    AverageEvaluationScore:   avgOverall,
    AverageAccuracyScore:     avgAccuracy,
    AverageFluencyScore:      avgFluency,
    AverageIntegrityScore:    avgIntegrity,
    SLevelCount:              evalStats.SLevelCount,
    ALevelCount:              evalStats.ALevelCount,
    BLevelCount:              evalStats.BLevelCount,
    CLevelCount:              evalStats.CLevelCount,
    ImprovementRate:          scoreChange,
    Strengths:                llmData.Highlights,        // 来自 LLM
    Weaknesses:               llmData.ImprovementAreas,  // 来自 LLM
    Recommendations:          &llmData.FullText,         // 完整报告
    CreatedAt:                time.Now(),
    UpdatedAt:                time.Now(),
}

if err := s.reportRepo.Create(ctx, report); err != nil {
    s.logger.Error("save report failed", zap.Error(err))
    // 保存失败不影响返回
}
```

#### 步骤 8：构建响应
```go
// 8. 构建完整响应
return &ReportMVPResponse{
    ReportID:        reportID,
    ReportType:      req.ReportType,
    PeriodStartDate: req.StartDate,
    PeriodEndDate:   req.EndDate,
    
    // 区域 1：活跃度
    ActivityStats: &ActivityStats{
        ConversationCount: totalConversations,
        EvaluationCount:   totalEvaluations,
        TotalMinutes:      totalMinutes,
        ActiveDays:        activeDays,
    },
    
    // 区域 2：雷达图
    AbilityRadar: &AbilityRadar{
        AccuracyScore:  avgAccuracy,
        FluencyScore:   avgFluency,
        IntegrityScore: avgIntegrity,
        Summary:        llmData.AbilityAnalysis,
    },
    
    // 区域 3：进步趋势
    ProgressStats: &ProgressStats{
        OverallScoreChange: int(scoreChange),
        PreviousScore:      previousScore,
        CurrentScore:       avgOverall,
        Highlights:         []string{llmData.ProgressHighlight},
        LevelImprovement:   fmt.Sprintf("S/A 级评测增加 %d 次", 
            (evalStats.SLevelCount + evalStats.ALevelCount) - 
            (prevEvalStats.SLevelCount + prevEvalStats.ALevelCount)),
    },
    
    // 区域 4：给孩子的卡片
    KidFriendlyCard: &KidFriendlyCard{
        EncouragementText: llmData.EncouragementText,
        Highlights:        llmData.Highlights,
        SmallGoal:         llmData.SmallGoal,
    },
    
    // 区域 5：难词卡片
    DifficultWords: difficultWords,
    
    // 区域 6：完整报告
    FullReport: &FullReport{
        PeriodSummary:     llmData.PeriodSummary,
        AbilityAnalysis:   llmData.AbilityAnalysis,
        ProgressHighlight: llmData.ProgressHighlight,
        ImprovementAreas:  llmData.ImprovementAreas,
        Recommendations:   llmData.Recommendations,
        FullText:          llmData.FullText,
    },
}, nil
```

---

## 🔑 关键要求（必须遵守）

### 1. Repository 方法需求

**ConversationRepository 需要新增**:
```go
GetStatsByUserAndDateRange(ctx, userID, startDate, endDate) (*ConversationStats, error)
```

**EvaluationRepository 需要新增**:
```go
GetStatsByUserAndDateRange(ctx, userID, startDate, endDate) (*EvaluationStats, error)
```

这两个方法需要执行复杂的 SQL 聚合查询。

### 2. LLM 返回 JSON 格式

LLM 必须返回严格的 JSON 格式，便于解析。如果解析失败，使用模板降级。

### 3. 问题单词聚合逻辑

```go
// 从所有评测记录的 problem_words JSON 数组中
// 统计每个单词的出现频率
// 取 Top 5
```

### 4. 儿童友好语言

所有文本必须：
- 正向鼓励
- 避免负面词汇
- 温和委婉
- 适合 6-12 岁理解

### 5. 降级策略

```
LLM 失败 → 使用模板文本
TTS 失败 → 跳过该单词
OSS 失败 → 跳过该单词
数据库保存失败 → 不影响返回
```

### 6. 日期计算

```go
// weekly: end_date - 7 天
// monthly: end_date - 1 个月
```

---

## ✅ 验证清单

- [ ] Response 结构体完整（6 个区域）
- [ ] LLM Prompt 模板定义（包含所有统计数据）
- [ ] Handler 日期计算正确
- [ ] Service 8 个步骤完整
- [ ] Repository 聚合查询实现
- [ ] 问题单词 TTS 生成
- [ ] 数据库保存（learning_reports 表）
- [ ] 降级策略完整
- [ ] 编译通过：`go build ./internal/...`

---

## 🚀 生成顺序（严格按照）

### 第 1 步：生成响应结构体
```
生成所有 Response 相关结构体（9 个）
验证：go build ./internal/service/...
```

### 第 2 步：生成 LLM Prompt 模板
```
生成 internal/infrastructure/llm/report_prompts.go
包含 GenerateReportPrompt() 和 ReportStats
验证：go build ./internal/infrastructure/llm/...
```

### 第 3 步：生成 Repository 聚合查询方法
```
在 conversation_repository.go 添加 GetStatsByUserAndDateRange()
在 evaluation_repository.go 添加 GetStatsByUserAndDateRange()
验证：go build ./internal/repository/...
```

### 第 4 步：生成 Handler
```
生成 ReportHandler.ReportMVP() 方法
验证：go build ./internal/handler/...
```

### 第 5 步：生成 Service
```
生成 reportServiceImpl.ReportMVP() 方法（8 个步骤）
验证：go build ./internal/service/...
```

### 第 6 步：最终测试
```bash
go run cmd/server/main.go

curl -X POST http://localhost:8080/api/v1/report/MVP \
  -H "Content-Type: application/json" \
  -d '{
    "report_type": "weekly"
  }'
```

---

## 📊 预期输出示例

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "report_id": "report_xxx",
    "report_type": "weekly",
    "period_start_date": "2024-02-12",
    "period_end_date": "2024-02-18",
    "activity_stats": {
      "conversation_count": 12,
      "evaluation_count": 8,
      "total_minutes": 45,
      "active_days": 5
    },
    "ability_radar": {
      "accuracy_score": 75.0,
      "fluency_score": 82.0,
      "integrity_score": 88.0,
      "summary": "整体表现很棒！"
    },
    "progress_stats": {
      "overall_score_change": 7,
      "previous_score": 75.0,
      "current_score": 82.0,
      "highlights": ["流利度提升 +8 分"],
      "level_improvement": "S/A 级评测增加 3 次"
    },
    "kid_friendly_card": {
      "encouragement_text": "这周你更敢开口了，真棒！",
      "highlights": ["流利度提升", "发音更清晰"],
      "small_goal": "每天跟读 5 分钟"
    },
    "difficult_words": [
      {
        "word": "apples",
        "frequency": 5,
        "demo_audio_url": "https://oss.xxx.com/report/words/xxx.mp3"
      },
      {
        "word": "beautiful",
        "frequency": 3,
        "demo_audio_url": "https://oss.xxx.com/report/words/xxx.mp3"
      }
    ],
    "full_report": {
      "period_summary": "这周你完成了 12 次对话和 8 次评测...",
      "ability_analysis": "准确度良好，流利度进步明显...",
      "progress_highlight": "相比上周，流利度提升了 8 分！",
      "improvement_areas": ["发音准确度还可以更好"],
      "recommendations": ["每天跟读 5 分钟", "慢速朗读练习"],
      "full_text": "这周你完成了 12 次对话练习...（100-150 字）"
    }
  }
}
```

---

现在请按照这个 Prompt，分 6 步生成代码！
