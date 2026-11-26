package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const UsageSnapshotUnassignedDeptName = "未归属"

type UsageSnapshotModelStat struct {
	ModelName        string `json:"model_name"`
	RequestCount     int64  `json:"request_count"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	Quota            int64  `json:"quota"`
}

type UsageSnapshot struct {
	Id                int    `json:"id" gorm:"primaryKey"`
	TenantId          int    `json:"tenant_id" gorm:"type:int;not null;default:0;index:idx_usage_snapshots_tenant_window,priority:1;index:idx_usage_snapshots_tenant_dept_window,priority:1;uniqueIndex:uq_usage_snapshots_bucket,priority:1"`
	DeptId            *int   `json:"dept_id" gorm:"type:int;index:idx_usage_snapshots_tenant_dept_window,priority:2;uniqueIndex:uq_usage_snapshots_bucket,priority:2"`
	DeptName          string `json:"dept_name" gorm:"type:varchar(255);not null;default:'';uniqueIndex:uq_usage_snapshots_bucket,priority:3"`
	WindowStart       int64  `json:"window_start" gorm:"type:bigint;not null;index:idx_usage_snapshots_tenant_window,priority:2;index:idx_usage_snapshots_tenant_dept_window,priority:3;uniqueIndex:uq_usage_snapshots_bucket,priority:4"`
	WindowEnd         int64  `json:"window_end" gorm:"type:bigint;not null;index:idx_usage_snapshots_tenant_window,priority:3;index:idx_usage_snapshots_tenant_dept_window,priority:4;uniqueIndex:uq_usage_snapshots_bucket,priority:5"`
	RequestCount      int64  `json:"request_count" gorm:"type:bigint;not null;default:0"`
	PromptTokens      int64  `json:"prompt_tokens" gorm:"type:bigint;not null;default:0"`
	CompletionTokens  int64  `json:"completion_tokens" gorm:"type:bigint;not null;default:0"`
	Quota             int64  `json:"quota" gorm:"type:bigint;not null;default:0"`
	UserCount         int64  `json:"user_count" gorm:"type:bigint;not null;default:0"`
	ModelDistribution string `json:"model_distribution" gorm:"type:text;not null"`
	UserIds           string `json:"-" gorm:"column:user_ids;type:text;not null"`
	CreatedAt         int64  `json:"created_at" gorm:"type:bigint;not null;default:0"`
	UpdatedAt         int64  `json:"updated_at" gorm:"type:bigint;not null;default:0"`
}

func (UsageSnapshot) TableName() string {
	return "enterprise_usage_snapshots"
}

func (s *UsageSnapshot) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if s.CreatedAt == 0 {
		s.CreatedAt = now
	}
	if s.UpdatedAt == 0 {
		s.UpdatedAt = now
	}
	return s.normalize()
}

func (s *UsageSnapshot) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = time.Now().Unix()
	return s.normalize()
}

func (s *UsageSnapshot) SetModelDistribution(stats []UsageSnapshotModelStat) error {
	if stats == nil {
		stats = []UsageSnapshotModelStat{}
	}
	data, err := common.Marshal(stats)
	if err != nil {
		return err
	}
	s.ModelDistribution = string(data)
	return nil
}

func (s UsageSnapshot) ParsedModelDistribution() ([]UsageSnapshotModelStat, error) {
	if s.ModelDistribution == "" {
		return []UsageSnapshotModelStat{}, nil
	}
	var stats []UsageSnapshotModelStat
	if err := common.UnmarshalJsonStr(s.ModelDistribution, &stats); err != nil {
		return nil, err
	}
	if stats == nil {
		return []UsageSnapshotModelStat{}, nil
	}
	return stats, nil
}

func (s *UsageSnapshot) SetUserIds(userIds []int) error {
	if userIds == nil {
		userIds = []int{}
	}
	data, err := common.Marshal(userIds)
	if err != nil {
		return err
	}
	s.UserIds = string(data)
	s.UserCount = int64(len(userIds))
	return nil
}

func (s UsageSnapshot) ParsedUserIds() ([]int, error) {
	if s.UserIds == "" {
		return []int{}, nil
	}
	var userIds []int
	if err := common.UnmarshalJsonStr(s.UserIds, &userIds); err != nil {
		return nil, err
	}
	if userIds == nil {
		return []int{}, nil
	}
	return userIds, nil
}

func (s *UsageSnapshot) normalize() error {
	if s.DeptId == nil && s.DeptName == "" {
		s.DeptName = UsageSnapshotUnassignedDeptName
	}
	if s.ModelDistribution == "" {
		if err := s.SetModelDistribution(nil); err != nil {
			return err
		}
	} else {
		stats, err := s.ParsedModelDistribution()
		if err != nil {
			return err
		}
		if err := s.SetModelDistribution(stats); err != nil {
			return err
		}
	}
	if s.UserIds == "" {
		return s.SetUserIds(nil)
	}
	userIds, err := s.ParsedUserIds()
	if err != nil {
		return err
	}
	return s.SetUserIds(userIds)
}
