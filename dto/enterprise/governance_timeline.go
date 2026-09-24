package enterprise

type GovernanceTimelineQuery struct {
	TenantId     *int    `form:"tenant_id,omitempty"`
	DepartmentId *int    `form:"department_id,omitempty"`
	SourceId     *int    `form:"source_id,omitempty"`
	ActorId      *int    `form:"actor_id,omitempty"`
	ActionType   *string `form:"action_type,omitempty"`
	SourceType   *string `form:"source_type,omitempty"`
	Status       *string `form:"status,omitempty"`
	StartAt      *int64  `form:"start_at,omitempty"`
	EndAt        *int64  `form:"end_at,omitempty"`
	Page         *int    `form:"page,omitempty"`
	PageSize     *int    `form:"page_size,omitempty"`
}

type GovernanceTimelineTarget struct {
	DepartmentId   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
	UserId         int    `json:"user_id"`
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`
	ObjectType     string `json:"object_type"`
	ObjectId       string `json:"object_id"`
}

type GovernanceTimelineItem struct {
	TraceId       string                   `json:"trace_id"`
	SourceType    string                   `json:"source_type"`
	SourceId      int                      `json:"source_id"`
	ActionType    string                   `json:"action_type"`
	TenantId      int                      `json:"tenant_id"`
	ActorId       int                      `json:"actor_id"`
	ActorName     string                   `json:"actor_name"`
	Target        GovernanceTimelineTarget `json:"target"`
	QuotaDelta    int64                    `json:"quota_delta"`
	BeforeQuota   int64                    `json:"before_quota"`
	AfterQuota    int64                    `json:"after_quota"`
	Status        string                   `json:"status"`
	OccurredAt    int64                    `json:"occurred_at"`
	DetailRoute   string                   `json:"detail_route"`
	DetailAPIPath string                   `json:"detail_api_path"`
	Summary       string                   `json:"summary"`
}

type GovernanceTimelineResponse struct {
	Items    []GovernanceTimelineItem `json:"items"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}
