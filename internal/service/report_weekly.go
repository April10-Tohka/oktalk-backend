package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/model"
	"pronunciation-correction-system/internal/pkg/uuid"
)

// ErrWeeklyReportInsufficient 本周有效记录不足 3 条
var ErrWeeklyReportInsufficient = errors.New("本周学习记录不足，至少需要 3 条")

// ErrReportForbidden 无权访问他人报告
var ErrReportForbidden = errors.New("forbidden")

// GenerateWeeklyReport 同步生成周报并落库
func (s *reportServiceImpl) GenerateWeeklyReport(ctx context.Context, userID string) (*WeeklyReportData, string, error) {
	if userID == "" {
		return nil, "", errors.New("user id is empty")
	}
	now := time.Now()
	loc := now.Location()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, loc)
	weekEnd := weekStart.AddDate(0, 0, 7).Add(-time.Nanosecond)

	// 统计本周有效发音评测次数
	evalCount, err := s.repos.LearningReport.CountEvaluations(ctx, userID, weekStart, weekEnd)
	if err != nil {
		return nil, "", err
	}
	// 统计本周场景对话次数
	convCount, err := s.repos.LearningReport.CountConversations(ctx, userID, weekStart, weekEnd)
	if err != nil {
		return nil, "", err
	}
	// 如果本周有效发音评测次数和场景对话次数之和小于 3，则返回错误
	if evalCount+convCount < 3 {
		return nil, "", ErrWeeklyReportInsufficient
	}

	// 查找本周最新周报
	existing, err := s.repos.LearningReport.FindLatestByUserAndPeriod(ctx, userID, "weekly", weekStart, weekEnd)
	if err != nil {
		return nil, "", err
	}
	// 如果本周最新周报存在且内容不为空且创建时间距现在小于 2 小时，则返回本周最新周报
	if existing != nil && existing.Content != "" && time.Since(existing.CreatedAt) < 2*time.Hour {
		var data WeeklyReportData
		if err := json.Unmarshal([]byte(existing.Content), &data); err == nil {
			return &data, existing.ID, nil // 返回本周最新周报
		}
	}
	// 统计本周坚持学习天数
	persistDays, err := s.repos.LearningReport.CountPersistenceDays(ctx, userID, weekStart, weekEnd)
	if err != nil {
		return nil, "", err
	}
	// 统计本周发音评测次数、场景对话次数和坚持学习天数
	activity := ReportActivity{
		EvaluationCount:   evalCount,
		ConversationCount: convCount,
		PersistenceDays:   persistDays,
	}

	// 计算本周发音评测的平均准确度、流利度、完整度和标准度
	avgA, avgF, avgI, avgS, err := s.repos.LearningReport.GetAvgScores(ctx, userID, weekStart, weekEnd)
	if err != nil {
		return nil, "", err
	}
	// 构建雷达图
	radar := ReportRadar{
		AccuracyScore:  scoreToPercent(avgA),
		FluencyScore:   scoreToPercent(avgF),
		IntegrityScore: scoreToPercent(avgI),
		StandardScore:  scoreToPercent(avgS),
	}

	// 获取本周高频难词
	rawList, err := s.repos.LearningReport.GetProblemWordsList(ctx, userID, weekStart, weekEnd)
	if err != nil {
		return nil, "", err
	}
	hardWords := buildHardWordsTop4(rawList, s.pronunciationLoader)

	// 获取本周场景对话的一次通过率与完成场景数
	passRate, completed, err := s.repos.LearningReport.GetSceneStats(ctx, userID, weekStart, weekEnd)
	if err != nil {
		return nil, "", err
	}
	// 构建场景对话表现
	scene := ReportScene{PassRate: passRate, CompletedScenes: completed}

	// 生成本周鼓励语
	encourage := s.llmWeeklyEncourage(ctx, activity, radar.AccuracyScore)
	// 生成本周完整报告
	full := s.llmWeeklyFull(ctx, weekStart, weekEnd, activity, radar, scene, hardWords)

	// 构建周报数据
	data := WeeklyReportData{
		WeekStart:  weekStart.Format("2006-01-02"),
		WeekEnd:    weekEnd.Format("2006-01-02"),
		Activity:   activity,
		Radar:      radar,
		HardWords:  hardWords,
		Scene:      scene,
		Encourage:  encourage,
		FullReport: full,
	}

	// 将周报数据序列化为 JSON
	contentJSON, err := json.Marshal(data)
	if err != nil {
		return nil, "", err
	}

	// 生成新的报告 ID
	reportID := uuid.New()
	// 创建新的周报记录
	rep := &model.LearningReport{
		ID:              reportID,
		UserID:          userID,
		ReportType:      "weekly",
		PeriodStartDate: weekStart,
		PeriodEndDate:   weekEnd,
		Content:         string(contentJSON),
		IsLatest:        true,
	}
	// 将周报记录保存到数据库
	if err := s.repos.LearningReport.Create(ctx, rep); err != nil {
		return nil, "", err
	}
	if err := s.repos.LearningReport.UpdateIsLatest(ctx, userID, weekStart, reportID); err != nil {
		s.logger.Warn("weekly report UpdateIsLatest failed", slog.String("error", err.Error()))
	}
	// 返回周报数据和报告 ID
	return &data, reportID, nil
}

