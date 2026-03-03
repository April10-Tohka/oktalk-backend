// Package service 提供智能学习报告业务逻辑
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"pronunciation-correction-system/internal/cache"
	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	llmPrompts "pronunciation-correction-system/internal/infrastructure/llm"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/pkg/uuid"
	"pronunciation-correction-system/internal/worker"
)

// ===== 请求结构 =====

// ReportMVPRequest MVP 同步生成学习报告请求
type ReportMVPRequest struct {
	ReportType string // weekly / monthly / custom
	StartDate  string // YYYY-MM-DD（可选）
	EndDate    string // YYYY-MM-DD（可选）
	UserID     string
}

// GenerateReportRequest 异步生成学习报告请求
type GenerateReportRequest struct {
	ReportType         string // daily / weekly / monthly / custom
	StartDate          string
	EndDate            string
	IncludeEvaluations bool
	IncludeChatStats   bool
	CustomPrompt       string
	UserID             string
}

// ===== 响应结构 =====

// ReportMVPResponse MVP 报告响应（完全对应前端 6 个区域）
type ReportMVPResponse struct {
	// === 顶部：基本信息 ===
	ReportID        string `json:"report_id"`
	ReportType      string `json:"report_type"`       // weekly/monthly
	PeriodStartDate string `json:"period_start_date"` // 2024-02-12
	PeriodEndDate   string `json:"period_end_date"`   // 2024-02-18

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
	ConversationCount int `json:"conversation_count"` // 对话次数
	EvaluationCount   int `json:"evaluation_count"`   // 评测次数
	TotalMinutes      int `json:"total_minutes"`      // 学习时长（分钟）
	ActiveDays        int `json:"active_days"`        // 活跃天数
}

// AbilityRadar 能力雷达图（对应区域 2）
type AbilityRadar struct {
	AccuracyScore  float64 `json:"accuracy_score"`  // 准确度
	FluencyScore   float64 `json:"fluency_score"`   // 流利度
	IntegrityScore float64 `json:"integrity_score"` // 完整度
	Summary        string  `json:"summary"`         // 一句总结
}

// ProgressStats 进步趋势（对应区域 3）
type ProgressStats struct {
	OverallScoreChange int      `json:"overall_score_change"` // 综合分变化
	PreviousScore      float64  `json:"previous_score"`       // 上期分数
	CurrentScore       float64  `json:"current_score"`        // 本期分数
	Highlights         []string `json:"highlights"`           // 进步亮点
	LevelImprovement   string   `json:"level_improvement"`    // 等级变化描述
}

// KidFriendlyCard 给孩子的鼓励卡片（对应区域 4）
type KidFriendlyCard struct {
	EncouragementText string   `json:"encouragement_text"` // 一句话鼓励
	Highlights        []string `json:"highlights"`         // 亮点列表
	SmallGoal         string   `json:"small_goal"`         // 小目标
}

// DifficultWord 难词卡片（对应区域 5）
type DifficultWord struct {
	Word         string `json:"word"`           // 单词
	Frequency    int    `json:"frequency"`      // 出现次数
	DemoAudioURL string `json:"demo_audio_url"` // 示范音频 URL
}

// FullReport 完整报告（对应区域 6，给家长看）
type FullReport struct {
	PeriodSummary     string   `json:"period_summary"`     // 周期小结
	AbilityAnalysis   string   `json:"ability_analysis"`   // 能力分析
	ProgressHighlight string   `json:"progress_highlight"` // 进步亮点
	ImprovementAreas  []string `json:"improvement_areas"`  // 改进点
	Recommendations   []string `json:"recommendations"`    // 建议
	FullText          string   `json:"full_text"`          // 完整文本
}

// ===== 其他响应结构（保留） =====

// ReportStatusResponse 报告生成状态
type ReportStatusResponse struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
}

// ReportDetailResponse 报告完整详情（也作为异步轮询响应）
type ReportDetailResponse struct {
	ReportID     string `json:"report_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
	ReportType   string `json:"report_type,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	StartDate    string `json:"start_date,omitempty"`
	EndDate      string `json:"end_date,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	// 完整报告内容（status=success 时填充）
	Report *ReportMVPResponse `json:"report,omitempty"`
}

// ReportSummary 报告列表摘要
type ReportSummary struct {
	ReportID   string `json:"report_id"`
	ReportType string `json:"report_type"`
	CreatedAt  string `json:"created_at"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
}

// DashboardResponse 学习统计面板响应
type DashboardResponse struct {
	TotalConversations int       `json:"total_conversations"`
	TotalEvaluations   int       `json:"total_evaluations"`
	TotalStudyMinutes  int       `json:"total_study_minutes"`
	AverageScore       float64   `json:"average_score"`
	RecentEvaluations  int       `json:"recent_evaluations"`
	ScoreTrend         []float64 `json:"score_trend"`
}

