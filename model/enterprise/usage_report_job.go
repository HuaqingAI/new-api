package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	UsageReportFrequencyDaily   = "daily"
	UsageReportFrequencyWeekly  = "weekly"
	UsageReportFrequencyMonthly = "monthly"

	UsageReportRangeToday      = "today"
	UsageReportRangeLast7Days  = "last7d"
	UsageReportRangeLast30Days = "last30d"

	UsageReportStatusPending = "pending"
	UsageReportStatusRunning = "running"
	UsageReportStatusSuccess = "success"
	UsageReportStatusFailed  = "failed"
)

type UsageReportTopDepartment struct {
	DeptId       *int   `json:"dept_id"`
	DeptName     string `json:"dept_name"`
	RequestCount int64  `json:"request_count"`
	Quota        int64  `json:"quota"`
	UserCount    int64  `json:"user_count"`
}

type UsageReportGrowthDepartment struct {
	DeptId               *int    `json:"dept_id"`
	DeptName             string  `json:"dept_name"`
	RequestCount         int64   `json:"request_count"`
	PreviousRequestCount int64   `json:"previous_request_count"`
	Quota                int64   `json:"quota"`
	PreviousQuota        int64   `json:"previous_quota"`
	RequestGrowthRate    float64 `json:"request_growth_rate"`
	QuotaGrowthRate      float64 `json:"quota_growth_rate"`
}

type UsageReportSnapshot struct {
	WindowStart         int64                         `json:"window_start"`
	WindowEnd           int64                         `json:"window_end"`
	DepartmentCount     int64                         `json:"department_count"`
	RequestCount        int64                         `json:"request_count"`
	PromptTokens        int64                         `json:"prompt_tokens"`
	CompletionTokens    int64                         `json:"completion_tokens"`
	Quota               int64                         `json:"quota"`
	UserCount           int64                         `json:"user_count"`
	TopDepartments      []UsageReportTopDepartment    `json:"top_departments"`
	GrowthDepartments   []UsageReportGrowthDepartment `json:"growth_departments"`
	PreviousWindowStart int64                         `json:"previous_window_start"`
	PreviousWindowEnd   int64                         `json:"previous_window_end"`
}

type UsageReportJob struct {
	Id              int    `json:"id" gorm:"primaryKey"`
	TenantId        int    `json:"tenant_id" gorm:"type:int;not null;default:0;uniqueIndex:uq_usage_report_job_tenant,priority:1"`
	Receivers       string `json:"receivers" gorm:"type:text;not null"`
	Frequency       string `json:"frequency" gorm:"type:varchar(32);not null;default:'daily'"`
	RangeType       string `json:"range_type" gorm:"type:varchar(32);not null;default:'last7d'"`
	Enabled         bool   `json:"enabled" gorm:"not null;default:true"`
	Status          string `json:"status" gorm:"type:varchar(32);not null;default:'pending';index:idx_usage_report_jobs_status"`
	LastRunAt       int64  `json:"last_run_at" gorm:"type:bigint;not null;default:0"`
	NextRunAt       int64  `json:"next_run_at" gorm:"type:bigint;not null;default:0;index:idx_usage_report_jobs_next_run"`
	LastSuccessAt   int64  `json:"last_success_at" gorm:"type:bigint;not null;default:0"`
	LastWindowStart int64  `json:"last_window_start" gorm:"type:bigint;not null;default:0"`
	LastWindowEnd   int64  `json:"last_window_end" gorm:"type:bigint;not null;default:0"`
	RunCount        int64  `json:"run_count" gorm:"type:bigint;not null;default:0"`
	FailureCount    int64  `json:"failure_count" gorm:"type:bigint;not null;default:0"`
	ErrorReason     string `json:"error_reason" gorm:"type:text;not null"`
	LastSnapshot    string `json:"last_snapshot" gorm:"type:text;not null"`
	CreatedAt       int64  `json:"created_at" gorm:"type:bigint;not null;default:0"`
	UpdatedAt       int64  `json:"updated_at" gorm:"type:bigint;not null;default:0"`
}

func (UsageReportJob) TableName() string {
	return "enterprise_usage_report_jobs"
}

func (j *UsageReportJob) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if j.CreatedAt == 0 {
		j.CreatedAt = now
	}
	if j.UpdatedAt == 0 {
		j.UpdatedAt = now
	}
	return j.normalize()
}

func (j *UsageReportJob) BeforeUpdate(tx *gorm.DB) error {
	j.UpdatedAt = time.Now().Unix()
	return j.normalize()
}

func (j *UsageReportJob) SetReceivers(receivers []string) error {
	if receivers == nil {
		receivers = []string{}
	}
	data, err := common.Marshal(receivers)
	if err != nil {
		return err
	}
	j.Receivers = string(data)
	return nil
}

func (j UsageReportJob) ParsedReceivers() ([]string, error) {
	if j.Receivers == "" {
		return []string{}, nil
	}
	var receivers []string
	if err := common.UnmarshalJsonStr(j.Receivers, &receivers); err != nil {
		return nil, err
	}
	if receivers == nil {
		return []string{}, nil
	}
	return receivers, nil
}

func (j *UsageReportJob) SetLastSnapshot(snapshot *UsageReportSnapshot) error {
	if snapshot == nil {
		data, err := common.Marshal(map[string]any{})
		if err != nil {
			return err
		}
		j.LastSnapshot = string(data)
		return nil
	}
	data, err := common.Marshal(snapshot)
	if err != nil {
		return err
	}
	j.LastSnapshot = string(data)
	return nil
}

func (j UsageReportJob) ParsedLastSnapshot() (*UsageReportSnapshot, error) {
	if j.LastSnapshot == "" {
		return nil, nil
	}
	var snapshot UsageReportSnapshot
	if err := common.UnmarshalJsonStr(j.LastSnapshot, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (j *UsageReportJob) normalize() error {
	if j.Receivers == "" {
		if err := j.SetReceivers(nil); err != nil {
			return err
		}
	} else {
		receivers, err := j.ParsedReceivers()
		if err != nil {
			return err
		}
		if err := j.SetReceivers(receivers); err != nil {
			return err
		}
	}
	if j.LastSnapshot == "" {
		if err := j.SetLastSnapshot(nil); err != nil {
			return err
		}
	} else {
		snapshot, err := j.ParsedLastSnapshot()
		if err != nil {
			return err
		}
		if err := j.SetLastSnapshot(snapshot); err != nil {
			return err
		}
	}
	if j.Frequency == "" {
		j.Frequency = UsageReportFrequencyDaily
	}
	if j.RangeType == "" {
		j.RangeType = UsageReportRangeLast7Days
	}
	if j.Status == "" {
		j.Status = UsageReportStatusPending
	}
	return nil
}
