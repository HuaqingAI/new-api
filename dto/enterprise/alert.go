package enterprise

type AlertEventQuery struct {
	TenantId     *int    `form:"tenant_id"`
	EventId      *int    `form:"event_id,omitempty"`
	DepartmentId *int    `form:"department_id,omitempty"`
	UnassignedOnly *bool `form:"unassigned_only,omitempty"`
	UserId       *int    `form:"user_id,omitempty"`
	Username     *string `form:"username,omitempty"`
	ModelName    *string `form:"model_name,omitempty"`
	RiskType     *string `form:"risk_type,omitempty"`
	From         *int64  `form:"from,omitempty"`
	To           *int64  `form:"to,omitempty"`
	Page         *int    `form:"page,omitempty"`
	PageSize     *int    `form:"page_size,omitempty"`
}

type AlertEventDepartmentSnapshot struct {
	DepartmentId   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
	ExternalSource string `json:"external_source"`
	Status         int    `json:"status"`
}

type AlertEventItem struct {
	Id                 int                            `json:"id"`
	TenantId           int                            `json:"tenant_id"`
	UserId             int                            `json:"user_id"`
	Username           string                         `json:"username"`
	RequestId          string                         `json:"request_id"`
	ModelName          string                         `json:"model_name"`
	RiskType           string                         `json:"risk_type"`
	ActionResult       string                         `json:"action_result"`
	CreatedAt          int64                          `json:"created_at"`
	DepartmentSnapshot []AlertEventDepartmentSnapshot `json:"department_snapshot"`
	Summary            string                         `json:"summary"`
}