// ===== Service 接口 =====

// ReportService 智能学习报告业务接口
type ReportService interface {
	// ReportMVP 同步生成学习报告 MVP（统计 + LLM 生成）
	ReportMVP(ctx context.Context, req *ReportMVPRequest) (*ReportMVPResponse, error)

	// GenerateReport 提交异步报告生成任务
	GenerateReport(ctx context.Context, req *GenerateReportRequest) (reportID string, err error)

	// GetReportStatus 查询报告生成进度
	GetReportStatus(ctx context.Context, reportID string) (*ReportStatusResponse, error)

	// GetReport 获取报告完整详情
	GetReport(ctx context.Context, reportID, userID string) (*ReportDetailResponse, error)

	// GetReportList 获取用户报告列表
	GetReportList(ctx context.Context, userID string, page, pageSize int) ([]*ReportSummary, int64, error)

	// DeleteReport 删除报告
	DeleteReport(ctx context.Context, reportID, userID string) error

	// GetDashboard 获取学习统计面板
	GetDashboard(ctx context.Context, userID string) (*DashboardResponse, error)
}

// ===== Service 实现 =====

// reportServiceImpl Report Service 实现
type reportServiceImpl struct {
	repos         *db.Repositories
	llmProvider   domain.LLMProvider
	ttsProvider   domain.TTSProvider
	ossProvider   domain.OSSProvider
	taskCache     *cache.TaskCache
	reportCache   *cache.ReportCache
	workerManager *worker.Manager
	logger        *slog.Logger
}

// NewReportService 创建 ReportService
func NewReportService(
	repos *db.Repositories,
	llmProvider domain.LLMProvider,
	ttsProvider domain.TTSProvider,
	ossProvider domain.OSSProvider,
	taskCache *cache.TaskCache,
	reportCache *cache.ReportCache,
	workerMgr *worker.Manager,
	logger *slog.Logger,
) ReportService {
	return &reportServiceImpl{
		repos:         repos,
		llmProvider:   llmProvider,
		ttsProvider:   ttsProvider,
		ossProvider:   ossProvider,
		taskCache:     taskCache,
		reportCache:   reportCache,
		workerManager: workerMgr,
		logger:        logger,
	}
}

