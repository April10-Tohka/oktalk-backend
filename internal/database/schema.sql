-- OKTalk 系统数据库初始化脚本
-- 创建数据库
CREATE DATABASE IF NOT EXISTS oktalk_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE oktalk_db;

-- ==================== 1. 用户基础信息表 ====================
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY COMMENT '用户ID (UUID)',
    username VARCHAR(100) NOT NULL UNIQUE COMMENT '用户名',
    email VARCHAR(255) NOT NULL UNIQUE COMMENT '邮箱',
    password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
    phone VARCHAR(20) COMMENT '手机号',
    avatar_url VARCHAR(500) COMMENT '头像URL',
    grade INT COMMENT '年级(1-6代表小学年级)',
    status ENUM('active','suspended','deleted') NOT NULL DEFAULT 'active' COMMENT '账户状态',
    language VARCHAR(10) NOT NULL DEFAULT 'en' COMMENT '语言偏好: en/zh',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL COMMENT '软删除时间',
    INDEX idx_created_at (created_at),
    INDEX idx_deleted_at (deleted_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户基础信息表';

-- ==================== 2. 用户扩展信息表 ====================
CREATE TABLE IF NOT EXISTS user_profiles (
    id VARCHAR(36) PRIMARY KEY COMMENT '主键ID (UUID)',
    user_id VARCHAR(36) NOT NULL UNIQUE COMMENT '用户ID (FK)',
    full_name VARCHAR(100) COMMENT '真实姓名',
    age INT COMMENT '年龄',
    gender ENUM('male','female','other') COMMENT '性别',
    bio TEXT COMMENT '个人简介',
    total_conversations INT NOT NULL DEFAULT 0 COMMENT '总对话数',
    total_evaluations INT NOT NULL DEFAULT 0 COMMENT '总评测数',
    total_reports INT NOT NULL DEFAULT 0 COMMENT '总报告数',
    average_evaluation_score FLOAT NOT NULL DEFAULT 0.0 COMMENT '平均评测分数',
    last_conversation_at TIMESTAMP NULL COMMENT '上次对话时间',
    last_evaluation_at TIMESTAMP NULL COMMENT '上次评测时间',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_last_conversation_at (last_conversation_at),
    INDEX idx_last_evaluation_at (last_evaluation_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户扩展信息表';

-- ==================== 3. 语音对话记录表 ====================
CREATE TABLE IF NOT EXISTS voice_conversations (
    id VARCHAR(36) PRIMARY KEY COMMENT '对话ID (UUID)',
    user_id VARCHAR(36) NOT NULL COMMENT '用户ID (FK)',
    topic VARCHAR(200) NOT NULL COMMENT '对话主题',
    difficulty_level ENUM('beginner','intermediate','advanced') NOT NULL DEFAULT 'beginner' COMMENT '难度级别',
    conversation_type ENUM('free_talk','scenario','question_answer') NOT NULL DEFAULT 'free_talk' COMMENT '对话类型',
    message_count INT NOT NULL DEFAULT 0 COMMENT '消息总数',
    duration_seconds INT NOT NULL DEFAULT 0 COMMENT '对话时长(秒)',
    status ENUM('active','completed','paused') NOT NULL DEFAULT 'active' COMMENT '状态',
    language_pair VARCHAR(20) NOT NULL DEFAULT 'en-zh' COMMENT '语言对',
    summary TEXT COMMENT '对话摘要',
    ai_model VARCHAR(100) NOT NULL DEFAULT 'gpt-4' COMMENT 'AI模型名称',
    score INT COMMENT '对话评分(0-100)',
    feedback TEXT COMMENT 'AI反馈内容',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL COMMENT '软删除时间',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_status (status),
    INDEX idx_difficulty_level (difficulty_level),
    INDEX idx_conversation_type (conversation_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='语音对话记录表';

-- ==================== 4. 对话消息明细表 ====================
CREATE TABLE IF NOT EXISTS conversation_messages (
    id VARCHAR(36) PRIMARY KEY COMMENT '消息ID (UUID)',
    conversation_id VARCHAR(36) NOT NULL COMMENT '对话ID (FK)',
    sender_type ENUM('user','ai') NOT NULL COMMENT '发送者类型',
    sender_id VARCHAR(36) COMMENT '发送者ID(仅user时有值)',
    message_text LONGTEXT NOT NULL COMMENT '消息文本',
    message_audio_url VARCHAR(500) COMMENT '消息音频URL',
    message_audio_duration INT COMMENT '音频时长(秒)',
    ai_response_text LONGTEXT COMMENT 'AI回复文本',
    ai_response_audio_url VARCHAR(500) COMMENT 'AI回复音频URL',
    pronunciation_score INT COMMENT '发音评分(0-100)',
    pronunciation_feedback TEXT COMMENT '发音反馈',
    sequence_number INT NOT NULL COMMENT '消息序号',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (conversation_id) REFERENCES voice_conversations(id) ON DELETE CASCADE,
    INDEX idx_conversation_id (conversation_id),
    INDEX idx_sequence_number (sequence_number),
    INDEX idx_sender_type (sender_type),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话消息明细表';

-- ==================== 5. 发音评测记录表 ====================
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

-- ==================== 6. 评测详细数据表 ====================
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

-- ==================== 7. 反馈记录表 ====================
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

-- ==================== 8. 学习报告表 ====================
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

-- ==================== 9. 报告统计明细表 ====================
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

-- ==================== 10. 系统配置表 ====================
CREATE TABLE IF NOT EXISTS system_settings (
    id VARCHAR(36) PRIMARY KEY COMMENT '主键ID (UUID)',
    config_key VARCHAR(100) NOT NULL UNIQUE COMMENT '配置键',
    config_value LONGTEXT NOT NULL COMMENT '配置值(支持JSON)',
    config_type ENUM('string','int','float','json','boolean') NOT NULL COMMENT '配置类型',
    description VARCHAR(500) COMMENT '配置描述',
    is_editable BOOLEAN NOT NULL DEFAULT TRUE COMMENT '是否可编辑',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_config_key (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表';

-- ==================== 插入初始系统配置 ====================
INSERT INTO system_settings (id, config_key, config_value, config_type, description) VALUES
('set_001', 'feedback_s_level_min_score', '90', 'int', 'S级反馈最低分数'),
('set_002', 'feedback_a_level_min_score', '70', 'int', 'A级反馈最低分数'),
('set_003', 'feedback_b_level_min_score', '50', 'int', 'B级反馈最低分数'),
('set_004', 'feedback_c_level_min_score', '0', 'int', 'C级反馈最低分数'),
('set_005', 'forbidden_words', '["bad","wrong","terrible","fail","mistake","error","poor","awful"]', 'json', '禁用词列表'),
('set_006', 'report_generation_schedule', '0 0 * * 1', 'string', '报告生成时间表(cron)');
