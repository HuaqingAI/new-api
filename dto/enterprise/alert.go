package enterprise

type AlertEventQuery struct {
	TenantId     *int    `form:"tenant_id"`
	DepartmentId *int    `form:"department_id,omitempty"`
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
	Id                  int                      `json:"id"`
	TenantId            int                      `json:"tenant_id"`
	Name                string                   `json:"name"`
	Enabled             bool                     `json:"enabled"`
	RiskTypes           []string                 `json:"risk_types"`
	DepartmentIds       []int                    `json:"department_ids"`
	ChannelConfigs      []AlertRuleChannelConfigItem `json:"channel_configs"`
	DedupeWindowSeconds int                      `json:"dedupe_window_seconds"`
	CreatedBy           int                      `json:"created_by"`
	UpdatedBy           int                      `json:"updated_by"`
	CreatedAt           int64                    `json:"created_at"`
	UpdatedAt           int64                    `json:"updated_at"`
}

type AlertRulesQuery struct {
	TenantId *int `form:"tenant_id,omitempty"`
}

type AlertRuleUpsertRequest struct {
	Id                  *int                     `json:"id,omitempty"`
	TenantId            *int                     `json:"tenant_id,omitempty"`
	Name                *string                  `json:"name,omitempty"`
	Enabled             *bool                    `json:"enabled,omitempty"`
	RiskTypes           []string                 `json:"risk_types"`
	DepartmentIds       []int                    `json:"department_ids"`
	ChannelConfigs      []AlertRuleChannelConfigInput `json:"channel_configs"`
	DedupeWindowSeconds *int                     `json:"dedupe_window_seconds,omitempty"`
}

type AlertRulesResponse struct {
	Items []AlertRuleItem `json:"items"`
	Total int             `json:"total"`
}

type AlertRuleResponse struct {
	Item AlertRuleItem `json:"item"`
}