type AlertEventsResponse struct {
	Items    []AlertEventItem `json:"items"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type DepartmentRiskSummaryQuery struct {
	TenantId     *int    `form:"tenant_id,omitempty"`
	From         int64   `form:"from"`
	To           int64   `form:"to"`
	SummarySort  *string `form:"summary_sort,omitempty"`
	SummaryOrder *string `form:"summary_order,omitempty"`
}

type DepartmentRiskEventEntry struct {
	DetailRoute   string `json:"detail_route"`
	DetailAPIPath string `json:"detail_api_path"`
	DepartmentId  *int   `json:"department_id,omitempty"`
	DepartmentName string `json:"department_name"`
	From          int64  `json:"from"`
	To            int64  `json:"to"`
	UnassignedOnly bool  `json:"unassigned_only"`
}

type DepartmentRiskSummaryItem struct {
	DeptId            *int                     `json:"dept_id"`
	DeptName          string                   `json:"dept_name"`
	IsUnassigned      bool                     `json:"is_unassigned"`
	WindowStart       int64                    `json:"window_start"`
	WindowEnd         int64                    `json:"window_end"`
	RiskEventCount    int64                    `json:"risk_event_count"`
	TotalRequestCount int64                    `json:"total_request_count"`
	RiskRate          float64                  `json:"risk_rate"`
	EventEntry        DepartmentRiskEventEntry `json:"event_entry"`
}

type DepartmentRiskTrendPoint struct {
	WindowStart                int64   `json:"window_start"`
	WindowEnd                  int64   `json:"window_end"`
	RiskEventCount             int64   `json:"risk_event_count"`
	TotalRequestCount          int64   `json:"total_request_count"`
	RiskRate                   float64 `json:"risk_rate"`
	UnassignedRiskEventCount   int64   `json:"unassigned_risk_event_count"`
	UnassignedTotalRequestCount int64  `json:"unassigned_total_request_count"`
}

type DepartmentRiskFormula struct {
	Expression       string `json:"expression"`
	NumeratorLabel   string `json:"numerator_label"`
	DenominatorLabel string `json:"denominator_label"`
}

type DepartmentRiskSummaryResponse struct {
	Items         []DepartmentRiskSummaryItem `json:"items"`
	TopDepartments []DepartmentRiskSummaryItem `json:"top_departments"`
	Trend         []DepartmentRiskTrendPoint `json:"trend"`
	Unassigned    DepartmentRiskSummaryItem  `json:"unassigned"`
	Formula       DepartmentRiskFormula      `json:"formula"`
	DisclaimerKey string                     `json:"disclaimer_key"`
}

type AlertDeliveriesQuery struct {
	TenantId       *int    `form:"tenant_id,omitempty"`
	RuleId         *int    `form:"rule_id,omitempty"`
	EventId        *int    `form:"event_id,omitempty"`
	ManualParentId *int    `form:"manual_parent_id,omitempty"`
	ChannelType    *string `form:"channel_type,omitempty"`
	Status         *string `form:"status,omitempty"`
	TriggerSource  *string `form:"trigger_source,omitempty"`
	Page           *int    `form:"page,omitempty"`
	PageSize       *int    `form:"page_size,omitempty"`
}

type AlertDeliveryTraceItem struct {
	EventId            int                            `json:"event_id"`
	RequestId          string                         `json:"request_id"`
	TenantId           int                            `json:"tenant_id"`
	Username           string                         `json:"username"`
	ModelName          string                         `json:"model_name"`
	RiskType           string                         `json:"risk_type"`
	ActionResult       string                         `json:"action_result"`
	EventCreatedAt     int64                          `json:"event_created_at"`
	DepartmentSnapshot []AlertEventDepartmentSnapshot `json:"department_snapshot"`
	DepartmentSummary  string                         `json:"department_summary"`
	EventSummary       string                         `json:"event_summary"`
	RuleId             int                            `json:"rule_id"`
	RuleName           string                         `json:"rule_name"`
	DetailRoute        string                         `json:"detail_route"`
	DetailAPIPath      string                         `json:"detail_api_path"`
}

type AlertDeliveryItem struct {
	Id             int                     `json:"id"`
	TenantId       int                     `json:"tenant_id"`
	EventId        int                     `json:"event_id"`
	RuleId         int                     `json:"rule_id"`
	ChannelType    string                  `json:"channel_type"`
	Status         string                  `json:"status"`
	AttemptCount   int                     `json:"attempt_count"`
	MaxAttempts    int                     `json:"max_attempts"`
	NextRetryAt    int64                   `json:"next_retry_at"`
	LastAttemptAt  int64                   `json:"last_attempt_at"`
	SentAt         int64                   `json:"sent_at"`
	FinalFailedAt  int64                   `json:"final_failed_at"`
	ErrorReason    string                  `json:"error_reason"`
	DedupeKey      string                  `json:"dedupe_key"`
	TriggerSource  string                  `json:"trigger_source"`
	ManualParentId *int                    `json:"manual_parent_id,omitempty"`
	TraceSummary   string                  `json:"trace_summary"`
	CreatedAt      int64                   `json:"created_at"`
	UpdatedAt      int64                   `json:"updated_at"`
	Trace          *AlertDeliveryTraceItem `json:"trace,omitempty"`
}

type AlertDeliveriesResponse struct {
	Items    []AlertDeliveryItem `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

type AlertDeliveryResendQuery struct {
	TenantId *int `form:"tenant_id,omitempty"`
}

type AlertDeliveryResendResponse struct {
	Item    AlertDeliveryItem `json:"item"`
	Created bool              `json:"created"`
}

type AlertRuleChannelConfigInput struct {
	Type                string   `json:"type"`
	Enabled             *bool    `json:"enabled,omitempty"`
	Receivers           []string `json:"receivers,omitempty"`
	WebhookURL          *string  `json:"webhook_url,omitempty"`
	WebhookSecret       *string  `json:"webhook_secret,omitempty"`
	DingTalkRobotURL    *string  `json:"dingtalk_robot_url,omitempty"`
	DingTalkRobotSecret *string  `json:"dingtalk_robot_secret,omitempty"`
}

type AlertRuleChannelConfigItem struct {
	Type                     string   `json:"type"`
	Enabled                  bool     `json:"enabled"`
	Receivers                []string `json:"receivers"`
	WebhookURL               *string  `json:"webhook_url,omitempty"`
	WebhookSecretConfigured  bool     `json:"webhook_secret_configured"`
	WebhookSecretMasked      string   `json:"webhook_secret_masked,omitempty"`
	DingTalkRobotURL         *string  `json:"dingtalk_robot_url,omitempty"`
	DingTalkSecretConfigured bool     `json:"dingtalk_robot_secret_configured"`
	DingTalkSecretMasked     string   `json:"dingtalk_robot_secret_masked,omitempty"`
}

type AlertRuleItem struct {
	Id                  int                          `json:"id"`
	TenantId            int                          `json:"tenant_id"`
	Name                string                       `json:"name"`
	Enabled             bool                         `json:"enabled"`
	RiskTypes           []string                     `json:"risk_types"`
	DepartmentIds       []int                        `json:"department_ids"`
	ChannelConfigs      []AlertRuleChannelConfigItem `json:"channel_configs"`
	DedupeWindowSeconds int                          `json:"dedupe_window_seconds"`
	CreatedBy           int                          `json:"created_by"`
	UpdatedBy           int                          `json:"updated_by"`
	CreatedAt           int64                        `json:"created_at"`
	UpdatedAt           int64                        `json:"updated_at"`
}

type AlertRulesQuery struct {
	TenantId *int `form:"tenant_id,omitempty"`
}

type AlertRuleUpsertRequest struct {
	Id                  *int                          `json:"id,omitempty"`
	TenantId            *int                          `json:"tenant_id,omitempty"`
	Name                *string                       `json:"name,omitempty"`
	Enabled             *bool                         `json:"enabled,omitempty"`
	RiskTypes           []string                      `json:"risk_types"`
	DepartmentIds       []int                         `json:"department_ids"`
	ChannelConfigs      []AlertRuleChannelConfigInput `json:"channel_configs"`
	DedupeWindowSeconds *int                          `json:"dedupe_window_seconds,omitempty"`
}

type AlertRulesResponse struct {
	Items []AlertRuleItem `json:"items"`
	Total int             `json:"total"`
}

type AlertRuleResponse struct {
	Item AlertRuleItem `json:"item"`
}
