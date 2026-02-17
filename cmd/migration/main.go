// Package main 提供数据库迁移工具
// 支持以下操作：
//   - up:     执行 GORM AutoMigrate（创建/更新表结构）
//   - seed:   初始化系统默认配置数据
//   - sql:    执行指定的 SQL 迁移文件
//   - all:    依次执行 up + seed（完整初始化）
//   - status: 检查数据库连接状态
//
// 使用示例:
//
//	go run cmd/migration/main.go -action=up
//	go run cmd/migration/main.go -action=seed
//	go run cmd/migration/main.go -action=sql -file=migrations/001_initial_schema.sql
//	go run cmd/migration/main.go -action=all
//	go run cmd/migration/main.go -action=status
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"pronunciation-correction-system/internal/config"
	"pronunciation-correction-system/internal/db"
)

func main() {
	// 命令行参数
	action := flag.String("action", "up", "Migration action: up, seed, sql, all, status")
	sqlFile := flag.String("file", "", "SQL file path (used with -action=sql)")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %v", err)
	}

	// 初始化数据库连接
	database, err := db.Init(cfg)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect to database: %v", err)
	}
	defer func() {
		if closeErr := db.Close(database); closeErr != nil {
			log.Printf("[WARN] Failed to close database: %v", closeErr)
		}
	}()

	log.Printf("[INFO] Connected to database: %s@%s:%d/%s",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 执行操作
	switch *action {
	case "up":
		runMigrate(database)
	case "seed":
		runSeed(database)
	case "sql":
		if *sqlFile == "" {
			log.Fatal("[FATAL] -file parameter is required for sql action")
		}
		runSQL(database, *sqlFile)
	case "all":
		runMigrate(database)
		runSeed(database)
	case "status":
		runStatus(database)
	default:
		log.Fatalf("[FATAL] Unknown action: %s. Available: up, seed, sql, all, status", *action)
	}
}

// runMigrate 执行 GORM AutoMigrate（创建/更新表结构）
func runMigrate(database *gorm.DB) {
	log.Println("========================================")
	log.Println("[INFO] Running GORM AutoMigrate...")
	log.Println("========================================")

	start := time.Now()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("[FATAL] Migration failed: %v", err)
	}

	log.Printf("[INFO] AutoMigrate completed successfully (duration: %v)", time.Since(start))
}

// runSeed 初始化系统默认配置数据
func runSeed(database *gorm.DB) {
	log.Println("========================================")
	log.Println("[INFO] Seeding default system settings...")
	log.Println("========================================")

	start := time.Now()
	ctx := context.Background()

	if err := db.InitSystemSettings(ctx, database); err != nil {
		log.Fatalf("[FATAL] Seed failed: %v", err)
	}

	log.Printf("[INFO] Seed completed successfully (duration: %v)", time.Since(start))
}

// runSQL 执行 SQL 迁移文件
func runSQL(database *gorm.DB, filePath string) {
	log.Println("========================================")
	log.Printf("[INFO] Executing SQL file: %s", filePath)
	log.Println("========================================")

	start := time.Now()

	// 读取 SQL 文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to read SQL file: %v", err)
	}

	sqlContent := string(content)
	if strings.TrimSpace(sqlContent) == "" {
		log.Println("[WARN] SQL file is empty, skipping")
		return
	}

	// 按分号分割 SQL 语句并逐条执行
	statements := splitSQLStatements(sqlContent)
	total := len(statements)
	succeeded := 0
	skipped := 0

	log.Printf("[INFO] Found %d SQL statements to execute", total)

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			skipped++
			continue
		}

		// 显示语句的前 80 个字符
		preview := stmt
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		log.Printf("[INFO] (%d/%d) Executing: %s", i+1, total, preview)

		if err := database.Exec(stmt).Error; err != nil {
			log.Fatalf("[FATAL] Statement %d failed: %v\nSQL: %s", i+1, err, stmt)
		}
		succeeded++
	}

	log.Printf("[INFO] SQL execution completed: %d succeeded, %d skipped (duration: %v)",
		succeeded, skipped, time.Since(start))
}

// runStatus 检查数据库连接状态
func runStatus(database *gorm.DB) {
	log.Println("========================================")
	log.Println("[INFO] Checking database status...")
	log.Println("========================================")

	// 检查连接
	if err := db.Ping(database); err != nil {
		log.Fatalf("[FATAL] Database connection failed: %v", err)
	}
	log.Println("[INFO] Database connection: OK")

	// 查询已存在的表
	var tables []string
	if err := database.Raw("SHOW TABLES").Scan(&tables).Error; err != nil {
		log.Printf("[WARN] Failed to list tables: %v", err)
	} else {
		log.Printf("[INFO] Tables found (%d):", len(tables))
		for _, t := range tables {
			// 查询行数
			var count int64
			database.Raw(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", t)).Scan(&count)
			log.Printf("  - %-35s (%d rows)", t, count)
		}
	}
}

// splitSQLStatements 按分号分割 SQL 语句
// 会忽略注释行和空行
func splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inSingleLineComment := false

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 跳过空行
		if trimmed == "" {
			continue
		}

		// 跳过单行注释（-- 开头）
		if strings.HasPrefix(trimmed, "--") {
			inSingleLineComment = false
			continue
		}

		_ = inSingleLineComment // 保留以供未来扩展

		// 追加当前行
		current.WriteString(line)
		current.WriteString("\n")

		// 检查是否以分号结尾（语句结束）
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			// 移除末尾分号
			stmt = strings.TrimSuffix(stmt, ";")
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	// 处理没有以分号结尾的最后一条语句
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		statements = append(statements, remaining)
	}

	return statements
}
