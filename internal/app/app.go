// Package app 提供应用程序依赖注入和生命周期管理
// 负责初始化所有组件、建立依赖关系、提供优雅关闭
package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/cache"
	cacheRedis "pronunciation-correction-system/internal/cache/redis"
	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/db"
	"pronunciation-correction-system/internal/domain"
	"pronunciation-correction-system/internal/handler"
	infraASR "pronunciation-correction-system/internal/infrastructure/asr/aliyun"
	infraXF "pronunciation-correction-system/internal/infrastructure/evalution/xf"
	infraLLM "pronunciation-correction-system/internal/infrastructure/llm/qwen"
	infraOSS "pronunciation-correction-system/internal/infrastructure/oss/aliyun"
	infraTTS "pronunciation-correction-system/internal/infrastructure/tts/aliyun"
	"pronunciation-correction-system/internal/service"
)

// App 应用程序实例，持有所有依赖
type App struct {
	Config *config.Config

	// 基础设施
	DB           *gorm.DB
	RedisClient  *cacheRedis.Client
	CacheManager *cache.Manager

	// 数据库仓库
	Repos *db.Repositories

	// 外部服务适配器（通过 domain 接口引用）
	ASRProvider        domain.ASRProvider
	EvaluationProvider domain.EvaluationProvider
	LLMProvider        domain.LLMProvider
	TTSProvider        domain.TTSProvider
	OSSProvider        domain.OSSProvider

	// 服务层
	AuthService     service.AuthService
	UserService     service.UserService
	ChatService     service.ChatService
	EvaluateService service.EvaluateService
	ReportService   service.ReportService

	// Handler 层
	Handlers *handler.Handlers
}

// New 创建并初始化应用程序实例
func New(cfg *config.Config) (*App, error) {
	app := &App{
		Config: cfg,
	}

	// 按顺序初始化各组件
	if err := app.initDatabase(); err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	if err := app.initRedis(); err != nil {
		// Redis 不可用时降级运行
		log.Printf("[App] Warning: Redis init failed: %v, running without cache", err)
	}

	app.initInfrastructure()
	app.initRepositories()
	app.initServices()
	app.initHandlers()

	log.Println("[App] Application initialized successfully")
	return app, nil
}

// initDatabase 初始化数据库连接
func (a *App) initDatabase() error {
	database, err := db.Init(a.Config)
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	a.DB = database

	// 执行数据库迁移
	if err := db.Migrate(database); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("[App] Database initialized")
	return nil
}

// initRedis 初始化 Redis 和缓存管理器
func (a *App) initRedis() error {
	mgr, err := cache.NewManager(&cache.ManagerConfig{
		Redis:        &a.Config.Redis,
		DefaultQuota: 50,
	})
	if err != nil {
		return err
	}

	a.CacheManager = mgr
	a.RedisClient = mgr.GetClient()

	log.Println("[App] Redis and CacheManager initialized")
	return nil
}