func (s *reportServiceImpl) ReportMVP(ctx context.Context, req *ReportMVPRequest) (*ReportMVPResponse, error) {
	// ─── 基础校验 ───
	if req == nil {
		return nil, errors.New("report mvp request is nil")
	}
	if req.UserID == "" {
		return nil, errors.New("user id is empty")
	}

	// ─── 1. 计算时间范围 ───
	now := time.Now()
	var periodStart, periodEnd time.Time
	reportType := req.ReportType
	if reportType == "" {
		reportType = "weekly"
	}
	switch reportType {
	case "monthly":
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 1, 0)
	default: // weekly
		reportType = "weekly"
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		periodStart = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 7)
	}

	s.logger.Info("ReportMVP started",
		slog.String("user_id", req.UserID),
		slog.String("type", reportType),
		slog.String("start", periodStart.Format("2006-01-02")),
		slog.String("end", periodEnd.Format("2006-01-02")),
	)

	// ─── 2. 统计对话数据 ───
	convStats, err := s.repos.VoiceConversation.GetStatsByUserAndDateRange(ctx, req.UserID, periodStart, periodEnd)
	if err != nil {
		logger.WarnContext(ctx, "query conversation stats failed, use zero values", "error", err)
		convStats = &db.ConversationStats{}
	}

	// ─── 3. 统计评测数据 ───
	evalStats, err := s.repos.PronunciationEvaluation.GetStatsByUserAndDateRange(ctx, req.UserID, periodStart, periodEnd)
	if err != nil {
		logger.WarnContext(ctx, "query evaluation stats failed, use zero values", "error", err)
		evalStats = &db.EvaluationStats{ProblemWords: make(map[string]int)}
	}

	// ─── 4. 查询上期数据（进步对比） ───
	var prevStart, prevEnd time.Time
	switch reportType {
	case "monthly":
		prevStart = periodStart.AddDate(0, -1, 0)
		prevEnd = periodStart
	default:
		prevStart = periodStart.AddDate(0, 0, -7)
		prevEnd = periodStart
	}
	prevAvgScore, _ := s.repos.PronunciationEvaluation.GetAverageScoreByUserIDAndDateRange(ctx, req.UserID, prevStart, prevEnd)
	currentScore := evalStats.AvgOverallScore
	scoreChange := currentScore - prevAvgScore

	// ─── 5. 高频问题单词 Top 5 ───
	type wordFreq struct {
		Word      string
		Frequency int
	}
	var sortedWords []wordFreq
	for w, f := range evalStats.ProblemWords {
		sortedWords = append(sortedWords, wordFreq{Word: w, Frequency: f})
	}
	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].Frequency > sortedWords[j].Frequency
	})
	if len(sortedWords) > 5 {
		sortedWords = sortedWords[:5]
	}

	// ─── 6. LLM 生成报告 ───
	llmStats := &llmPrompts.ReportStats{
		StartDate:         periodStart.Format("2006-01-02"),
		EndDate:           periodEnd.Format("2006-01-02"),
		ConversationCount: convStats.TotalCount,
		EvaluationCount:   evalStats.TotalCount,
		TotalMinutes:      convStats.TotalDurationSeconds / 60,
		ActiveDays:        convStats.ActiveDays,
		AvgAccuracy:       evalStats.AvgAccuracyScore,
		AvgFluency:        evalStats.AvgFluencyScore,
		AvgIntegrity:      evalStats.AvgIntegrityScore,
		AvgOverall:        currentScore,
		SCount:            evalStats.SLevelCount,
		ACount:            evalStats.ALevelCount,
		BCount:            evalStats.BLevelCount,
		CCount:            evalStats.CLevelCount,
		PreviousScore:     prevAvgScore,
		CurrentScore:      currentScore,
		ScoreChange:       scoreChange,
		ProblemWords:      evalStats.ProblemWords,
	}

	systemPrompt, userMsg := llmPrompts.GenerateReportPrompt(llmStats)
	var reportData llmPrompts.LLMReportData

	llmResponse, err := s.llmProvider.Chat(ctx, systemPrompt, userMsg)
	if err != nil {
		logger.WarnContext(ctx, "LLM report generation failed, use defaults", "error", err)
		reportData = llmPrompts.GetDefaultReportData()
	} else {
		// 尝试解析 JSON
		if parseErr := json.Unmarshal([]byte(llmResponse), &reportData); parseErr != nil {
			logger.WarnContext(ctx, "LLM response JSON parse failed, use defaults", "error", parseErr, "response", llmResponse)
			reportData = llmPrompts.GetDefaultReportData()
		}
	}

	// ─── 7. TTS 生成问题单词示范音频 + OSS 上传 ───
	var difficultWords []DifficultWord
	for _, wf := range sortedWords {
		dw := DifficultWord{
			Word:      wf.Word,
			Frequency: wf.Frequency,
		}

		// TTS 合成
		if s.ttsProvider != nil && s.ossProvider != nil {
			ttsData, ttsErr := s.ttsProvider.Synthesize(ctx, wf.Word, nil)
			if ttsErr == nil && len(ttsData) > 0 {
				ossKey := fmt.Sprintf("report/demo/%s/%s_%s.mp3",
					req.UserID, wf.Word, time.Now().Format("20060102150405"))
				audioURL, ossErr := s.ossProvider.UploadBytes(ctx, ossKey, ttsData, "audio/mpeg")
				if ossErr == nil {
					dw.DemoAudioURL = audioURL
				} else {
					logger.WarnContext(ctx, "OSS upload demo audio failed", "word", wf.Word, "error", ossErr)
				}
			} else if ttsErr != nil {
				logger.WarnContext(ctx, "TTS synthesize demo word failed", "word", wf.Word, "error", ttsErr)
			}
		}

		difficultWords = append(difficultWords, dw)
	}

	// ─── 8. 保存报告到数据库 ───
	reportID := uuid.New()
	totalMinutes := convStats.TotalDurationSeconds / 60

	reportModel := &model.LearningReport{
		ID:              reportID,
		UserID:          req.UserID,
		ReportType:      reportType,
		PeriodStartDate: periodStart,
		PeriodEndDate:   periodEnd,
		// 统计
		TotalConversations: convStats.TotalCount,
		TotalEvaluations:   evalStats.TotalCount,
		TotalStudyMinutes:  totalMinutes,
		// 平均分数
		AverageEvaluationScore: currentScore,
		// 雷达图
		AverageAccuracyScore:  evalStats.AvgAccuracyScore,
		AverageFluencyScore:   evalStats.AvgFluencyScore,
		AverageIntegrityScore: evalStats.AvgIntegrityScore,
		// 分级统计
		SLevelCount: evalStats.SLevelCount,
		ALevelCount: evalStats.ALevelCount,
		BLevelCount: evalStats.BLevelCount,
		CLevelCount: evalStats.CLevelCount,
		// 进步
		ImprovementRate: scoreChange,
		// AI 建议
		Recommendations: toStringPtr(reportData.FullText),
		Strengths:       model.StringArray(reportData.Highlights),
		Weaknesses:      model.StringArray(reportData.ImprovementAreas),
	}
	if err := s.repos.LearningReport.Create(ctx, reportModel); err != nil {
		logger.WarnContext(ctx, "save report to db failed", "error", err)
		// 不中断返回，报告内容仍可返回给前端
	}

	// ─── 9. 组装响应 ───
	resp := &ReportMVPResponse{
		ReportID:        reportID,
		ReportType:      reportType,
		PeriodStartDate: periodStart.Format("2006-01-02"),
		PeriodEndDate:   periodEnd.Format("2006-01-02"),

		ActivityStats: &ActivityStats{
			ConversationCount: convStats.TotalCount,
			EvaluationCount:   evalStats.TotalCount,
			TotalMinutes:      totalMinutes,
			ActiveDays:        convStats.ActiveDays,
		},

		AbilityRadar: &AbilityRadar{
			AccuracyScore:  evalStats.AvgAccuracyScore,
			FluencyScore:   evalStats.AvgFluencyScore,
			IntegrityScore: evalStats.AvgIntegrityScore,
			Summary:        reportData.AbilityAnalysis,
		},

		ProgressStats: &ProgressStats{
			OverallScoreChange: int(scoreChange),
			PreviousScore:      prevAvgScore,
			CurrentScore:       currentScore,
			Highlights:         reportData.Highlights,
			LevelImprovement:   reportData.ProgressHighlight,
		},

		KidFriendlyCard: &KidFriendlyCard{
			EncouragementText: reportData.EncouragementText,
			Highlights:        reportData.Highlights,
			SmallGoal:         reportData.SmallGoal,
		},

		DifficultWords: difficultWords,

		FullReport: &FullReport{
			PeriodSummary:     reportData.PeriodSummary,
			AbilityAnalysis:   reportData.AbilityAnalysis,
			ProgressHighlight: reportData.ProgressHighlight,
			ImprovementAreas:  reportData.ImprovementAreas,
			Recommendations:   reportData.Recommendations,
			FullText:          reportData.FullText,
		},
	}

	// 保证 slice 字段非 nil
	if resp.DifficultWords == nil {
		resp.DifficultWords = []DifficultWord{}
	}

	s.logger.Info("ReportMVP completed", slog.String("report_id", reportID))
	return resp, nil
}

