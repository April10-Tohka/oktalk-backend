// Package config 提供应用程序配置管理
// 定义所有配置结构体和默认值
package config

// Config 应用程序主配置结构体
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	LLM        LLMConfig        `mapstructure:"llm"`
	ASR        ASRConfig        `mapstructure:"asr"`
	Evaluation EvaluationConfig `mapstructure:"evaluation"`
	TTS        TTSConfig        `mapstructure:"tts"`
	OSS        OSSConfig        `mapstructure:"oss"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Log        LogConfig        `mapstructure:"log"`
}

// ===================== 服务器 & 基础设施 =====================

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Port        int    `mapstructure:"port"`
	Mode        string `mapstructure:"mode"` // debug, release, test
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

// RedisConfig Redis 缓存配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// LogConfig 日志配置
type LogConfig struct {
	// 环境：development, production
	Environment string `yaml:"environment" mapstructure:"environment"`
	// 日志级别：debug, info, warn, error
	Level string `yaml:"level" mapstructure:"level"`
	// 控制台输出配置
	Console ConsoleConfig `yaml:"console" mapstructure:"console"`
	// 文件输出配置
	File FileConfig `yaml:"file" mapstructure:"file"`
}

// ConsoleConfig 控制台输出配置
type ConsoleConfig struct {
	// 是否启用控制台输出
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	// 是否彩色输出
	Colorful bool `yaml:"colorful" mapstructure:"colorful"`
}

// FileConfig 文件输出配置
type FileConfig struct {
	// 是否启用文件输出
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	// 日志文件路径
	Filename string `yaml:"filename" mapstructure:"filename"`
	// 单个文件最大大小（MB）
	MaxSize int `yaml:"max_size" mapstructure:"max_size"`
	// 保留的旧日志文件数量
	MaxBackups int `yaml:"max_backups" mapstructure:"max_backups"`
	// 保留的旧日志文件天数
	MaxAge int `yaml:"max_age" mapstructure:"max_age"`
	// 是否压缩旧日志文件
	Compress bool `yaml:"compress" mapstructure:"compress"`
}

// DefaultConfig 默认日志配置
func DefaultConfig() *LogConfig {
	return &LogConfig{
		Environment: "development",
		Level:       "debug",
		Console: ConsoleConfig{
			Enabled:  true,
			Colorful: true,
		},
		File: FileConfig{
			Enabled:    true,
			Filename:   "logs/app.log",
			MaxSize:    100,
			MaxBackups: 30,
			MaxAge:     7,
			Compress:   true,
		},
	}
}

// ===================== LLM 大语言模型 =====================

// LLMConfig LLM 模块配置（支持多 Provider 切换）
type LLMConfig struct {
	ActiveProvider string         `mapstructure:"active_provider"` // qwen, gemini, deepseek, doubao
	Qwen           QwenConfig     `mapstructure:"qwen"`
	Gemini         GeminiConfig   `mapstructure:"gemini"`
	DeepSeek       DeepSeekConfig `mapstructure:"deepseek"`
}

// QwenConfig 通义千问配置
type QwenConfig struct {
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Endpoint string `mapstructure:"endpoint"`
	BaseURL  string `mapstructure:"base_url"`
}

// GeminiConfig Google Gemini 配置
type GeminiConfig struct {
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Endpoint string `mapstructure:"endpoint"`
}

// DeepSeekConfig DeepSeek 配置
type DeepSeekConfig struct {
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Endpoint string `mapstructure:"endpoint"`
}

// ===================== ASR 语音识别 =====================

// ASRConfig ASR 语音识别模块配置（支持多 Provider 切换）
type ASRConfig struct {
	ActiveProvider string          `mapstructure:"active_provider"` // aliyun, xunfei
	Aliyun         AliyunASRConfig `mapstructure:"aliyun"`
}

// AliyunASRConfig 阿里云 FUN-ASR 语音识别配置
type AliyunASRConfig struct {
	APIKey   string `mapstructure:"api_key"`  // DashScope API Key
	Model    string `mapstructure:"model"`    // 模型名称，如 paraformer-realtime-v2
	Endpoint string `mapstructure:"endpoint"` // WebSocket 地址（留空使用默认）
}

// EvaluationConfig 语音评测配置（支持多 Provider 切换）
type EvaluationConfig struct {
	ActiveProvider string       `mapstructure:"active_provider"` //  xunfei
	XunFei         XunFeiConfig `mapstructure:"xunfei"`
}

// XunFeiConfig 科大讯飞 API 配置（语音评测）
type XunFeiConfig struct {
	AppID     string `mapstructure:"app_id"`
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
}

// ===================== TTS 语音合成 =====================

// TTSConfig TTS 语音合成模块配置（支持多 Provider 切换）
type TTSConfig struct {
	ActiveProvider string          `mapstructure:"active_provider"` // aliyun
	Aliyun         AliyunTTSConfig `mapstructure:"aliyun"`
}

// AliyunTTSConfig 阿里云 CosyVoice TTS 配置（WebSocket 方式）
type AliyunTTSConfig struct {
	APIKey         string            `mapstructure:"api_key"`         // DashScope API Key
	Model          string            `mapstructure:"model"`           // 模型名称，如 cosyvoice-v3-flash
	Region         string            `mapstructure:"region"`          // 地域：beijing
	Endpoint       string            `mapstructure:"endpoint"`        // WebSocket 地址（留空使用默认）
	DefaultOptions TTSDefaultOptions `mapstructure:"default_options"` // 默认合成参数
}

// TTSDefaultOptions TTS 默认合成参数
type TTSDefaultOptions struct {
	Voice      string  `mapstructure:"voice"`       // 音色：longanyang, longxiaochun 等
	Format     string  `mapstructure:"format"`      // 格式：mp3, wav, pcm
	SampleRate int     `mapstructure:"sample_rate"` // 采样率：8000, 16000, 22050, 24000, 48000
	Volume     int     `mapstructure:"volume"`      // 音量：0-100
	Rate       float64 `mapstructure:"rate"`        // 语速：0.5-2.0
	Pitch      float64 `mapstructure:"pitch"`       // 音调：0.5-2.0
}

// ===================== OSS 对象存储 =====================

// OSSConfig OSS 对象存储模块配置（支持多 Provider 切换）
type OSSConfig struct {
	ActiveProvider string          `mapstructure:"active_provider"` // aliyun, huawei
	Aliyun         AliyunOSSConfig `mapstructure:"aliyun"`
}

// AliyunOSSConfig 阿里云 OSS 配置
type AliyunOSSConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Bucket          string `mapstructure:"bucket"`
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	CDNDomain       string `mapstructure:"cdn_domain"` // CDN 加速域名（可选）
}
