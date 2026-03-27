
---

## 智能周报：完整设计文档

### 一、字段换算规则

```
百分制分数 = round(原始分 × 20)，clamp 到 [0, 100]

示例：
AccuracyScore  4.833651 → round(96.67) = 97
FluencyScore   4.337938 → round(86.76) = 87
IntegrityScore 5.0      → round(100)   = 100
StandardScore  3.516544 → round(70.33) = 70
TotalScore     4.553227 → round(91.06) = 91
```

---

### 二、报告五个区域的完整数据定义

**区域 1：学习活跃度**

| 字段 | SQL | 表 |
|------|-----|----|
| `evaluation_count` | COUNT(*) WHERE is_rejected=false AND created_at BETWEEN ? AND ? | pronunciation_records |
| `conversation_count` | COUNT(*) WHERE created_at BETWEEN ? AND ? | scene_messages |
| `persistence_days` | COUNT(DISTINCT DATE(created_at)) 两表 UNION | 两表 |

**区域 2：能力雷达图（4维度，百分制）**

| 字段 | SQL |
|------|-----|
| `accuracy_score` | round(AVG(accuracy_score) × 20)，无记录则 0 |
| `fluency_score` | round(AVG(fluency_score) × 20)，无记录则 0 |
| `integrity_score` | round(AVG(integrity_score) × 20)，无记录则 0 |
| `standard_score` | round(AVG(standard_score) × 20)，无记录则 0 |

来源：pronunciation_records WHERE is_rejected=false AND created_at BETWEEN ? AND ?

**区域 3：难词卡片（最多4个）**

```
1. 查本周所有 pronunciation_records.problem_words（is_rejected=false）
2. 每条是 JSON 数组字符串，Go 层 json.Unmarshal → []string
3. 用 map[string]int 统计词频
4. 按频率降序取前 4 个
5. 从 PronunciationLoader 配置文件中查找 standard_audio_url
   遍历所有 unit 的所有 item，strings.EqualFold(item.Content, word) 匹配
   找不到则 audio_url = ""
```

**区域 4：场景对话表现**

| 字段 | SQL |
|------|-----|
| `scene_pass_rate` | COUNT(match_result IN ('rule_pass','llm_pass') AND attempt=1) / COUNT(*) × 100，无记录则 0 |
| `completed_scenes` | COUNT(*) FROM scene_sessions WHERE status='completed' AND created_at BETWEEN ? AND ? |

**区域 5：LLM 生成内容（两次调用）**

第一次：生成鼓励卡片（encourage_text）
第二次：生成完整报告（summary + strengths + improvements）

---

### 三、API 设计

```
POST /api/v1/report/generate     生成报告（同步，直接返回完整内容）
GET  /api/v1/report/list         获取历史报告列表
GET  /api/v1/report/:report_id   获取报告详情
```

---

### 四、learning_reports 表需要的字段

扫描现有 learning_reports model，需要确认以下字段存在，不存在则补充：

```go
Content  string `gorm:"type:longtext"                        json:"content"`   // 完整报告JSON
IsLatest bool   `gorm:"type:boolean;not null;default:true"   json:"is_latest"` // 是否最新
```

---




## 前置扫描（必须先做）

```
1. 查看 internal/model/learning_report.go，记录所有现有字段
   重点确认：content、is_latest 字段是否存在
2. 查看现有 LearningReport Repository 接口的所有方法
3. 查看现有 report Service 文件，记录已有方法
4. 查看现有 report Handler 文件，记录已有方法和路由
5. 查看 PronunciationLoader 的方法签名
   重点：GetAll() 和 GetByID() 的返回结构，item.Content 和 item.StandardAudioURL 字段名
6. 查看现有 LLM Provider 接口的调用方式
7. 查看现有统一响应封装函数签名
8. 查看 JWT 中间件注入 user_id 的方式
9. 查看路由注册方式
10. 读完后再开始写代码
```

---

## 一、Model 层

### 1.1 确认/补充 learning_reports 字段

扫描 `internal/model/learning_report.go`，若缺少以下字段则在结构体末尾追加：

```go
// 若不存在则新增（不修改现有字段）
Content  string `gorm:"type:longtext"                                      json:"content"`
IsLatest bool   `gorm:"type:boolean;not null;default:true;index"           json:"is_latest"`
```

### 1.2 Migration SQL

在项目现有 migration 机制中追加：

