package agentplatform

type CreateKnowledgeRequest struct {
	DisplayName         string `json:"display_name" binding:"required"`
	Description         string `json:"description,omitempty"`
	ExternalKnowledgeId string `json:"external_knowledge_id,omitempty"`
	OwnerUserId         int    `json:"owner_user_id,omitempty"`
	TenantId            *int   `json:"tenant_id,omitempty"`
}

type UpdateKnowledgeRequest struct {
	DisplayName         string `json:"display_name" binding:"required"`
	Description         string `json:"description,omitempty"`
	ExternalKnowledgeId string `json:"external_knowledge_id,omitempty"`
}

type KnowledgeQuery struct {
	OwnerUserId *int `form:"owner_user_id" json:"owner_user_id,omitempty"`
	TenantId    *int `form:"tenant_id" json:"tenant_id,omitempty"`
	Page        *int `form:"page" json:"page,omitempty"`
	PageSize    *int `form:"page_size" json:"page_size,omitempty"`
}

type KnowledgeItem = ResourceItem

type KnowledgeDetailItem struct {
	ResourceItem
	ExternalKnowledgeId string `json:"external_knowledge_id"`
}

type KnowledgeListResponse struct {
	Items    []KnowledgeDetailItem `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}
