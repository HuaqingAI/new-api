package enterprise

type SubmitQuotaRequestRequest struct {
	TenantId           *int   `json:"tenant_id,omitempty"`
	DepartmentId       int    `json:"department_id"`
	DepartmentBudgetId int    `json:"department_budget_id"`
	BudgetMode         string `json:"budget_mode,omitempty"`
	RequestedQuota     *int64 `json:"requested_quota,omitempty"`
	RequestReason      string `json:"request_reason,omitempty"`
	IdempotencyKey     string `json:"idempotency_key,omitempty"`
}

type DecideQuotaRequestRequest struct {
	TenantId       *int   `json:"tenant_id,omitempty"`
	Action         string `json:"action"`
	ApprovedQuota  *int64 `json:"approved_quota,omitempty"`
	ApprovalReason string `json:"approval_reason,omitempty"`
	RejectedReason string `json:"rejected_reason,omitempty"`
}

type QuotaRequestListQuery struct {
	TenantId        *int   `form:"tenant_id"`
	DepartmentId    *int   `form:"department_id"`
	RequesterUserId *int   `form:"requester_user_id"`
	IncludePending  *bool  `form:"include_pending,omitempty"`
	Limit           *int   `form:"limit,omitempty"`
	View            string `form:"view,omitempty"`
	Status          string `form:"status,omitempty"`
	Page            *int   `form:"page,omitempty"`
	PageSize        *int   `form:"page_size,omitempty"`
}

type QuotaRequestItem struct {
	Id                   int    `json:"id"`
	TenantId             int    `json:"tenant_id"`
	DepartmentId         int    `json:"department_id"`
	DepartmentName       string `json:"department_name"`
	DepartmentBudgetId   int    `json:"department_budget_id"`
	BudgetScopeType      string `json:"budget_scope_type"`
	BudgetName           string `json:"budget_name"`
	BudgetMode           string `json:"budget_mode"`
	RequesterUserId      int    `json:"requester_user_id"`
	RequesterUsername    string `json:"requester_username"`
	RequesterDisplayName string `json:"requester_display_name"`
	RequestedQuota       int64  `json:"requested_quota"`
	ApprovedQuota        int64  `json:"approved_quota"`
	Status               string `json:"status"`
	ApproverUserId       int    `json:"approver_user_id"`
	ApproverUsername     string `json:"approver_username"`
	ApprovalReason       string `json:"approval_reason"`
	RequestReason        string `json:"request_reason"`
	AllocationId         int    `json:"allocation_id"`
	OwnerCountSnapshot   int    `json:"owner_count_snapshot"`
	Fallback             string `json:"fallback"`
	SubmittedAt          int64  `json:"submitted_at"`
	ApprovedAt           int64  `json:"approved_at"`
	RejectedAt           int64  `json:"rejected_at"`
	FulfilledAt          int64  `json:"fulfilled_at"`
	ProcessedAt          int64  `json:"processed_at"`
	ExpiresAt            int64  `json:"expires_at"`
	CreatedAt            int64  `json:"created_at"`
	UpdatedAt            int64  `json:"updated_at"`
}

type QuotaRequestResponse struct {
	Item       *QuotaRequestItem    `json:"item"`
	Allocation *QuotaAllocationItem `json:"allocation,omitempty"`
}

type QuotaRequestListResponse struct {
	Items    []QuotaRequestItem `json:"items"`
	Total    int64              `json:"total,omitempty"`
	Page     int                `json:"page,omitempty"`
	PageSize int                `json:"page_size,omitempty"`
	Scope    string             `json:"scope,omitempty"`
}

type QuotaRequestCapabilityBudgetItem struct {
	Id             int     `json:"id"`
	TenantId       int     `json:"tenant_id"`
	DepartmentId   int     `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	ScopeType      string  `json:"scope_type"`
	Name           string  `json:"name"`
	IsPublic       bool    `json:"is_public"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	TotalQuota     int64   `json:"total_quota"`
	Remaining      int64   `json:"remaining"`
	AllocatedTotal int64   `json:"allocated_total"`
	CycleQuota     int64   `json:"cycle_quota"`
	CycleType      string  `json:"cycle_type"`
	CycleStartedAt int64   `json:"cycle_started_at"`
	CustomSeconds  int64   `json:"custom_seconds"`
	ExpiresAt      int64   `json:"expires_at"`
	ParentStatus   string  `json:"parent_status"`
	UsageRatio     float64 `json:"usage_ratio"`
	ThresholdState string  `json:"threshold_state"`
	CreatedAt      int64   `json:"created_at"`
	UpdatedAt      int64   `json:"updated_at"`
}

type QuotaRequestCapabilityResponse struct {
	CanSubmit bool                               `json:"can_submit"`
	CanGovern bool                               `json:"can_govern"`
	Budgets   []QuotaRequestCapabilityBudgetItem `json:"budgets"`
}
