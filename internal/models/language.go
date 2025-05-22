package models

// Language constants
const (
	LangSimplifiedChinese  = "zh_CN"
	LangTraditionalChinese = "zh_TW"
	LangEnglish            = "en"
)

// Translation is a map of message keys to translated text
type Translation map[string]string

// Translations stores all language translations
var Translations = map[string]Translation{
	LangSimplifiedChinese: {
		"help_title":                      "TG-AntiSpam Bot 帮助",
		"help_description":                "此机器人可以帮助保护您的群组免受垃圾消息和恶意用户的骚扰。",
		"general_commands":                "通用命令:",
		"settings_commands":               "群组管理命令:",
		"help_cmd_help":                   "/help - 显示此帮助消息",
		"help_cmd_settings":               "/settings - 显示群组设置",
		"help_cmd_toggle_premium":         "/toggle_premium - 切换默认禁止Premium用户",
		"help_cmd_toggle_cas":             "/toggle_cas - 切换启用CAS验证",
		"help_cmd_toggle_notifications":   "/toggle_notifications - 切换发送管理员通知",
		"help_cmd_language":               "/language - 设置机器人语言",
		"help_cmd_language_group":         "/language_group - 设置群组语言",
		"help_cmd_toggle_random_username": "/toggle_random_username - 切换默认禁止随机用户名用户",
		"help_cmd_toggle_emoji_name":      "/toggle_emoji_name - 切换默认禁止姓名表情符号用户",
		"help_cmd_toggle_bio_link":        "/toggle_bio_link - 切换默认禁止个人简介可疑链接用户",
		"help_cmd_self_unban":             "/self_unban - 自助解封",
		"help_note":                       "注意: 只有群组管理员才能更改设置。",

		// Command descriptions for Telegram command menu
		"cmd_desc_help":                   "显示帮助信息",
		"cmd_desc_settings":               "查看/修改群组设置",
		"cmd_desc_toggle_premium":         "切换默认封禁Premium用户",
		"cmd_desc_toggle_cas":             "切换默认启用CAS验证",
		"cmd_desc_toggle_notifications":   "切换通知管理员",
		"cmd_desc_language":               "设置机器人语言",
		"cmd_desc_toggle_random_username": "切换默认封禁随机用户名用户",
		"cmd_desc_toggle_emoji_name":      "切换默认封禁姓名表情符号用户",
		"cmd_desc_toggle_bio_link":        "切换默认封禁个人简介可疑链接用户",
		"cmd_desc_self_unban":             "自助解封",

		"user_not_admin":               "您不是群组管理员，无法使用该指令",
		"empty_group_list":             "群组记录为空，请将机器人添加到您管理的群组中并设置为管理员",
		"get_ban_records_error":        "获取封禁记录失败，请稍后再试",
		"no_ban_records":               "您没有待解封的记录",
		"cannot_unban_for_other_users": "不能为其他用户解除限制",
		"select_group_to_unban":        "请选择要解除限制的群组：",

		"settings_title":           "%s 的设置",
		"settings_bot_status":      "机器人状态:",
		"settings_active":          "✅ 已激活",
		"settings_current":         "当前设置:",
		"settings_ban_premium":     "- 默认封禁Premium用户: %s",
		"settings_cas":             "- CAS验证: %s",
		"settings_random_username": "- 随机用户名封禁: %s",
		"settings_emoji_name":      "- 姓名表情符号封禁: %s",
		"settings_bio_link":        "- 个人简介可疑链接封禁: %s",
		"settings_notifications":   "- 通知管理员: %s",
		"settings_language":        "- 语言: %s",

		"enabled":  "✅ 启用",
		"disabled": "❌ 禁用",

		"language_select":  "请选择语言:",
		"language_updated": "机器人语言已更新为: %s",

		"warning_title":                  "⚠️ <b>安全提醒</b> [%s]",
		"warning_restricted":             "用户 %s 已被限制发送消息和媒体的权限",
		"warning_reason":                 "<b>原因</b>: %s",
		"warning_unban_button":           "解除限制",
		"warning_self_unban_button":      "自行解除限制",
		"warning_self_unban_instruction": "%s 由于安全原因你已被限制发言，如需解除限制请联系管理员或者点击下面的按钮自助解封。",
		"warning_user_unbanned":          "✅ 用户已成功解封",
		"warning_unbanned_message":       "✅ <b>用户已解封</b>\n用户 %s 已被解除限制，现在可以正常发言。",
		"ban_user":                       "封禁用户",
		"user_banned":                    "✅ 用户已被成功限制",
		"warning_banned_message":         "⛔ <b>用户已被限制</b>\n用户 %s 已被限制发送消息和媒体的权限。",
		"math_verification":              "请计算以下算术题：\n%d %s %d = ?",
		"math_verification_success":      "✅ 验证成功！您已被解除限制。",
		"math_verification_failed":       "❌ 验证失败，请重试。",

		// New translations for private chat mode
		"please_use_private_chat": "请在私聊中使用此命令，以管理您的群组设置",
		"select_group":            "请选择您要管理的群组:",
		"enter_group_id":          "请输入群组ID，格式为纯数字（例如：-1001234567890）",
		"action":                  "操作",
		"no_admin_groups":         "您不是任何群组的管理员，或者机器人未添加到您管理的群组中",
		"group_not_found":         "找不到此群组，或机器人不是该群组的成员",
		"invalid_group_id":        "无效的群组ID，请输入正确的数字格式",
		"invalid_format":          "无效的格式，请重试",
		"unknown_action":          "未知操作，请重试",
		"action_completed":        "操作已完成",

		// 用户被限制的原因
		"reason_premium_user":    "Premium用户",
		"reason_random_username": "随机用户名",
		"reason_emoji_name":      "姓名含有表情符号",
		"reason_bio_link":        "个人简介包含可疑链接",
		"reason_cas_blacklisted": "用户在 CAS 黑名单中",
		"reason_ai_spam":         "被 AI 判定为垃圾消息",
		"reason_join_group":      "加入群组",

		// Toggle action response messages
		"premium_ban_enabled":          "已启用默认封禁Premium用户",
		"premium_ban_disabled":         "已禁用默认封禁Premium用户",
		"cas_enabled":                  "已启用CAS验证",
		"cas_disabled":                 "已禁用CAS验证",
		"random_username_ban_enabled":  "已启用随机用户名封禁",
		"random_username_ban_disabled": "已禁用随机用户名封禁",
		"emoji_name_ban_enabled":       "已启用姓名表情符号封禁",
		"emoji_name_ban_disabled":      "已禁用姓名表情符号封禁",
		"bio_link_ban_enabled":         "已启用个人简介可疑链接封禁",
		"bio_link_ban_disabled":        "已禁用个人简介可疑链接封禁",
		"notifications_enabled":        "已启用管理员通知",
		"notifications_disabled":       "已禁用管理员通知",

		// Other messages for action responses
		"toggle_premium":         "切换Premium用户封禁",
		"toggle_cas":             "切换CAS验证",
		"toggle_random_username": "切换随机用户名封禁",
		"toggle_emoji_name":      "切换姓名表情符号封禁",
		"toggle_bio_link":        "切换个人简介可疑链接封禁",
		"toggle_notifications":   "切换管理员通知",
		"change_language":        "更改语言",

		"select_language": "请选择机器人使用的语言:",
	},

	LangTraditionalChinese: {
		"help_title":                      "TG-AntiSpam Bot 幫助",
		"help_description":                "此機器人可以幫助保護您的群組免受垃圾消息和惡意用戶的騷擾。",
		"general_commands":                "通用命令:",
		"settings_commands":               "群組管理命令:",
		"help_cmd_help":                   "/help - 顯示此幫助消息",
		"help_cmd_settings":               "/settings - 顯示群組設置",
		"help_cmd_toggle_premium":         "/toggle_premium - 切換默認禁止Premium用戶",
		"help_cmd_toggle_cas":             "/toggle_cas - 切換默認啟用CAS驗證",
		"help_cmd_toggle_random_username": "/toggle_random_username - 切換默認禁止隨機用戶名用户",
		"help_cmd_toggle_emoji_name":      "/toggle_emoji_name - 切換默認禁止名字包含表情符號用户",
		"help_cmd_toggle_bio_link":        "/toggle_bio_link - 切換默認禁止個人簡介包含可疑連結用户",
		"help_cmd_toggle_notifications":   "/toggle_notifications - 切換發送管理員通知",
		"help_cmd_self_unban":             "/self_unban - 自助解封",
		"help_cmd_language":               "/language - 設置機器人語言",
		"help_cmd_language_group":         "/language_group - 設置群組語言",
		"help_note":                       "注意: 只有群組管理員才能更改設置。",

		// Command descriptions for Telegram command menu
		"cmd_desc_help":                   "顯示幫助信息",
		"cmd_desc_settings":               "查看/修改群組設置",
		"cmd_desc_toggle_premium":         "切換默認封禁Premium用戶",
		"cmd_desc_toggle_cas":             "切換默認啟用CAS驗證",
		"cmd_desc_toggle_random_username": "切換默認封禁隨機用戶名用户",
		"cmd_desc_toggle_emoji_name":      "切換姓名表情符號封禁",
		"cmd_desc_toggle_bio_link":        "切換個人簡介可疑連結封禁",
		"cmd_desc_toggle_notifications":   "切換管理員通知",
		"cmd_desc_language":               "設置機器人語言",
		"cmd_desc_self_unban":             "自助解封",

		"user_not_admin":               "您不是群組管理員，無法使用該指令",
		"empty_group_list":             "群組記錄為空，請將機器人添加到您管理的群組中並設置為管理員",
		"get_ban_records_error":        "獲取封禁記錄失敗，請稍後再試",
		"no_ban_records":               "您沒有待解封的記錄",
		"cannot_unban_for_other_users": "不能為其他用戶解除限制",
		"select_group_to_unban":        "請選擇要解除限制的群組：",

		"settings_title":           "%s 的設置",
		"settings_bot_status":      "機器人狀態:",
		"settings_active":          "✅ 已激活",
		"settings_current":         "當前設置:",
		"settings_ban_premium":     "- 默認封禁Premium用戶: %s",
		"settings_cas":             "- CAS驗證: %s",
		"settings_random_username": "- 隨機用戶名封禁: %s",
		"settings_emoji_name":      "- 姓名表情符號封禁: %s",
		"settings_bio_link":        "- 個人簡介可疑連結封禁: %s",
		"settings_notifications":   "- 管理員通知: %s",
		"settings_language":        "- 語言: %s",

		"enabled":  "✅ 啟用",
		"disabled": "❌ 禁用",

		"setting_premium":         "封禁Premium用戶",
		"setting_cas":             "CAS驗證",
		"setting_random_username": "封禁隨機用戶名用戶",
		"setting_emoji_name":      "封禁名字包含表情符號用戶",
		"setting_bio_link":        "封禁個人簡介包含可疑連結用戶",
		"setting_notifications":   "通知管理員",
		"setting_language":        "語言",

		"language_select":  "請選擇語言:",
		"language_updated": "機器人語言已更新為: %s",

		"warning_title":                  "⚠️ <b>安全提醒</b> [%s]",
		"warning_restricted":             "用戶 %s 已被限制發送消息和媒體的權限",
		"warning_reason":                 "<b>原因</b>: %s",
		"warning_unban_button":           "解除限制",
		"warning_self_unban_button":      "自行解除限制",
		"warning_self_unban_instruction": "%s 由于安全原因你已被限制发言，如需解除限制请联系管理员或者点击下面的按钮自助解封。",
		"warning_user_unbanned":          "✅ 用戶已成功解封",
		"warning_unbanned_message":       "✅ <b>用戶已解封</b>\n用戶 %s 已被解除限制，現在可以正常發言。",
		"ban_user":                       "封禁用户",
		"user_banned":                    "✅ 用戶已被成功限制",
		"warning_banned_message":         "⛔ <b>用戶已被限制</b>\n用戶 %s 已被限制發送消息和媒體的權限。",
		"math_verification":              "請計算以下算術題：\n%d %s %d = ?",
		"math_verification_success":      "✅ 驗證成功！您已被解除限制。",
		"math_verification_failed":       "❌ 驗證失敗，請重試。",

		// New translations for private chat mode
		"please_use_private_chat": "請在私聊中使用此命令，以管理您的群組設置",
		"select_group":            "請選擇您要管理的群組:",
		"enter_group_id":          "請輸入群組ID，格式為純數字（例如：-1001234567890）",
		"action":                  "操作",
		"no_admin_groups":         "您不是任何群組的管理員，或者機器人未添加到您管理的群組中",
		"group_not_found":         "找不到此群組，或機器人不是該群組的成員",
		"invalid_group_id":        "無效的群組ID，請輸入正確的數字格式",
		"invalid_format":          "無效的格式，請重試",
		"unknown_action":          "未知操作，請重試",
		"action_completed":        "操作已完成",

		// 用户被限制的原因
		"reason_premium_user":    "Premium用戶",
		"reason_random_username": "隨機用戶名",
		"reason_emoji_name":      "姓名含有表情符號",
		"reason_bio_link":        "個人簡介包含可疑連結",
		"reason_cas_blacklisted": "用戶在 CAS 黑名單中",
		"reason_ai_spam":         "被 AI 判定為垃圾訊息",
		"reason_join_group":      "加入群組",
		// Toggle action response messages
		"premium_ban_enabled":          "已啟用默認封禁Premium用戶",
		"premium_ban_disabled":         "已禁用默認封禁Premium用戶",
		"cas_enabled":                  "已啟用CAS驗證",
		"cas_disabled":                 "已禁用CAS驗證",
		"random_username_ban_enabled":  "已啟用隨機用戶名封禁",
		"random_username_ban_disabled": "已禁用隨機用戶名封禁",
		"emoji_name_ban_enabled":       "已啟用姓名表情符號封禁",
		"emoji_name_ban_disabled":      "已禁用姓名表情符號封禁",
		"bio_link_ban_enabled":         "已啟用個人簡介可疑連結封禁",
		"bio_link_ban_disabled":        "已禁用個人簡介可疑連結封禁",
		"notifications_enabled":        "已啟用管理員通知",
		"notifications_disabled":       "已禁用管理員通知",

		// Other messages for action responses
		"toggle_premium":         "切換Premium用戶封禁",
		"toggle_cas":             "切換CAS驗證",
		"toggle_random_username": "切換隨機用戶名封禁",
		"toggle_emoji_name":      "切換姓名表情符號封禁",
		"toggle_bio_link":        "切換個人簡介可疑連結封禁",
		"toggle_notifications":   "切換管理員通知",
		"change_language":        "更改語言",

		"select_language": "請選擇機器人使用的語言:",
	},

	LangEnglish: {
		"help_title":                      "TG-AntiSpam Bot Help",
		"help_description":                "This bot helps protect your group from spam messages and malicious users.",
		"general_commands":                "General commands:",
		"settings_commands":               "Group management commands:",
		"help_cmd_help":                   "/help - Show this help message",
		"help_cmd_settings":               "/settings - Display group settings",
		"help_cmd_toggle_premium":         "/toggle_premium - Toggle default banning of Premium users",
		"help_cmd_toggle_cas":             "/toggle_cas - Toggle CAS verification",
		"help_cmd_toggle_random_username": "/toggle_random_username - Toggle default banning of random username",
		"help_cmd_toggle_emoji_name":      "/toggle_emoji_name - Toggle default banning of name containing emojis",
		"help_cmd_toggle_bio_link":        "/toggle_bio_link - Toggle default banning of bio containing suspicious links",
		"help_cmd_toggle_notifications":   "/toggle_notifications - Toggle admin notifications",
		"help_cmd_language":               "/language - Set bot language",
		"help_cmd_language_group":         "/language_group - Set group language",
		"help_cmd_self_unban":             "/self_unban - Self-unban",
		"help_note":                       "Note: Only group administrators can change settings.",

		// Command descriptions for Telegram command menu
		"cmd_desc_help":                   "Show help information",
		"cmd_desc_settings":               "View/modify group settings",
		"cmd_desc_toggle_premium":         "Toggle Premium user banning",
		"cmd_desc_toggle_cas":             "Toggle CAS verification",
		"cmd_desc_toggle_notifications":   "Toggle admin notifications",
		"cmd_desc_language":               "Set bot language",
		"cmd_desc_toggle_random_username": "Toggle random username banning",
		"cmd_desc_toggle_emoji_name":      "Toggle name emoji banning",
		"cmd_desc_toggle_bio_link":        "Toggle bio link banning",
		"cmd_desc_self_unban":             "Self-unban",

		"user_not_admin":               "You are not an admin in this group, and cannot use this command",
		"empty_group_list":             "Group list is empty, please add the bot to your groups and set it as an administrator",
		"get_ban_records_error":        "Failed to get ban records, please try again later",
		"no_ban_records":               "You have no ban records to unban",
		"cannot_unban_for_other_users": "You cannot unban for other users",
		"select_group_to_unban":        "Please select the group to unban from:",

		"settings_title":           "Settings for %s",
		"settings_bot_status":      "Bot Status:",
		"settings_active":          "✅ Active",
		"settings_current":         "Current Settings:",
		"settings_ban_premium":     "- Ban Premium Users by Default: %s",
		"settings_cas":             "- CAS Verification: %s",
		"settings_random_username": "- Random Username Check: %s",
		"settings_emoji_name":      "- Name Emoji Check: %s",
		"settings_bio_link":        "- Bio Link Check: %s",
		"settings_notifications":   "- Admin Notifications: %s",
		"settings_language":        "- Language: %s",

		"settings_cmd_premium":         "/toggle_premium - Toggle Premium user ban",
		"settings_cmd_cas":             "/toggle_cas - Toggle CAS verification",
		"settings_cmd_notifications":   "/toggle_notifications - Toggle admin notifications",
		"settings_cmd_language":        "/language - Set bot language",
		"settings_cmd_random_username": "/toggle_random_username - Toggle random username banning",
		"settings_cmd_emoji_name":      "/toggle_emoji_name - Toggle name emoji banning",
		"settings_cmd_bio_link":        "/toggle_bio_link - Toggle bio link banning",

		"enabled":  "✅ Enabled",
		"disabled": "❌ Disabled",

		"setting_premium":         "Ban Premium users",
		"setting_cas":             "CAS verification",
		"setting_random_username": "Ban random username",
		"setting_emoji_name":      "Ban name containing emojis",
		"setting_bio_link":        "Ban bio containing suspicious links",
		"setting_notifications":   "Notify admins",
		"setting_language":        "Language",

		"language_select":  "Please select a language:",
		"language_updated": "Bot language updated to: %s",

		"warning_title":                  "⚠️ <b>Security Alert</b> [%s]",
		"warning_restricted":             "User %s has been restricted from sending messages and media",
		"warning_reason":                 "<b>Reason</b>: %s",
		"warning_unban_button":           "Remove Restriction",
		"warning_self_unban_button":      "Self-unban",
		"warning_self_unban_instruction": "%s has been restricted from sending messages and media due to security reasons, please contact the administrator or click the button below to self-unban.",
		"warning_user_unbanned":          "✅ User successfully unrestricted",
		"warning_unbanned_message":       "✅ <b>User Unrestricted</b>\nUser %s has been unrestricted and can now send messages normally.",
		"ban_user":                       "Ban User",
		"user_banned":                    "✅ User successfully restricted",
		"warning_banned_message":         "⛔ <b>User Restricted</b>\nUser %s has been restricted from sending messages and media.",
		"math_verification":              "Please calculate the following math problem: %d %s %d = ?",
		"math_verification_success":      "✅ Verification successful! You have been unrestricted.",
		"math_verification_failed":       "❌ Verification failed, please try again.",

		// New translations for private chat mode
		"please_use_private_chat": "Please use this command in a private chat with the bot to manage your group settings",
		"select_group":            "Please select a group to manage:",
		"enter_group_id":          "Please enter the Group ID as a numeric value (e.g., -1001234567890)",
		"action":                  "Action",
		"no_admin_groups":         "You are not an admin in any groups, or the bot has not been added to groups you administer",
		"group_not_found":         "Group not found, or the bot is not a member of that group",
		"invalid_group_id":        "Invalid group ID, please enter a valid numeric format",
		"invalid_format":          "Invalid format, please try again",
		"unknown_action":          "Unknown action, please try again",
		"action_completed":        "Action completed successfully",

		// 用户被限制的原因
		"reason_premium_user":    "Premium user",
		"reason_random_username": "random username",
		"reason_emoji_name":      "Name contains emojis",
		"reason_bio_link":        "Bio contains suspicious links",
		"reason_cas_blacklisted": "User is on the CAS blacklist",
		"reason_ai_spam":         "Classified as spam by AI",
		"reason_join_group":      "Join group",

		// Toggle action response messages
		"premium_ban_enabled":          "Premium user ban enabled",
		"premium_ban_disabled":         "Premium user ban disabled",
		"cas_enabled":                  "CAS verification enabled",
		"cas_disabled":                 "CAS verification disabled",
		"random_username_ban_enabled":  "Random username ban enabled",
		"random_username_ban_disabled": "Random username ban disabled",
		"emoji_name_ban_enabled":       "Name emoji ban enabled",
		"emoji_name_ban_disabled":      "Name emoji ban disabled",
		"bio_link_ban_enabled":         "Bio suspicious link ban enabled",
		"bio_link_ban_disabled":        "Bio suspicious link ban disabled",
		"notifications_enabled":        "Admin notifications enabled",
		"notifications_disabled":       "Admin notifications disabled",

		// Other messages for action responses
		"toggle_premium":         "Toggle Premium user ban",
		"toggle_cas":             "Toggle CAS verification",
		"toggle_random_username": "Toggle random username ban",
		"toggle_emoji_name":      "Toggle name emoji ban",
		"toggle_bio_link":        "Toggle bio suspicious link ban",
		"toggle_notifications":   "Toggle admin notifications",
		"change_language":        "Change language",

		"select_language": "Please select a language for the bot:",
	},
}

// GetTranslation returns the correct translation for a given language code and key
func GetTranslation(lang, key string) string {
	// Default to Simplified Chinese if language not supported
	if _, ok := Translations[lang]; !ok {
		lang = LangSimplifiedChinese
	}

	// Get translation for key
	if translation, ok := Translations[lang][key]; ok {
		return translation
	}

	// Fall back to Simplified Chinese if key not found in specified language
	if translation, ok := Translations[LangSimplifiedChinese][key]; ok {
		return translation
	}

	// Return the key itself if translation not found
	return key
}

// GetLanguageName returns the localized name of a language code
func GetLanguageName(lang, langCode string) string {
	switch langCode {
	case LangSimplifiedChinese:
		return "简体中文"
	case LangTraditionalChinese:
		return "繁体中文"
	case LangEnglish:
		return "English"
	default:
		return langCode
	}
}
