-- ============================================================================
-- 由 internal/model 推导的 MySQL 8.0+ DDL（utf8mb4）
-- 表创建顺序：先 users，再依赖 users 的表
-- ============================================================================

SET NAMES utf8mb4;

-- ----------------------------------------------------------------------------
-- users
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `users` (
    `id`               VARCHAR(36)  NOT NULL COMMENT '用户ID UUID',
    `username`         VARCHAR(100) NOT NULL COMMENT '用户名',
    `phone`            VARCHAR(20)  DEFAULT NULL COMMENT '手机号',
    `register_source`  VARCHAR(20)  NOT NULL COMMENT 'sms/wechat',
    `status`           VARCHAR(20)  NOT NULL DEFAULT 'active' COMMENT 'active/banned/deactivated',
    `avatar_url`       VARCHAR(500) DEFAULT NULL COMMENT '头像URL',
    `grade`            INT          DEFAULT NULL COMMENT '年级1-6',
    `created_at`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`       TIMESTAMP    NULL DEFAULT NULL COMMENT '软删除',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_users_username` (`username`),
    UNIQUE KEY `uk_users_phone` (`phone`),
    KEY `idx_users_created_at` (`created_at`),
    KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户基础信息';

-- ----------------------------------------------------------------------------
-- user_profiles
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `user_profiles` (
    `id`                         VARCHAR(36) NOT NULL COMMENT '主键UUID',
    `user_id`                    VARCHAR(36) NOT NULL COMMENT '用户ID',
    `age`                        INT         DEFAULT NULL,
    `gender`                     ENUM('male','female') DEFAULT NULL,
    `bio`                        TEXT        DEFAULT NULL,
    `total_conversations`        INT         NOT NULL DEFAULT 0,
    `total_evaluations`          INT         NOT NULL DEFAULT 0,
    `total_reports`              INT         NOT NULL DEFAULT 0,
    `total_study_minutes`        INT         NOT NULL DEFAULT 0,
    `average_evaluation_score`   DOUBLE      NOT NULL DEFAULT 0,
    `last_conversation_at`       TIMESTAMP   NULL DEFAULT NULL,
    `last_evaluation_at`         TIMESTAMP   NULL DEFAULT NULL,
    `created_at`                 TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`                 TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_profiles_user_id` (`user_id`),
    KEY `idx_user_profiles_last_conversation_at` (`last_conversation_at`),
    KEY `idx_user_profiles_last_evaluation_at` (`last_evaluation_at`),
    CONSTRAINT `fk_user_profiles_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户扩展信息';

-- ----------------------------------------------------------------------------
-- user_wechat_bindings
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `user_wechat_bindings` (
    `id`                 VARCHAR(36)  NOT NULL,
    `user_id`            VARCHAR(36)  NOT NULL,
    `open_id`            VARCHAR(100) NOT NULL,
    `wechat_nickname`    VARCHAR(100) DEFAULT NULL,
    `wechat_avatar_url`  VARCHAR(500) DEFAULT NULL,
    `created_at`         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_wechat_bindings_user_id` (`user_id`),
    UNIQUE KEY `uk_user_wechat_bindings_open_id` (`open_id`),
    CONSTRAINT `fk_user_wechat_bindings_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='微信绑定';

-- ----------------------------------------------------------------------------
-- user_login_logs
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `user_login_logs` (
    `id`          VARCHAR(36)  NOT NULL,
    `user_id`     VARCHAR(36)  NOT NULL,
    `login_type`  VARCHAR(20)  NOT NULL,
    `result`      VARCHAR(20)  NOT NULL,
    `fail_reason` VARCHAR(200) DEFAULT NULL,
    `ip`          VARCHAR(45)  NOT NULL,
    `platform`    VARCHAR(20)  DEFAULT NULL,
    `device_id`   VARCHAR(100) DEFAULT NULL,
    `user_agent`  VARCHAR(500) DEFAULT NULL,
    `created_at`  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_login_logs_user_id` (`user_id`),
    KEY `idx_user_login_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='登录日志';

-- ----------------------------------------------------------------------------
-- sessions（登录会话）
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `sessions` (
    `id`         VARCHAR(36)  NOT NULL,
    `user_id`    VARCHAR(36)  NOT NULL,
    `token`      VARCHAR(255) NOT NULL,
    `user_agent` VARCHAR(255) DEFAULT NULL,
    `ip`         VARCHAR(50)  DEFAULT NULL,
    `expires_at` TIMESTAMP    NOT NULL,
    `created_at` TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sessions_token` (`token`),
    KEY `idx_sessions_user_id` (`user_id`),
    KEY `idx_sessions_expires_at` (`expires_at`),
    CONSTRAINT `fk_sessions_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户登录会话';

-- ----------------------------------------------------------------------------
-- voice_conversations
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_conversations` (
    `id`                 VARCHAR(36) NOT NULL,
    `user_id`            VARCHAR(36) NOT NULL,
    `topic`              VARCHAR(200) NOT NULL,
    `difficulty_level`   ENUM('beginner','intermediate','advanced') NOT NULL DEFAULT 'beginner',
    `conversation_type`  ENUM('free_talk','question_answer') NOT NULL DEFAULT 'free_talk',
    `message_count`      INT         NOT NULL DEFAULT 0,
    `duration_seconds`   INT         NOT NULL DEFAULT 0,
    `status`             ENUM('active','completed','paused') NOT NULL DEFAULT 'active',
    `summary`            TEXT        DEFAULT NULL,
    `score`              INT         DEFAULT NULL,
    `feedback`           TEXT        DEFAULT NULL,
    `created_at`         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`         TIMESTAMP   NULL DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_voice_conversations_user_id` (`user_id`),
    KEY `idx_voice_conversations_difficulty_level` (`difficulty_level`),
    KEY `idx_voice_conversations_conversation_type` (`conversation_type`),
    KEY `idx_voice_conversations_status` (`status`),
    KEY `idx_voice_conversations_created_at` (`created_at`),
    KEY `idx_voice_conversations_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_voice_conversations_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='语音对话会话';

-- ----------------------------------------------------------------------------
-- conversation_messages
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `conversation_messages` (
    `id`              VARCHAR(36) NOT NULL,
    `conversation_id` VARCHAR(36) NOT NULL,
    `sender_type`     ENUM('user','ai') NOT NULL,
    `turn_id`         INT         NOT NULL,
    `message_text`    LONGTEXT    NOT NULL,
    `audio_url`       VARCHAR(500) DEFAULT NULL,
    `audio_duration`  INT         DEFAULT NULL,
    `sequence_number` INT         NOT NULL,
    `latency_ms`      INT         DEFAULT NULL,
    `created_at`      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_conversation_messages_conversation_id` (`conversation_id`),
    KEY `idx_conversation_messages_sender_type` (`sender_type`),
    KEY `idx_conversation_messages_turn_id` (`turn_id`),
    KEY `idx_conversation_messages_created_at` (`created_at`),
    KEY `idx_conversation_messages_turn` (`conversation_id`, `turn_id`),
    CONSTRAINT `fk_conversation_messages_conversation_id` FOREIGN KEY (`conversation_id`) REFERENCES `voice_conversations` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话消息';

-- ----------------------------------------------------------------------------
-- pronunciation_evaluations
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `pronunciation_evaluations` (
    `id`                       VARCHAR(36) NOT NULL,
    `user_id`                  VARCHAR(36) NOT NULL,
    `target_text`              VARCHAR(500) NOT NULL,
    `recognized_text`          VARCHAR(500) DEFAULT NULL,
    `audio_url`                VARCHAR(500) DEFAULT NULL,
    `audio_duration`           INT         DEFAULT NULL,
    `overall_score`            INT         NOT NULL DEFAULT 0,
    `accuracy_score`           INT         NOT NULL DEFAULT 0,
    `fluency_score`            INT         NOT NULL DEFAULT 0,
    `integrity_score`          INT         NOT NULL DEFAULT 0,
    `feedback_level`           ENUM('S','A','B','C') NOT NULL DEFAULT 'C',
    `feedback_text`            TEXT        DEFAULT NULL,
    `feedback_audio_url`       VARCHAR(500) DEFAULT NULL,
    `problem_words`            JSON        DEFAULT NULL,
    `problem_word_audio_urls`  JSON        DEFAULT NULL,
    `demo_sentence_audio_url`  VARCHAR(500) DEFAULT NULL,
    `difficulty_level`         ENUM('beginner','intermediate','advanced') NOT NULL DEFAULT 'beginner',
    `assessment_sid`           VARCHAR(100) DEFAULT NULL,
    `speech_assessment_json`   LONGTEXT    DEFAULT NULL,
    `status`                   ENUM('pending','processing','completed','failed') NOT NULL DEFAULT 'pending',
    `error_message`            VARCHAR(500) DEFAULT NULL,
    `created_at`               TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`               TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_pronunciation_evaluations_user_id` (`user_id`),
    KEY `idx_pronunciation_evaluations_feedback_level` (`feedback_level`),
    KEY `idx_pronunciation_evaluations_status` (`status`),
    KEY `idx_pronunciation_evaluations_created_at` (`created_at`),
    CONSTRAINT `fk_pronunciation_evaluations_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发音评测';

-- ----------------------------------------------------------------------------
-- learning_reports
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `learning_reports` (
    `id`                           VARCHAR(36) NOT NULL,
    `user_id`                      VARCHAR(36) NOT NULL,
    `report_type`                  ENUM('weekly','monthly','custom') NOT NULL DEFAULT 'weekly',
    `period_start_date`            DATE        NOT NULL,
    `period_end_date`              DATE        NOT NULL,
    `task_id`                      VARCHAR(100) NOT NULL DEFAULT '',
    `total_conversations`          INT         NOT NULL DEFAULT 0,
    `total_evaluations`            INT         NOT NULL DEFAULT 0,
    `total_study_minutes`          INT         NOT NULL DEFAULT 0,
    `average_conversation_score`   DOUBLE      NOT NULL DEFAULT 0,
    `average_evaluation_score`   DOUBLE      NOT NULL DEFAULT 0,
    `average_accuracy_score`       DOUBLE      NOT NULL DEFAULT 0,
    `average_fluency_score`        DOUBLE      NOT NULL DEFAULT 0,
    `average_integrity_score`      DOUBLE      NOT NULL DEFAULT 0,
    `s_level_count`                INT         NOT NULL DEFAULT 0,
    `a_level_count`                INT         NOT NULL DEFAULT 0,
    `b_level_count`                INT         NOT NULL DEFAULT 0,
    `c_level_count`                INT         NOT NULL DEFAULT 0,
    `improvement_rate`             DOUBLE      NOT NULL DEFAULT 0,
    `most_practiced_topics`        JSON        DEFAULT NULL,
    `problem_words`                JSON        DEFAULT NULL,
    `strengths`                    JSON        DEFAULT NULL,
    `weaknesses`                   JSON        DEFAULT NULL,
    `recommendations`              LONGTEXT    DEFAULT NULL,
    `ai_model`                     VARCHAR(100) DEFAULT NULL,
    `created_at`                   TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`                   TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `content`                      LONGTEXT    DEFAULT NULL,
    `is_latest`                    TINYINT(1)  NOT NULL DEFAULT 1,
    PRIMARY KEY (`id`),
    KEY `idx_learning_reports_user_id` (`user_id`),
    KEY `idx_learning_reports_report_type` (`report_type`),
    KEY `idx_learning_reports_period_start_date` (`period_start_date`),
    KEY `idx_learning_reports_created_at` (`created_at`),
    KEY `idx_learning_reports_is_latest` (`is_latest`),
    KEY `idx_lr_user_latest` (`user_id`, `is_latest`),
    CONSTRAINT `fk_learning_reports_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='学习报告';

-- ----------------------------------------------------------------------------
-- system_settings
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `system_settings` (
    `id`           VARCHAR(36) NOT NULL,
    `config_key`   VARCHAR(100) NOT NULL,
    `config_value` LONGTEXT    NOT NULL,
    `config_type`  ENUM('string','int','float','json','boolean') NOT NULL,
    `description`  VARCHAR(500) DEFAULT NULL,
    `is_editable`  TINYINT(1)  NOT NULL DEFAULT 1,
    `created_at`   TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_system_settings_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置';

-- ----------------------------------------------------------------------------
-- pronunciation_sessions
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `pronunciation_sessions` (
    `id`             VARCHAR(36) NOT NULL,
    `user_id`        VARCHAR(36) NOT NULL,
    `unit_id`        VARCHAR(50) NOT NULL,
    `current_index`  INT         NOT NULL DEFAULT 1,
    `status`         VARCHAR(20) NOT NULL DEFAULT 'ongoing',
    `created_at`     TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`     TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_pronunciation_sessions_user_id` (`user_id`),
    KEY `idx_pronunciation_sessions_status` (`status`),
    CONSTRAINT `fk_pronunciation_sessions_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发音练习会话';

-- ----------------------------------------------------------------------------
-- pronunciation_records
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `pronunciation_records` (
    `id`              VARCHAR(36)   NOT NULL,
    `session_id`      VARCHAR(36)   NOT NULL,
    `user_id`         VARCHAR(36)   NOT NULL,
    `unit_id`         VARCHAR(50)   NOT NULL,
    `item_id`         INT           NOT NULL,
    `content`         VARCHAR(500)  NOT NULL,
    `practice_type`   VARCHAR(20)   NOT NULL,
    `raw_score`       DECIMAL(3,1)  NOT NULL,
    `stars`           INT           NOT NULL,
    `problem_words`   JSON          DEFAULT NULL,
    `user_audio_url`  VARCHAR(500) DEFAULT NULL,
    `ai_encourage`    TEXT          DEFAULT NULL,
    `ai_problem_tip`  TEXT          DEFAULT NULL,
    `ai_suggestion`   TEXT          DEFAULT NULL,
    `ai_audio_url`    VARCHAR(500) DEFAULT NULL,
    `is_rejected`     TINYINT(1)    NOT NULL DEFAULT 0,
    `accuracy_score`  DECIMAL(3,1)  NOT NULL DEFAULT 0,
    `fluency`         DECIMAL(3,1)  NOT NULL DEFAULT 0,
    `integrity`       DECIMAL(3,1)  NOT NULL DEFAULT 0,
    `standard_score`  DECIMAL(3,1)  NOT NULL DEFAULT 0,
    `created_at`      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_pronunciation_records_session_id` (`session_id`),
    KEY `idx_pronunciation_records_user_id` (`user_id`),
    KEY `idx_pronunciation_records_is_rejected` (`is_rejected`),
    CONSTRAINT `fk_pronunciation_records_session_id` FOREIGN KEY (`session_id`) REFERENCES `pronunciation_sessions` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_pronunciation_records_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发音练习单次记录';

-- ----------------------------------------------------------------------------
-- scene_sessions
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `scene_sessions` (
    `id`           VARCHAR(36) NOT NULL,
    `user_id`      VARCHAR(36) NOT NULL,
    `scene_id`     VARCHAR(50) NOT NULL,
    `current_step` INT         NOT NULL DEFAULT 1,
    `status`       VARCHAR(20) NOT NULL DEFAULT 'active',
    `created_at`   TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_scene_sessions_user_id` (`user_id`),
    KEY `idx_scene_sessions_status` (`status`),
    CONSTRAINT `fk_scene_sessions_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='场景引导会话';

-- ----------------------------------------------------------------------------
-- scene_messages
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `scene_messages` (
    `id`             VARCHAR(36)  NOT NULL,
    `session_id`     VARCHAR(36)  NOT NULL,
    `user_id`        VARCHAR(36)  NOT NULL,
    `scene_id`       VARCHAR(50)  NOT NULL,
    `step_id`        INT          NOT NULL,
    `attempt`        INT          NOT NULL DEFAULT 1,
    `user_text`      TEXT         DEFAULT NULL,
    `user_audio_url` VARCHAR(500) DEFAULT NULL,
    `match_result`   VARCHAR(20)  NOT NULL,
    `ai_reply_text`  TEXT         DEFAULT NULL,
    `ai_audio_url`   VARCHAR(500) DEFAULT NULL,
    `llm_status`     VARCHAR(10)  DEFAULT NULL,
    `step_advanced`  TINYINT(1)   DEFAULT 0,
    `created_at`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_scene_messages_session_id` (`session_id`),
    KEY `idx_scene_messages_user_scene_step` (`user_id`, `scene_id`, `step_id`),
    CONSTRAINT `fk_scene_messages_session_id` FOREIGN KEY (`session_id`) REFERENCES `scene_sessions` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_scene_messages_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='场景引导消息';
