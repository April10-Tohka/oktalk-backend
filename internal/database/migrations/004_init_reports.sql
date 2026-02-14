-- 迁移文件: 004_init_reports.sql
-- 用途: 创建学习报告相关表

-- +migrate Up
-- 1. 学习报告表
CREATE TABLE IF NOT EXISTS learning_reports (
    id VARCHAR(36) PRIMARY KEY COMMENT '报告ID (UUID)',
    user_id VARCHAR(36) NOT NULL COMMENT '用户ID (FK)',
    report_type ENUM('weekly','monthly','custom') NOT NULL DEFAULT 'weekly' COMMENT '报告类型',
    period_start_date DATE NOT NULL COMMENT '统计周期起始日期',
    period_end_date DATE NOT NULL COMMENT '统计周期结束日期',
    total_conversations INT NOT NULL DEFAULT 0 COMMENT '总对话数',
    total_evaluations INT NOT NULL DEFAULT 0 COMMENT '总评测数',
    total_study_minutes INT NOT NULL DEFAULT 0 COMMENT '总学习时长(分钟)',
    average_conversation_score FLOAT NOT NULL DEFAULT 0.0 COMMENT '平均对话评分',
    average_evaluation_score FLOAT NOT NULL DEFAULT 0.0 COMMENT '平均评测分数',
    s_level_count INT NOT NULL DEFAULT 0 COMMENT 'S级评测数',
    a_level_count INT NOT NULL DEFAULT 0 COMMENT 'A级评测数',
    b_level_count INT NOT NULL DEFAULT 0 COMMENT 'B级评测数',
    c_level_count INT NOT NULL DEFAULT 0 COMMENT 'C级评测数',
    improvement_rate FLOAT NOT NULL DEFAULT 0.0 COMMENT '进步率(%)',
    strengths JSON COMMENT '优势总结(JSON数组)',
    weaknesses JSON COMMENT '不足总结(JSON数组)',
    recommendations LONGTEXT COMMENT '改进建议',
    report_content LONGTEXT COMMENT '完整报告内容(HTML/Markdown)',
    generated_by VARCHAR(50) NOT NULL DEFAULT 'system' COMMENT '生成者: system/manual',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_report_type (report_type),
    INDEX idx_period_start_date (period_start_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='学习报告表';

-- 2. 报告统计明细表
CREATE TABLE IF NOT EXISTS report_statistics (
    id VARCHAR(36) PRIMARY KEY COMMENT '主键ID (UUID)',
    report_id VARCHAR(36) NOT NULL COMMENT '报告ID (FK)',
    stat_date DATE NOT NULL COMMENT '统计日期',
    daily_conversations INT NOT NULL DEFAULT 0 COMMENT '当日对话数',
    daily_evaluations INT NOT NULL DEFAULT 0 COMMENT '当日评测数',
    daily_study_minutes INT NOT NULL DEFAULT 0 COMMENT '当日学习时长(分钟)',
    daily_avg_eval_score FLOAT NOT NULL DEFAULT 0.0 COMMENT '当日平均评测分数',
    daily_s_level_count INT NOT NULL DEFAULT 0 COMMENT '当日S级数',
    daily_a_level_count INT NOT NULL DEFAULT 0 COMMENT '当日A级数',
    daily_b_level_count INT NOT NULL DEFAULT 0 COMMENT '当日B级数',
    daily_c_level_count INT NOT NULL DEFAULT 0 COMMENT '当日C级数',
    topic_breakdown JSON COMMENT '主题分布(JSON)',
    difficulty_breakdown JSON COMMENT '难度分布(JSON)',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (report_id) REFERENCES learning_reports(id) ON DELETE CASCADE,
    INDEX idx_report_id (report_id),
    INDEX idx_stat_date (stat_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='报告统计明细表';

-- +migrate Down
DROP TABLE IF EXISTS report_statistics;
DROP TABLE IF EXISTS learning_reports;
