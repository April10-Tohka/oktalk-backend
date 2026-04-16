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
	return &cfg, nil
}