// toIntPtr 辅助：int → *int
func toIntPtr(v int) *int { return &v }

// toStringPtr 辅助：string → *string
func toStringPtr(v string) *string { return &v }

func (s *reportServiceImpl) GenerateReport(ctx context.Context, req *GenerateReportRequest) (string, error) {
	// 步骤 1：基础校验
	if req == nil {
		return "", errors.New("generate report request is nil")
	}
	if req.UserID == "" {
		return "", errors.New("user id is empty")
	}

	// 步骤 2：校验 report_type
	reportType := strings.TrimSpace(req.ReportType)
	if reportType == "" {
		reportType = "weekly"
	}
	if reportType != "weekly" && reportType != "monthly" {
		return "", fmt.Errorf("invalid report_type: %s (must be weekly or monthly)", reportType)
	}

	// 步骤 3：确定日期范围
	now := time.Now()
	var periodStart, periodEnd time.Time
	switch reportType {
	case "monthly":
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
	default: // weekly
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		periodStart = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 7).Add(-time.Nanosecond)
	}

	// 步骤 4：检查数据量是否足够
	evalStats, _ := s.repos.PronunciationEvaluation.GetStatsByUserAndDateRange(ctx, req.UserID, periodStart, periodEnd)
	convStats, _ := s.repos.VoiceConversation.GetStatsByUserAndDateRange(ctx, req.UserID, periodStart, periodEnd)
	activityCount := 0
	if evalStats != nil {
		activityCount += evalStats.TotalCount
	}
	if convStats != nil {
		activityCount += convStats.TotalCount
	}
	if activityCount < 3 {
		return "", fmt.Errorf("数据不足，至少需要 3 条学习记录（当前 %d 条）", activityCount)
	}

	// 步骤 5: 生成报告ID 并预先落库
	reportID := uuid.New()
	if s.repos != nil {
		reportModel := &model.LearningReport{
			ID:         reportID,
			UserID:     req.UserID,
			ReportType: reportType,
		}
		if err := s.repos.LearningReport.Create(ctx, reportModel); err != nil {
			return "", fmt.Errorf("create report model: %w", err)
		}
	}

	// 步骤 5：序列化 Payload
	payload, err := json.Marshal(&reportPayloadData{
		ReportType: reportType,
		StartDate:  periodStart.Format("2006-01-02"),
		EndDate:    periodEnd.Format("2006-01-02"),
		UserID:     req.UserID,
		ReportID:   reportID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal report payload: %w", err)
	}

	// 步骤 6：构建 Task 并提交
	task := &worker.Task{
		Type:     "report",
		UserID:   req.UserID,
		Payload:  payload,
		DomainID: reportID,
	}

	taskID, err := s.workerManager.SubmitTask(ctx, task)
	if err != nil {
		logger.ErrorContext(ctx, "submit report task failed", "error", err)
		return "", fmt.Errorf("submit task: %w", err)
	}

	logger.InfoContext(ctx, "report task submitted",
		"task_id", taskID, "user_id", req.UserID, "type", reportType,
		"start", periodStart.Format("2006-01-02"), "end", periodEnd.Format("2006-01-02"))

	return taskID, nil
}