// initInfrastructure 初始化基础设施层适配器（具体实现 → domain 接口）
// 根据配置中的 ActiveProvider 选择具体实现
func (a *App) initInfrastructure() {
	// ASR: 根据 active_provider 选择语音识别实现
	switch a.Config.ASR.ActiveProvider {
	case "aliyun":
		a.ASRProvider = infraASR.NewAliyunASRAdapter(a.Config.ASR.Aliyun)
	default:
		a.ASRProvider = infraASR.NewAliyunASRAdapter(a.Config.ASR.Aliyun)
	}

	// SpeechAssessor: 语音评测（讯飞）
	a.EvaluationProvider = infraXF.NewXFEvaluationAdapter(a.Config.Evaluation.XunFei)

	// LLM: 根据 active_provider 选择大语言模型实现
	switch a.Config.LLM.ActiveProvider {
	case "qwen":
		a.LLMProvider = infraLLM.NewQwenAdapter(a.Config.LLM.Qwen)
	default:
		a.LLMProvider = infraLLM.NewQwenAdapter(a.Config.LLM.Qwen)
	}

	// TTS: 根据 active_provider 选择语音合成实现
	switch a.Config.TTS.ActiveProvider {
	case "aliyun":
		a.TTSProvider = infraTTS.NewAliyunTTSAdapter(a.Config.TTS.Aliyun)
	default:
		a.TTSProvider = infraTTS.NewAliyunTTSAdapter(a.Config.TTS.Aliyun)
	}

	// OSS: 根据 active_provider 选择对象存储实现
	switch a.Config.OSS.ActiveProvider {
	case "aliyun":
		ossAdapter, err := infraOSS.NewAliyunOSSAdapter(a.Config.OSS.Aliyun)
		if err != nil {
			log.Fatalf("[App] Failed to create Aliyun OSS adapter: %v", err)
		}
		a.OSSProvider = ossAdapter
	default:
		ossAdapter, err := infraOSS.NewAliyunOSSAdapter(a.Config.OSS.Aliyun)
		if err != nil {
			log.Fatalf("[App] Failed to create Aliyun OSS adapter: %v", err)
		}
		a.OSSProvider = ossAdapter
	}

	log.Println("[App] Infrastructure adapters initialized")
}

// initRepositories 初始化数据库仓库
func (a *App) initRepositories() {
	a.Repos = db.NewRepositories(a.DB)
	log.Println("[App] Repositories initialized")
}

// initServices 初始化业务服务
// TODO: Step2 注入真实依赖（Repos, Provider 等）
func (a *App) initServices() {
	a.AuthService = service.NewAuthService(nil)
	a.UserService = service.NewUserService(nil)
	a.ChatService = service.NewChatService(nil)
	a.EvaluateService = service.NewEvaluateService(nil)
	a.ReportService = service.NewReportService(nil)

	log.Println("[App] Services initialized")
}

// initHandlers 初始化 HTTP Handler
func (a *App) initHandlers() {
	a.Handlers = &handler.Handlers{
		Auth:     handler.NewAuthHandler(a.AuthService),
		User:     handler.NewUserHandler(a.UserService),
		Chat:     handler.NewChatHandler(a.ChatService),
		Evaluate: handler.NewEvaluateHandler(a.EvaluateService),
		Report:   handler.NewReportHandler(a.ReportService),
		System:   handler.NewSystemHandler(),
	}
	log.Println("[App] Handlers initialized")
}

// Close 优雅关闭所有资源
func (a *App) Close() {
	log.Println("[App] Shutting down...")

	// 关闭外部服务适配器
	if a.ASRProvider != nil {
		_ = a.ASRProvider.Close()
	}
	if a.EvaluationProvider != nil {
		_ = a.EvaluationProvider.Close()
	}
	if a.LLMProvider != nil {
		_ = a.LLMProvider.Close()
	}
	if a.TTSProvider != nil {
		_ = a.TTSProvider.Close()
	}
	if a.OSSProvider != nil {
		_ = a.OSSProvider.Close()
	}

	// 关闭缓存
	if a.CacheManager != nil {
		_ = a.CacheManager.Close()
	}

	// 关闭数据库
	if a.DB != nil {
		_ = db.Close(a.DB)
	}

	log.Println("[App] All resources closed")
}

// HealthCheck 检查各组件健康状态
func (a *App) HealthCheck(ctx context.Context) map[string]string {
	status := map[string]string{
		"app": "ok",
	}

	// 检查数据库
	if err := db.Ping(a.DB); err != nil {
		status["database"] = fmt.Sprintf("error: %v", err)
	} else {
		status["database"] = "ok"
	}

	// 检查 Redis
	if a.RedisClient != nil {
		timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := a.RedisClient.Ping(timeoutCtx); err != nil {
			status["redis"] = fmt.Sprintf("error: %v", err)
		} else {
			status["redis"] = "ok"
		}
	} else {
		status["redis"] = "not configured"
	}

	return status
}
