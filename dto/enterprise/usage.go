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

type DepartmentUsageReportConfigRequest struct {
	TenantId  *int     `json:"tenant_id,omitempty"`
	Receivers []string `json:"receivers"`
	Frequency *string  `json:"frequency,omitempty"`
	RangeType *string  `json:"range_type,omitempty"`
	Enabled   *bool    `json:"enabled,omitempty"`
}

type DepartmentUsageReportSummary struct {
	WindowStart      int64 `json:"window_start"`
	WindowEnd        int64 `json:"window_end"`
	DepartmentCount  int64 `json:"department_count"`
	RequestCount     int64 `json:"request_count"`
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	Quota            int64 `json:"quota"`
	UserCount        int64 `json:"user_count"`
}

type DepartmentUsageReportTopDepartment struct {
	DeptId       *int   `json:"dept_id"`
	DeptName     string `json:"dept_name"`
	RequestCount int64  `json:"request_count"`
	Quota        int64  `json:"quota"`
	UserCount    int64  `json:"user_count"`
}

type DepartmentUsageReportGrowthDepartment struct {
	DeptId               *int    `json:"dept_id"`
	DeptName             string  `json:"dept_name"`
	RequestCount         int64   `json:"request_count"`
	PreviousRequestCount int64   `json:"previous_request_count"`
	Quota                int64   `json:"quota"`
	PreviousQuota        int64   `json:"previous_quota"`
	RequestGrowthRate    float64 `json:"request_growth_rate"`
	QuotaGrowthRate      float64 `json:"quota_growth_rate"`
}

type DepartmentUsageReportSnapshot struct {
	WindowStart         int64                                   `json:"window_start"`
	WindowEnd           int64                                   `json:"window_end"`
	PreviousWindowStart int64                                   `json:"previous_window_start"`
	PreviousWindowEnd   int64                                   `json:"previous_window_end"`
	DepartmentCount     int64                                   `json:"department_count"`
	RequestCount        int64                                   `json:"request_count"`
	PromptTokens        int64                                   `json:"prompt_tokens"`
	CompletionTokens    int64                                   `json:"completion_tokens"`
	Quota               int64                                   `json:"quota"`
	UserCount           int64                                   `json:"user_count"`
	TopDepartments      []DepartmentUsageReportTopDepartment    `json:"top_departments"`
	GrowthDepartments   []DepartmentUsageReportGrowthDepartment `json:"growth_departments"`
}

type DepartmentUsageReportJobItem struct {
	Id              int                            `json:"id"`
	TenantId        int                            `json:"tenant_id"`
	Receivers       []string                       `json:"receivers"`
	Frequency       string                         `json:"frequency"`
	RangeType       string                         `json:"range_type"`
	Enabled         bool                           `json:"enabled"`
	Status          string                         `json:"status"`
	LastRunAt       int64                          `json:"last_run_at"`
	NextRunAt       int64                          `json:"next_run_at"`
	LastSuccessAt   int64                          `json:"last_success_at"`
	LastWindowStart int64                          `json:"last_window_start"`
	LastWindowEnd   int64                          `json:"last_window_end"`
	RunCount        int64                          `json:"run_count"`
	FailureCount    int64                          `json:"failure_count"`
	ErrorReason     string                         `json:"error_reason"`
	LastSnapshot    *DepartmentUsageReportSnapshot `json:"last_snapshot,omitempty"`
	CreatedAt       int64                          `json:"created_at"`
	UpdatedAt       int64                          `json:"updated_at"`
}

type DepartmentUsageReportConfigResponse struct {
	Item DepartmentUsageReportJobItem `json:"item"`
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
	DisplayName      string `json:"display_name"`
	RequestCount     int64  `json:"request_count"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TokenCount       int64  `json:"token_count"`
	Quota            int64  `json:"quota"`
}

type DepartmentUsageLogUserOption struct {
	UserId      int    `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
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
	DepartmentId    *int                           `json:"department_id"`
	DepartmentName  string                         `json:"department_name"`
	StartTimestamp  int64                          `json:"start_timestamp"`
	EndTimestamp    int64                          `json:"end_timestamp"`
	Username        string                         `json:"username"`
	UsernameOptions []string                       `json:"username_options"`
	UserOptions     []DepartmentUsageLogUserOption `json:"user_options"`
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
