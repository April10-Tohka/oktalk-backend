-- 智能周报同步：pronunciation_records 细分指标 + learning_reports 内容字段
-- MySQL 8+：可逐条执行；若列已存在会报错，需 DBA 按需跳过

ALTER TABLE pronunciation_records
    ADD COLUMN is_rejected     TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '科大讯飞拒识',
    ADD COLUMN accuracy_score  DECIMAL(3,1) NOT NULL DEFAULT 0 COMMENT '准确度 0-5',
    ADD COLUMN fluency         DECIMAL(3,1) NOT NULL DEFAULT 0 COMMENT '流利度 0-5',
    ADD COLUMN integrity       DECIMAL(3,1) NOT NULL DEFAULT 0 COMMENT '完整度 0-5',
    ADD COLUMN standard_score    DECIMAL(3,1) NOT NULL DEFAULT 0 COMMENT '标准度 0-5';

ALTER TABLE learning_reports
    ADD COLUMN content   LONGTEXT     NULL COMMENT '完整周报 JSON',
    ADD COLUMN is_latest TINYINT(1)   NOT NULL DEFAULT 1 COMMENT '是否该周期最新';

CREATE INDEX idx_lr_user_latest ON learning_reports (user_id, is_latest);
