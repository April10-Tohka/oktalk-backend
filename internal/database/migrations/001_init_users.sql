-- 迁移文件: 001_init_users.sql
-- 用途: 创建用户相关表

-- +migrate Up
-- 1. 用户基础信息表
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

-- 2. 用户扩展信息表
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

-- +migrate Down
DROP TABLE IF EXISTS user_profiles;
DROP TABLE IF EXISTS users;
