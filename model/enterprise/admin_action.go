package enterprise

import (
	"time"

	"gorm.io/gorm"
)

type AdminAction struct {
	ActionId    int    `json:"action_id" gorm:"column:action_id;primaryKey;autoIncrement"`
	TenantId    int    `json:"tenant_id" gorm:"type:int;not null;default:0;index:idx_ent_admin_actions_tenant"`
	ActorId     int    `json:"actor_id" gorm:"type:int;not null;index:idx_ent_admin_actions_actor"`
	ActionType  string `json:"action_type" gorm:"type:varchar(96);not null;index:idx_ent_admin_actions_action"`
	ObjectType  string `json:"object_type" gorm:"type:varchar(64);not null;index:idx_ent_admin_actions_object"`
	ObjectId    string `json:"object_id" gorm:"type:varchar(128);not null;default:'';index:idx_ent_admin_actions_object"`
	DiffSummary string `json:"diff_summary" gorm:"type:varchar(1024);not null;default:''"`
	Payload     string `json:"payload" gorm:"type:text;not null;default:'{}'"`
	CreatedAt   int64  `json:"created_at" gorm:"type:bigint;not null;default:0;index:idx_ent_admin_actions_created"`
}

func (AdminAction) TableName() string {
	return "enterprise_admin_actions"
}

func (a *AdminAction) BeforeCreate(tx *gorm.DB) error {
	if a.CreatedAt == 0 {
		a.CreatedAt = time.Now().Unix()
	}
	return nil
}