```sql
-- 若字段不存在则添加
ALTER TABLE learning_reports
    ADD COLUMN IF NOT EXISTS content   LONGTEXT      COMMENT '完整报告JSON内容',
    ADD COLUMN IF NOT EXISTS is_latest BOOLEAN NOT NULL DEFAULT TRUE COMMENT '是否为该周期最新报告';

CREATE INDEX IF NOT EXISTS idx_lr_user_latest
    ON learning_reports(user_id, is_latest);
```

---

## 二、数据结构定义

新建 `internal/service/report_types.go`（若项目有统一的 types 目录则放在对应位置）：

```go
package service

// WeeklyReportData 周报完整数据结构，序列化后存入 learning_reports.content
type WeeklyReportData struct {
    WeekStart string              `json:"week_start"` // "2026-03-16"
    WeekEnd   string              `json:"week_end"`   // "2026-03-22"
    Activity  ReportActivity      `json:"activity"`
    Radar     ReportRadar         `json:"radar"`
    HardWords []ReportHardWord    `json:"hard_words"`
    Scene     ReportScene         `json:"scene"`
    Encourage ReportEncourage     `json:"encourage"`
    FullReport ReportFullContent  `json:"full_report"`
}

type ReportActivity struct {
    EvaluationCount   int `json:"evaluation_count"`
    ConversationCount int `json:"conversation_count"`
    PersistenceDays   int `json:"persistence_days"`
}

type ReportRadar struct {
    AccuracyScore  int `json:"accuracy_score"`  // 0-100
    FluencyScore   int `json:"fluency_score"`
    IntegrityScore int `json:"integrity_score"`
    StandardScore  int `json:"standard_score"`
}

type ReportHardWord struct {
    Word     string `json:"word"`
    Count    int    `json:"count"`
    AudioURL string `json:"audio_url"`
}

type ReportScene struct {
    PassRate        int `json:"pass_rate"`        // 0-100
    CompletedScenes int `json:"completed_scenes"`
}

type ReportEncourage struct {
    EncourageText string `json:"encourage_text"`
}

type ReportFullContent struct {
    Summary      string   `json:"summary"`
    Strengths    []string `json:"strengths"`
    Improvements []string `json:"improvements"`
}

// HardWordStat 难词统计中间结构（内部使用）
type HardWordStat struct {
    Word  string
    Count int
}
```

---

## 三、Repository 层

在现有 LearningReport Repository 接口中追加以下方法，并用 GORM 实现。
**不新建文件，追加到现有 Repository 接口和实现文件中。**

```go
// ── 报告 CRUD ────────────────────────────────────────────

// FindByID 根据报告 ID 查询，不存在返回 nil, nil
FindByID(ctx context.Context, reportID string) (*model.LearningReport, error)

// ListLatestByUserID 查询用户所有 is_latest=true 的报告
// ORDER BY period_start_date DESC
ListLatestByUserID(ctx context.Context, userID string) ([]*model.LearningReport, error)

// UpdateContent 更新报告内容（content 字段）
// UPDATE learning_reports SET content=? WHERE id=?
UpdateContent(ctx context.Context, reportID string, content string) error

// UpdateIsLatest 将同周期旧报告标记为非最新
// UPDATE learning_reports SET is_latest=false
// WHERE user_id=? AND report_type='weekly' AND period_start_date=? AND id != excludeID
UpdateIsLatest(ctx context.Context, userID string, periodStart time.Time, excludeID string) error

// ── 统计查询（供报告 Worker 使用）────────────────────────

// CountEvaluations 统计本周有效发音评测次数（is_rejected=false）
// SELECT COUNT(*) FROM pronunciation_records
// WHERE user_id=? AND is_rejected=false AND created_at BETWEEN ? AND ?
CountEvaluations(ctx context.Context, userID string, start, end time.Time) (int, error)

// CountConversations 统计本周场景对话次数
// SELECT COUNT(*) FROM scene_messages
// WHERE user_id=? AND created_at BETWEEN ? AND ?
CountConversations(ctx context.Context, userID string, start, end time.Time) (int, error)

// CountPersistenceDays 统计本周有练习记录的天数（两表取并集）
// SELECT COUNT(DISTINCT DATE(created_at)) FROM (
//   SELECT created_at FROM pronunciation_records
//   WHERE user_id=? AND created_at BETWEEN ? AND ?
//   UNION
//   SELECT created_at FROM scene_messages
//   WHERE user_id=? AND created_at BETWEEN ? AND ?
// ) t
CountPersistenceDays(ctx context.Context, userID string, start, end time.Time) (int, error)

// GetAvgScores 查询本周发音评测各维度平均分（原始值 0-5）
// SELECT AVG(accuracy_score), AVG(fluency_score), AVG(integrity_score), AVG(standard_score)
// FROM pronunciation_records
// WHERE user_id=? AND is_rejected=false AND created_at BETWEEN ? AND ?
// 无记录时四个值均返回 0
GetAvgScores(ctx context.Context, userID string, start, end time.Time) (
    accuracy, fluency, integrity, standard float64, err error,
)

// GetProblemWordsList 查询本周所有有效评测的 problem_words 字段原始值列表
// SELECT problem_words FROM pronunciation_records
// WHERE user_id=? AND is_rejected=false AND created_at BETWEEN ? AND ?
// 返回 []string，每个元素是 JSON 数组字符串，如 `["pork","chicken"]`
// 空的 problem_words 跳过
GetProblemWordsList(ctx context.Context, userID string, start, end time.Time) ([]string, error)

// GetSceneStats 查询本周场景对话统计
// pass_rate: COUNT(match_result IN ('rule_pass','llm_pass') AND attempt=1) / COUNT(*) × 100
// completed_scenes: COUNT(*) FROM scene_sessions WHERE status='completed' AND created_at BETWEEN ? AND ?
// 无 scene_messages 记录时 pass_rate=0，无 scene_sessions 记录时 completed_scenes=0
GetSceneStats(ctx context.Context, userID string, start, end time.Time) (passRate, completedScenes int, err error)
```

