package i18n

// Catalog is the complete table of translatable strings rendered by the
// backend. Keys are namespaced (err.*, auth.*, setup.*, …) so adding a new
// message in one domain doesn't risk colliding with another, and so that a
// missing entry inside an entire domain shows up as a tight cluster of
// "key.path" leaks in the UI rather than scattered single bugs.
//
// Adding a new message:
//  1. Pick a namespace prefix (or add one and document it here).
//  2. Add a constant in keys.go if Go code references the message.
//  3. Add an entry below with both English and Chinese translations.
//  4. Use [T] / [internal/handler.M] / the template `T` helper to render it.
//
// Removing or renaming a key is a code search away — keys are used as
// constant names in keys.go and as raw strings in templates, so grepping the
// repo finds every reference.
//
// Convention: English entries are the source of truth; Chinese entries
// translate them. When a translator is unsure, leaving the Chinese entry
// blank or omitted causes [T] to fall back to English at runtime — which is
// always preferable to shipping a wrong translation.
//
// Format strings: messages whose runtime form interpolates a value use
// fmt.Sprintf-style verbs (typically `%s`). Both locale entries must agree
// on the verb count and order.
var Catalog = map[string]map[Locale]string{
	// ── Generic envelope ────────────────────────────────────────────────
	"common.success": {
		LocaleEN: "success",
		LocaleZH: "成功",
	},
	"err.bad_request": {
		LocaleEN: "bad request",
		LocaleZH: "请求无效",
	},
	"err.unauthorized": {
		LocaleEN: "unauthorized",
		LocaleZH: "未授权",
	},
	"err.forbidden": {
		LocaleEN: "forbidden",
		LocaleZH: "无权操作",
	},
	"err.not_found": {
		LocaleEN: "not found",
		LocaleZH: "未找到",
	},
	"err.internal": {
		LocaleEN: "internal server error",
		LocaleZH: "服务器内部错误",
	},
	"err.invalid_request": {
		LocaleEN: "invalid request",
		LocaleZH: "请求无效",
	},
	"err.invalid_payload": {
		LocaleEN: "invalid payload: %s",
		LocaleZH: "请求体无效：%s",
	},
	"err.missing_user_context": {
		LocaleEN: "missing user context",
		LocaleZH: "缺少用户上下文",
	},
	"err.missing_request_context": {
		LocaleEN: "missing request context",
		LocaleZH: "缺少请求上下文",
	},

	// ── Auth ────────────────────────────────────────────────────────────
	"auth.username_password_required": {
		LocaleEN: "username and password are required",
		LocaleZH: "用户名和密码必填",
	},
	"auth.login_failed": {
		LocaleEN: "login failed",
		LocaleZH: "登录失败",
	},
	"auth.invalid_refresh_token": {
		LocaleEN: "invalid refresh token",
		LocaleZH: "刷新令牌无效",
	},
	"auth.refresh_token_required": {
		LocaleEN: "refresh_token is required",
		LocaleZH: "refresh_token 必填",
	},
	"auth.refresh_session_failed": {
		LocaleEN: "failed to refresh session",
		LocaleZH: "刷新会话失败",
	},
	"auth.invalid_registration": {
		LocaleEN: "invalid registration data: %s",
		LocaleZH: "注册数据无效：%s",
	},
	"auth.registration_failed": {
		LocaleEN: "registration failed",
		LocaleZH: "注册失败",
	},
	"auth.invalid_profile": {
		LocaleEN: "invalid profile data: %s",
		LocaleZH: "资料数据无效：%s",
	},
	"auth.update_profile_failed": {
		LocaleEN: "update profile failed",
		LocaleZH: "更新资料失败",
	},
	"auth.invalid_password": {
		LocaleEN: "invalid password data: %s",
		LocaleZH: "密码数据无效：%s",
	},
	"auth.change_password_failed": {
		LocaleEN: "change password failed",
		LocaleZH: "修改密码失败",
	},
	"auth.set_password_failed": {
		LocaleEN: "set password failed",
		LocaleZH: "设置密码失败",
	},
	"auth.hash_password_failed": {
		LocaleEN: "failed to hash password",
		LocaleZH: "密码加密失败",
	},
	"auth.revoke_sessions_failed": {
		LocaleEN: "failed to revoke existing sessions",
		LocaleZH: "撤销现有会话失败",
	},
	"auth.load_current_user_failed": {
		LocaleEN: "failed to load current user",
		LocaleZH: "加载当前用户失败",
	},
	"auth.reload_user_failed": {
		LocaleEN: "failed to reload user",
		LocaleZH: "重新加载用户失败",
	},
	"auth.save_nickname_failed": {
		LocaleEN: "failed to save user nickname",
		LocaleZH: "保存用户昵称失败",
	},
	"auth.user_not_found": {
		LocaleEN: "user not found",
		LocaleZH: "用户不存在",
	},
	"auth.identifier_required": {
		LocaleEN: "identifier is required",
		LocaleZH: "标识符必填",
	},

	// ── 2FA ─────────────────────────────────────────────────────────────
	"twofa.code_required": {
		LocaleEN: "verification code is required",
		LocaleZH: "验证码必填",
	},
	"twofa.password_and_code_required": {
		LocaleEN: "password and code are required",
		LocaleZH: "密码和验证码必填",
	},
	"twofa.code_incorrect": {
		LocaleEN: "verification code is incorrect",
		LocaleZH: "验证码不正确",
	},
	"twofa.code_expired": {
		LocaleEN: "verification code has expired",
		LocaleZH: "验证码已过期",
	},
	"twofa.start_setup_failed": {
		LocaleEN: "failed to start 2fa setup",
		LocaleZH: "启动两步验证失败",
	},
	"twofa.enable_failed": {
		LocaleEN: "failed to enable 2fa",
		LocaleZH: "启用两步验证失败",
	},
	"twofa.disable_failed": {
		LocaleEN: "failed to disable 2fa",
		LocaleZH: "停用两步验证失败",
	},
	"twofa.verify_failed": {
		LocaleEN: "failed to verify 2fa",
		LocaleZH: "校验两步验证失败",
	},
	"twofa.reset_failed": {
		LocaleEN: "failed to reset 2fa",
		LocaleZH: "重置两步验证失败",
	},
	"twofa.challenge_required": {
		LocaleEN: "challenge_token and code are required",
		LocaleZH: "challenge_token 和验证码必填",
	},

	// ── Email / verification / password reset ──────────────────────────
	"email.invalid": {
		LocaleEN: "invalid email: %s",
		LocaleZH: "邮箱无效：%s",
	},
	"email.invalid_address": {
		LocaleEN: "invalid email address",
		LocaleZH: "邮箱地址无效",
	},
	"email.already_registered": {
		LocaleEN: "email is already registered",
		LocaleZH: "邮箱已被注册",
	},
	"email.new_matches_current": {
		LocaleEN: "new email matches current email",
		LocaleZH: "新邮箱与当前邮箱相同",
	},
	"email.no_pending_verification": {
		LocaleEN: "no pending verification for this email",
		LocaleZH: "该邮箱没有待验证的请求",
	},
	"email.no_pending_reset": {
		LocaleEN: "no pending reset for this account",
		LocaleZH: "该账号没有待处理的重置请求",
	},
	"email.send_verification_failed": {
		LocaleEN: "send verification email failed: %s",
		LocaleZH: "发送验证邮件失败：%s",
	},
	"email.send_reset_failed": {
		LocaleEN: "send reset email failed: %s",
		LocaleZH: "发送重置邮件失败：%s",
	},
	"email.confirm_change_failed": {
		LocaleEN: "confirm email change failed: %s",
		LocaleZH: "确认邮箱变更失败：%s",
	},
	"email.confirm_reset_failed": {
		LocaleEN: "confirm password reset failed: %s",
		LocaleZH: "确认密码重置失败：%s",
	},
	"email.invalid_reset_data": {
		LocaleEN: "invalid reset data: %s",
		LocaleZH: "重置数据无效：%s",
	},
	"email.change_service_unavailable": {
		LocaleEN: "email change service unavailable",
		LocaleZH: "邮箱变更服务不可用",
	},
	"email.password_reset_unavailable": {
		LocaleEN: "password reset service unavailable",
		LocaleZH: "密码重置服务不可用",
	},
	"email.service_not_configured": {
		LocaleEN: "email service is not configured",
		LocaleZH: "邮件服务尚未配置",
	},
	"email.rate_limited": {
		LocaleEN: "please wait before requesting another code",
		LocaleZH: "请稍候再请求新的验证码",
	},
	"email.test_send_failed": {
		LocaleEN: "test email send failed: %s",
		LocaleZH: "测试邮件发送失败：%s",
	},
	"email.validate_failed": {
		LocaleEN: "failed to validate user email",
		LocaleZH: "校验用户邮箱失败",
	},

	// ── OAuth / social login ───────────────────────────────────────────
	"oauth.invalid_provider": {
		LocaleEN: "invalid oauth provider data: %s",
		LocaleZH: "OAuth 提供方数据无效：%s",
	},
	"oauth.update_provider_failed": {
		LocaleEN: "failed to update oauth provider",
		LocaleZH: "更新 OAuth 提供方失败",
	},
	"oauth.load_providers_failed": {
		LocaleEN: "failed to load oauth providers",
		LocaleZH: "加载 OAuth 提供方失败",
	},
	"oauth.exchange_failed": {
		LocaleEN: "exchange oauth ticket failed",
		LocaleZH: "交换 OAuth 票据失败",
	},
	"oauth.invalid_onboarding": {
		LocaleEN: "invalid onboarding data: %s",
		LocaleZH: "引导数据无效：%s",
	},
	"oauth.onboarding_failed": {
		LocaleEN: "oauth onboarding failed",
		LocaleZH: "OAuth 引导失败",
	},
	"oauth.ticket_required": {
		LocaleEN: "ticket is required",
		LocaleZH: "票据必填",
	},
	"oauth.social_not_configured": {
		LocaleEN: "social login is not configured",
		LocaleZH: "社交登录尚未配置",
	},
	"oauth.list_identities_failed": {
		LocaleEN: "failed to list identities",
		LocaleZH: "获取已绑定身份失败",
	},
	"oauth.unlink_identity_failed": {
		LocaleEN: "failed to unlink identity",
		LocaleZH: "解绑身份失败",
	},
	"oauth.identity_not_found": {
		LocaleEN: "identity not found",
		LocaleZH: "身份不存在",
	},

	// ── Setup / install wizard ─────────────────────────────────────────
	"setup.already_initialized": {
		LocaleEN: "system is already initialized",
		LocaleZH: "系统已完成初始化",
	},
	"setup.invalid_data": {
		LocaleEN: "invalid setup data: %s",
		LocaleZH: "安装数据无效：%s",
	},
	"setup.save_site_settings_failed": {
		LocaleEN: "failed to save site settings",
		LocaleZH: "保存站点设置失败",
	},
	"setup.create_admin_failed": {
		LocaleEN: "failed to create admin user: %s",
		LocaleZH: "创建管理员失败：%s",
	},
	"setup.invalid_storage": {
		LocaleEN: "invalid %s storage config: %s",
		LocaleZH: "%s 存储配置无效：%s",
	},
	"setup.storage_validation_failed": {
		LocaleEN: "storage config validation failed: %s",
		LocaleZH: "存储配置校验失败：%s",
	},
	"setup.create_storage_failed": {
		LocaleEN: "failed to create storage config",
		LocaleZH: "创建存储配置失败",
	},
	"setup.invalid_database": {
		LocaleEN: "invalid database config: %s",
		LocaleZH: "数据库配置无效：%s",
	},
	"setup.save_database_failed": {
		LocaleEN: "save database config: %s",
		LocaleZH: "保存数据库配置失败：%s",
	},

	// ── Files / albums / storage ────────────────────────────────────────
	"file.not_found": {
		LocaleEN: "file not found",
		LocaleZH: "文件不存在",
	},
	"file.thumbnail_not_found": {
		LocaleEN: "thumbnail not found",
		LocaleZH: "缩略图不存在",
	},
	"file.read_failed": {
		LocaleEN: "failed to read file",
		LocaleZH: "读取文件失败",
	},
	"file.delete_failed": {
		LocaleEN: "failed to delete file",
		LocaleZH: "删除文件失败",
	},
	"file.move_failed": {
		LocaleEN: "failed to move file",
		LocaleZH: "移动文件失败",
	},
	"file.list_failed": {
		LocaleEN: "failed to list files",
		LocaleZH: "获取文件列表失败",
	},
	"file.not_owner": {
		LocaleEN: "not the owner of this file",
		LocaleZH: "您不是该文件的所有者",
	},
	"file.invalid_target_folder": {
		LocaleEN: "invalid target folder",
		LocaleZH: "目标文件夹无效",
	},
	"file.ids_required": {
		LocaleEN: "ids is required",
		LocaleZH: "ids 必填",
	},
	"album.not_found": {
		LocaleEN: "album not found",
		LocaleZH: "相册不存在",
	},
	"album.folder_not_found": {
		LocaleEN: "folder not found",
		LocaleZH: "文件夹不存在",
	},
	"album.not_owner": {
		LocaleEN: "not the owner of this album",
		LocaleZH: "您不是该相册的所有者",
	},
	"album.invalid_data": {
		LocaleEN: "invalid album data: %s",
		LocaleZH: "相册数据无效：%s",
	},
	"album.invalid_parent_folder": {
		LocaleEN: "invalid parent folder",
		LocaleZH: "父文件夹无效",
	},
	"album.folder_self_parent": {
		LocaleEN: "folder cannot be its own parent",
		LocaleZH: "文件夹不能作为自己的父级",
	},
	"album.descendant_move": {
		LocaleEN: "cannot move folder into its descendant",
		LocaleZH: "无法将文件夹移入其后代",
	},
	"album.create_failed": {
		LocaleEN: "failed to create album",
		LocaleZH: "创建相册失败",
	},
	"album.update_failed": {
		LocaleEN: "failed to update album",
		LocaleZH: "更新相册失败",
	},
	"album.delete_failed": {
		LocaleEN: "failed to delete album",
		LocaleZH: "删除相册失败",
	},
	"album.list_failed": {
		LocaleEN: "failed to list albums",
		LocaleZH: "获取相册列表失败",
	},
	"album.load_folder_failed": {
		LocaleEN: "failed to load folder path",
		LocaleZH: "加载文件夹路径失败",
	},
	"storage.not_found": {
		LocaleEN: "storage config not found",
		LocaleZH: "存储配置不存在",
	},
	"storage.invalid_config": {
		LocaleEN: "invalid storage config: %s",
		LocaleZH: "存储配置无效：%s",
	},
	"storage.parse_config_failed": {
		LocaleEN: "failed to parse storage config",
		LocaleZH: "解析存储配置失败",
	},
	"storage.create_failed": {
		LocaleEN: "failed to create storage config",
		LocaleZH: "创建存储配置失败",
	},
	"storage.update_failed": {
		LocaleEN: "failed to update storage config",
		LocaleZH: "更新存储配置失败",
	},
	"storage.delete_failed": {
		LocaleEN: "failed to delete storage config",
		LocaleZH: "删除存储配置失败",
	},
	"storage.list_failed": {
		LocaleEN: "failed to list storage configs",
		LocaleZH: "获取存储配置列表失败",
	},
	"storage.set_default_failed": {
		LocaleEN: "failed to set default storage: %s",
		LocaleZH: "设置默认存储失败：%s",
	},
	"storage.reorder_failed": {
		LocaleEN: "failed to reorder storages: %s",
		LocaleZH: "调整存储顺序失败：%s",
	},
	"storage.invalid_reorder": {
		LocaleEN: "invalid reorder payload: %s",
		LocaleZH: "排序数据无效：%s",
	},
	"storage.ordered_ids_empty": {
		LocaleEN: "ordered_ids must not be empty",
		LocaleZH: "ordered_ids 不能为空",
	},

	// ── Tokens ──────────────────────────────────────────────────────────
	"token.invalid_data": {
		LocaleEN: "invalid token data: %s",
		LocaleZH: "令牌数据无效：%s",
	},
	"token.create_failed": {
		LocaleEN: "failed to create token",
		LocaleZH: "创建令牌失败",
	},
	"token.list_failed": {
		LocaleEN: "failed to list tokens",
		LocaleZH: "获取令牌列表失败",
	},
	"token.not_found": {
		LocaleEN: "token not found",
		LocaleZH: "令牌不存在",
	},

	// ── Users / settings ────────────────────────────────────────────────
	"user.not_found": {
		LocaleEN: "user not found",
		LocaleZH: "用户不存在",
	},
	"user.invalid_data": {
		LocaleEN: "invalid user data: %s",
		LocaleZH: "用户数据无效：%s",
	},
	"user.create_failed": {
		LocaleEN: "failed to create user: %s",
		LocaleZH: "创建用户失败：%s",
	},
	"user.update_failed": {
		LocaleEN: "failed to update user",
		LocaleZH: "更新用户失败",
	},
	"user.delete_failed": {
		LocaleEN: "failed to delete user",
		LocaleZH: "删除用户失败",
	},
	"user.list_failed": {
		LocaleEN: "failed to list users",
		LocaleZH: "获取用户列表失败",
	},
	"settings.invalid_data": {
		LocaleEN: "invalid settings data: %s",
		LocaleZH: "设置数据无效：%s",
	},
	"settings.invalid_value": {
		LocaleEN: "invalid %s: %s",
		LocaleZH: "%s 无效：%s",
	},
	"settings.invalid_empty": {
		LocaleEN: "invalid %s: cannot be empty",
		LocaleZH: "%s 无效：不能为空",
	},
	"settings.get_failed": {
		LocaleEN: "failed to get settings",
		LocaleZH: "获取设置失败",
	},
	"settings.update_failed": {
		LocaleEN: "failed to update settings",
		LocaleZH: "更新设置失败",
	},
	"settings.cannot_modify": {
		LocaleEN: "%s cannot be modified through the settings API",
		LocaleZH: "%s 无法通过设置 API 修改",
	},
	"settings.cannot_be_empty": {
		LocaleEN: "%s cannot be empty",
		LocaleZH: "%s 不能为空",
	},
	"settings.invalid_path_pattern": {
		LocaleEN: "invalid upload.path_pattern: %s",
		LocaleZH: "upload.path_pattern 无效：%s",
	},
	"settings.invalid_max_file_size": {
		LocaleEN: "invalid upload.max_file_size_mb: %s",
		LocaleZH: "upload.max_file_size_mb 无效：%s",
	},
	"settings.invalid_dangerous_extensions": {
		LocaleEN: "invalid upload.dangerous_extension_rules: %s",
		LocaleZH: "upload.dangerous_extension_rules 无效：%s",
	},
	"settings.invalid_dangerous_rename": {
		LocaleEN: "invalid upload.dangerous_rename_suffix: %s",
		LocaleZH: "upload.dangerous_rename_suffix 无效：%s",
	},
	"settings.capacity_negative": {
		LocaleEN: "capacity_limit_bytes must not be negative",
		LocaleZH: "capacity_limit_bytes 不能为负",
	},
	"settings.access_stats_failed": {
		LocaleEN: "failed to get access stats",
		LocaleZH: "获取访问统计失败",
	},
	"settings.access_heatmap_failed": {
		LocaleEN: "failed to get access heatmap stats",
		LocaleZH: "获取访问热力图失败",
	},
	"settings.upload_stats_failed": {
		LocaleEN: "failed to get upload stats",
		LocaleZH: "获取上传统计失败",
	},
	"settings.upload_heatmap_failed": {
		LocaleEN: "failed to get upload heatmap stats",
		LocaleZH: "获取上传热力图失败",
	},
	"settings.stats_failed": {
		LocaleEN: "failed to get stats",
		LocaleZH: "获取统计数据失败",
	},

	// ── Admin page-title decoration ─────────────────────────────────────
	"admin.page_title_suffix": {
		LocaleEN: "%s admin",
		LocaleZH: "%s 管理后台",
	},

	// ── Templates: layout / nav / footer ────────────────────────────────
	"nav.explore": {
		LocaleEN: "Explore",
		LocaleZH: "探索",
	},
	"nav.upload": {
		LocaleEN: "Upload",
		LocaleZH: "上传",
	},
	"nav.login": {
		LocaleEN: "Sign in",
		LocaleZH: "登录",
	},
	"nav.dashboard": {
		LocaleEN: "Dashboard",
		LocaleZH: "控制台",
	},
	"nav.language": {
		LocaleEN: "Language",
		LocaleZH: "语言",
	},

	// ── Templates: index / landing hero ─────────────────────────────────
	"index.hero_eyebrow": {
		LocaleEN: "Open source · self-hosted · single-binary launch",
		LocaleZH: "开源 · 自部署 · 单文件极速启动",
	},
	"index.stats_files": {
		LocaleEN: "%s files",
		LocaleZH: "%s 个文件",
	},
	"index.hero_title_1": {
		LocaleEN: "Drop in. Share out.",
		LocaleZH: "拖入即享，秒级分发",
	},
	"index.hero_title_2": {
		LocaleEN: "Your files, your domain.",
		LocaleZH: "你的文件，你的域名",
	},
	"index.hero_subtitle": {
		LocaleEN: "Lightweight image and file hosting that puts you in control of storage, links and access.",
		LocaleZH: "轻量级图床与文件托管，存储、外链与访问尽在掌握。",
	},
	"index.cta_upload": {
		LocaleEN: "Upload now",
		LocaleZH: "立即上传",
	},
	"index.cta_explore": {
		LocaleEN: "Browse public gallery",
		LocaleZH: "浏览公开画廊",
	},
	"index.feature_drag_title": {
		LocaleEN: "Drag, drop, done",
		LocaleZH: "拖拽即用",
	},
	"index.feature_drag_body": {
		LocaleEN: "Multi-file upload with progress, retries and instant share links — no clicks wasted.",
		LocaleZH: "多文件上传，带进度与重试，分享链接即刻生成。",
	},
	"index.feature_storage_title": {
		LocaleEN: "Bring your storage",
		LocaleZH: "存储自由切换",
	},
	"index.feature_storage_body": {
		LocaleEN: "Local disk, S3, OSS, FTP — pick a backend per album or fall back to a default.",
		LocaleZH: "本地、S3、OSS、FTP 任意挑选，可按相册指定，也可设默认后端。",
	},
	"index.feature_links_title": {
		LocaleEN: "Links that don't break",
		LocaleZH: "永不失效的链接",
	},
	"index.feature_links_body": {
		LocaleEN: "Hash-based URLs, optional source-CDN passthrough and per-file access control.",
		LocaleZH: "基于哈希的 URL、源站直连穿透、按文件粒度的访问控制。",
	},

	// ── Templates: explore page ─────────────────────────────────────────
	"explore.title": {
		LocaleEN: "Explore",
		LocaleZH: "探索广场",
	},
	"explore.subtitle": {
		LocaleEN: "Recent public uploads from the community.",
		LocaleZH: "近期来自社区的公开上传。",
	},
	"explore.empty_title": {
		LocaleEN: "Nothing here yet",
		LocaleZH: "暂无内容",
	},
	"explore.empty_body": {
		LocaleEN: "Be the first — upload a file and toggle it public to share with everyone.",
		LocaleZH: "成为第一个分享者——上传文件并设为公开即可。",
	},
	"explore.disabled_title": {
		LocaleEN: "Public gallery is off",
		LocaleZH: "公开画廊已关闭",
	},
	"explore.disabled_body": {
		LocaleEN: "An admin disabled the public gallery on this instance. Sign in to see your own files.",
		LocaleZH: "管理员已关闭本站的公开画廊。请登录查看您自己的文件。",
	},
	"explore.load_more": {
		LocaleEN: "Load more",
		LocaleZH: "加载更多",
	},
	"explore.loading": {
		LocaleEN: "Loading…",
		LocaleZH: "加载中…",
	},
	"explore.no_more": {
		LocaleEN: "No more files",
		LocaleZH: "没有更多文件了",
	},
	"explore.load_failed": {
		LocaleEN: "Failed to load — tap to retry",
		LocaleZH: "加载失败 —— 点击重试",
	},
	"explore.tab_all": {
		LocaleEN: "All",
		LocaleZH: "全部",
	},
	"explore.tab_image": {
		LocaleEN: "Images",
		LocaleZH: "图片",
	},
	"explore.tab_video": {
		LocaleEN: "Videos",
		LocaleZH: "视频",
	},
	"explore.tab_audio": {
		LocaleEN: "Audio",
		LocaleZH: "音频",
	},
	"explore.tab_file": {
		LocaleEN: "Files",
		LocaleZH: "文件",
	},
	"explore.empty_files": {
		LocaleEN: "No public files yet",
		LocaleZH: "暂无公开文件",
	},
	"explore.type_pdf": {
		LocaleEN: "PDF document",
		LocaleZH: "PDF 文档",
	},
	"explore.type_word": {
		LocaleEN: "Word document",
		LocaleZH: "Word 文档",
	},
	"explore.type_odt": {
		LocaleEN: "ODT document",
		LocaleZH: "ODT 文档",
	},
	"explore.type_rtf": {
		LocaleEN: "RTF document",
		LocaleZH: "RTF 文档",
	},
	"explore.type_pages": {
		LocaleEN: "Pages document",
		LocaleZH: "Pages 文档",
	},
	"explore.type_excel": {
		LocaleEN: "Excel spreadsheet",
		LocaleZH: "Excel 表格",
	},
	"explore.type_ods": {
		LocaleEN: "ODS spreadsheet",
		LocaleZH: "ODS 表格",
	},
	"explore.type_numbers": {
		LocaleEN: "Numbers spreadsheet",
		LocaleZH: "Numbers 表格",
	},
	"explore.type_csv": {
		LocaleEN: "CSV spreadsheet",
		LocaleZH: "CSV 表格",
	},
	"explore.type_ppt": {
		LocaleEN: "PowerPoint presentation",
		LocaleZH: "PPT 演示",
	},
	"explore.type_odp": {
		LocaleEN: "ODP presentation",
		LocaleZH: "ODP 演示",
	},
	"explore.type_keynote": {
		LocaleEN: "Keynote presentation",
		LocaleZH: "Keynote 演示",
	},
	"explore.type_text": {
		LocaleEN: "Plain text",
		LocaleZH: "纯文本",
	},
	"explore.type_log": {
		LocaleEN: "Log file",
		LocaleZH: "日志文件",
	},
	"explore.type_ini": {
		LocaleEN: "INI config",
		LocaleZH: "INI 配置",
	},
	"explore.type_config": {
		LocaleEN: "Config file",
		LocaleZH: "配置文件",
	},
	"explore.type_env": {
		LocaleEN: "Env file",
		LocaleZH: "环境变量",
	},
	"explore.type_properties": {
		LocaleEN: "Properties file",
		LocaleZH: "属性文件",
	},
	"explore.type_c_header": {
		LocaleEN: "C header",
		LocaleZH: "C 头文件",
	},
	"explore.type_cpp_header": {
		LocaleEN: "C++ header",
		LocaleZH: "C++ 头文件",
	},
	"explore.type_shell": {
		LocaleEN: "Shell",
		LocaleZH: "Shell",
	},
	"explore.type_batch": {
		LocaleEN: "Batch script",
		LocaleZH: "批处理",
	},
	"explore.type_database": {
		LocaleEN: "Database",
		LocaleZH: "数据库",
	},
	"explore.type_zip_archive": {
		LocaleEN: "ZIP archive",
		LocaleZH: "ZIP 压缩包",
	},
	"explore.type_rar_archive": {
		LocaleEN: "RAR archive",
		LocaleZH: "RAR 压缩包",
	},
	"explore.type_7z_archive": {
		LocaleEN: "7z archive",
		LocaleZH: "7z 压缩包",
	},
	"explore.type_tar_archive": {
		LocaleEN: "TAR archive",
		LocaleZH: "TAR 归档",
	},
	"explore.type_gz_archive": {
		LocaleEN: "GZ archive",
		LocaleZH: "GZ 压缩包",
	},
	"explore.type_tgz_archive": {
		LocaleEN: "TGZ archive",
		LocaleZH: "TGZ 压缩包",
	},
	"explore.type_bz2_archive": {
		LocaleEN: "BZ2 archive",
		LocaleZH: "BZ2 压缩包",
	},
	"explore.type_xz_archive": {
		LocaleEN: "XZ archive",
		LocaleZH: "XZ 压缩包",
	},
	"explore.type_truetype_font": {
		LocaleEN: "TrueType font",
		LocaleZH: "TrueType 字体",
	},
	"explore.type_opentype_font": {
		LocaleEN: "OpenType font",
		LocaleZH: "OpenType 字体",
	},
	"explore.type_woff_font": {
		LocaleEN: "WOFF font",
		LocaleZH: "WOFF 字体",
	},
	"explore.type_woff2_font": {
		LocaleEN: "WOFF2 font",
		LocaleZH: "WOFF2 字体",
	},
	"explore.type_executable": {
		LocaleEN: "Executable",
		LocaleZH: "可执行文件",
	},
	"explore.type_installer": {
		LocaleEN: "Installer",
		LocaleZH: "安装包",
	},
	"explore.type_dmg_image": {
		LocaleEN: "DMG image",
		LocaleZH: "DMG 镜像",
	},
	"explore.type_application": {
		LocaleEN: "Application",
		LocaleZH: "应用程序",
	},
	"explore.type_deb_installer": {
		LocaleEN: "DEB installer",
		LocaleZH: "DEB 安装包",
	},
	"explore.type_rpm_installer": {
		LocaleEN: "RPM installer",
		LocaleZH: "RPM 安装包",
	},
	"explore.type_apk_installer": {
		LocaleEN: "APK installer",
		LocaleZH: "APK 安装包",
	},
	"explore.type_ipa_installer": {
		LocaleEN: "IPA installer",
		LocaleZH: "IPA 安装包",
	},
	"explore.type_jar_archive": {
		LocaleEN: "JAR archive",
		LocaleZH: "JAR 包",
	},
	"explore.type_war_archive": {
		LocaleEN: "WAR archive",
		LocaleZH: "WAR 包",
	},
	"explore.type_epub_book": {
		LocaleEN: "EPUB book",
		LocaleZH: "EPUB 电子书",
	},
	"explore.type_mobi_book": {
		LocaleEN: "Mobi book",
		LocaleZH: "Mobi 电子书",
	},
	"explore.type_kindle_book": {
		LocaleEN: "Kindle book",
		LocaleZH: "Kindle 电子书",
	},
	"explore.type_iso_image": {
		LocaleEN: "ISO image",
		LocaleZH: "ISO 镜像",
	},
	"explore.type_disk_image": {
		LocaleEN: "Disk image",
		LocaleZH: "磁盘映像",
	},
	"explore.type_vmdk_disk": {
		LocaleEN: "VMDK virtual disk",
		LocaleZH: "VMDK 虚拟盘",
	},
	"explore.type_pem_cert": {
		LocaleEN: "PEM certificate",
		LocaleZH: "PEM 证书",
	},
	"explore.type_cert": {
		LocaleEN: "Certificate",
		LocaleZH: "证书文件",
	},
	"explore.type_pkcs12_cert": {
		LocaleEN: "PKCS12 certificate",
		LocaleZH: "PKCS12 证书",
	},
	"explore.type_pfx_cert": {
		LocaleEN: "PFX certificate",
		LocaleZH: "PFX 证书",
	},
	"explore.type_lock": {
		LocaleEN: "Lock file",
		LocaleZH: "Lock 文件",
	},
	"explore.type_image": {
		LocaleEN: "Image",
		LocaleZH: "图片",
	},
	"explore.type_video": {
		LocaleEN: "Video",
		LocaleZH: "视频",
	},
	"explore.type_audio": {
		LocaleEN: "Audio",
		LocaleZH: "音频",
	},
	"explore.type_text_file": {
		LocaleEN: "Text file",
		LocaleZH: "文本文件",
	},
	"explore.type_unknown": {
		LocaleEN: "Unknown type",
		LocaleZH: "未知类型",
	},
	"explore.type_ext_suffix": {
		LocaleEN: "%s file",
		LocaleZH: "%s 文件",
	},

	// ── Templates: upload page ──────────────────────────────────────────
	"upload.title": {
		LocaleEN: "Upload",
		LocaleZH: "上传文件",
	},
	"upload.subtitle": {
		LocaleEN: "Drag files into the drop zone or click to choose. Up to %s per file.",
		LocaleZH: "把文件拖入拖放区，或点击选择。单文件最大 %s。",
	},
	"upload.dropzone_hint": {
		LocaleEN: "Drop files here or click to browse",
		LocaleZH: "将文件拖到此处或点击选择",
	},
	"upload.dropzone_sub": {
		LocaleEN: "Multiple files supported · max %s each",
		LocaleZH: "支持多文件 · 单文件最大 %s",
	},
	"upload.choose_files": {
		LocaleEN: "Choose files",
		LocaleZH: "选择文件",
	},
	"upload.start": {
		LocaleEN: "Start upload",
		LocaleZH: "开始上传",
	},
	"upload.clear": {
		LocaleEN: "Clear",
		LocaleZH: "清空",
	},
	"upload.uploading": {
		LocaleEN: "Uploading…",
		LocaleZH: "上传中…",
	},
	"upload.success_title": {
		LocaleEN: "Upload complete",
		LocaleZH: "上传完成",
	},
	"upload.success_body": {
		LocaleEN: "Copy the link below or open the share page for more formats.",
		LocaleZH: "复制下方链接，或打开分享页查看更多格式。",
	},
	"upload.copy_link": {
		LocaleEN: "Copy link",
		LocaleZH: "复制链接",
	},
	"upload.copied": {
		LocaleEN: "Copied",
		LocaleZH: "已复制",
	},
	"upload.open_share": {
		LocaleEN: "Open share page",
		LocaleZH: "打开分享页",
	},
	"upload.upload_more": {
		LocaleEN: "Upload another",
		LocaleZH: "继续上传",
	},
	"upload.guest_disabled_title": {
		LocaleEN: "Guest upload is off",
		LocaleZH: "游客上传已关闭",
	},
	"upload.guest_disabled_body": {
		LocaleEN: "Sign in to upload, or ask an admin to enable guest uploads.",
		LocaleZH: "请登录后上传，或请管理员开启游客上传。",
	},
	"upload.error_too_large": {
		LocaleEN: "%s exceeds the %s limit",
		LocaleZH: "%s 超过 %s 上限",
	},
	"upload.error_generic": {
		LocaleEN: "Upload failed",
		LocaleZH: "上传失败",
	},
	"upload.subtitle_short": {
		LocaleEN: "Drag or click to choose files. Share links appear right after upload.",
		LocaleZH: "拖拽或点击选择文件，上传后即刻获取分享链接。",
	},
	"upload.dropzone_sub_long": {
		LocaleEN: "Images, video, audio and documents · max %s per file",
		LocaleZH: "支持图片、视频、音频和文档，单文件最大 %s",
	},
	"upload.guest_disabled_body_pre": {
		LocaleEN: "Please ",
		LocaleZH: "请",
	},
	"upload.guest_disabled_login_link": {
		LocaleEN: "sign in",
		LocaleZH: "登录",
	},
	"upload.guest_disabled_body_post": {
		LocaleEN: " to upload, or ask an admin to enable guest uploads.",
		LocaleZH: "后上传文件，或联系管理员开启游客上传。",
	},
	"upload.unnamed_file": {
		LocaleEN: "Untitled file",
		LocaleZH: "未命名文件",
	},
	"upload.rate_limited": {
		LocaleEN: "Upload rate limited — please retry shortly",
		LocaleZH: "上传请求过于频繁，请稍后再试",
	},
	"upload.processing": {
		LocaleEN: "Processing",
		LocaleZH: "处理中",
	},
	"upload.done": {
		LocaleEN: "Done",
		LocaleZH: "完成",
	},
	"upload.failed": {
		LocaleEN: "Failed",
		LocaleZH: "失败",
	},
	"upload.toast_success": {
		LocaleEN: "Uploaded: %s",
		LocaleZH: "上传成功：%s",
	},
	"upload.toast_failed": {
		LocaleEN: "Upload failed: %s",
		LocaleZH: "上传失败：%s",
	},
	"upload.toast_failed_name": {
		LocaleEN: "Upload failed: %s",
		LocaleZH: "上传失败：%s",
	},
	"upload.toast_network_error": {
		LocaleEN: "Network error: %s",
		LocaleZH: "网络错误：%s",
	},
	"upload.error_too_large_with_name": {
		LocaleEN: "%s exceeds the per-file limit (%s)",
		LocaleZH: "%s 超过单文件大小上限（%s）",
	},
	"upload.copy_label_default": {
		LocaleEN: "Content",
		LocaleZH: "内容",
	},
	"upload.copied_suffix": {
		LocaleEN: "%s copied",
		LocaleZH: "%s已复制",
	},
	"upload.copy_empty": {
		LocaleEN: "Nothing to copy",
		LocaleZH: "没有可复制的内容",
	},
	"upload.copy_failed": {
		LocaleEN: "Copy failed — please copy manually",
		LocaleZH: "复制失败，请手动复制",
	},
	"upload.uploaded_badge": {
		LocaleEN: "Uploaded",
		LocaleZH: "已上传",
	},
	"upload.link_short": {
		LocaleEN: "Short link",
		LocaleZH: "短链接",
	},
	"upload.link_source": {
		LocaleEN: "Source URL",
		LocaleZH: "源 URL",
	},
	"upload.copy_btn": {
		LocaleEN: "Copy",
		LocaleZH: "复制",
	},
	"upload.copy_tooltip": {
		LocaleEN: "Click to copy",
		LocaleZH: "点击复制",
	},
	"upload.type_html_page": {
		LocaleEN: "HTML page",
		LocaleZH: "HTML 页面",
	},
	"upload.type_css_style": {
		LocaleEN: "CSS stylesheet",
		LocaleZH: "CSS 样式",
	},
	"upload.type_scss_style": {
		LocaleEN: "SCSS stylesheet",
		LocaleZH: "SCSS 样式",
	},
	"upload.type_sass_style": {
		LocaleEN: "Sass stylesheet",
		LocaleZH: "Sass 样式",
	},
	"upload.type_less_style": {
		LocaleEN: "Less stylesheet",
		LocaleZH: "Less 样式",
	},
	"upload.type_vue_file": {
		LocaleEN: "Vue file",
		LocaleZH: "Vue 文件",
	},
	"upload.type_svelte_file": {
		LocaleEN: "Svelte file",
		LocaleZH: "Svelte 文件",
	},
	"upload.type_shell_script": {
		LocaleEN: "Shell script",
		LocaleZH: "Shell 脚本",
	},
	"upload.type_bash_script": {
		LocaleEN: "Bash script",
		LocaleZH: "Bash 脚本",
	},
	"upload.type_zsh_script": {
		LocaleEN: "Zsh script",
		LocaleZH: "Zsh 脚本",
	},
	"upload.type_fish_script": {
		LocaleEN: "Fish script",
		LocaleZH: "Fish 脚本",
	},
	"upload.type_powershell_script": {
		LocaleEN: "PowerShell script",
		LocaleZH: "PowerShell 脚本",
	},
	"upload.type_sql_file": {
		LocaleEN: "SQL file",
		LocaleZH: "SQL 文件",
	},
	"upload.type_db_file": {
		LocaleEN: "Database file",
		LocaleZH: "数据库文件",
	},
	"upload.type_sqlite_db": {
		LocaleEN: "SQLite database",
		LocaleZH: "SQLite 数据库",
	},
	"upload.type_targz_archive": {
		LocaleEN: "TAR.GZ archive",
		LocaleZH: "TAR.GZ 压缩包",
	},
	"upload.type_tarbz2_archive": {
		LocaleEN: "TAR.BZ2 archive",
		LocaleZH: "TAR.BZ2 压缩包",
	},
	"upload.type_tarxz_archive": {
		LocaleEN: "TAR.XZ archive",
		LocaleZH: "TAR.XZ 压缩包",
	},
	"upload.type_psd_file": {
		LocaleEN: "Photoshop file",
		LocaleZH: "Photoshop 文件",
	},
	"upload.type_ai_file": {
		LocaleEN: "Illustrator file",
		LocaleZH: "Illustrator 文件",
	},
	"upload.type_sketch_file": {
		LocaleEN: "Sketch file",
		LocaleZH: "Sketch 文件",
	},
	"upload.type_fig_file": {
		LocaleEN: "Figma file",
		LocaleZH: "Figma 文件",
	},
	"upload.type_xd_file": {
		LocaleEN: "Adobe XD file",
		LocaleZH: "Adobe XD 文件",
	},
	"upload.type_file": {
		LocaleEN: "File",
		LocaleZH: "文件",
	},

	// ── Templates: share page ───────────────────────────────────────────
	"share.not_found_title": {
		LocaleEN: "File not found",
		LocaleZH: "文件不存在",
	},
	"share.not_found_body": {
		LocaleEN: "The file may have been deleted, or the link is wrong.",
		LocaleZH: "文件可能已被删除，或链接错误。",
	},
	"share.not_found_back": {
		LocaleEN: "Back to home",
		LocaleZH: "返回首页",
	},
	"share.download": {
		LocaleEN: "Download",
		LocaleZH: "下载",
	},
	"share.copy": {
		LocaleEN: "Copy",
		LocaleZH: "复制",
	},
	"share.copied": {
		LocaleEN: "Copied",
		LocaleZH: "已复制",
	},
	"share.uploaded_at": {
		LocaleEN: "Uploaded %s",
		LocaleZH: "上传于 %s",
	},
	"share.size": {
		LocaleEN: "Size",
		LocaleZH: "大小",
	},
	"share.dimensions": {
		LocaleEN: "Dimensions",
		LocaleZH: "尺寸",
	},
	"share.duration": {
		LocaleEN: "Duration",
		LocaleZH: "时长",
	},
	"share.type": {
		LocaleEN: "Type",
		LocaleZH: "类型",
	},
	"share.preview_truncated": {
		LocaleEN: "Preview truncated · download for full content",
		LocaleZH: "预览已截断 · 下载查看完整内容",
	},
	"share.source_url_label": {
		LocaleEN: "Source URL",
		LocaleZH: "源站 URL",
	},
	"share.copy_link": {
		LocaleEN: "Copy link",
		LocaleZH: "复制链接",
	},
	"share.preview_truncated_256k": {
		LocaleEN: "Truncated · showing first 256 KB",
		LocaleZH: "已截断 · 仅展示前 256 KB",
	},
	"share.preview_failed": {
		LocaleEN: "Preview failed — download to view the full file.",
		LocaleZH: "预览加载失败，请下载文件查看。",
	},
	"share.no_preview_title": {
		LocaleEN: "Inline preview unavailable",
		LocaleZH: "该文件无法在线预览",
	},
	"share.no_preview_body": {
		LocaleEN: "Browsers can't embed this file type — download a copy to open it locally.",
		LocaleZH: "浏览器不支持这种文件的内嵌预览，请下载到本地后打开。",
	},
	"share.open_new_tab": {
		LocaleEN: "Open in new tab",
		LocaleZH: "新标签页打开",
	},
	"share.link_formats": {
		LocaleEN: "Link formats",
		LocaleZH: "链接格式",
	},

	// ── Footer ──────────────────────────────────────────────────────────
	"footer.upload": {
		LocaleEN: "Upload",
		LocaleZH: "上传",
	},
	"footer.explore": {
		LocaleEN: "Explore",
		LocaleZH: "探索",
	},
	"footer.dashboard": {
		LocaleEN: "Dashboard",
		LocaleZH: "控制台",
	},
	"footer.login": {
		LocaleEN: "Sign in",
		LocaleZH: "登录",
	},

	// ── Public surfaces (gallery / guest upload toggles) ───────────────
	"public.gallery_disabled": {
		LocaleEN: "public gallery is disabled",
		LocaleZH: "公开画廊已关闭",
	},
	"public.guest_upload_disabled": {
		LocaleEN: "guest upload is disabled",
		LocaleZH: "游客上传已关闭",
	},

	// ── Setup wizard (template) ─────────────────────────────────────────
	"setup_page.title": {
		LocaleEN: "%s install wizard",
		LocaleZH: "%s 安装向导",
	},
	// Doc-title slug (no site-name interpolation) — used for the
	// browser tab title on the install wizard.
	"setup_page.doc_title": {
		LocaleEN: "Install wizard",
		LocaleZH: "安装向导",
	},
	"setup_page.subtitle": {
		LocaleEN: "Two minutes to set up database / site / admin / storage.",
		LocaleZH: "两分钟完成 数据库 / 站点 / 管理员 / 存储 配置",
	},
	"setup_page.step_db": {
		LocaleEN: "Database",
		LocaleZH: "数据库",
	},
	"setup_page.step_site": {
		LocaleEN: "Site",
		LocaleZH: "站点",
	},
	"setup_page.step_admin": {
		LocaleEN: "Admin",
		LocaleZH: "管理员",
	},
	"setup_page.step_storage": {
		LocaleEN: "Storage",
		LocaleZH: "存储",
	},
	"setup_page.db_section": {
		LocaleEN: "Database",
		LocaleZH: "数据库",
	},
	"setup_page.db_current": {
		LocaleEN: "Current %s · %s",
		LocaleZH: "当前 %s · %s",
	},
	"setup_page.db_current_prefix": {
		LocaleEN: "Current",
		LocaleZH: "当前",
	},
	"setup_page.db_intro": {
		LocaleEN: "SQLite needs no operations and fits a single user / small team. MySQL / PostgreSQL fit multi-instance deploys.",
		LocaleZH: "SQLite 单文件零运维，适合个人 / 小团队；MySQL / PostgreSQL 适合多实例部署。",
	},
	"setup_page.db_no_switch": {
		LocaleEN: "Read-only environment — set KITE_DB_DRIVER + KITE_DSN env vars and restart instead.",
		LocaleZH: "运行环境无法写配置文件，切换需设置 KITE_DB_DRIVER + KITE_DSN 后重启。",
	},
	"setup_page.driver_label": {
		LocaleEN: "Driver",
		LocaleZH: "驱动",
	},
	"setup_page.driver_sqlite_hint": {
		LocaleEN: "Default · single file · zero ops",
		LocaleZH: "默认 · 单文件 · 零运维",
	},
	"setup_page.driver_mysql_name": {
		LocaleEN: "MySQL / MariaDB",
		LocaleZH: "MySQL / MariaDB",
	},
	"setup_page.driver_mysql_hint": {
		LocaleEN: "Remote · multi-instance · compatible",
		LocaleZH: "远程 · 多实例 · 兼容",
	},
	"setup_page.driver_postgres_hint": {
		LocaleEN: "Remote · strong consistency · rich types",
		LocaleZH: "远程 · 强一致 · 富类型",
	},
	"setup_page.field_path": {
		LocaleEN: "Database file path",
		LocaleZH: "数据库文件路径",
	},
	"setup_page.field_host": {
		LocaleEN: "Host",
		LocaleZH: "主机",
	},
	"setup_page.field_port": {
		LocaleEN: "Port",
		LocaleZH: "端口",
	},
	"setup_page.field_user": {
		LocaleEN: "Username",
		LocaleZH: "用户名",
	},
	"setup_page.field_pass": {
		LocaleEN: "Password",
		LocaleZH: "密码",
	},
	"setup_page.field_db": {
		LocaleEN: "Database name",
		LocaleZH: "数据库名",
	},
	"setup_page.field_ssl": {
		LocaleEN: "SSL mode",
		LocaleZH: "SSL 模式",
	},
	"setup_page.btn_test": {
		LocaleEN: "Test connection",
		LocaleZH: "测试连接",
	},
	"setup_page.btn_save_restart": {
		LocaleEN: "Save and restart",
		LocaleZH: "保存配置并重启",
	},
	"setup_page.btn_next": {
		LocaleEN: "Next →",
		LocaleZH: "下一步 →",
	},
	"setup_page.btn_prev": {
		LocaleEN: "← Previous",
		LocaleZH: "← 上一步",
	},
	"setup_page.btn_finish": {
		LocaleEN: "Finish install ✓",
		LocaleZH: "完成安装 ✓",
	},
	"setup_page.site_section": {
		LocaleEN: "Site",
		LocaleZH: "站点信息",
	},
	"setup_page.site_intro": {
		LocaleEN: "Shown in the title bar and share links. You can change these later under Admin → Settings.",
		LocaleZH: "显示在标题栏与分享链接里。安装完成后可以在「后台 → 设置」修改。",
	},
	"setup_page.site_name_label": {
		LocaleEN: "Site name",
		LocaleZH: "站点名称",
	},
	"setup_page.site_url_label": {
		LocaleEN: "Site URL",
		LocaleZH: "站点 URL",
	},
	"setup_page.site_url_hint": {
		LocaleEN: "Used to build absolute share links · include scheme · no trailing slash",
		LocaleZH: "用于生成绝对外链，含协议且不带末尾斜杠",
	},
	"setup_page.admin_section": {
		LocaleEN: "Admin account",
		LocaleZH: "管理员账号",
	},
	"setup_page.admin_intro": {
		LocaleEN: "The first account that can sign into the admin panel. Don't lose it — password recovery requires SMTP.",
		LocaleZH: "第一个能登录后台的账号，务必牢记。找回密码需要先配置 SMTP。",
	},
	"setup_page.admin_username": {
		LocaleEN: "Username",
		LocaleZH: "用户名",
	},
	"setup_page.admin_email": {
		LocaleEN: "Email",
		LocaleZH: "邮箱",
	},
	"setup_page.admin_password": {
		LocaleEN: "Password",
		LocaleZH: "密码",
	},
	"setup_page.admin_password_hint": {
		LocaleEN: "At least 6 characters",
		LocaleZH: "至少 6 位",
	},
	"setup_page.admin_password_confirm": {
		LocaleEN: "Confirm password",
		LocaleZH: "确认密码",
	},
	"setup_page.storage_section": {
		LocaleEN: "Storage backend",
		LocaleZH: "存储后端",
	},
	"setup_page.storage_intro": {
		LocaleEN: "Where uploaded files land. Start with local storage; you can add S3 / OSS / FTP later under Admin → Storage.",
		LocaleZH: "上传文件落地位置。先用本地存储跑起来，安装完成后到「后台 → 存储」加 S3 / OSS / FTP。",
	},
	"setup_page.storage_type_label": {
		LocaleEN: "Type",
		LocaleZH: "类型",
	},
	"setup_page.storage_local": {
		LocaleEN: "Local storage (recommended)",
		LocaleZH: "本地存储（推荐）",
	},
	"setup_page.storage_path_label": {
		LocaleEN: "Storage path",
		LocaleZH: "存储路径",
	},
	"setup_page.storage_path_hint": {
		LocaleEN: "Relative to the working directory · created automatically",
		LocaleZH: "相对工作目录，不存在自动创建",
	},
	"setup_page.done_title": {
		LocaleEN: "Install complete",
		LocaleZH: "安装完成",
	},
	"setup_page.done_body": {
		LocaleEN: "Redirecting to the sign-in page…",
		LocaleZH: "即将跳转登录页…",
	},
	"setup_page.done_login_now": {
		LocaleEN: "Sign in now →",
		LocaleZH: "现在登录 →",
	},
	"setup_page.restart_title": {
		LocaleEN: "Database config saved — please restart",
		LocaleZH: "数据库配置已保存，请重启服务",
	},
	"setup_page.restart_body": {
		LocaleEN: "The new connection string was written to %s. Reopen this page after the restart to continue.",
		LocaleZH: "新的连接串已写入 %s。重启进程后再次打开本页面即可继续后续步骤。",
	},
	"setup_page.restart_body_pre": {
		LocaleEN: "The new connection string was written to ",
		LocaleZH: "新的连接串已写入 ",
	},
	"setup_page.restart_body_post": {
		LocaleEN: ". Reopen this page after the restart to continue.",
		LocaleZH: "。重启进程后再次打开本页面即可继续后续步骤。",
	},
	"setup_page.restart_run_directly": {
		LocaleEN: "Run directly",
		LocaleZH: "直接运行",
	},
	"setup_page.restart_run_directly_cmd": {
		LocaleEN: "After ^C, run ./kite again",
		LocaleZH: "^C 后再次执行 ./kite",
	},
	"setup_page.restart_done": {
		LocaleEN: "Restarted — refresh this page ↻",
		LocaleZH: "重启完成，刷新本页 ↻",
	},
	"setup_page.err_site_name_required": {
		LocaleEN: "Site name is required",
		LocaleZH: "请填写站点名称",
	},
	"setup_page.err_site_url_invalid": {
		LocaleEN: "Site URL must start with http:// or https://",
		LocaleZH: "站点 URL 需以 http:// 或 https:// 开头",
	},
	"setup_page.err_username_short": {
		LocaleEN: "Username must be at least 3 characters",
		LocaleZH: "用户名至少 3 个字符",
	},
	"setup_page.err_username_charset": {
		LocaleEN: "Username may only contain letters, digits, dots and hyphens",
		LocaleZH: "用户名只能包含字母、数字、点和连字符",
	},
	"setup_page.err_email_invalid": {
		LocaleEN: "Email format is invalid",
		LocaleZH: "邮箱格式不正确",
	},
	"setup_page.err_password_short": {
		LocaleEN: "Password must be at least 6 characters",
		LocaleZH: "密码至少 6 位",
	},
	"setup_page.err_password_mismatch": {
		LocaleEN: "Passwords do not match",
		LocaleZH: "两次输入的密码不一致",
	},
	"setup_page.err_dsn_required": {
		LocaleEN: "Please fill in the connection details",
		LocaleZH: "请填写连接信息",
	},
	"setup_page.status_connecting": {
		LocaleEN: "Connecting…",
		LocaleZH: "连接中…",
	},
	"setup_page.status_connected": {
		LocaleEN: "Connection ok ✓",
		LocaleZH: "连接成功 ✓",
	},
	"setup_page.status_connect_failed": {
		LocaleEN: "Connection failed",
		LocaleZH: "连接失败",
	},
	"setup_page.status_saving": {
		LocaleEN: "Saving…",
		LocaleZH: "保存中…",
	},
	"setup_page.status_save_failed": {
		LocaleEN: "Save failed",
		LocaleZH: "保存失败",
	},
	"setup_page.status_same_as_current": {
		LocaleEN: "Same as current — go to the next step",
		LocaleZH: "与当前一致，直接进入下一步",
	},
	"setup_page.status_installing": {
		LocaleEN: "Installing…",
		LocaleZH: "安装中…",
	},
	"setup_page.status_install_failed": {
		LocaleEN: "Install failed — please check the form",
		LocaleZH: "安装失败，请检查表单",
	},
	"setup_page.status_install_request_failed": {
		LocaleEN: "Install request failed",
		LocaleZH: "安装请求失败",
	},
	"setup_page.status_connect_anomaly": {
		LocaleEN: "Connection error",
		LocaleZH: "连接异常",
	},
	"setup_page.status_save_anomaly": {
		LocaleEN: "Save error",
		LocaleZH: "保存异常",
	},
	"setup_page.driver_sqlite_name": {
		LocaleEN: "SQLite",
		LocaleZH: "SQLite",
	},
	"setup_page.driver_postgres_name": {
		LocaleEN: "PostgreSQL",
		LocaleZH: "PostgreSQL",
	},
}
