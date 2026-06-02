package agentplatform

type CreateSkillRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	OwnerUserId int    `json:"owner_user_id" binding:"required"`
	TenantId    *int   `json:"tenant_id,omitempty"`
}

type UpdateSkillRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
}

type SkillQuery struct {
	OwnerUserId *int `form:"owner_user_id" json:"owner_user_id,omitempty"`
	TenantId    *int `form:"tenant_id" json:"tenant_id,omitempty"`
	Page        *int `form:"page" json:"page,omitempty"`
	PageSize    *int `form:"page_size" json:"page_size,omitempty"`
}

type SkillItem = ResourceItem

type SkillListResponse struct {
	Items    []SkillItem `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
