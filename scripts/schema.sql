-- TG-AntiSpam Database Schema

-- Create GroupInfo table
CREATE TABLE IF NOT EXISTS `group_info` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `group_id` bigint(20) NOT NULL,
  `group_name` varchar(255) DEFAULT NULL,
  `group_link` varchar(255) DEFAULT NULL,
  `admin_id` bigint(20) DEFAULT NULL,
  `language` varchar(8) NOT NULL DEFAULT 'zh_CN',  -- 新增字段
  `is_admin` tinyint(1) DEFAULT 0,
  `enable_notification` tinyint(1) DEFAULT 1,
  `ban_premium` tinyint(1) DEFAULT 1,
  `ban_random_username` tinyint(1) DEFAULT 1,
  `ban_emoji_name` tinyint(1) DEFAULT 1,
  `ban_bio_link` tinyint(1) DEFAULT 1,
  `enable_cas` tinyint(1) DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_group_id` (`group_id`, `admin_id`),
  KEY `idx_admin_id` (`admin_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;