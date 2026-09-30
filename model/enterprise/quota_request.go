package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	QuotaRequestStatusSubmitted = "submitted"
	QuotaRequestStatusApproved  = "approved"
	QuotaRequestStatusRejected  = "rejected"
	QuotaRequestStatusFulfilled = "fulfilled"
	QuotaRequestStatusCanceled  = "canceled"
	QuotaRequestStatusExpired   = "expired"
)

type QuotaRequest struct {
	Id                  int    `json:"id" gorm:"primaryKey"`
	TenantId            int    `json:"tenant_id" gorm:"not null;default:0;index:idx_ent_quota_req_tenant;uniqueIndex:uq_ent_quota_req_idempotency,priority:1"`
	DepartmentId        int    `json:"department_id" gorm:"not null;index:idx_ent_quota_req_department"`
	DepartmentBudgetId  int    `json:"department_budget_id" gorm:"not null;default:0;index:idx_ent_quota_req_budget"`
	BudgetMode          string `json:"budget_mode" gorm:"type:varchar(32);not null;default:'';index:idx_ent_quota_req_budget_mode"`
	BudgetScopeSnapshot string `json:"budget_scope_snapshot" gorm:"type:varchar(16);not null;default:''"`
	BudgetNameSnapshot  string `json:"budget_name_snapshot" gorm:"type:varchar(128);not null;default:''"`
	RequesterUserId     int    `json:"requester_user_id" gorm:"not null;index:idx_ent_quota_req_requester;uniqueIndex:uq_ent_quota_req_idempotency,priority:2"`
	RequestedQuota      int64  `json:"requested_quota" gorm:"type:bigint;not null;default:0"`
	ApprovedQuota       int64  `json:"approved_quota" gorm:"type:bigint;not null;default:0"`
	Status              string `json:"status" gorm:"type:varchar(32);not null;default:'submitted';index:idx_ent_quota_req_status"`
	ApproverUserId      int    `json:"approver_user_id" gorm:"not null;default:0;index:idx_ent_quota_req_approver"`
	RequestReason       string `json:"request_reason" gorm:"type:text"`
	ApprovalReason      string `json:"approval_reason" gorm:"type:text"`
	AllocationId        int    `json:"allocation_id" gorm:"not null;default:0;index:idx_ent_quota_req_allocation"`
	IdempotencyKey      string `json:"idempotency_key" gorm:"type:varchar(96);not null;default:'';uniqueIndex:uq_ent_quota_req_idempotency,priority:3"`
	OwnerCountSnapshot  int    `json:"owner_count_snapshot" gorm:"not null;default:0"`
	Fallback            string `json:"fallback" gorm:"type:varchar(32);not null;default:'';index:idx_ent_quota_req_fallback"`
	SubmittedAt         int64  `json:"submitted_at" gorm:"type:bigint;not null;default:0"`
	ApprovedAt          int64  `json:"approved_at" gorm:"type:bigint;not null;default:0"`
	RejectedAt          int64  `json:"rejected_at" gorm:"type:bigint;not null;default:0"`
	FulfilledAt         int64  `json:"fulfilled_at" gorm:"type:bigint;not null;default:0"`
	ProcessedAt         int64  `json:"processed_at" gorm:"type:bigint;not null;default:0"`
	ExpiresAt           int64  `json:"expires_at" gorm:"type:bigint;not null;default:0"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint"`
}

func (QuotaRequest) TableName() string {
	return "enterprise_quota_requests"
}

func (q *QuotaRequest) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if q.CreatedAt == 0 {
		q.CreatedAt = now
	}
	if q.UpdatedAt == 0 {
		q.UpdatedAt = now
	}
	if q.SubmittedAt == 0 {
		q.SubmittedAt = now
	}
	if q.Status == "" {
		q.Status = QuotaRequestStatusSubmitted
	}
	if q.IdempotencyKey == "" {
		q.IdempotencyKey = common.GetUUID()
	}
	return nil
}

func (q *QuotaRequest) BeforeUpdate(tx *gorm.DB) error {
	q.UpdatedAt = common.GetTimestamp()
	return nil
}