---

## 四、Service 层

### 4.1 GenerateWeeklyReport（核心方法，同步实现）

在现有 report Service 文件中新增此方法（或重写现有的 GenerateReport）：

```
入参：ctx context.Context, userID string
返回：*WeeklyReportData, reportID string, error

步骤 1：计算本周时间范围
  now := time.Now()
  weekday := int(now.Weekday())
  if weekday == 0 { weekday = 7 }  // Sunday=0 → 7
  weekStart = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
  weekEnd   = time.Date(now.Year(), now.Month(), now.Day()-weekday+7, 23, 59, 59, 999999999, now.Location())

步骤 2：数据量检查
  evalCount  = repo.CountEvaluations(ctx, userID, weekStart, weekEnd)
  convCount  = repo.CountConversations(ctx, userID, weekStart, weekEnd)
  if evalCount + convCount < 3:
    return nil, "", errors.New("本周学习记录不足，至少需要 3 条")

步骤 3：幂等检查（冷却期 2 小时）
  查询最近一条该用户本周的报告：
    SELECT * FROM learning_reports
    WHERE user_id=? AND report_type='weekly' AND period_start_date=weekStart
    ORDER BY created_at DESC LIMIT 1
  若存在且 time.Since(latestReport.CreatedAt) < 2*time.Hour:
    反序列化 latestReport.Content → WeeklyReportData
    返回已有数据和 report_id（幂等）

步骤 4：计算活跃度数据
  persistenceDays = repo.CountPersistenceDays(ctx, userID, weekStart, weekEnd)
  activity = ReportActivity{
    EvaluationCount:   evalCount,
    ConversationCount: convCount,
    PersistenceDays:   persistenceDays,
  }

步骤 5：计算雷达图数据
  accuracy, fluency, integrity, standard = repo.GetAvgScores(ctx, userID, weekStart, weekEnd)
  // 换算百分制：round(原始分 × 20)，clamp 到 [0, 100]
  toPercent := func(v float64) int {
    result := int(math.Round(v * 20))
    if result < 0 { return 0 }
    if result > 100 { return 100 }
    return result
  }
  radar = ReportRadar{
    AccuracyScore:  toPercent(accuracy),
    FluencyScore:   toPercent(fluency),
    IntegrityScore: toPercent(integrity),
    StandardScore:  toPercent(standard),
  }

步骤 6：计算难词卡片
  rawList = repo.GetProblemWordsList(ctx, userID, weekStart, weekEnd)
  wordFreq := map[string]int{}
  for _, raw := range rawList:
    var words []string
    json.Unmarshal([]byte(raw), &words)
    for _, w := range words:
      wordFreq[strings.ToLower(w)]++
  // 排序取前4
  sorted = sort wordFreq by count DESC, take first 4
  hardWords = []ReportHardWord{}
  for _, stat := range sorted:
    audioURL = findAudioURL(stat.Word)  // 见步骤 6.1
    hardWords = append(hardWords, ReportHardWord{Word: stat.Word, Count: stat.Count, AudioURL: audioURL})

步骤 6.1：查找 standard_audio_url（内部辅助函数）
  func findAudioURL(word string) string:
    for _, unit := range pronunciationLoader.GetAll():
      for _, item := range unit.Items:
        if strings.EqualFold(item.Content, word):
          return item.StandardAudioURL
    return ""

步骤 7：查询场景对话表现
  passRate, completedScenes = repo.GetSceneStats(ctx, userID, weekStart, weekEnd)
  scene = ReportScene{PassRate: passRate, CompletedScenes: completedScenes}

步骤 8：调用 LLM 生成鼓励卡片
  prompt = buildEncouragePrompt(activity, radar.AccuracyScore)
  llmResult = llmProvider.Chat(ctx, prompt)
  解析 JSON → { "encourage_text": "..." }
  解析失败 → fallback: "你这周练习很认真！🌟 继续加油，你的英语越来越棒了！💪"
  encourage = ReportEncourage{EncourageText: encourageText}

步骤 9：调用 LLM 生成完整报告总结
  prompt = buildFullReportPrompt(weekStart, weekEnd, activity, radar, scene, hardWords)
  llmResult = llmProvider.Chat(ctx, prompt)
  解析 JSON → { "summary": "...", "strengths": [...], "improvements": [...] }
  解析失败 → fallback:
    summary: "这周学习很认真！发音练习和对话都有参与，继续保持！"
    strengths: ["坚持练习"]
    improvements: ["多练习发音"]

步骤 10：组装完整报告数据
  reportData = WeeklyReportData{
    WeekStart:  weekStart.Format("2006-01-02"),
    WeekEnd:    weekEnd.Format("2006-01-02"),
    Activity:   activity,
    Radar:      radar,
    HardWords:  hardWords,
    Scene:      scene,
    Encourage:  encourage,
    FullReport: fullReport,
  }

步骤 11：落库
  contentJSON = json.Marshal(reportData)
  新建 LearningReport 记录：
    id = UUID()
    user_id = userID
    report_type = "weekly"
    period_start_date = weekStart
    period_end_date   = weekEnd
    content = contentJSON
    is_latest = true
  INSERT INTO learning_reports
  repo.UpdateIsLatest(ctx, userID, weekStart, newReportID)  // 旧报告标记为非最新

步骤 12：返回 reportData 和 newReportID
```

