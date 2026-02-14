// Package main 提供数据库迁移工具
// 用于创建和更新数据库表结构
package main

import (
	"flag"
	"log"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/db"
)

func main() {
	// 命令行参数
	action := flag.String("action", "up", "Migration action: up, down, status")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库连接
	database, err := db.Init(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 执行迁移操作
	switch *action {
	case "up":
		log.Println("Running migrations...")
		if err := db.Migrate(database); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migrations completed successfully")
	case "down":
		log.Println("Rolling back migrations...")
		// TODO: 实现 Rollback 逻辑
		log.Println("Rollback not yet implemented")
	case "status":
		log.Println("Checking migration status...")
		if err := db.Ping(database); err != nil {
			log.Fatalf("Database connection failed: %v", err)
		}
		log.Println("Database connection OK")
	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}