// reportStageDescriptions 报告生成阶段描述
var reportStageDescriptions = map[string]string{
	"queued":                     "报告任务已进入队列",
	"fetching_eval_data":         "正在查询评测数据...",
	"fetching_chat_data":         "正在查询对话数据...",
	"analyzing_eval":             "正在分析评测统计...",
	"analyzing_chat":             "正在分析对话统计...",
	"calculating_progress":       "正在计算学习进度...",
	"generating_recommendations": "正在生成学习建议...",
	"generating_report":          "正在生成完整报告...",
	"saving":                     "正在保存报告...",
	"completed":                  "报告生成完成",
}

func (s *reportServiceImpl) GetReportStatus(ctx context.Context, reportID string) (*ReportStatusResponse, error) {
	if reportID == "" {
		return nil, errors.New("report_id is empty")
	}

	meta, err := s.taskCache.GetTaskMeta(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("query task status: %w", err)
	}
	if meta == nil {
		return nil, fmt.Errorf("report task not found: %s", reportID)
	}

	stage := meta.CurrentStage
	msg := reportStageDescriptions[stage]
	if msg == "" {
		msg = "处理中..."
	}

	return &ReportStatusResponse{
		ReportID: reportID,
		Status:   meta.Status,
		Message:  msg,
	}, nil
}

func (s *reportServiceImpl) GetReport(ctx context.Context, reportID, userID string) (*ReportDetailResponse, error) {
	if reportID == "" {
		return nil, errors.New("report_id is empty")
	}

	// 步骤 1：从缓存查询任务状态
	meta, err := s.taskCache.GetTaskMeta(ctx, reportID)
	if err != nil {
		logger.ErrorContext(ctx, "get report task meta failed", "report_id", reportID, "error", err)
	}

	// 步骤 2：TaskMeta 存在 → 按状态处理
	if meta != nil {
		switch meta.Status {
		case "pending":
			stage := meta.CurrentStage
			if stage == "" {
				stage = "queued"
			}
			return &ReportDetailResponse{
				ReportID: reportID,
				Status:   "pending",
				Message:  reportStageDescriptions[stage],
			}, nil

		case "processing":
			stage := meta.CurrentStage
			msg := reportStageDescriptions[stage]
			if msg == "" {
				msg = "正在生成报告..."
			}
			return &ReportDetailResponse{
				ReportID: reportID,
				Status:   "processing",
				Message:  msg,
			}, nil

		case "success":
			// 从 reportCache 获取完整报告
			reportData, found, cacheErr := s.reportCache.GetReportResult(ctx, reportID)
			if cacheErr != nil {
				logger.ErrorContext(ctx, "get report from cache failed", "error", cacheErr)
			}
			if found && reportData != nil {
				// 将 map 序列化再反序列化为 ReportMVPResponse
				jsonBytes, _ := json.Marshal(reportData)
				var mvpResp ReportMVPResponse
				if parseErr := json.Unmarshal(jsonBytes, &mvpResp); parseErr == nil {
					return &ReportDetailResponse{
						ReportID:   reportID,
						Status:     "success",
						ReportType: mvpResp.ReportType,
						StartDate:  mvpResp.PeriodStartDate,
						EndDate:    mvpResp.PeriodEndDate,
						CreatedAt:  time.Now().Format(time.RFC3339),
						Report:     &mvpResp,
					}, nil
				}
			}

			// 缓存未命中 → 降级到数据库
			return s.getReportFromDB(ctx, reportID, userID)

		case "failed":
			return &ReportDetailResponse{
				ReportID:     reportID,
				Status:       "failed",
				ErrorMessage: meta.Error,
				Message:      "报告生成失败，请稍后重试",
			}, nil
		}
	}

	// 步骤 3：TaskMeta 不存在 → 查数据库历史报告
	return s.getReportFromDB(ctx, reportID, userID)
}

