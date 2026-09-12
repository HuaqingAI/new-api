package agentplatform

type CreateSkillRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description,omitempty"`
	OwnerUserId int    `json:"owner_user_id,omitempty"`
	TenantId    *int   `json:"tenant_id,omitempty"`
}

type UpdateSkillRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description,omitempty"`
}

type SkillQuery struct {
	OwnerUserId *int `form:"owner_user_id" json:"owner_user_id,omitempty"`
	TenantId    *int `form:"tenant_id" json:"tenant_id,omitempty"`
	Page        *int `form:"page" json:"page,omitempty"`
	PageSize    *int `form:"page_size" json:"page_size,omitempty"`
}

type SkillItem = ResourceItem

type SkillDetailItem struct {
	ResourceItem
	FileName  string `json:"file_name"`
	Sha256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type SkillListResponse struct {
	Items    []SkillItem `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
