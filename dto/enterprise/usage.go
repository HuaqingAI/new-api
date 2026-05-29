package enterprise

type DepartmentUsageSummaryQuery struct {
	TenantId *int  `form:"tenant_id"`
	From     int64 `form:"from"`
	To       int64 `form:"to"`
}

type UsageModelDistributionItem struct {
	ModelName        string `json:"model_name"`
	RequestCount     int64  `json:"request_count"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	Quota            int64  `json:"quota"`
}

type DepartmentUsageSummaryItem struct {
	DeptId            *int                         `json:"dept_id"`
	DeptName          string                       `json:"dept_name"`
	WindowStart       int64                        `json:"window_start"`
	WindowEnd         int64                        `json:"window_end"`
	RequestCount      int64                        `json:"request_count"`
	PromptTokens      int64                        `json:"prompt_tokens"`
	CompletionTokens  int64                        `json:"completion_tokens"`
	Quota             int64                        `json:"quota"`
	UserCount         int64                        `json:"user_count"`
	ModelDistribution []UsageModelDistributionItem `json:"model_distribution"`
}

type DepartmentUsageSummaryResponse struct {
	Items []DepartmentUsageSummaryItem `json:"items"`
}
