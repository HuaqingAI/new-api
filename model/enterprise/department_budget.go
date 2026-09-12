package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	DepartmentBudgetTypeBalance      = "balance"
	DepartmentBudgetTypeSubscription = "subscription"

	DepartmentBudgetScopeDepartment = "department"
	DepartmentBudgetScopePublic     = "public"

	DepartmentBudgetStatusActive  = "active"
	DepartmentBudgetStatusPaused  = "paused"
	DepartmentBudgetStatusRevoked = "revoked"
	DepartmentBudgetStatusExpired = "expired"
)

type DepartmentBudget struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	TenantId       int    `json:"tenant_id" gorm:"not null;default:0;index:idx_ent_dept_budgets_tenant"`
	DepartmentId   int    `json:"department_id" gorm:"not null;index:idx_ent_dept_budgets_department"`
	ScopeType      string `json:"scope_type" gorm:"type:varchar(16);not null;default:'department';index:idx_ent_dept_budgets_scope"`
	Name           string `json:"name" gorm:"type:varchar(128);not null;default:''"`
	Type           string `json:"type" gorm:"type:varchar(32);not null;index:idx_ent_dept_budgets_type"`
	Status         string `json:"status" gorm:"type:varchar(32);not null;index:idx_ent_dept_budgets_status"`
	TotalQuota     int64  `json:"total_quota" gorm:"type:bigint;not null;default:0"`
	Remaining      int64  `json:"remaining" gorm:"type:bigint;not null;default:0"`
	AllocatedTotal int64  `json:"allocated_total" gorm:"type:bigint;not null;default:0"`
	CycleQuota     int64  `json:"cycle_quota" gorm:"type:bigint;not null;default:0"`
	CycleType      string `json:"cycle_type" gorm:"type:varchar(16);not null;default:'never'"`
	CycleStartedAt int64  `json:"cycle_started_at" gorm:"type:bigint;not null;default:0"`
	CustomSeconds  int64  `json:"custom_seconds" gorm:"type:bigint;not null;default:0"`
	ExpiresAt      int64  `json:"expires_at" gorm:"type:bigint;not null;default:0"`
	ParentStatus   string `json:"parent_status" gorm:"type:varchar(32);not null;default:'';index:idx_ent_dept_budgets_parent_status"`
	CreatedAt      int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt      int64  `json:"updated_at" gorm:"bigint"`
}

func (DepartmentBudget) TableName() string {
	return "enterprise_department_budgets"
}

func (b *DepartmentBudget) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if b.CreatedAt == 0 {
		b.CreatedAt = now
	}
	if b.UpdatedAt == 0 {
		b.UpdatedAt = now
	}
	if b.Type == "" {
		b.Type = DepartmentBudgetTypeBalance
	}
	if b.ScopeType == "" {
		b.ScopeType = DepartmentBudgetScopeDepartment
	}
	if b.Status == "" {
		b.Status = DepartmentBudgetStatusActive
	}
	if b.CycleType == "" {
		b.CycleType = "never"
	}
	if b.Type == DepartmentBudgetTypeBalance {
		if b.Remaining == 0 && b.TotalQuota > 0 {
			b.Remaining = b.TotalQuota
		}
	}
	if b.Type == DepartmentBudgetTypeSubscription && b.Remaining == 0 && b.CycleQuota > 0 {
		b.Remaining = b.CycleQuota
	}
	return nil
}

func (b *DepartmentBudget) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = common.GetTimestamp()
	return nil
}
