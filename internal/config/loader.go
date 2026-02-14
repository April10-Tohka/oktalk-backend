// Package config 提供配置加载功能
// 支持从环境变量和配置文件加载配置
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load 加载应用程序配置
// 优先级: 环境变量 > 配置文件 > 默认值
func Load() (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 获取运行环境
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	// 配置文件路径
	v.SetConfigName(env)
	v.SetConfigType("yaml")
	v.AddConfigPath("./internal/config/env")
	v.AddConfigPath("./config/env")
	v.AddConfigPath(".")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// 配置文件不存在时使用默认值和环境变量
	}

	// 绑定环境变量
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 解析配置到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	fmt.Println("cfg:", cfg)
	return &cfg, nil
}

// setDefaults 设置配置默认值
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.name", "pronunciation-correction-system")

	// 数据库默认配置
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.user", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "pronunciation_db")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)

	// Redis 默认配置
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// LLM 默认配置
	v.SetDefault("llm.active_provider", "qwen")
	v.SetDefault("llm.qwen.model", "qwen-turbo")

	// ASR 默认配置
	v.SetDefault("asr.active_provider", "xunfei")

	// TTS 默认配置
	v.SetDefault("tts.active_provider", "aliyun")
	v.SetDefault("tts.aliyun.model", "cosyvoice-v3-flash")
	v.SetDefault("tts.aliyun.region", "beijing")
	v.SetDefault("tts.aliyun.default_options.voice", "longanyang")
	v.SetDefault("tts.aliyun.default_options.format", "mp3")
	v.SetDefault("tts.aliyun.default_options.sample_rate", 22050)
	v.SetDefault("tts.aliyun.default_options.volume", 50)
	v.SetDefault("tts.aliyun.default_options.rate", 1.0)
	v.SetDefault("tts.aliyun.default_options.pitch", 1.0)

	// OSS 默认配置
	v.SetDefault("oss.active_provider", "aliyun")

	// JWT 默认配置
	v.SetDefault("jwt.expire_hours", 24)

	// 日志默认配置
	v.SetDefault("log.level", "debug")
	v.SetDefault("log.output", "stdout")
}
