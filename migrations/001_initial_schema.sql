-- ============================================================================
-- OKTalk AI 发音纠正系统 - 数据库初始化脚本
-- 版本: v2.0（最终确定版）
-- 数据库: MySQL 8.0+
-- 字符集: utf8mb4_unicode_ci
-- 表数量: 7 张表
-- ============================================================================

-- 设置字符集
SET NAMES utf8mb4;
SET CHARACTER_SET_CLIENT = utf8mb4;
SET CHARACTER_SET_RESULTS = utf8mb4;

-- ============================================================================
-- 表 1：users（用户基础信息表）
-- 用途：存储用户的基础账户信息
-- ============================================================================
CREATE TABLE IF NOT EXISTS `users` (
    `id`            VARCHAR(36)     NOT NULL                    COMMENT '用户ID (UUID)',
    `username`      VARCHAR(100)    NOT NULL                    COMMENT '用户名',
    `password_hash` VARCHAR(255)    NOT NULL                    COMMENT '密码哈希值',
    `phone`         VARCHAR(20)     DEFAULT NULL                COMMENT '手机号',
    `avatar_url`    VARCHAR(500)    DEFAULT NULL                COMMENT '头像URL',
    `grade`         INT             DEFAULT NULL                COMMENT '年级 (1-6 代表小学1-6年级)',
    `created_at`    TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    `updated_at`    TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`    TIMESTAMP       DEFAULT NULL                COMMENT '软删除时间',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_users_username` (`username`),
    UNIQUE KEY `uk_users_phone` (`phone`),
    INDEX `idx_users_created_at` (`created_at`),
    INDEX `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户基础信息表';

-- ============================================================================
-- 表 2：user_profiles（用户扩展信息表）
-- 用途：存储用户的详细个人信息和学习统计数据
-- ============================================================================
CREATE TABLE IF NOT EXISTS `user_profiles` (
    `id`                        VARCHAR(36)     NOT NULL                    COMMENT '主键ID (UUID)',
    `user_id`                   VARCHAR(36)     NOT NULL                    COMMENT '用户ID (FK → users.id)',
    `age`                       INT             DEFAULT NULL                COMMENT '年龄',
    `gender`                    ENUM('male','female') DEFAULT NULL          COMMENT '性别',
    `bio`                       TEXT            DEFAULT NULL                COMMENT '个人简介',
    `total_conversations`       INT             NOT NULL DEFAULT 0          COMMENT '总对话数',
    `total_evaluations`         INT             NOT NULL DEFAULT 0          COMMENT '总评测数',
    `total_reports`             INT             NOT NULL DEFAULT 0          COMMENT '总报告数',
    `total_study_minutes`       INT             NOT NULL DEFAULT 0          COMMENT '累计学习时长（分钟）',
    `average_evaluation_score`  FLOAT           NOT NULL DEFAULT 0.0        COMMENT '平均评测分数',
    `last_conversation_at`      TIMESTAMP       DEFAULT NULL                COMMENT '上次对话时间',
    `last_evaluation_at`        TIMESTAMP       DEFAULT NULL                COMMENT '上次评测时间',
    `created_at`                TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    `updated_at`                TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_profiles_user_id` (`user_id`),
    INDEX `idx_user_profiles_last_conversation_at` (`last_conversation_at`),
    INDEX `idx_user_profiles_last_evaluation_at` (`last_evaluation_at`),
    CONSTRAINT `fk_user_profiles_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户扩展信息表';

-- ============================================================================
-- 表 3：voice_conversations（语音对话记录表）
-- 用途：存储用户的 AI 语音对话会话
-- ============================================================================
CREATE TABLE IF NOT EXISTS `voice_conversations` (
    `id`                VARCHAR(36)     NOT NULL                    COMMENT '对话ID (UUID)',
    `user_id`           VARCHAR(36)     NOT NULL                    COMMENT '用户ID (FK → users.id)',
    `topic`             VARCHAR(200)    NOT NULL                    COMMENT '对话主题',
    `difficulty_level`  ENUM('beginner','intermediate','advanced') NOT NULL DEFAULT 'beginner' COMMENT '难度等级',
    `conversation_type` ENUM('free_talk','question_answer')        NOT NULL DEFAULT 'free_talk' COMMENT '对话类型',
    `message_count`     INT             NOT NULL DEFAULT 0          COMMENT '消息总数（包括用户和AI）',
    `duration_seconds`  INT             NOT NULL DEFAULT 0          COMMENT '对话时长（秒）',
    `status`            ENUM('active','completed','paused')        NOT NULL DEFAULT 'active' COMMENT '对话状态',
    `summary`           TEXT            DEFAULT NULL                COMMENT '对话摘要（AI异步生成）',
    `score`             INT             DEFAULT NULL                COMMENT '对话评分（0-100）',
    `feedback`          TEXT            DEFAULT NULL                COMMENT 'AI反馈内容',
    `created_at`        TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    `updated_at`        TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`        TIMESTAMP       DEFAULT NULL                COMMENT '软删除时间',

    PRIMARY KEY (`id`),
    INDEX `idx_voice_conversations_user_id` (`user_id`),
    INDEX `idx_voice_conversations_created_at` (`created_at`),
    INDEX `idx_voice_conversations_type_status` (`conversation_type`, `status`),
    INDEX `idx_voice_conversations_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_voice_conversations_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='语音对话记录表';

-- ============================================================================
-- 表 4：conversation_messages（对话消息明细表）
-- 用途：存储对话中的每一条消息（用户或AI），每条消息单独一条记录
-- ============================================================================
CREATE TABLE IF NOT EXISTS `conversation_messages` (
    `id`                VARCHAR(36)     NOT NULL                    COMMENT '消息ID (UUID)',
    `conversation_id`   VARCHAR(36)     NOT NULL                    COMMENT '对话ID (FK → voice_conversations.id)',
    `sender_type`       ENUM('user','ai') NOT NULL                  COMMENT '发送者类型',
    `message_text`      LONGTEXT        NOT NULL                    COMMENT '消息文本内容',
    `audio_url`         VARCHAR(500)    DEFAULT NULL                COMMENT '音频URL',
    `audio_duration`    INT             DEFAULT NULL                COMMENT '音频时长（秒）',
    `sequence_number`   INT             NOT NULL                    COMMENT '消息序号（从1开始）',
    `latency_ms`        INT             DEFAULT NULL                COMMENT '处理延迟（毫秒）',
    `created_at`        TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    `updated_at`        TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    PRIMARY KEY (`id`),
    INDEX `idx_conversation_messages_conv_seq` (`conversation_id`, `sequence_number`),
    INDEX `idx_conversation_messages_sender_created` (`sender_type`, `created_at`),
    CONSTRAINT `fk_conversation_messages_conversation_id` FOREIGN KEY (`conversation_id`) REFERENCES `voice_conversations` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话消息明细表';

-- ============================================================================
-- 表 5：pronunciation_evaluations（发音评测记录表）
-- 用途：存储用户的发音评测结果和分级反馈数据
--
-- 分级反馈机制：
--   S 级 (90-100): 纯鼓励，不提供示范音频
--   A 级 (70-89):  鼓励+诊断，提供问题单词示范音频
--   B 级 (50-69):  诊断+示范，提供问题单词示范音频
--   C 级 (0-49):   完整示范，提供整句示范音频
-- ============================================================================
CREATE TABLE IF NOT EXISTS `pronunciation_evaluations` (
    `id`                        VARCHAR(36)     NOT NULL                    COMMENT '评测ID (UUID)',
    `user_id`                   VARCHAR(36)     NOT NULL                    COMMENT '用户ID (FK → users.id)',
    `target_text`               VARCHAR(500)    NOT NULL                    COMMENT '目标朗读文本',
    `recognized_text`           VARCHAR(500)    DEFAULT NULL                COMMENT '识别出的文本',
    `audio_url`                 VARCHAR(500)    DEFAULT NULL                COMMENT '原始录音URL',
    `audio_duration`            INT             DEFAULT NULL                COMMENT '录音时长（秒）',

    -- 评分字段
    `overall_score`             INT             NOT NULL DEFAULT 0          COMMENT '综合评分（0-100）',
    `accuracy_score`            INT             NOT NULL DEFAULT 0          COMMENT '准确度评分（0-100）',
    `fluency_score`             INT             NOT NULL DEFAULT 0          COMMENT '流利度评分（0-100）',
    `integrity_score`           INT             NOT NULL DEFAULT 0          COMMENT '完整度评分（0-100）',

    -- 反馈字段
    `feedback_level`            ENUM('S','A','B','C') NOT NULL DEFAULT 'C'  COMMENT '反馈级别',
    `feedback_text`             TEXT            DEFAULT NULL                COMMENT '反馈文本内容（LLM生成）',
    `feedback_audio_url`        VARCHAR(500)    DEFAULT NULL                COMMENT '反馈音频URL（TTS生成）',

    -- 问题单词字段（A/B级使用）
    `problem_words`             JSON            DEFAULT NULL                COMMENT '问题单词列表 (JSON数组)',
    `problem_word_audio_urls`   JSON            DEFAULT NULL                COMMENT '问题单词示范音频URL (JSON对象)',

    -- 整句示范字段（C级使用）
    `demo_sentence_audio_url`   VARCHAR(500)    DEFAULT NULL                COMMENT '整句示范音频URL（仅C级）',

    -- 其他字段
    `difficulty_level`          ENUM('beginner','intermediate','advanced') NOT NULL DEFAULT 'beginner' COMMENT '难度级别',
    `assessment_sid`            VARCHAR(100)    DEFAULT NULL                COMMENT '语音评测会话ID',
    `speech_assessment_json`    LONGTEXT        DEFAULT NULL                COMMENT '语音评测原始数据（JSON）',
    `status`                    ENUM('pending','processing','completed','failed') NOT NULL DEFAULT 'pending' COMMENT '评测状态',
    `error_message`             VARCHAR(500)    DEFAULT NULL                COMMENT '错误信息（失败时）',
    `created_at`                TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    `updated_at`                TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    PRIMARY KEY (`id`),
    INDEX `idx_pronunciation_evaluations_user_created` (`user_id`, `created_at`),
    INDEX `idx_pronunciation_evaluations_level_score` (`feedback_level`, `overall_score`),
    INDEX `idx_pronunciation_evaluations_status_created` (`status`, `created_at`),
    CONSTRAINT `fk_pronunciation_evaluations_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发音评测记录表';

-- ============================================================================
-- 表 6：learning_reports（学习报告表）
-- 用途：存储用户的学习报告数据，支持雷达图分析
--
-- 雷达图三维分析：
--   1. 准确度（Accuracy）: average_accuracy_score
--   2. 流利度（Fluency）:  average_fluency_score
--   3. 完整度（Integrity）: average_integrity_score
-- ============================================================================
CREATE TABLE IF NOT EXISTS `learning_reports` (
    `id`                            VARCHAR(36)     NOT NULL                    COMMENT '报告ID (UUID)',
    `user_id`                       VARCHAR(36)     NOT NULL                    COMMENT '用户ID (FK → users.id)',
    `report_type`                   ENUM('weekly','monthly','custom') NOT NULL DEFAULT 'weekly' COMMENT '报告类型',
    `period_start_date`             DATE            NOT NULL                    COMMENT '周期起始日期',
    `period_end_date`               DATE            NOT NULL                    COMMENT '周期结束日期',

    -- 统计数据
    `total_conversations`           INT             NOT NULL DEFAULT 0          COMMENT '总对话数',
    `total_evaluations`             INT             NOT NULL DEFAULT 0          COMMENT '总评测数',
    `total_study_minutes`           INT             NOT NULL DEFAULT 0          COMMENT '总学习时长（分钟）',

    -- 平均分数
    `average_conversation_score`    FLOAT           NOT NULL DEFAULT 0.0        COMMENT '平均对话评分',
    `average_evaluation_score`      FLOAT           NOT NULL DEFAULT 0.0        COMMENT '平均评测分数',

    -- 雷达图数据
    `average_accuracy_score`        FLOAT           NOT NULL DEFAULT 0.0        COMMENT '平均准确度评分（雷达图）',
    `average_fluency_score`         FLOAT           NOT NULL DEFAULT 0.0        COMMENT '平均流利度评分（雷达图）',
    `average_integrity_score`       FLOAT           NOT NULL DEFAULT 0.0        COMMENT '平均完整度评分（雷达图）',

    -- 分级统计
    `s_level_count`                 INT             NOT NULL DEFAULT 0          COMMENT 'S级评测数',
    `a_level_count`                 INT             NOT NULL DEFAULT 0          COMMENT 'A级评测数',
    `b_level_count`                 INT             NOT NULL DEFAULT 0          COMMENT 'B级评测数',
    `c_level_count`                 INT             NOT NULL DEFAULT 0          COMMENT 'C级评测数',

    -- 进步分析
    `improvement_rate`              FLOAT           NOT NULL DEFAULT 0.0        COMMENT '进步率（百分比）',

    -- JSON 分析字段
    `most_practiced_topics`         JSON            DEFAULT NULL                COMMENT '最常练习的主题 (JSON数组)',
    `problem_words`                 JSON            DEFAULT NULL                COMMENT '高频问题单词 (JSON数组)',
    `strengths`                     JSON            DEFAULT NULL                COMMENT '优势分析 (JSON数组)',
    `weaknesses`                    JSON            DEFAULT NULL                COMMENT '不足分析 (JSON数组)',

    -- AI 建议
    `recommendations`               LONGTEXT        DEFAULT NULL                COMMENT 'AI生成的学习建议',
    `ai_model`                      VARCHAR(100)    DEFAULT NULL                COMMENT '生成建议使用的AI模型',

    -- 时间戳
    `created_at`                    TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    `updated_at`                    TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    PRIMARY KEY (`id`),
    INDEX `idx_learning_reports_user_created` (`user_id`, `created_at`),
    INDEX `idx_learning_reports_type_period` (`report_type`, `period_start_date`),
    CONSTRAINT `fk_learning_reports_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='学习报告表';

-- ============================================================================
-- 表 7：system_settings（系统配置表）
-- 用途：存储系统级配置参数
-- ============================================================================
CREATE TABLE IF NOT EXISTS `system_settings` (
    `id`            VARCHAR(36)     NOT NULL                    COMMENT '主键ID (UUID)',
    `config_key`    VARCHAR(100)    NOT NULL                    COMMENT '配置键',
    `config_value`  LONGTEXT        NOT NULL                    COMMENT '配置值',
    `config_type`   ENUM('string','int','float','json','boolean') NOT NULL DEFAULT 'string' COMMENT '配置类型',
    `description`   VARCHAR(500)    DEFAULT NULL                COMMENT '配置描述',
    `is_editable`   BOOLEAN         NOT NULL DEFAULT TRUE       COMMENT '是否可编辑',
    `created_at`    TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    `updated_at`    TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_system_settings_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表';

-- ============================================================================
-- 初始化默认系统配置
-- ============================================================================
INSERT INTO `system_settings` (`id`, `config_key`, `config_value`, `config_type`, `description`, `is_editable`)
VALUES
    ('set_001', 'feedback_s_level_min_score', '90', 'int', 'S级反馈最低分数', TRUE),
    ('set_002', 'feedback_a_level_min_score', '70', 'int', 'A级反馈最低分数', TRUE),
    ('set_003', 'feedback_b_level_min_score', '50', 'int', 'B级反馈最低分数', TRUE),
    ('set_004', 'feedback_c_level_min_score', '0',  'int', 'C级反馈最低分数', TRUE),
    ('set_005', 'forbidden_words', '["bad","wrong","terrible","fail","mistake","error","poor","awful"]', 'json', '禁用词列表（反馈生成时过滤）', TRUE),
    ('set_006', 'report_generation_schedule', '0 0 * * 1', 'string', '报告生成时间表(cron表达式，每周一)', TRUE),
    ('set_007', 'max_audio_duration_seconds', '60', 'int', '最大录音时长（秒）', TRUE),
    ('set_008', 'max_conversation_messages', '50', 'int', '单次对话最大消息数', TRUE),
    ('set_009', 'default_tts_voice', 'longanyang', 'string', '默认TTS音色', TRUE),
    ('set_010', 'default_llm_model', 'qwen-plus', 'string', '默认LLM模型', TRUE)
ON DUPLICATE KEY UPDATE `updated_at` = CURRENT_TIMESTAMP;
