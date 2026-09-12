package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	BudgetDelegationStatusActive     = "active"
	BudgetDelegationStatusSuperseded = "superseded"
	BudgetDelegationStatusRevoked    = "revoked"
	BudgetDelegationStatusExpired    = "expired"
	BudgetDelegationStatusClosed     = "closed"
)

type BudgetDelegation struct {
	Id                         int    `json:"id" gorm:"primaryKey"`
	TenantId                   int    `json:"tenant_id" gorm:"not null;default:0;index:idx_ent_budget_deleg_tenant"`
	SourceDepartmentId         int    `json:"source_department_id" gorm:"not null;index:idx_ent_budget_deleg_source_dept"`
	SourceBudgetId             int    `json:"source_budget_id" gorm:"not null;index:idx_ent_budget_deleg_source_budget"`
	TargetDepartmentId         int    `json:"target_department_id" gorm:"not null;index:idx_ent_budget_deleg_target_dept"`
	TargetBudgetId             int    `json:"target_budget_id" gorm:"not null;index:idx_ent_budget_deleg_target_budget"`
	ActorId                    int    `json:"actor_id" gorm:"not null;default:0;index:idx_ent_budget_deleg_actor"`
	CommittedQuota             int64  `json:"committed_quota" gorm:"type:bigint;not null;default:0"`
	BudgetTypeSnapshot         string `json:"budget_type_snapshot" gorm:"type:varchar(32);not null;default:''"`
	CycleTypeSnapshot          string `json:"cycle_type_snapshot" gorm:"type:varchar(16);not null;default:''"`
	BeforeSourceBudgetSnapshot string `json:"before_source_budget_snapshot" gorm:"type:text"`
	AfterSourceBudgetSnapshot  string `json:"after_source_budget_snapshot" gorm:"type:text"`
	BeforeTargetBudgetSnapshot string `json:"before_target_budget_snapshot" gorm:"type:text"`
	AfterTargetBudgetSnapshot  string `json:"after_target_budget_snapshot" gorm:"type:text"`
	Status                     string `json:"status" gorm:"type:varchar(32);not null;default:'active';index:idx_ent_budget_deleg_status"`
	SupersededById             int    `json:"superseded_by_id" gorm:"not null;default:0;index:idx_ent_budget_deleg_superseded_by"`
	ProcessedAt                int64  `json:"processed_at" gorm:"type:bigint;not null;default:0;index:idx_ent_budget_deleg_processed_at"`
	Reason                     string `json:"reason" gorm:"type:text"`
	CreatedAt                  int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt                  int64  `json:"updated_at" gorm:"bigint"`
}

func (BudgetDelegation) TableName() string {
	return "enterprise_budget_delegations"
}

func (d *BudgetDelegation) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if d.CreatedAt == 0 {
		d.CreatedAt = now
	}
	if d.UpdatedAt == 0 {
		d.UpdatedAt = now
	}
	if d.Status == "" {
		d.Status = BudgetDelegationStatusActive
	}
	return nil
}

func (d *BudgetDelegation) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = common.GetTimestamp()
	return nil
}
