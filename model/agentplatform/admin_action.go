package agentplatform

import (
	"time"

	"gorm.io/gorm"
)

type AdminAction struct {
	Id             int       `json:"id" gorm:"primaryKey"`
	ActorUserId    int       `json:"actor_user_id" gorm:"type:int;not null;index:idx_ap_admin_action_actor"`
	ActionType     string    `json:"action_type" gorm:"type:varchar(64);not null;index:idx_ap_admin_action_action"`
	ObjectType     string    `json:"object_type" gorm:"type:varchar(32);not null;index:idx_ap_admin_action_object"`
	ObjectId       string    `json:"object_id" gorm:"type:varchar(128);not null;index:idx_ap_admin_action_object"`
	BeforeStatus   string    `json:"before_status" gorm:"type:varchar(16);not null"`
	AfterStatus    string    `json:"after_status" gorm:"type:varchar(16);not null"`
	TargetVersion  string    `json:"target_version" gorm:"type:varchar(64);not null"`
	RequestId      string    `json:"request_id" gorm:"type:varchar(128);not null"`
	Result         string    `json:"result" gorm:"type:varchar(32);not null"`
	ErrorSummary   string    `json:"error_summary" gorm:"type:varchar(1024);not null"`
	Payload        string    `json:"payload" gorm:"type:text;not null"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AdminAction) TableName() string {
	return "agent_platform_admin_actions"
}

func (a *AdminAction) BeforeCreate(tx *gorm.DB) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if a.Payload == "" {
		a.Payload = "{}"
	}
	return nil
}
