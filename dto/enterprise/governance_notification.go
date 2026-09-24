package enterprise

type GovernanceNotificationQuery struct {
	TenantId        *int    `form:"tenant_id,omitempty"`
	DepartmentId    *int    `form:"department_id,omitempty"`
	RecipientUserId *int    `form:"recipient_user_id,omitempty"`
	SourceType      *string `form:"source_type,omitempty"`
	SourceId        *int    `form:"source_id,omitempty"`
	ActionType      *string `form:"action_type,omitempty"`
	Status          *string `form:"status,omitempty"`
	Page            *int    `form:"page,omitempty"`
	PageSize        *int    `form:"page_size,omitempty"`
}

type GovernanceNotificationTraceItem struct {
	TraceId            string `json:"trace_id"`
	SourceType         string `json:"source_type"`
	SourceId           int    `json:"source_id"`
	ActionType         string `json:"action_type"`
	TenantId           int    `json:"tenant_id"`
	DepartmentId       int    `json:"department_id"`
	DepartmentName     string `json:"department_name"`
	BudgetId           int    `json:"budget_id"`
	AllocationId       int    `json:"allocation_id"`
	RequestId          int    `json:"request_id"`
	ActorId            int    `json:"actor_id"`
	ActorName          string `json:"actor_name"`
	TargetUserId       int    `json:"target_user_id"`
	TargetUsername     string `json:"target_username"`
	TargetDisplayName  string `json:"target_display_name"`
	QuotaDelta         int64  `json:"quota_delta"`
	CommittedQuota     int64  `json:"committed_quota"`
	RequestedQuota     int64  `json:"requested_quota"`
	ApprovedQuota      int64  `json:"approved_quota"`
	Status             string `json:"status"`
	Fallback           string `json:"fallback,omitempty"`
	OccurredAt         int64  `json:"occurred_at"`
	DetailRoute        string `json:"detail_route"`
	DetailAPIPath      string `json:"detail_api_path"`
	Summary            string `json:"summary"`
	RecipientUserId    int    `json:"recipient_user_id"`
	RecipientKind      string `json:"recipient_kind"`
	NotificationTarget string `json:"notification_target,omitempty"`
}

type GovernanceNotificationItem struct {
	Id              int                              `json:"id"`
	TenantId        int                              `json:"tenant_id"`
	SourceType      string                           `json:"source_type"`
	SourceId        int                              `json:"source_id"`
	TraceId         string                           `json:"trace_id"`
	ActionType      string                           `json:"action_type"`
	RecipientUserId int                              `json:"recipient_user_id"`
	RecipientKind   string                           `json:"recipient_kind"`
	ChannelType     string                           `json:"channel_type"`
	Status          string                           `json:"status"`
	AttemptCount    int                              `json:"attempt_count"`
	MaxAttempts     int                              `json:"max_attempts"`
	NextRetryAt     int64                            `json:"next_retry_at"`
	LastAttemptAt   int64                            `json:"last_attempt_at"`
	SentAt          int64                            `json:"sent_at"`
	FinalFailedAt   int64                            `json:"final_failed_at"`
	ErrorReason     string                           `json:"error_reason"`
	DedupeKey       string                           `json:"dedupe_key"`
	TriggerSource   string                           `json:"trigger_source"`
	ManualParentId  *int                             `json:"manual_parent_id,omitempty"`
	TraceSummary    string                           `json:"trace_summary"`
	Trace           *GovernanceNotificationTraceItem `json:"trace,omitempty"`
	CreatedAt       int64                            `json:"created_at"`
	UpdatedAt       int64                            `json:"updated_at"`
}

type GovernanceNotificationResponse struct {
	Items    []GovernanceNotificationItem `json:"items"`
	Total    int                          `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}

type GovernanceNotificationResendQuery struct {
	TenantId *int `form:"tenant_id,omitempty"`
}

type GovernanceNotificationResendResponse struct {
	Item    GovernanceNotificationItem `json:"item"`
	Created bool                       `json:"created"`
}
