package enterprise

type CreateDepartmentBudgetRequest struct {
	TenantId       *int   `json:"tenant_id,omitempty"`
	Type           string `json:"type"`
	TotalQuota     *int64 `json:"total_quota,omitempty"`
	CycleQuota     *int64 `json:"cycle_quota,omitempty"`
	CycleType      string `json:"cycle_type,omitempty"`
	CycleStartedAt *int64 `json:"cycle_started_at,omitempty"`
	CustomSeconds  *int64 `json:"custom_seconds,omitempty"`
	ExpiresAt      *int64 `json:"expires_at,omitempty"`
}

type DepartmentBudgetListQuery struct {
	TenantId           *int   `form:"tenant_id"`
	SortBy             string `form:"sort_by"`
	SortOrder          string `form:"sort_order"`
	IncludeDescendants *bool  `form:"include_descendants,omitempty"`
}

type DepartmentBudgetThresholds struct {
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}

type DepartmentBudgetItem struct {
	Id             int     `json:"id"`
	TenantId       int     `json:"tenant_id"`
	DepartmentId   int     `json:"department_id"`
	DepartmentName string  `json:"department_name"`
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

type DepartmentBudgetWalletDetail struct {
	AllocationId             int    `json:"allocation_id"`
	AllocationStatus         string `json:"allocation_status"`
	TargetUserId             int    `json:"target_user_id"`
	TargetUsername           string `json:"target_username"`
	TargetDisplayName        string `json:"target_display_name"`
	WalletId                 int    `json:"wallet_id"`
	WalletStatus             string `json:"wallet_status"`
	Quota                    int64  `json:"quota"`
	RemainQuota              int64  `json:"remain_quota"`
	CycleType                string `json:"cycle_type"`
	CycleStartedAt           int64  `json:"cycle_started_at"`
	NextResetTime            int64  `json:"next_reset_time"`
	ExpiresAt                int64  `json:"expires_at"`
	SourceAllocationId       int    `json:"source_allocation_id"`
	SourceParentBudgetId     int    `json:"source_parent_budget_id"`
	SourceParentBudgetType   string `json:"source_parent_budget_type"`
	SourceParentBudgetStatus string `json:"source_parent_budget_status"`
	CommittedQuota           int64  `json:"committed_quota"`
	ProcessedAt              int64  `json:"processed_at"`
	CreatedAt                int64  `json:"created_at"`
	UpdatedAt                int64  `json:"updated_at"`
	Reason                   string `json:"reason"`
}

type DepartmentBudgetResponse struct {
	Item *DepartmentBudgetItem `json:"item"`
}

type DepartmentBudgetListResponse struct {
	Items               []DepartmentBudgetItem     `json:"items"`
	Thresholds          DepartmentBudgetThresholds `json:"thresholds"`
	ScopeDepartmentId   *int                       `json:"scope_department_id,omitempty"`
	ScopeDepartmentName string                     `json:"scope_department_name"`
	IncludeDescendants  bool                       `json:"include_descendants"`
	ScopeDepartmentIds  []int                      `json:"scope_department_ids"`
}

type DepartmentBudgetDetailResponse struct {
	Budget     *DepartmentBudgetItem          `json:"budget"`
	Wallets    []DepartmentBudgetWalletDetail `json:"wallets"`
	Thresholds DepartmentBudgetThresholds     `json:"thresholds"`
}
