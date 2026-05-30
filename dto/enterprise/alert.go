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