// getReportFromDB 从数据库获取已生成的历史报告
func (s *reportServiceImpl) getReportFromDB(ctx context.Context, reportID, userID string) (*ReportDetailResponse, error) {
	report, err := s.repos.LearningReport.GetByID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("report not found: %s", reportID)
	}

	// 权限检查
	if userID != "" && report.UserID != userID {
		return nil, fmt.Errorf("forbidden: report does not belong to user")
	}

	// 映射 LearningReport → ReportMVPResponse
	mvpResp := &ReportMVPResponse{
		ReportID:        report.ID,
		ReportType:      report.ReportType,
		PeriodStartDate: report.PeriodStartDate.Format("2006-01-02"),
		PeriodEndDate:   report.PeriodEndDate.Format("2006-01-02"),
		ActivityStats: &ActivityStats{
			ConversationCount: report.TotalConversations,
			EvaluationCount:   report.TotalEvaluations,
			TotalMinutes:      report.TotalStudyMinutes,
		},
		AbilityRadar: &AbilityRadar{
			AccuracyScore:  report.AverageAccuracyScore,
			FluencyScore:   report.AverageFluencyScore,
			IntegrityScore: report.AverageIntegrityScore,
		},
		ProgressStats: &ProgressStats{
			OverallScoreChange: int(report.ImprovementRate),
			CurrentScore:       report.AverageEvaluationScore,
		},
		DifficultWords: []DifficultWord{},
	}
	if report.Recommendations != nil {
		mvpResp.FullReport = &FullReport{
			FullText: *report.Recommendations,
		}
		if report.Strengths != nil {
			mvpResp.FullReport.ImprovementAreas = []string(report.Strengths)
		}
		if report.Weaknesses != nil {
			mvpResp.FullReport.Recommendations = []string(report.Weaknesses)
		}
	}

	return &ReportDetailResponse{
		ReportID:   report.ID,
		Status:     "success",
		ReportType: report.ReportType,
		StartDate:  report.PeriodStartDate.Format("2006-01-02"),
		EndDate:    report.PeriodEndDate.Format("2006-01-02"),
		CreatedAt:  report.CreatedAt.Format(time.RFC3339),
		Report:     mvpResp,
	}, nil
}

func (s *reportServiceImpl) GetReportList(ctx context.Context, userID string, page, pageSize int) ([]*ReportSummary, int64, error) {
	// TODO: Step2 实现
	// 1. 查询 learning_reports 表
	// 2. 按创建时间降序排列
	// 3. 分页返回报告摘要
	return nil, 0, nil
}

func (s *reportServiceImpl) DeleteReport(ctx context.Context, reportID, userID string) error {
	// TODO: Step2 实现
	// 1. 验证用户对该报告的所有权
	// 2. 软删除报告记录
	return nil
}

func (s *reportServiceImpl) GetDashboard(ctx context.Context, userID string) (*DashboardResponse, error) {
	// TODO: Step2 实现
	// 1. 查询用户近 N 天的对话统计
	// 2. 查询用户近 N 天的评测统计
	// 3. 计算总学习时长、平均分、分数趋势
	// 4. 返回统计面板数据
	return nil, nil
}

// ===================== ReportTaskProcessor =====================

