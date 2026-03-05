-- 002_cleanup_old_columns.sql
-- 清理 v1.0 遗留的旧列（v2.0 不再需要 password_hash, email 等字段）

ALTER TABLE users DROP COLUMN password_hash;
