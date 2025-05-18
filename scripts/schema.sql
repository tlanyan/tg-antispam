-- TG-AntiSpam Database Schema

-- Create GroupInfo table
CREATE TABLE IF NOT EXISTS `group_info` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `group_id` bigint(20) NOT NULL,
  `group_name` varchar(255) DEFAULT NULL,
  `group_link` varchar(255) DEFAULT NULL,
  `admin_id` bigint(20) DEFAULT NULL,
  `language` varchar(8) NOT NULL DEFAULT 'zh_CN',
  `is_admin` tinyint(1) DEFAULT 0,
  `enable_notification` tinyint(1) DEFAULT 1,
  `ban_premium` tinyint(1) DEFAULT 1,
  `ban_random_username` tinyint(1) DEFAULT 1,
  `ban_emoji_name` tinyint(1) DEFAULT 1,
  `ban_bio_link` tinyint(1) DEFAULT 1,
  `enable_cas` tinyint(1) DEFAULT 1,
  `enable_aicheck` tinyint(1) DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_group_id` (`group_id`, `admin_id`),
  KEY `idx_admin_id` (`admin_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create BanRecord table
CREATE TABLE IF NOT EXISTS `ban_record` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `group_id` bigint(20) NOT NULL,
  `user_id` bigint(20) NOT NULL,
  `reason` text NOT NULL,
  `is_unbanned` tinyint(1) DEFAULT 0,
  `unbanned_by` varchar(255) DEFAULT '',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create PendingMessage table
CREATE TABLE IF NOT EXISTS `pending_message` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `chat_id` bigint(20) NOT NULL,
  `message_id` int(11) NOT NULL,
  `delete_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_chat_message` (`chat_id`, `message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;