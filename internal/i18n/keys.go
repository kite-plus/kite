package i18n

// This file enumerates the catalogue keys referenced from Go source. Keys
// that only show up in templates can stay as raw strings — the template
// author is referencing the English fragment they typed anyway, so a typo
// surfaces immediately.
//
// Constants here exist for two reasons:
//   - Refactoring safety: renaming a key is a single edit + compile failures
//     for every stale reference, instead of grepping for string literals.
//   - Discoverability: a Go developer adding a new error message can browse
//     this file to see what's already covered before inventing a duplicate.
//
// Keep the constant order aligned with [Catalog] (same domain blocks) so
// drift between the two files is visible at a glance during review.

// Generic envelope.
const (
	KeySuccess              = "common.success"
	KeyErrBadRequest        = "err.bad_request"
	KeyErrUnauthorized      = "err.unauthorized"
	KeyErrForbidden         = "err.forbidden"
	KeyErrNotFound          = "err.not_found"
	KeyErrInternal          = "err.internal"
	KeyErrInvalidRequest    = "err.invalid_request"
	KeyErrInvalidPayload    = "err.invalid_payload"
	KeyErrMissingUserCtx    = "err.missing_user_context"
	KeyErrMissingRequestCtx = "err.missing_request_context"
)

// Auth.
const (
	KeyAuthUsernamePasswordRequired = "auth.username_password_required"
	KeyAuthLoginFailed              = "auth.login_failed"
	KeyAuthInvalidRefreshToken      = "auth.invalid_refresh_token"
	KeyAuthRefreshTokenRequired     = "auth.refresh_token_required"
	KeyAuthRefreshSessionFailed     = "auth.refresh_session_failed"
	KeyAuthInvalidRegistration      = "auth.invalid_registration"
	KeyAuthRegistrationFailed       = "auth.registration_failed"
	KeyAuthInvalidProfile           = "auth.invalid_profile"
	KeyAuthUpdateProfileFailed      = "auth.update_profile_failed"
	KeyAuthInvalidPassword          = "auth.invalid_password"
	KeyAuthChangePasswordFailed     = "auth.change_password_failed"
	KeyAuthSetPasswordFailed        = "auth.set_password_failed"
	KeyAuthHashPasswordFailed       = "auth.hash_password_failed"
	KeyAuthRevokeSessionsFailed     = "auth.revoke_sessions_failed"
	KeyAuthLoadCurrentUserFailed    = "auth.load_current_user_failed"
	KeyAuthReloadUserFailed         = "auth.reload_user_failed"
	KeyAuthSaveNicknameFailed       = "auth.save_nickname_failed"
	KeyAuthUserNotFound             = "auth.user_not_found"
	KeyAuthIdentifierRequired       = "auth.identifier_required"
)

// 2FA.
const (
	KeyTwoFACodeRequired            = "twofa.code_required"
	KeyTwoFAPasswordAndCodeRequired = "twofa.password_and_code_required"
	KeyTwoFACodeIncorrect           = "twofa.code_incorrect"
	KeyTwoFACodeExpired             = "twofa.code_expired"
	KeyTwoFAStartSetupFailed        = "twofa.start_setup_failed"
	KeyTwoFAEnableFailed            = "twofa.enable_failed"
	KeyTwoFADisableFailed           = "twofa.disable_failed"
	KeyTwoFAVerifyFailed            = "twofa.verify_failed"
	KeyTwoFAResetFailed             = "twofa.reset_failed"
	KeyTwoFAChallengeRequired       = "twofa.challenge_required"
)

// Email / verification / password reset.
const (
	KeyEmailInvalid                  = "email.invalid"
	KeyEmailInvalidAddress           = "email.invalid_address"
	KeyEmailAlreadyRegistered        = "email.already_registered"
	KeyEmailNewMatchesCurrent        = "email.new_matches_current"
	KeyEmailNoPendingVerification    = "email.no_pending_verification"
	KeyEmailNoPendingReset           = "email.no_pending_reset"
	KeyEmailSendVerificationFailed   = "email.send_verification_failed"
	KeyEmailSendResetFailed          = "email.send_reset_failed"
	KeyEmailConfirmChangeFailed      = "email.confirm_change_failed"
	KeyEmailConfirmResetFailed       = "email.confirm_reset_failed"
	KeyEmailInvalidResetData         = "email.invalid_reset_data"
	KeyEmailChangeServiceUnavailable = "email.change_service_unavailable"
	KeyEmailPasswordResetUnavailable = "email.password_reset_unavailable"
	KeyEmailServiceNotConfigured     = "email.service_not_configured"
	KeyEmailRateLimited              = "email.rate_limited"
	KeyEmailTestSendFailed           = "email.test_send_failed"
	KeyEmailValidateFailed           = "email.validate_failed"
)

// OAuth / social login.
const (
	KeyOAuthInvalidProvider      = "oauth.invalid_provider"
	KeyOAuthUpdateProviderFailed = "oauth.update_provider_failed"
	KeyOAuthLoadProvidersFailed  = "oauth.load_providers_failed"
	KeyOAuthExchangeFailed       = "oauth.exchange_failed"
	KeyOAuthInvalidOnboarding    = "oauth.invalid_onboarding"
	KeyOAuthOnboardingFailed     = "oauth.onboarding_failed"
	KeyOAuthTicketRequired       = "oauth.ticket_required"
	KeyOAuthSocialNotConfigured  = "oauth.social_not_configured"
	KeyOAuthListIdentitiesFailed = "oauth.list_identities_failed"
	KeyOAuthUnlinkIdentityFailed = "oauth.unlink_identity_failed"
	KeyOAuthIdentityNotFound     = "oauth.identity_not_found"
)

