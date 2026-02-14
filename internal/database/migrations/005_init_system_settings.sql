-- 迁移文件: 005_init_system_settings.sql
-- 用途: 创建系统配置表并插入初始数据

-- +migrate Up
-- 系统配置表
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

-- 插入初始系统配置
INSERT INTO system_settings (id, config_key, config_value, config_type, description) VALUES
('set_001', 'feedback_s_level_min_score', '90', 'int', 'S级反馈最低分数'),
('set_002', 'feedback_a_level_min_score', '70', 'int', 'A级反馈最低分数'),
('set_003', 'feedback_b_level_min_score', '50', 'int', 'B级反馈最低分数'),
('set_004', 'feedback_c_level_min_score', '0', 'int', 'C级反馈最低分数'),
('set_005', 'forbidden_words', '["bad","wrong","terrible","fail","mistake","error","poor","awful"]', 'json', '禁用词列表'),
('set_006', 'report_generation_schedule', '0 0 * * 1', 'string', '报告生成时间表(cron)');

-- +migrate Down
DROP TABLE IF EXISTS system_settings;
