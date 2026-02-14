-- 002_init_evaluations.sql
-- 创建评测相关表

-- 评测记录表
CREATE TABLE IF NOT EXISTS `evaluations` (
    `id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NOT NULL,
    `text_id` VARCHAR(36) DEFAULT NULL,
    `text` TEXT NOT NULL COMMENT '评测文本',
    `audio_url` VARCHAR(255) DEFAULT '' COMMENT '用户音频URL',
    `score` DECIMAL(5,2) DEFAULT 0 COMMENT '综合评分',
    `level` VARCHAR(10) DEFAULT '' COMMENT '评分等级',
    `accuracy` DECIMAL(5,2) DEFAULT 0 COMMENT '准确度',
    `fluency` DECIMAL(5,2) DEFAULT 0 COMMENT '流利度',
    `completeness` DECIMAL(5,2) DEFAULT 0 COMMENT '完整度',
    `intonation` DECIMAL(5,2) DEFAULT 0 COMMENT '语调',
    `duration` INT DEFAULT 0 COMMENT '音频时长(毫秒)',
    `details` JSON DEFAULT NULL COMMENT '详细评测数据',
    `status` TINYINT DEFAULT 1 COMMENT '1:成功 0:失败 2:处理中',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_text_id` (`text_id`),
    KEY `idx_score` (`score`),
    KEY `idx_created_at` (`created_at`),
    KEY `idx_user_created` (`user_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发音评测记录表';

-- 练习文本表
CREATE TABLE IF NOT EXISTS `texts` (
    `id` VARCHAR(36) NOT NULL,
    `content` TEXT NOT NULL COMMENT '文本内容',
    `translation` TEXT DEFAULT NULL COMMENT '中文翻译',
    `phonetic` TEXT DEFAULT NULL COMMENT '音标',
    `level` VARCHAR(20) DEFAULT '' COMMENT '难度等级',
    `scenario` VARCHAR(50) DEFAULT '' COMMENT '场景分类',
    `tags` VARCHAR(255) DEFAULT '' COMMENT '标签',
    `demo_audio_url` VARCHAR(255) DEFAULT '' COMMENT '示范音频URL',
    `duration` INT DEFAULT 0 COMMENT '示范音频时长(毫秒)',
    `usage_count` INT DEFAULT 0 COMMENT '使用次数',
    `status` TINYINT DEFAULT 1 COMMENT '1:启用 0:禁用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_level` (`level`),
    KEY `idx_scenario` (`scenario`),
    KEY `idx_status` (`status`),
    KEY `idx_usage_count` (`usage_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='练习文本表';
