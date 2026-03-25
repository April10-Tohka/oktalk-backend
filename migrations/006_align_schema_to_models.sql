-- ============================================================================
-- 006_align_schema_to_models.sql
-- 目的：将数据库表结构对齐至 internal/model 中的 GORM 模型定义
-- 前置：已执行 001～005（或具备等价的 users / voice_conversations / pronunciation_* 等表）
-- 数据库：MySQL 8.0.12+（需支持 ADD COLUMN IF NOT EXISTS、DROP COLUMN IF EXISTS）
-- ============================================================================

SET NAMES utf8mb4;

-- ----------------------------------------------------------------------------
-- 1. sessions（用户登录会话）— 模型：model.Session，表名 sessions
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `sessions` (
    `id`         VARCHAR(36)  NOT NULL COMMENT '会话ID',
    `user_id`    VARCHAR(36)  NOT NULL COMMENT '用户ID',
    `token`      VARCHAR(255) NOT NULL COMMENT '会话令牌',
    `user_agent` VARCHAR(255) DEFAULT NULL COMMENT 'User-Agent',
    `ip`         VARCHAR(50)  DEFAULT NULL COMMENT '客户端IP',
    `expires_at` TIMESTAMP    NOT NULL COMMENT '过期时间',
    `created_at` TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sessions_token` (`token`),
    KEY `idx_sessions_user_id` (`user_id`),
    KEY `idx_sessions_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户登录会话';

-- ----------------------------------------------------------------------------
-- 2. conversation_messages — 模型：ConversationMessage.TurnID
-- ----------------------------------------------------------------------------
ALTER TABLE `conversation_messages`
    ADD COLUMN IF NOT EXISTS `turn_id` INT NOT NULL DEFAULT 1 COMMENT '轮次ID（同轮 user/ai 相同）' AFTER `sender_type`;

UPDATE `conversation_messages`
SET `turn_id` = GREATEST(1, (`sequence_number` + 1) DIV 2)
WHERE `turn_id` = 1;

SET @idx_cm_turn := (
    SELECT COUNT(*) FROM information_schema.statistics
     WHERE table_schema = DATABASE() AND table_name = 'conversation_messages' AND index_name = 'idx_conversation_messages_turn'
);
SET @sql_idx_cm := IF(@idx_cm_turn = 0,
    'CREATE INDEX `idx_conversation_messages_turn` ON `conversation_messages` (`conversation_id`, `turn_id`)',
    'SELECT 1');
PREPARE stmt_idx_cm FROM @sql_idx_cm;
EXECUTE stmt_idx_cm;
DEALLOCATE PREPARE stmt_idx_cm;

-- ----------------------------------------------------------------------------
-- 3. learning_reports — 模型：LearningReport.TaskID / Content / IsLatest
-- ----------------------------------------------------------------------------
ALTER TABLE `learning_reports`
    ADD COLUMN IF NOT EXISTS `task_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '异步任务ID（幂等）' AFTER `period_end_date`;

ALTER TABLE `learning_reports`
    ADD COLUMN IF NOT EXISTS `content` LONGTEXT COMMENT '完整报告JSON' AFTER `task_id`;

ALTER TABLE `learning_reports`
    ADD COLUMN IF NOT EXISTS `is_latest` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否该周期最新' AFTER `content`;

SET @idx_lr := (
    SELECT COUNT(*) FROM information_schema.statistics
     WHERE table_schema = DATABASE() AND table_name = 'learning_reports' AND index_name = 'idx_lr_user_latest'
);
SET @sql_idx_lr := IF(@idx_lr = 0,
    'CREATE INDEX `idx_lr_user_latest` ON `learning_reports` (`user_id`, `is_latest`)',
    'SELECT 1');
PREPARE stmt_idx_lr FROM @sql_idx_lr;
EXECUTE stmt_idx_lr;
DEALLOCATE PREPARE stmt_idx_lr;

-- ----------------------------------------------------------------------------
-- 4. pronunciation_records — 模型：PronunciationRecord
--    从 004+005（raw_score / fluency / integrity / phone_score）迁移到 total_score 与各 *_score 列
-- ----------------------------------------------------------------------------
DELIMITER $$

DROP PROCEDURE IF EXISTS `sp_align_pronunciation_records`$$

CREATE PROCEDURE `sp_align_pronunciation_records`()
BEGIN
    DECLARE v_raw   INT DEFAULT 0;
    DECLARE v_total INT DEFAULT 0;
    DECLARE v_acc   INT DEFAULT 0;
    DECLARE v_flu   INT DEFAULT 0;
    DECLARE v_int   INT DEFAULT 0;
    DECLARE v_phone INT DEFAULT 0;

    SELECT COUNT(*) INTO v_raw FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'pronunciation_records' AND COLUMN_NAME = 'raw_score';

    SELECT COUNT(*) INTO v_total FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'pronunciation_records' AND COLUMN_NAME = 'total_score';

    SELECT COUNT(*) INTO v_acc FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'pronunciation_records' AND COLUMN_NAME = 'accuracy_score';

    SELECT COUNT(*) INTO v_flu FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'pronunciation_records' AND COLUMN_NAME = 'fluency';

    SELECT COUNT(*) INTO v_int FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'pronunciation_records' AND COLUMN_NAME = 'integrity';

    SELECT COUNT(*) INTO v_phone FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'pronunciation_records' AND COLUMN_NAME = 'phone_score';

    IF v_total = 0 THEN
        ALTER TABLE `pronunciation_records`
            ADD COLUMN `total_score` DECIMAL(4,2) NULL COMMENT '综合分' AFTER `practice_type`;
    END IF;

    IF v_acc = 0 THEN
        ALTER TABLE `pronunciation_records`
            ADD COLUMN `accuracy_score` DECIMAL(4,2) NOT NULL DEFAULT 0 COMMENT '准确度' AFTER `ai_audio_url`,
            ADD COLUMN `fluency_score` DECIMAL(4,2) NOT NULL DEFAULT 0 COMMENT '流利度' AFTER `accuracy_score`,
            ADD COLUMN `integrity_score` DECIMAL(4,2) NOT NULL DEFAULT 0 COMMENT '完整度' AFTER `fluency_score`,
            ADD COLUMN `standard_score` DECIMAL(4,2) NOT NULL DEFAULT 0 COMMENT '标准度' AFTER `integrity_score`,
            ADD COLUMN `except_info` INT NOT NULL DEFAULT 0 COMMENT '异常信息' AFTER `standard_score`,
            ADD COLUMN `is_rejected` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否被拒绝' AFTER `except_info`,
            ADD COLUMN `raw_xml` LONGTEXT NULL COMMENT '原始评测XML' AFTER `is_rejected`;
    END IF;

    IF v_raw > 0 THEN
        SET @q := 'UPDATE `pronunciation_records` SET `total_score` = `raw_score` WHERE `total_score` IS NULL';
        PREPARE ps FROM @q;
        EXECUTE ps;
        DEALLOCATE PREPARE ps;
    END IF;

    IF v_flu > 0 THEN
        SET @q := 'UPDATE `pronunciation_records` SET `fluency_score` = COALESCE(`fluency`, 0)';
        PREPARE ps FROM @q;
        EXECUTE ps;
        DEALLOCATE PREPARE ps;
    END IF;

    IF v_int > 0 THEN
        SET @q := 'UPDATE `pronunciation_records` SET `integrity_score` = COALESCE(`integrity`, 0)';
        PREPARE ps FROM @q;
        EXECUTE ps;
        DEALLOCATE PREPARE ps;
    END IF;

    IF v_phone > 0 THEN
        SET @q := 'UPDATE `pronunciation_records` SET `standard_score` = COALESCE(`phone_score`, 0)';
        PREPARE ps FROM @q;
        EXECUTE ps;
        DEALLOCATE PREPARE ps;
    END IF;

    IF EXISTS (SELECT 1 FROM `pronunciation_records` WHERE `total_score` IS NULL LIMIT 1) THEN
        UPDATE `pronunciation_records` SET `total_score` = 0 WHERE `total_score` IS NULL;
    END IF;

    ALTER TABLE `pronunciation_records`
        MODIFY COLUMN `total_score` DECIMAL(4,2) NOT NULL COMMENT '综合分';

    IF v_raw > 0 THEN
        ALTER TABLE `pronunciation_records` DROP COLUMN IF EXISTS `raw_score`;
    END IF;
    IF v_flu > 0 THEN
        ALTER TABLE `pronunciation_records` DROP COLUMN IF EXISTS `fluency`;
    END IF;
    IF v_int > 0 THEN
        ALTER TABLE `pronunciation_records` DROP COLUMN IF EXISTS `integrity`;
    END IF;
    IF v_phone > 0 THEN
        ALTER TABLE `pronunciation_records` DROP COLUMN IF EXISTS `phone_score`;
    END IF;
END$$

DELIMITER ;

CALL `sp_align_pronunciation_records`();
DROP PROCEDURE IF EXISTS `sp_align_pronunciation_records`;

-- user_wechat_bindings.union_id：迁移 001 已包含；模型未映射该字段时 GORM 会忽略，无需删除。
