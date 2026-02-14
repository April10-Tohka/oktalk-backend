-- 003_init_feedbacks.sql
-- 创建反馈和报告相关表

-- 反馈表
CREATE TABLE IF NOT EXISTS `feedbacks` (
    `id` VARCHAR(36) NOT NULL,
    `evaluation_id` VARCHAR(36) NOT NULL,
    `text` TEXT DEFAULT NULL COMMENT '反馈文本',
    `level` VARCHAR(20) DEFAULT '' COMMENT '反馈等级',
    `suggestions` TEXT DEFAULT NULL COMMENT '改进建议',
    `demo_audio_url` VARCHAR(255) DEFAULT '' COMMENT '示范音频URL',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_evaluation_id` (`evaluation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='评测反馈表';

-- 学习报告表
CREATE TABLE IF NOT EXISTS `reports` (
    `id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NOT NULL,
    `type` VARCHAR(20) NOT NULL COMMENT 'weekly, monthly',
    `title` VARCHAR(100) DEFAULT '' COMMENT '报告标题',
    `summary` TEXT DEFAULT NULL COMMENT '报告摘要',
    `content` JSON DEFAULT NULL COMMENT '报告详细内容',
    `start_date` DATE DEFAULT NULL COMMENT '报告起始日期',
    `end_date` DATE DEFAULT NULL COMMENT '报告结束日期',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_type` (`type`),
    KEY `idx_created_at` (`created_at`),
    KEY `idx_user_type_created` (`user_id`, `type`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='学习报告表';

-- 对话会话表
CREATE TABLE IF NOT EXISTS `chat_sessions` (
    `id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NOT NULL,
    `title` VARCHAR(100) DEFAULT '' COMMENT '会话标题',
    `scenario` VARCHAR(50) DEFAULT '' COMMENT '对话场景',
    `status` TINYINT DEFAULT 1 COMMENT '1:活跃 0:已结束',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_status` (`status`),
    KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话会话表';

-- 对话消息表
CREATE TABLE IF NOT EXISTS `chat_messages` (
    `id` VARCHAR(36) NOT NULL,
    `session_id` VARCHAR(36) NOT NULL,
    `role` VARCHAR(20) NOT NULL COMMENT 'user, assistant, system',
    `content` TEXT NOT NULL COMMENT '消息内容',
    `audio_url` VARCHAR(255) DEFAULT '' COMMENT '语音消息URL',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_session_id` (`session_id`),
    KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话消息表';
