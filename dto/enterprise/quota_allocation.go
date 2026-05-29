package enterprise

type CreateQuotaAllocationRequest struct {
	TenantId           *int    `json:"tenant_id,omitempty"`
	DepartmentBudgetId int     `json:"department_budget_id"`
	DepartmentId       int     `json:"department_id"`
	TargetUserId       int     `json:"target_user_id"`
	CommittedQuota     *int64  `json:"committed_quota,omitempty"`
	Reason             string  `json:"reason,omitempty"`
}

type ReorderEnterpriseWalletRequest struct {
	UserSubscriptionId int `json:"user_subscription_id"`
	TargetSortOrder    int `json:"target_sort_order"`
}

type QuotaAllocationItem struct {
	Id                     int    `json:"id"`
	TenantId               int    `json:"tenant_id"`
	DepartmentBudgetId     int    `json:"department_budget_id"`
	DepartmentId           int    `json:"department_id"`
	TargetUserId           int    `json:"target_user_id"`
	WalletId               int    `json:"wallet_id"`
	ActorId                int    `json:"actor_id"`
	CommittedQuota         int64  `json:"committed_quota"`
	BudgetTypeSnapshot     string `json:"budget_type_snapshot"`
	CycleTypeSnapshot      string `json:"cycle_type_snapshot"`
	CycleStartedAtSnapshot int64  `json:"cycle_started_at_snapshot"`
	CustomSecondsSnapshot  int64  `json:"custom_seconds_snapshot"`
	ExpiresAtSnapshot      int64  `json:"expires_at_snapshot"`
	Reason                 string `json:"reason"`
	Status                 string `json:"status"`
	ProcessedAt            int64  `json:"processed_at"`
	CreatedAt              int64  `json:"created_at"`
	UpdatedAt              int64  `json:"updated_at"`
}

type RevokeQuotaAllocationRequest struct {
	TenantId     *int   `json:"tenant_id,omitempty"`
	DepartmentId int    `json:"department_id"`
	Reason       string `json:"reason,omitempty"`
}

type QuotaAllocationResponse struct {
	Item *QuotaAllocationItem `json:"item"`
}

type QuotaAllocationListResponse struct {
	Items []QuotaAllocationItem `json:"items"`
}
