-- 迁移文件: 002_init_conversations.sql
-- 用途: 创建语音对话相关表

-- +migrate Up
-- 1. 语音对话记录表
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

-- 2. 对话消息明细表
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

-- +migrate Down
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS voice_conversations;
