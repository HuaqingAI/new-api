package enterprise

type DingTalkIdentity struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	TenantId       int    `json:"tenant_id" gorm:"type:int;not null;default:0;index;uniqueIndex:uq_enterprise_dingtalk_identity_key;uniqueIndex:uq_enterprise_dingtalk_identity_user"`
	CorpId         string `json:"corp_id" gorm:"type:varchar(128);not null;default:'';index"`
	IdentityKey    string `json:"-" gorm:"type:varchar(160);not null;default:'';uniqueIndex:uq_enterprise_dingtalk_identity_key"`
	UnionId        string `json:"union_id" gorm:"type:varchar(128);not null;default:'';index"`
	OpenId         string `json:"open_id" gorm:"type:varchar(128);not null;default:'';index"`
	ExternalUserId string `json:"external_user_id" gorm:"type:varchar(128);not null;default:'';index"`
	UserId         int    `json:"user_id" gorm:"type:int;not null;index;uniqueIndex:uq_enterprise_dingtalk_identity_user"`
	Status         int    `json:"status" gorm:"type:int;not null;default:1;index"`
	LastLoginAt    int64  `json:"last_login_at" gorm:"type:bigint;not null;default:0"`
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt      int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (DingTalkIdentity) TableName() string {
	return "enterprise_dingtalk_identities"
}