func scoreToPercent(v float64) int {
	x := int(math.Round(v * 20))
	if x < 0 {
		return 0
	}
	if x > 100 {
		return 100
	}
	return x
}

func buildHardWordsTop4(rawList []string, loader *config.PronunciationLoader) []ReportHardWord {
	freq := make(map[string]int)
	for _, raw := range rawList {
		var words []string
		if err := json.Unmarshal([]byte(raw), &words); err != nil {
			continue
		}
		for _, w := range words {
			w = strings.TrimSpace(w)
			if w == "" {
				continue
			}
			freq[strings.ToLower(w)]++
		}
	}
	type pair struct {
		w string
		c int
	}
	var pairs []pair
	for w, c := range freq {
		pairs = append(pairs, pair{w: w, c: c})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].c != pairs[j].c {
			return pairs[i].c > pairs[j].c
		}
		return pairs[i].w < pairs[j].w
	})
	var out []ReportHardWord
	for i := 0; i < len(pairs) && i < 4; i++ {
		out = append(out, ReportHardWord{
			Word:     pairs[i].w,
			Count:    pairs[i].c,
			AudioURL: findStandardAudioURL(loader, pairs[i].w),
		})
	}
	return out
}

func findStandardAudioURL(loader *config.PronunciationLoader, word string) string {
	if loader == nil {
		return ""
	}
	for _, u := range loader.GetAll() {
		for i := range u.Items {
			if strings.EqualFold(u.Items[i].Content, word) {
				return u.Items[i].StandardAudioURL
			}
		}
	}
	return ""
}

func (s *reportServiceImpl) llmWeeklyEncourage(ctx context.Context, activity ReportActivity, accuracyPct int) ReportEncourage {
	sys := `你是一位儿童英语学习助手，说话简短活泼，多用 emoji，语气温暖鼓励。
只能返回 JSON，不允许返回任何其他内容。`
	usr := fmt.Sprintf(`小朋友这周的学习情况：
- 发音评测次数：%d 次
- 场景对话次数：%d 次
- 坚持学习天数：%d 天
- 发音准确度：%d 分（满分100）

请生成一段简短鼓励，只返回：
{"encourage_text": "两句话以内，儿童友好，多 emoji"}`,
		activity.EvaluationCount, activity.ConversationCount, activity.PersistenceDays, accuracyPct)
	raw, err := s.llmProvider.Chat(ctx, sys, usr)
	if err != nil {
		s.logger.Warn("weekly encourage LLM failed", slog.String("error", err.Error()))
		return ReportEncourage{EncourageText: "你这周练习很认真！🌟 继续加油，你的英语越来越棒了！💪"}
	}
	js := extractJSONObject(raw)
	var wrap struct {
		EncourageText string `json:"encourage_text"`
	}
	if err := json.Unmarshal([]byte(js), &wrap); err != nil || wrap.EncourageText == "" {
		return ReportEncourage{EncourageText: "你这周练习很认真！🌟 继续加油，你的英语越来越棒了！💪"}
	}
	return ReportEncourage{EncourageText: wrap.EncourageText}
}