### 4.2 LLM Prompt 构建函数

在 Service 文件中定义以下两个内部函数：

```go
func buildEncouragePrompt(activity ReportActivity, accuracyScore int) string {
    system := `你是一位儿童英语学习助手，说话简短活泼，多用 emoji，语气温暖鼓励。
只能返回 JSON，不允许返回任何其他内容。`

    user := fmt.Sprintf(`小朋友这周的学习情况：
- 发音评测次数：%d 次
- 场景对话次数：%d 次
- 坚持学习天数：%d 天
- 发音准确度：%d 分（满分100）

请生成一段简短鼓励，只返回：
{"encourage_text": "两句话以内，儿童友好，多 emoji"}`,
        activity.EvaluationCount,
        activity.ConversationCount,
        activity.PersistenceDays,
        accuracyScore)

    // 按项目现有 LLM Provider 的调用约定拼接 system+user
    // 如果 LLM Provider 接受 []ChatMessage，则构建对应结构
    return user  // 实际返回方式参照现有 LLM 调用代码
}

func buildFullReportPrompt(weekStart, weekEnd string,
    activity ReportActivity, radar ReportRadar,
    scene ReportScene, hardWords []ReportHardWord) string {

    hardWordsText := ""
    for _, hw := range hardWords {
        hardWordsText += fmt.Sprintf("- %s（出现 %d 次）\n", hw.Word, hw.Count)
    }
    if hardWordsText == "" {
        hardWordsText = "本周无高频难词，发音很棒！"
    }

    return fmt.Sprintf(`以下是小朋友本周（%s 至 %s）的英语学习数据：

学习活跃度：
- 发音评测：%d 次（有效）
- 场景对话：%d 次
- 坚持学习：%d 天（共7天）

发音能力评估（满分100）：
- 准确度：%d
- 流利度：%d
- 完整度：%d
- 标准度：%d

场景对话表现：
- 一次通过率：%d%%
- 完成场景数：%d

