package enterprise

type AdminActionQuery struct {
	TenantId   *int   `form:"tenant_id" json:"tenant_id,omitempty"`
	ActorId    *int   `form:"actor_id" json:"actor_id,omitempty"`
	ActionType string `form:"action_type" json:"action_type,omitempty"`
	ObjectType string `form:"object_type" json:"object_type,omitempty"`
	ObjectId   string `form:"object_id" json:"object_id,omitempty"`
	StartAt    *int64 `form:"start_at" json:"start_at,omitempty"`
	EndAt      *int64 `form:"end_at" json:"end_at,omitempty"`
	Page       *int   `form:"page" json:"page,omitempty"`
	PageSize   *int   `form:"page_size" json:"page_size,omitempty"`
}

type AdminActionItem struct {
	ActionId    int    `json:"action_id"`
	TenantId    int    `json:"tenant_id"`
	ActorId     int    `json:"actor_id"`
	ActionType  string `json:"action_type"`
	ObjectType  string `json:"object_type"`
	ObjectId    string `json:"object_id"`
	CreatedAt   int64  `json:"created_at"`
	DiffSummary string `json:"diff_summary"`
	Payload     string `json:"payload,omitempty"`
}

type AdminActionsResponse struct {
	Items    []AdminActionItem `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type DepartmentAdminRoleRequest struct {
	TenantId *int `json:"tenant_id,omitempty"`
	UserId   int  `json:"user_id"`
}
