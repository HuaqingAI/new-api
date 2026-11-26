package enterprise

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	UsageScopeTypeTenant     = "tenant"
	UsageScopeTypeDepartment = "department"
)

// UsageScopeSnapshot stores one de-duplicated usage bucket for a tenant or a
// department subtree. Direct department attribution remains in UsageSnapshot.
type UsageScopeSnapshot struct {
	Id                int    `json:"id" gorm:"primaryKey"`
	TenantId          int    `json:"tenant_id" gorm:"type:int;not null;default:0;index:idx_usage_scope_snapshots_tenant_window,priority:1;index:idx_usage_scope_snapshots_tenant_dept_window,priority:1;uniqueIndex:uq_usage_scope_snapshots_bucket,priority:1"`
	ScopeKey          string `json:"scope_key" gorm:"type:varchar(64);not null;uniqueIndex:uq_usage_scope_snapshots_bucket,priority:2"`
	ScopeType         string `json:"scope_type" gorm:"type:varchar(16);not null;index:idx_usage_scope_snapshots_tenant_window,priority:2"`
	DepartmentId      *int   `json:"department_id" gorm:"type:int;index:idx_usage_scope_snapshots_tenant_dept_window,priority:2"`
	DepartmentName    string `json:"department_name" gorm:"type:varchar(255);not null;default:''"`
	WindowStart       int64  `json:"window_start" gorm:"type:bigint;not null;index:idx_usage_scope_snapshots_tenant_window,priority:3;index:idx_usage_scope_snapshots_tenant_dept_window,priority:3;uniqueIndex:uq_usage_scope_snapshots_bucket,priority:3"`
	WindowEnd         int64  `json:"window_end" gorm:"type:bigint;not null;index:idx_usage_scope_snapshots_tenant_window,priority:4;index:idx_usage_scope_snapshots_tenant_dept_window,priority:4;uniqueIndex:uq_usage_scope_snapshots_bucket,priority:4"`
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

func (UsageScopeSnapshot) TableName() string {
	return "enterprise_usage_scope_snapshots"
}

func (s *UsageScopeSnapshot) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if s.CreatedAt == 0 {
		s.CreatedAt = now
	}
	if s.UpdatedAt == 0 {
		s.UpdatedAt = now
	}
	return s.normalize()
}

func (s *UsageScopeSnapshot) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = time.Now().Unix()
	return s.normalize()
}

func (s *UsageScopeSnapshot) SetModelDistribution(stats []UsageSnapshotModelStat) error {
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

func (s UsageScopeSnapshot) ParsedModelDistribution() ([]UsageSnapshotModelStat, error) {
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

func (s *UsageScopeSnapshot) SetUserIds(userIds []int) error {
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

func (s UsageScopeSnapshot) ParsedUserIds() ([]int, error) {
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

func (s *UsageScopeSnapshot) normalize() error {
	if s.ScopeType == "" {
		if s.DepartmentId == nil {
			s.ScopeType = UsageScopeTypeTenant
		} else {
			s.ScopeType = UsageScopeTypeDepartment
		}
	}
	if s.ScopeKey == "" {
		if s.ScopeType == UsageScopeTypeTenant {
			s.ScopeKey = UsageScopeTypeTenant
		} else if s.DepartmentId != nil {
			s.ScopeKey = fmt.Sprintf("department:%d", *s.DepartmentId)
		}
	}
	if s.ModelDistribution == "" {
		if err := s.SetModelDistribution(nil); err != nil {
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
