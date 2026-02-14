-- 迁移文件: 003_init_evaluations.sql
-- 用途: 创建发音评测相关表

-- +migrate Up
-- 1. 发音评测记录表
CREATE TABLE IF NOT EXISTS pronunciation_evaluations (
    id VARCHAR(36) PRIMARY KEY COMMENT '评测ID (UUID)',
    user_id VARCHAR(36) NOT NULL COMMENT '用户ID (FK)',
    target_text VARCHAR(500) NOT NULL COMMENT '目标朗读文本',
    recognized_text VARCHAR(500) COMMENT '识别出的文本',
    audio_url VARCHAR(500) COMMENT '原始录音URL',
    audio_duration INT COMMENT '录音时长(秒)',
    overall_score INT NOT NULL DEFAULT 0 COMMENT '综合评分(0-100)',
    accuracy_score INT NOT NULL DEFAULT 0 COMMENT '准确度评分(0-100)',
    fluency_score INT NOT NULL DEFAULT 0 COMMENT '流利度评分(0-100)',
    integrity_score INT NOT NULL DEFAULT 0 COMMENT '完整度评分(0-100)',
    feedback_level ENUM('S','A','B','C') NOT NULL DEFAULT 'C' COMMENT '反馈级别',
    feedback_text TEXT COMMENT '反馈文本',
    feedback_audio_url VARCHAR(500) COMMENT '反馈音频URL',
    demo_audio_type ENUM('word','sentence') COMMENT '示范音频类型',
    demo_audio_content VARCHAR(500) COMMENT '示范内容',
    demo_audio_url VARCHAR(500) COMMENT '示范音频URL',
    difficulty_level ENUM('beginner','intermediate','advanced') NOT NULL DEFAULT 'beginner' COMMENT '难度级别',
    speech_assessment_json LONGTEXT COMMENT '讯飞返回的原始评测数据(JSON)',
    status ENUM('pending','processing','completed','failed') NOT NULL DEFAULT 'completed' COMMENT '状态',
    error_message VARCHAR(500) COMMENT '错误信息(失败时)',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_feedback_level (feedback_level),
    INDEX idx_overall_score (overall_score),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发音评测记录表';

-- 2. 评测详细数据表
CREATE TABLE IF NOT EXISTS evaluation_details (
    id VARCHAR(36) PRIMARY KEY COMMENT '主键ID (UUID)',
    evaluation_id VARCHAR(36) NOT NULL COMMENT '评测ID (FK)',
    word_index INT NOT NULL COMMENT '单词序号(0开始)',
    word_text VARCHAR(100) NOT NULL COMMENT '单词文本',
    word_score INT NOT NULL DEFAULT 0 COMMENT '单词评分(0-100)',
    is_problem_word BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否问题单词(score<70)',
    phoneme_details JSON COMMENT '音素详情(JSON数组)',
    begin_time_ms INT COMMENT '单词开始时间(毫秒)',
    end_time_ms INT COMMENT '单词结束时间(毫秒)',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (evaluation_id) REFERENCES pronunciation_evaluations(id) ON DELETE CASCADE,
    INDEX idx_evaluation_id (evaluation_id),
    INDEX idx_word_index (word_index),
    INDEX idx_is_problem_word (is_problem_word)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='评测详细数据表';

-- 3. 反馈记录表
CREATE TABLE IF NOT EXISTS feedback_records (
    id VARCHAR(36) PRIMARY KEY COMMENT '反馈ID (UUID)',
    evaluation_id VARCHAR(36) NOT NULL UNIQUE COMMENT '评测ID (FK)',
    feedback_level ENUM('S','A','B','C') NOT NULL COMMENT '反馈级别',
    feedback_text TEXT NOT NULL COMMENT '反馈文本',
    feedback_audio_url VARCHAR(500) COMMENT '反馈音频URL',
    demo_type ENUM('word','sentence') COMMENT '示范类型',
    demo_content VARCHAR(500) COMMENT '示范内容',
    demo_audio_url VARCHAR(500) COMMENT '示范音频URL',
    generation_method VARCHAR(50) NOT NULL DEFAULT 'llm' COMMENT '生成方法: llm/template/manual',
    ai_model VARCHAR(100) COMMENT 'AI模型名称',
    generation_duration_ms INT COMMENT '生成耗时(毫秒)',
    status ENUM('generating','completed','failed') NOT NULL DEFAULT 'completed' COMMENT '状态',
    quality_score INT COMMENT '反馈质量评分(0-100)',
    user_feedback VARCHAR(500) COMMENT '用户对反馈的评价',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (evaluation_id) REFERENCES pronunciation_evaluations(id) ON DELETE CASCADE,
    INDEX idx_evaluation_id (evaluation_id),
    INDEX idx_feedback_level (feedback_level),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='反馈记录表';

-- +migrate Down
DROP TABLE IF EXISTS feedback_records;
DROP TABLE IF EXISTS evaluation_details;
DROP TABLE IF EXISTS pronunciation_evaluations;
