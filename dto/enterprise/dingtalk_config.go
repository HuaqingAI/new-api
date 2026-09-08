package enterprise

type DingTalkConfigRequest struct {
	TenantId        *int    `json:"tenant_id,omitempty"`
	CorpId          string  `json:"corp_id"`
	AppKey          string  `json:"app_key"`
	AppSecret       *string `json:"app_secret,omitempty"`
	CallbackUrl     string  `json:"callback_url"`
	SyncScope       string  `json:"sync_scope"`
	LoginEnabled    *bool   `json:"login_enabled,omitempty"`
	SyncEnabled     *bool   `json:"sync_enabled,omitempty"`
	AutoSyncOnLogin *bool   `json:"auto_sync_on_login,omitempty"`
}

type DingTalkConfigResponse struct {
	Id              int    `json:"id"`
	TenantId        int    `json:"tenant_id"`
	CorpId          string `json:"corp_id"`
	AppKey          string `json:"app_key"`
	CallbackUrl     string `json:"callback_url"`
	SyncScope       string `json:"sync_scope"`
	LoginEnabled    bool   `json:"login_enabled"`
	SyncEnabled     bool   `json:"sync_enabled"`
	AutoSyncOnLogin bool   `json:"auto_sync_on_login"`
	HasAppSecret    bool   `json:"has_app_secret"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}
