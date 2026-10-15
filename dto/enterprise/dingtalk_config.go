package enterprise

type DingTalkConfigRequest struct {
	TenantId                  *int    `json:"tenant_id,omitempty"`
	CorpId                    string  `json:"corp_id"`
	AppKey                    string  `json:"app_key"`
	AppSecret                 *string `json:"app_secret,omitempty"`
	CallbackUrl               string  `json:"callback_url"`
	SyncScope                 string  `json:"sync_scope"`
	LoginEnabled              *bool   `json:"login_enabled,omitempty"`
	SyncEnabled               *bool   `json:"sync_enabled,omitempty"`
	AutoSyncOnLogin           *bool   `json:"auto_sync_on_login,omitempty"`
	ScheduledFullSyncEnabled  *bool   `json:"scheduled_full_sync_enabled,omitempty"`
	ScheduledFullSyncCron     string  `json:"scheduled_full_sync_cron,omitempty"`
	ScheduledFullSyncTimezone string  `json:"scheduled_full_sync_timezone,omitempty"`
}

type DingTalkConfigResponse struct {
	Id                          int    `json:"id"`
	TenantId                    int    `json:"tenant_id"`
	CorpId                      string `json:"corp_id"`
	AppKey                      string `json:"app_key"`
	CallbackUrl                 string `json:"callback_url"`
	SyncScope                   string `json:"sync_scope"`
	LoginEnabled                bool   `json:"login_enabled"`
	SyncEnabled                 bool   `json:"sync_enabled"`
	AutoSyncOnLogin             bool   `json:"auto_sync_on_login"`
	ScheduledFullSyncEnabled    bool   `json:"scheduled_full_sync_enabled"`
	ScheduledFullSyncCron       string `json:"scheduled_full_sync_cron"`
	ScheduledFullSyncTimezone   string `json:"scheduled_full_sync_timezone"`
	ScheduledFullSyncNextRunAt  int64  `json:"scheduled_full_sync_next_run_at"`
	ScheduledFullSyncLastRunAt  int64  `json:"scheduled_full_sync_last_run_at"`
	ScheduledFullSyncLastTaskId int    `json:"scheduled_full_sync_last_task_id"`
	ScheduledFullSyncLastStatus string `json:"scheduled_full_sync_last_status"`
	ScheduledFullSyncLastError  string `json:"scheduled_full_sync_last_error"`
	ScheduledFullSyncRevision   int64  `json:"scheduled_full_sync_revision"`
	ScheduledFullSyncUpdatedAt  int64  `json:"scheduled_full_sync_updated_at"`
	HasAppSecret                bool   `json:"has_app_secret"`
	CreatedAt                   int64  `json:"created_at"`
	UpdatedAt                   int64  `json:"updated_at"`
}
