package enterprise

type CreateBudgetDelegationRequest struct {
	TenantId           *int   `json:"tenant_id,omitempty"`
	SourceDepartmentId int    `json:"source_department_id"`
	SourceBudgetId     int    `json:"source_budget_id"`
	TargetDepartmentId int    `json:"target_department_id"`
	TargetBudgetId     int    `json:"target_budget_id"`
	CommittedQuota     *int64 `json:"committed_quota,omitempty"`
	Reason             string `json:"reason,omitempty"`
}

type SupersedeBudgetDelegationRequest struct {
	TenantId           *int   `json:"tenant_id,omitempty"`
	SourceDepartmentId int    `json:"source_department_id"`
	NewCommittedQuota  *int64 `json:"new_committed_quota,omitempty"`
	Reason             string `json:"reason,omitempty"`
}

type BudgetDelegationListQuery struct {
	TenantId     *int `form:"tenant_id"`
	DepartmentId int  `form:"department_id"`
}

type BudgetDelegationItem struct {
	Id                         int    `json:"id"`
	TenantId                   int    `json:"tenant_id"`
	SourceDepartmentId         int    `json:"source_department_id"`
	SourceDepartmentName       string `json:"source_department_name"`
	SourceBudgetId             int    `json:"source_budget_id"`
	TargetDepartmentId         int    `json:"target_department_id"`
	TargetDepartmentName       string `json:"target_department_name"`
	TargetBudgetId             int    `json:"target_budget_id"`
	ActorId                    int    `json:"actor_id"`
	CommittedQuota             int64  `json:"committed_quota"`
	BudgetTypeSnapshot         string `json:"budget_type_snapshot"`
	CycleTypeSnapshot          string `json:"cycle_type_snapshot"`
	BeforeSourceBudgetSnapshot string `json:"before_source_budget_snapshot"`
	AfterSourceBudgetSnapshot  string `json:"after_source_budget_snapshot"`
	BeforeTargetBudgetSnapshot string `json:"before_target_budget_snapshot"`
	AfterTargetBudgetSnapshot  string `json:"after_target_budget_snapshot"`
	Status                     string `json:"status"`
	SupersededById             int    `json:"superseded_by_id"`
	ProcessedAt                int64  `json:"processed_at"`
	Reason                     string `json:"reason"`
	CreatedAt                  int64  `json:"created_at"`
	UpdatedAt                  int64  `json:"updated_at"`
}

type BudgetDelegationResponse struct {
	Item *BudgetDelegationItem `json:"item"`
}

type BudgetDelegationListResponse struct {
	Items []BudgetDelegationItem `json:"items"`
}
