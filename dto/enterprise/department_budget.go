package enterprise

type CreateDepartmentBudgetRequest struct {
	TenantId       *int    `json:"tenant_id,omitempty"`
	Type           string  `json:"type"`
	TotalQuota     *int64  `json:"total_quota,omitempty"`
	CycleQuota     *int64  `json:"cycle_quota,omitempty"`
	CycleType      string  `json:"cycle_type,omitempty"`
	CycleStartedAt *int64  `json:"cycle_started_at,omitempty"`
	CustomSeconds  *int64  `json:"custom_seconds,omitempty"`
	ExpiresAt      *int64  `json:"expires_at,omitempty"`
}

type DepartmentBudgetItem struct {
	Id             int    `json:"id"`
	TenantId       int    `json:"tenant_id"`
	DepartmentId   int    `json:"department_id"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	TotalQuota     int64  `json:"total_quota"`
	Remaining      int64  `json:"remaining"`
	CycleQuota     int64  `json:"cycle_quota"`
	CycleType      string `json:"cycle_type"`
	CycleStartedAt int64  `json:"cycle_started_at"`
	CustomSeconds  int64  `json:"custom_seconds"`
	ExpiresAt      int64  `json:"expires_at"`
	ParentStatus   string `json:"parent_status"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type DepartmentBudgetResponse struct {
	Item *DepartmentBudgetItem `json:"item"`
}