// Public surfaces (anonymous gallery / guest upload toggles).
const (
	KeyPublicGalleryDisabled     = "public.gallery_disabled"
	KeyPublicGuestUploadDisabled = "public.guest_upload_disabled"
)

// Setup wizard (handler messages).
const (
	KeySetupAlreadyInitialized      = "setup.already_initialized"
	KeySetupInvalidData             = "setup.invalid_data"
	KeySetupSaveSiteSettingsFailed  = "setup.save_site_settings_failed"
	KeySetupCreateAdminFailed       = "setup.create_admin_failed"
	KeySetupInvalidStorage          = "setup.invalid_storage"
	KeySetupStorageValidationFailed = "setup.storage_validation_failed"
	KeySetupCreateStorageFailed     = "setup.create_storage_failed"
	KeySetupInvalidDatabase         = "setup.invalid_database"
	KeySetupSaveDatabaseFailed      = "setup.save_database_failed"
)

// Files / albums / storage.
const (
	KeyFileNotFound            = "file.not_found"
	KeyFileThumbnailNotFound   = "file.thumbnail_not_found"
	KeyFileReadFailed          = "file.read_failed"
	KeyFileDeleteFailed        = "file.delete_failed"
	KeyFileMoveFailed          = "file.move_failed"
	KeyFileListFailed          = "file.list_failed"
	KeyFileNotOwner            = "file.not_owner"
	KeyFileInvalidTargetFolder = "file.invalid_target_folder"
	KeyFileIDsRequired         = "file.ids_required"

	KeyAlbumNotFound            = "album.not_found"
	KeyAlbumFolderNotFound      = "album.folder_not_found"
	KeyAlbumNotOwner            = "album.not_owner"
	KeyAlbumInvalidData         = "album.invalid_data"
	KeyAlbumInvalidParentFolder = "album.invalid_parent_folder"
	KeyAlbumFolderSelfParent    = "album.folder_self_parent"
	KeyAlbumDescendantMove      = "album.descendant_move"
	KeyAlbumCreateFailed        = "album.create_failed"
	KeyAlbumUpdateFailed        = "album.update_failed"
	KeyAlbumDeleteFailed        = "album.delete_failed"
	KeyAlbumListFailed          = "album.list_failed"
	KeyAlbumLoadFolderFailed    = "album.load_folder_failed"

	KeyStorageNotFound          = "storage.not_found"
	KeyStorageInvalidConfig     = "storage.invalid_config"
	KeyStorageParseConfigFailed = "storage.parse_config_failed"
	KeyStorageCreateFailed      = "storage.create_failed"
	KeyStorageUpdateFailed      = "storage.update_failed"
	KeyStorageDeleteFailed      = "storage.delete_failed"
	KeyStorageListFailed        = "storage.list_failed"
	KeyStorageSetDefaultFailed  = "storage.set_default_failed"
	KeyStorageReorderFailed     = "storage.reorder_failed"
	KeyStorageInvalidReorder    = "storage.invalid_reorder"
	KeyStorageOrderedIDsEmpty   = "storage.ordered_ids_empty"
)

// Tokens.
const (
	KeyTokenInvalidData  = "token.invalid_data"
	KeyTokenCreateFailed = "token.create_failed"
	KeyTokenListFailed   = "token.list_failed"
	KeyTokenNotFound     = "token.not_found"
)

// Users / settings.
const (
	KeyUserNotFound     = "user.not_found"
	KeyUserInvalidData  = "user.invalid_data"
	KeyUserCreateFailed = "user.create_failed"
	KeyUserUpdateFailed = "user.update_failed"
	KeyUserDeleteFailed = "user.delete_failed"
	KeyUserListFailed   = "user.list_failed"

	KeySettingsInvalidData               = "settings.invalid_data"
	KeySettingsInvalidValue              = "settings.invalid_value"
	KeySettingsInvalidEmpty              = "settings.invalid_empty"
	KeySettingsGetFailed                 = "settings.get_failed"
	KeySettingsUpdateFailed              = "settings.update_failed"
	KeySettingsCannotModify              = "settings.cannot_modify"
	KeySettingsCannotBeEmpty             = "settings.cannot_be_empty"
	KeySettingsInvalidPathPattern        = "settings.invalid_path_pattern"
	KeySettingsInvalidMaxFileSize        = "settings.invalid_max_file_size"
	KeySettingsInvalidDangerousExtension = "settings.invalid_dangerous_extensions"
	KeySettingsInvalidDangerousRename    = "settings.invalid_dangerous_rename"
	KeySettingsCapacityNegative          = "settings.capacity_negative"
	KeySettingsAccessStatsFailed         = "settings.access_stats_failed"
	KeySettingsAccessHeatmapFailed       = "settings.access_heatmap_failed"
	KeySettingsUploadStatsFailed         = "settings.upload_stats_failed"
	KeySettingsUploadHeatmapFailed       = "settings.upload_heatmap_failed"
	KeySettingsStatsFailed               = "settings.stats_failed"
)