// reportPayloadData 异步报告任务 payload 反序列化目标
type reportPayloadData struct {
	ReportType string `json:"report_type"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	UserID     string `json:"user_id"`
	ReportID   string `json:"report_id"`
}

// ReportTaskProcessor 实现 worker.TaskProcessor 接口
type ReportTaskProcessor struct {
	repos       *db.Repositories
	llmProvider domain.LLMProvider
	ttsProvider domain.TTSProvider
	ossProvider domain.OSSProvider
	taskCache   *cache.TaskCache
	logger      *slog.Logger
}

// NewReportTaskProcessor 创建 ReportTaskProcessor
func NewReportTaskProcessor(
	repos *db.Repositories,
	llm domain.LLMProvider,
	tts domain.TTSProvider,
	oss domain.OSSProvider,
	taskCache *cache.TaskCache,
	logger *slog.Logger,
) *ReportTaskProcessor {
	return &ReportTaskProcessor{
		repos:       repos,
		llmProvider: llm,
		ttsProvider: tts,
		ossProvider: oss,
		taskCache:   taskCache,
		logger:      logger,
	}
}

// Process 实现 worker.TaskProcessor 接口
func (p *ReportTaskProcessor) Process(ctx context.Context, task *worker.Task) (interface{}, string, error) {
	// 反序列化 payload
	var payload reportPayloadData
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return nil, "", fmt.Errorf("unmarshal report payload: %w", err)
	}
	reportID := payload.ReportID
	resultKey := fmt.Sprintf(cache.KeyReportResult, reportID)
	reportType := payload.ReportType

	// 解析日期范围
	periodStart, _ := time.Parse("2006-01-02", payload.StartDate)
	periodEnd, _ := time.Parse("2006-01-02", payload.EndDate)
	if periodEnd.IsZero() {
		periodEnd = periodStart.AddDate(0, 0, 7)
	}

	// ===== A1: 查询评测数据 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "fetching_eval_data")

	evalStats, err := p.repos.PronunciationEvaluation.GetStatsByUserAndDateRange(ctx, task.UserID, periodStart, periodEnd)
	if err != nil {
		p.logger.Warn("Report: query eval stats failed, use zeros", slog.String("error", err.Error()))
		evalStats = &db.EvaluationStats{ProblemWords: make(map[string]int)}
	}

	// ===== A2: 查询对话数据 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "fetching_chat_data")

	convStats, err := p.repos.VoiceConversation.GetStatsByUserAndDateRange(ctx, task.UserID, periodStart, periodEnd)
	if err != nil {
		p.logger.Warn("Report: query conv stats failed, use zeros", slog.String("error", err.Error()))
		convStats = &db.ConversationStats{}
	}

	// ===== A3: 计算评测统计 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "analyzing_eval")

	totalMinutes := convStats.TotalDurationSeconds / 60

	// 高频问题单词 Top 5
	type wordFreq struct {
		Word      string
		Frequency int
	}
	var sortedWords []wordFreq
	for w, f := range evalStats.ProblemWords {
		sortedWords = append(sortedWords, wordFreq{Word: w, Frequency: f})
	}
	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].Frequency > sortedWords[j].Frequency
	})
	if len(sortedWords) > 5 {
		sortedWords = sortedWords[:5]
	}

	// ===== A4: 计算进步趋势 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "calculating_progress")

	var prevStart, prevEnd time.Time
	switch reportType {
	case "monthly":
		prevStart = periodStart.AddDate(0, -1, 0)
		prevEnd = periodStart
	default:
		prevStart = periodStart.AddDate(0, 0, -7)
		prevEnd = periodStart
	}
	prevAvgScore, _ := p.repos.PronunciationEvaluation.GetAverageScoreByUserIDAndDateRange(ctx, task.UserID, prevStart, prevEnd)
	currentScore := evalStats.AvgOverallScore
	scoreChange := currentScore - prevAvgScore

	// ===== A5: LLM 生成报告 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "generating_report")

	llmStats := &llmPrompts.ReportStats{
		StartDate:         payload.StartDate,
		EndDate:           payload.EndDate,
		ConversationCount: convStats.TotalCount,
		EvaluationCount:   evalStats.TotalCount,
		TotalMinutes:      totalMinutes,
		ActiveDays:        convStats.ActiveDays,
		AvgAccuracy:       evalStats.AvgAccuracyScore,
		AvgFluency:        evalStats.AvgFluencyScore,
		AvgIntegrity:      evalStats.AvgIntegrityScore,
		AvgOverall:        currentScore,
		SCount:            evalStats.SLevelCount,
		ACount:            evalStats.ALevelCount,
		BCount:            evalStats.BLevelCount,
		CCount:            evalStats.CLevelCount,
		PreviousScore:     prevAvgScore,
		CurrentScore:      currentScore,
		ScoreChange:       scoreChange,
		ProblemWords:      evalStats.ProblemWords,
	}

	systemPrompt, userMsg := llmPrompts.GenerateReportPrompt(llmStats)
	var reportData llmPrompts.LLMReportData

	llmResponse, llmErr := p.llmProvider.Chat(ctx, systemPrompt, userMsg)
	if llmErr != nil {
		p.logger.Warn("Report LLM failed, use defaults", slog.String("error", llmErr.Error()))
		reportData = llmPrompts.GetDefaultReportData()
	} else {
		if parseErr := json.Unmarshal([]byte(llmResponse), &reportData); parseErr != nil {
			p.logger.Warn("Report LLM JSON parse failed, use defaults", slog.String("error", parseErr.Error()))
			reportData = llmPrompts.GetDefaultReportData()
		}
	}

	// ===== A6: TTS 问题单词 + OSS 上传 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "generating_recommendations")

	var difficultWords []DifficultWord
	for _, wf := range sortedWords {
		dw := DifficultWord{
			Word:      wf.Word,
			Frequency: wf.Frequency,
		}
		if p.ttsProvider != nil && p.ossProvider != nil {
			ttsData, ttsErr := p.ttsProvider.Synthesize(ctx, wf.Word, nil)
			if ttsErr == nil && len(ttsData) > 0 {
				ossKey := fmt.Sprintf("report/demo/%s/%s_%s.mp3",
					task.UserID, wf.Word, time.Now().Format("20060102150405"))
				audioURL, ossErr := p.ossProvider.UploadBytes(ctx, ossKey, ttsData, "audio/mpeg")
				if ossErr == nil {
					dw.DemoAudioURL = audioURL
				}
			}
		}
		difficultWords = append(difficultWords, dw)
	}

	// ===== A7: 保存报告到数据库 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "saving")

	if p.repos != nil {
		report, err := p.repos.LearningReport.GetByID(ctx, reportID)
		if err == nil && report != nil {
			report.PeriodStartDate = periodStart
			report.PeriodEndDate = periodEnd
			report.TotalConversations = convStats.TotalCount
			report.TotalEvaluations = evalStats.TotalCount
			report.TotalStudyMinutes = totalMinutes
			report.AverageEvaluationScore = currentScore
			report.AverageAccuracyScore = evalStats.AvgAccuracyScore
			report.AverageFluencyScore = evalStats.AvgFluencyScore
			report.AverageIntegrityScore = evalStats.AvgIntegrityScore
			report.SLevelCount = evalStats.SLevelCount
			report.ALevelCount = evalStats.ALevelCount
			report.BLevelCount = evalStats.BLevelCount
			report.CLevelCount = evalStats.CLevelCount
			report.ImprovementRate = scoreChange
			report.Recommendations = toStringPtr(reportData.FullText)
			report.Strengths = model.StringArray(reportData.Highlights)
			report.Weaknesses = model.StringArray(reportData.ImprovementAreas)
			if updateErr := p.repos.LearningReport.Update(ctx, report); updateErr != nil {
				p.logger.Error("update report db failed", slog.String("error", updateErr.Error()))
			}
		} else {
			// Fallback: create if missing
			reportModel := &model.LearningReport{
				ID:                     reportID,
				UserID:                 task.UserID,
				ReportType:             reportType,
				PeriodStartDate:        periodStart,
				PeriodEndDate:          periodEnd,
				TotalConversations:     convStats.TotalCount,
				TotalEvaluations:       evalStats.TotalCount,
				TotalStudyMinutes:      totalMinutes,
				AverageEvaluationScore: currentScore,
				AverageAccuracyScore:   evalStats.AvgAccuracyScore,
				AverageFluencyScore:    evalStats.AvgFluencyScore,
				AverageIntegrityScore:  evalStats.AvgIntegrityScore,
				SLevelCount:            evalStats.SLevelCount,
				ALevelCount:            evalStats.ALevelCount,
				BLevelCount:            evalStats.BLevelCount,
				CLevelCount:            evalStats.CLevelCount,
				ImprovementRate:        scoreChange,
				Recommendations:        toStringPtr(reportData.FullText),
				Strengths:              model.StringArray(reportData.Highlights),
				Weaknesses:             model.StringArray(reportData.ImprovementAreas),
			}

			for dbRetry := 0; dbRetry < 3; dbRetry++ {
				if saveErr := p.repos.LearningReport.Create(ctx, reportModel); saveErr != nil {
					p.logger.Error("save report db retry", slog.Int("attempt", dbRetry+1), slog.String("error", saveErr.Error()))
					time.Sleep(time.Duration(dbRetry+1) * 500 * time.Millisecond)
					continue
				}
				break
			}
		}

	}

	// ===== A8: 构建 ReportMVPResponse 并返回 =====
	_ = p.taskCache.UpdateTaskStage(ctx, task.ID, "completed")

	result := &ReportMVPResponse{
		ReportID:        reportID,
		ReportType:      reportType,
		PeriodStartDate: periodStart.Format("2006-01-02"),
		PeriodEndDate:   periodEnd.Format("2006-01-02"),

		ActivityStats: &ActivityStats{
			ConversationCount: convStats.TotalCount,
			EvaluationCount:   evalStats.TotalCount,
			TotalMinutes:      totalMinutes,
			ActiveDays:        convStats.ActiveDays,
		},

		AbilityRadar: &AbilityRadar{
			AccuracyScore:  evalStats.AvgAccuracyScore,
			FluencyScore:   evalStats.AvgFluencyScore,
			IntegrityScore: evalStats.AvgIntegrityScore,
			Summary:        reportData.AbilityAnalysis,
		},

		ProgressStats: &ProgressStats{
			OverallScoreChange: int(scoreChange),
			PreviousScore:      prevAvgScore,
			CurrentScore:       currentScore,
			Highlights:         reportData.Highlights,
			LevelImprovement:   reportData.ProgressHighlight,
		},

		KidFriendlyCard: &KidFriendlyCard{
			EncouragementText: reportData.EncouragementText,
			Highlights:        reportData.Highlights,
			SmallGoal:         reportData.SmallGoal,
		},

		DifficultWords: difficultWords,

		FullReport: &FullReport{
			PeriodSummary:     reportData.PeriodSummary,
			AbilityAnalysis:   reportData.AbilityAnalysis,
			ProgressHighlight: reportData.ProgressHighlight,
			ImprovementAreas:  reportData.ImprovementAreas,
			Recommendations:   reportData.Recommendations,
			FullText:          reportData.FullText,
		},
	}
	if result.DifficultWords == nil {
		result.DifficultWords = []DifficultWord{}
	}

	return result, resultKey, nil
}

// ===================== ReportResultPersister =====================

// ReportResultPersister 实现 worker.ResultPersister 接口
type ReportResultPersister struct {
	logger *slog.Logger
}

// NewReportResultPersister 创建 ReportResultPersister
func NewReportResultPersister(logger *slog.Logger) *ReportResultPersister {
	return &ReportResultPersister{logger: logger}
}

// SaveResult 将结果持久化
// ReportTaskProcessor.Process 中已经完成了 DB 写入，这里只做日志记录
func (p *ReportResultPersister) SaveResult(ctx context.Context, task *worker.Task, result interface{}) error {
	p.logger.Info("report result persisted (already saved in processor)",
		slog.String("task_id", task.ID),
		slog.String("type", task.Type),
	)
	return nil
}