func (s *reportServiceImpl) llmWeeklyFull(ctx context.Context, weekStart, weekEnd time.Time, activity ReportActivity, radar ReportRadar, scene ReportScene, hardWords []ReportHardWord) ReportFullContent {
	sys := `你是一位专业的儿童英语学习报告生成助手，语气友好温暖，面向家长和孩子。
只能返回 JSON，不允许返回任何其他内容。`
	hardWordsText := ""
	for _, hw := range hardWords {
		hardWordsText += fmt.Sprintf("- %s（出现 %d 次）\n", hw.Word, hw.Count)
	}
	if hardWordsText == "" {
		hardWordsText = "本周无高频难词，发音很棒！"
	}
	usr := fmt.Sprintf(`以下是小朋友本周（%s 至 %s）的英语学习数据：

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
		weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"),
		activity.EvaluationCount, activity.ConversationCount, activity.PersistenceDays,
		radar.AccuracyScore, radar.FluencyScore, radar.IntegrityScore, radar.StandardScore,
		scene.PassRate, scene.CompletedScenes,
		hardWordsText)
	raw, err := s.llmProvider.Chat(ctx, sys, usr)
	if err != nil {
		s.logger.Warn("weekly full LLM failed", slog.String("error", err.Error()))
		return fallbackWeeklyFull()
	}
	js := extractJSONObject(raw)
	var wrap ReportFullContent
	if err := json.Unmarshal([]byte(js), &wrap); err != nil || wrap.Summary == "" {
		return fallbackWeeklyFull()
	}
	if wrap.Strengths == nil {
		wrap.Strengths = []string{}
	}
	if wrap.Improvements == nil {
		wrap.Improvements = []string{}
	}
	return wrap
}

func fallbackWeeklyFull() ReportFullContent {
	return ReportFullContent{
		Summary:      "这周学习很认真！发音练习和对话都有参与，继续保持！",
		Strengths:    []string{"坚持练习"},
		Improvements: []string{"多练习发音"},
	}
}

// GetWeeklyReportList 最新周报列表（is_latest=true）
func (s *reportServiceImpl) GetWeeklyReportList(ctx context.Context, userID string) ([]WeeklyReportListItem, error) {
	list, err := s.repos.LearningReport.ListLatestByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]WeeklyReportListItem, 0, len(list))
	for _, r := range list {
		if r.ReportType != "weekly" || r.Content == "" {
			continue
		}
		var data WeeklyReportData
		if err := json.Unmarshal([]byte(r.Content), &data); err != nil {
			continue
		}
		out = append(out, WeeklyReportListItem{
			ReportID:        r.ID,
			WeekStart:       data.WeekStart,
			WeekEnd:         data.WeekEnd,
			CreatedAt:       r.CreatedAt.Format(time.RFC3339),
			AccuracyScore:   data.Radar.AccuracyScore,
			PersistenceDays: data.Activity.PersistenceDays,
			EvaluationCount: data.Activity.EvaluationCount,
		})
	}
	return out, nil
}

// GetWeeklyReportByID 详情（校验归属）
func (s *reportServiceImpl) GetWeeklyReportByID(ctx context.Context, reportID, userID string) (*WeeklyReportData, error) {
	rep, err := s.repos.LearningReport.FindReportByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	if rep == nil {
		return nil, nil
	}
	if rep.UserID != userID {
		return nil, ErrReportForbidden
	}
	if rep.Content == "" {
		return nil, errors.New("empty content")
	}
	var data WeeklyReportData
	if err := json.Unmarshal([]byte(rep.Content), &data); err != nil {
		return nil, err
	}
	return &data, nil
}
