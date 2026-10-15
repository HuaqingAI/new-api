package enterprise

type DingTalkSyncTask struct {
	Id                  int     `json:"id" gorm:"primaryKey"`
	TenantId            int     `json:"tenant_id" gorm:"type:int;not null;default:0;index;uniqueIndex:uq_enterprise_dingtalk_sync_tasks_scheduled_occurrence,priority:1;uniqueIndex:uq_enterprise_dingtalk_sync_tasks_active,priority:1"`
	Mode                string  `json:"mode" gorm:"type:varchar(32);not null;default:'full'"`
	Status              string  `json:"status" gorm:"type:varchar(32);not null;default:'pending';index"`
	Progress            int     `json:"progress" gorm:"type:int;not null;default:0"`
	DepartmentsCreated  int     `json:"departments_created" gorm:"type:int;not null;default:0"`
	DepartmentsUpdated  int     `json:"departments_updated" gorm:"type:int;not null;default:0"`
	DepartmentsDisabled int     `json:"departments_disabled" gorm:"type:int;not null;default:0"`
	UsersCreated        int     `json:"users_created" gorm:"type:int;not null;default:0"`
	UsersUpdated        int     `json:"users_updated" gorm:"type:int;not null;default:0"`
	MembershipsCreated  int     `json:"memberships_created" gorm:"type:int;not null;default:0"`
	MembershipsUpdated  int     `json:"memberships_updated" gorm:"type:int;not null;default:0"`
	MembershipsDisabled int     `json:"memberships_disabled" gorm:"type:int;not null;default:0"`
	SkippedCount        int     `json:"skipped_count" gorm:"type:int;not null;default:0"`
	FailedCount         int     `json:"failed_count" gorm:"type:int;not null;default:0"`
	ErrorSummary        string  `json:"error_summary" gorm:"type:varchar(1024);not null;default:''"`
	CreatedBy           int     `json:"created_by" gorm:"type:int;not null;default:0;index"`
	StartedAt           int64   `json:"started_at" gorm:"type:bigint;not null;default:0"`
	FinishedAt          int64   `json:"finished_at" gorm:"type:bigint;not null;default:0"`
	CreatedAt           int64   `json:"created_at" gorm:"autoCreateTime;column:created_at;index"`
	UpdatedAt           int64   `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
	TriggerSource       string  `json:"trigger_source" gorm:"type:varchar(32);not null;default:'manual';index"`
	ScheduledFor        *int64  `json:"scheduled_for,omitempty" gorm:"type:bigint;uniqueIndex:uq_enterprise_dingtalk_sync_tasks_scheduled_occurrence,priority:2"`
	ScheduleRevision    int64   `json:"schedule_revision" gorm:"type:bigint;not null;default:0"`
	ActiveKey           *string `json:"active_key,omitempty" gorm:"type:varchar(32);uniqueIndex:uq_enterprise_dingtalk_sync_tasks_active,priority:2"`
	RunnerId            string  `json:"runner_id" gorm:"type:varchar(128);not null;default:''"`
}

func (DingTalkSyncTask) TableName() string {
	return "enterprise_dingtalk_sync_tasks"
}

type DingTalkSyncLog struct {
	Id               int    `json:"id" gorm:"primaryKey"`
	TaskId           int    `json:"task_id" gorm:"type:int;not null;index"`
	TenantId         int    `json:"tenant_id" gorm:"type:int;not null;default:0;index"`
	ObjectType       string `json:"object_type" gorm:"type:varchar(32);not null;default:'';index"`
	ObjectExternalId string `json:"object_external_id" gorm:"type:varchar(128);not null;default:'';index"`
	Action           string `json:"action" gorm:"type:varchar(32);not null;default:'';index"`
	Status           string `json:"status" gorm:"type:varchar(32);not null;default:'';index"`
	Message          string `json:"message" gorm:"type:varchar(1024);not null;default:''"`
	CreatedAt        int64  `json:"created_at" gorm:"autoCreateTime;column:created_at;index"`
}

func (DingTalkSyncLog) TableName() string {
	return "enterprise_dingtalk_sync_logs"
}
