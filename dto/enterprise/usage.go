package enterprise

type DepartmentUsageSummaryQuery struct {
	TenantId     *int    `form:"tenant_id"`
	From         int64   `form:"from"`
	To           int64   `form:"to"`
	SummarySort  *string `form:"summary_sort,omitempty"`
	SummaryOrder *string `form:"summary_order,omitempty"`
}

type DepartmentUsageExportQuery struct {
	TenantId     *int    `form:"tenant_id"`
	From         int64   `form:"from"`
	To           int64   `form:"to"`
	SummarySort  *string `form:"summary_sort,omitempty"`
	SummaryOrder *string `form:"summary_order,omitempty"`
}

type DepartmentUsageDetailQuery struct {
	TenantId *int  `form:"tenant_id"`
	DeptId   *int  `form:"dept_id"`
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

type DepartmentUsageUserRankItem struct {
	UserId           int    `json:"user_id"`
	Username         string `json:"username"`
	RequestCount     int64  `json:"request_count"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TokenCount       int64  `json:"token_count"`
	Quota            int64  `json:"quota"`
}

type DepartmentUsageTrendPoint struct {
	WindowStart      int64 `json:"window_start"`
	WindowEnd        int64 `json:"window_end"`
	RequestCount     int64 `json:"request_count"`
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TokenCount       int64 `json:"token_count"`
	Quota            int64 `json:"quota"`
	UserCount        int64 `json:"user_count"`
}

type DepartmentUsageLogFilters struct {
	DepartmentId    *int     `json:"department_id"`
	DepartmentName  string   `json:"department_name"`
	StartTimestamp  int64    `json:"start_timestamp"`
	EndTimestamp    int64    `json:"end_timestamp"`
	Username        string   `json:"username"`
	UsernameOptions []string `json:"username_options"`
}

type DepartmentUsageLogEntryLink struct {
	Path    string                    `json:"path"`
	Section string                    `json:"section"`
	Filters DepartmentUsageLogFilters `json:"filters"`
}

type DepartmentUsageDetailResponse struct {
	DeptId            *int                          `json:"dept_id"`
	DeptName          string                        `json:"dept_name"`
	WindowStart       int64                         `json:"window_start"`
	WindowEnd         int64                         `json:"window_end"`
	RequestCount      int64                         `json:"request_count"`
	PromptTokens      int64                         `json:"prompt_tokens"`
	CompletionTokens  int64                         `json:"completion_tokens"`
	TokenCount        int64                         `json:"token_count"`
	Quota             int64                         `json:"quota"`
	UserCount         int64                         `json:"user_count"`
	UserRanking       []DepartmentUsageUserRankItem `json:"user_ranking"`
	ModelDistribution []UsageModelDistributionItem  `json:"model_distribution"`
	Trend             []DepartmentUsageTrendPoint   `json:"trend"`
	RecentLogsEntry   DepartmentUsageLogEntryLink   `json:"recent_logs_entry"`
}
