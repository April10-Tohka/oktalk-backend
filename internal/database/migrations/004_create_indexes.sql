-- 004_create_indexes.sql
-- 创建额外索引和优化

-- 用户表额外索引
CREATE INDEX IF NOT EXISTS `idx_users_level` ON `users` (`level`);

-- 评测表复合索引
CREATE INDEX IF NOT EXISTS `idx_evaluations_user_score` ON `evaluations` (`user_id`, `score`);
CREATE INDEX IF NOT EXISTS `idx_evaluations_user_level` ON `evaluations` (`user_id`, `level`);

-- 文本表全文索引（用于搜索）
-- ALTER TABLE `texts` ADD FULLTEXT INDEX `ft_content` (`content`);

-- 报告表复合索引
CREATE INDEX IF NOT EXISTS `idx_reports_user_dates` ON `reports` (`user_id`, `start_date`, `end_date`);

-- 添加外键约束（可选，根据需要启用）
-- ALTER TABLE `evaluations` ADD CONSTRAINT `fk_evaluations_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `evaluations` ADD CONSTRAINT `fk_evaluations_text` FOREIGN KEY (`text_id`) REFERENCES `texts` (`id`) ON DELETE SET NULL;
-- ALTER TABLE `feedbacks` ADD CONSTRAINT `fk_feedbacks_evaluation` FOREIGN KEY (`evaluation_id`) REFERENCES `evaluations` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `reports` ADD CONSTRAINT `fk_reports_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `sessions` ADD CONSTRAINT `fk_sessions_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `chat_sessions` ADD CONSTRAINT `fk_chat_sessions_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `chat_messages` ADD CONSTRAINT `fk_chat_messages_session` FOREIGN KEY (`session_id`) REFERENCES `chat_sessions` (`id`) ON DELETE CASCADE;