本周难词：
%s
请生成本周学习报告，只返回如下 JSON：
{"summary":"3-5句话总结，包含优点和改进建议","strengths":["优点1","优点2"],"improvements":["建议1"]}`,
        weekStart, weekEnd,
        activity.EvaluationCount, activity.ConversationCount, activity.PersistenceDays,
        radar.AccuracyScore, radar.FluencyScore, radar.IntegrityScore, radar.StandardScore,
        scene.PassRate, scene.CompletedScenes,
        hardWordsText)
}
```

---

## 五、Handler 层

在现有 report Handler 文件中新增/修改以下三个方法：

### 5.1 GenerateReport（修改现有或新增）

```
POST /api/v1/report/generate

逻辑：
1. 从 Context 取 user_id
2. 调用 reportService.GenerateWeeklyReport(ctx, userID)
   - 返回 400：数据不足
   - 返回 200：成功（幂等情况也返回 200）
3. 返回完整报告数据（同步，不返回 task_id）
```

**Response 200**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "report_id": "uuid",
    "week_start": "2026-03-16",
    "week_end": "2026-03-22",
    "activity": {
      "evaluation_count": 8,
      "conversation_count": 12,
      "persistence_days": 5
    },
    "radar": {
      "accuracy_score": 97,
      "fluency_score": 87,
      "integrity_score": 100,
      "standard_score": 70
    },
    "hard_words": [
      { "word": "chicken", "count": 3, "audio_url": "https://oss.../chicken.mp3" },
      { "word": "morning", "count": 2, "audio_url": "" }
    ],
    "scene": {
      "pass_rate": 75,
      "completed_scenes": 2
    },
    "encourage": {
      "encourage_text": "你这周超棒的！🌟 发音评测做了8次，继续加油！💪"
    },
    "full_report": {
      "summary": "这周你完成了12次对话和8次发音评测，非常努力！准确度和完整度表现优秀，下周可以多练习chicken和morning。",
      "strengths": ["发音准确度达到97分", "坚持学习5天"],
      "improvements": ["加强chicken的发音练习"]
    }
  }
}
```

**Response 400（数据不足）**
```json
{
  "code": 400,
  "message": "本周学习记录不足，至少需要 3 条",
  "data": null
}
```

### 5.2 GetReportList

```
GET /api/v1/report/list

逻辑：
1. 从 Context 取 user_id
2. repo.ListLatestByUserID(ctx, userID)
3. 每条报告反序列化 content 字段，提取摘要数据
4. 返回列表
```

**Response 200**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "reports": [
      {
        "report_id": "uuid",
        "week_start": "2026-03-16",
        "week_end": "2026-03-22",
        "created_at": "2026-03-23T10:00:00Z",
        "accuracy_score": 97,
        "persistence_days": 5,
        "evaluation_count": 8
      }
    ]
  }
}
```

### 5.3 GetReportDetail

```
GET /api/v1/report/:report_id

逻辑：
1. 从 Context 取 user_id
2. repo.FindByID(ctx, reportID)
   不存在 → 404
   report.UserID != user_id → 403
3. json.Unmarshal(report.Content) → WeeklyReportData
4. 返回完整报告（结构同 GenerateReport 的响应）
```

---

## 六、路由注册

参照现有路由注册方式，在需要 Auth 的路由组中确认/新增：

```go
reportGroup := v1.Group("/report", middleware.AuthMiddleware())
{
    reportGroup.POST("/generate",        reportHandler.GenerateReport)    // 新增或修改
    reportGroup.GET("/list",             reportHandler.GetReportList)     // 新增
    reportGroup.GET("/:report_id",       reportHandler.GetReportDetail)   // 新增
}
```

---

## 七、代码约束

1. **同步实现**：GenerateWeeklyReport 在 Service 里直接完成所有计算，不使用 Worker Pool，不返回 task_id
2. **统计方法追加到现有 Repository**：不新建 Repository 文件，所有统计方法追加到现有 LearningReportRepository 接口及其实现
3. **LLM 调用方式参照现有代码**：扫描项目现有 LLM 调用代码，使用相同的调用方式和参数结构，不自行假设接口
4. **JSON 解析失败必须 fallback**：两次 LLM 调用的结果解析失败时使用预设 fallback，不中断流程也不返回错误
5. **is_rejected=false 过滤**：所有 pronunciation_records 的统计查询都加上 `AND is_rejected=false` 条件
6. **难词查找找不到不报错**：`findAudioURL` 找不到匹配项时返回空字符串，不返回 error
7. **幂等返回已有数据**：冷却期内重复请求直接返回已有报告内容，不重新计算也不重新调用 LLM
8. **不修改现有功能代码**：只在报告相关文件中新增/修改，不改动发音纠正和场景引导的现有代码