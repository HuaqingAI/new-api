package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	QuotaAllocationStatusActive    = "active"
	QuotaAllocationStatusCancelled = "cancelled"
)

type QuotaAllocation struct {
	Id                     int    `json:"id" gorm:"primaryKey"`
	TenantId               int    `json:"tenant_id" gorm:"not null;default:0;index:idx_ent_quota_alloc_tenant"`
	DepartmentBudgetId     int    `json:"department_budget_id" gorm:"not null;index:idx_ent_quota_alloc_budget"`
	DepartmentId           int    `json:"department_id" gorm:"not null;index:idx_ent_quota_alloc_department"`
	TargetUserId           int    `json:"target_user_id" gorm:"not null;index:idx_ent_quota_alloc_target"`
	WalletId               int    `json:"wallet_id" gorm:"not null;default:0;index:idx_ent_quota_alloc_wallet"`
	ActorId                int    `json:"actor_id" gorm:"not null;default:0;index:idx_ent_quota_alloc_actor"`
	CommittedQuota         int64  `json:"committed_quota" gorm:"type:bigint;not null;default:0"`
	BudgetTypeSnapshot     string `json:"budget_type_snapshot" gorm:"type:varchar(32);not null;default:''"`
	CycleTypeSnapshot      string `json:"cycle_type_snapshot" gorm:"type:varchar(16);not null;default:''"`
	CycleStartedAtSnapshot int64  `json:"cycle_started_at_snapshot" gorm:"type:bigint;not null;default:0"`
	CustomSecondsSnapshot  int64  `json:"custom_seconds_snapshot" gorm:"type:bigint;not null;default:0"`
	ExpiresAtSnapshot      int64  `json:"expires_at_snapshot" gorm:"type:bigint;not null;default:0"`
	Reason                 string `json:"reason" gorm:"type:text"`
	BeforeBudgetSnapshot   string `json:"before_budget_snapshot" gorm:"type:text"`
	AfterBudgetSnapshot    string `json:"after_budget_snapshot" gorm:"type:text"`
	Status                 string `json:"status" gorm:"type:varchar(32);not null;default:'active';index:idx_ent_quota_alloc_status"`
	CreatedAt              int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt              int64  `json:"updated_at" gorm:"bigint"`
}

func (QuotaAllocation) TableName() string {
	return "enterprise_quota_allocations"
}

func (q *QuotaAllocation) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if q.CreatedAt == 0 {
		q.CreatedAt = now
	}
	if q.UpdatedAt == 0 {
		q.UpdatedAt = now
	}
	if q.Status == "" {
		q.Status = QuotaAllocationStatusActive
	}
	return nil
}

func (q *QuotaAllocation) BeforeUpdate(tx *gorm.DB) error {
	q.UpdatedAt = common.GetTimestamp()
	return nil
}
