package enterprise

type DingTalkConfig struct {
	Id              int    `json:"id" gorm:"primaryKey"`
	TenantId        int    `json:"tenant_id" gorm:"type:int;not null;default:0;uniqueIndex:uq_enterprise_dingtalk_configs_tenant"`
	CorpId          string `json:"corp_id" gorm:"type:varchar(128);not null;default:''"`
	AppKey          string `json:"app_key" gorm:"type:varchar(128);not null;default:''"`
	AppSecret       string `json:"-" gorm:"type:varchar(512);not null;default:''"`
	CallbackUrl     string `json:"callback_url" gorm:"type:varchar(1024);not null;default:''"`
	SyncScope       string `json:"sync_scope" gorm:"type:text;not null"`
	LoginEnabled    bool   `json:"login_enabled" gorm:"not null;default:false"`
	SyncEnabled     bool   `json:"sync_enabled" gorm:"not null;default:false"`
	AutoSyncOnLogin bool   `json:"auto_sync_on_login" gorm:"type:boolean;not null;default:false"`
	CreatedAt       int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt       int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (DingTalkConfig) TableName() string {
	return "enterprise_dingtalk_configs"
}
