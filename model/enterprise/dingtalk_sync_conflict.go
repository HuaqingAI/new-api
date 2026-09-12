package enterprise

type DingTalkSyncConflict struct {
	Id              int    `json:"id" gorm:"primaryKey"`
	TenantId        int    `json:"tenant_id" gorm:"type:int;not null;default:0;index;uniqueIndex:uq_enterprise_dingtalk_sync_conflict"`
	TaskId          int    `json:"task_id" gorm:"type:int;not null;default:0;index"`
	TriggerSource   string `json:"trigger_source" gorm:"type:varchar(32);not null;default:'full_sync';index"`
	ExternalUserId  string `json:"external_user_id" gorm:"type:varchar(128);not null;default:'';index;uniqueIndex:uq_enterprise_dingtalk_sync_conflict"`
	UnionId         string `json:"union_id" gorm:"type:varchar(128);not null;default:'';index"`
	Mobile          string `json:"mobile" gorm:"type:varchar(64);not null;default:'';index"`
	Email           string `json:"email" gorm:"type:varchar(255);not null;default:'';index"`
	Name            string `json:"name" gorm:"type:varchar(255);not null;default:''"`
	ConflictType    string `json:"conflict_type" gorm:"type:varchar(64);not null;default:'';index;uniqueIndex:uq_enterprise_dingtalk_sync_conflict"`
	CandidateUserId int    `json:"candidate_user_id" gorm:"type:int;not null;default:0;index"`
	Details         string `json:"details" gorm:"type:varchar(1024);not null;default:''"`
	Status          string `json:"status" gorm:"type:varchar(32);not null;default:'pending';index"`
	LastTaskId      int    `json:"last_task_id" gorm:"type:int;not null;default:0;index"`
	ResolvedBy      int    `json:"resolved_by" gorm:"type:int;not null;default:0"`
	ResolvedAt      int64  `json:"resolved_at" gorm:"type:bigint;not null;default:0"`
	CreatedAt       int64  `json:"created_at" gorm:"autoCreateTime;column:created_at;index"`
	UpdatedAt       int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (DingTalkSyncConflict) TableName() string {
	return "enterprise_dingtalk_sync_conflicts"
}
