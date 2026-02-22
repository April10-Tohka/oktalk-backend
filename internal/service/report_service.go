// Package service 提供智能学习报告业务逻辑
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	llmPrompts "pronunciation-correction-system/internal/infrastructure/llm"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/logger"
	"pronunciation-correction-system/internal/pkg/uuid"
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

// ReportDetailResponse 报告完整详情
type ReportDetailResponse struct {
	ReportID   string `json:"report_id"`
	ReportType string `json:"report_type"`
	UserID     string `json:"user_id"`
	CreatedAt  string `json:"created_at"`
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

// ===== 空实现 =====

// reportServiceImpl Report Service 实现
type reportServiceImpl struct {
	repos       *db.Repositories
	llmProvider domain.LLMProvider
	ttsProvider domain.TTSProvider
	ossProvider domain.OSSProvider
	logger      *slog.Logger
}

// NewReportService 创建 ReportService
func NewReportService(
	repos *db.Repositories,
	llmProvider domain.LLMProvider,
	ttsProvider domain.TTSProvider,
	ossProvider domain.OSSProvider,
	logger *slog.Logger,
) ReportService {
	return &reportServiceImpl{
		repos:       repos,
		llmProvider: llmProvider,
		ttsProvider: ttsProvider,
		ossProvider: ossProvider,
		logger:      logger,
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
	// TODO: Step3 实现异步任务
	// 1. 生成 report_id
	// 2. 创建异步任务（数据统计 → LLM 生成）
	// 3. 将任务提交到队列
	// 4. 返回 report_id
	return "", nil
}

func (s *reportServiceImpl) GetReportStatus(ctx context.Context, reportID string) (*ReportStatusResponse, error) {
	// TODO: Step3 实现
	// 1. 从缓存/数据库查询报告生成状态
	// 2. 返回进度信息
	return nil, nil
}

func (s *reportServiceImpl) GetReport(ctx context.Context, reportID, userID string) (*ReportDetailResponse, error) {
	// TODO: Step2 实现
	// 1. 验证用户对该报告的所有权
	// 2. 查询 learning_reports 表
	// 3. 解析 JSON 字段（summary, analysis, insights）
	// 4. 返回完整报告详情
	return nil, nil
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
