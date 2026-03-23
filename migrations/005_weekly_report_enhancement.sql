-- pronunciation_records：细分指标
ALTER TABLE pronunciation_records
    ADD COLUMN fluency     DECIMAL(3,1) NOT NULL DEFAULT 0 COMMENT '流利度 0-5',
    ADD COLUMN integrity   DECIMAL(3,1) NOT NULL DEFAULT 0 COMMENT '完整度 0-5',
    ADD COLUMN phone_score DECIMAL(3,1) NOT NULL DEFAULT 0 COMMENT '标准度 0-5';

-- learning_reports：周报 JSON 与最新标记（MySQL 8+ 可手动删重复列后执行）
ALTER TABLE learning_reports
    ADD COLUMN content   LONGTEXT COMMENT '完整报告JSON内容',
    ADD COLUMN is_latest BOOLEAN NOT NULL DEFAULT TRUE COMMENT '是否为该周期最新报告';

CREATE INDEX idx_lr_user_latest ON learning_reports(user_id, is_latest);
