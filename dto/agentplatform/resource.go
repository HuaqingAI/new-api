package agentplatform

type CreateResourceRequest struct {
	ResourceType string `json:"resource_type" binding:"required"`
	DisplayName  string `json:"display_name" binding:"required"`
	Description  string `json:"description,omitempty"`
	Avatar       string `json:"avatar,omitempty"`
	OwnerUserId  int    `json:"owner_user_id" binding:"required"`
	TenantId     *int   `json:"tenant_id,omitempty"`
}

type ResourceQuery struct {
	ResourceType string `form:"resource_type" json:"resource_type,omitempty"`
	OwnerUserId  *int   `form:"owner_user_id" json:"owner_user_id,omitempty"`
	TenantId     *int   `form:"tenant_id" json:"tenant_id,omitempty"`
	Page         *int   `form:"page" json:"page,omitempty"`
	PageSize     *int   `form:"page_size" json:"page_size,omitempty"`
}

type ResourceItem struct {
	Id            int    `json:"id"`
	ResourceId    string `json:"resource_id"`
	ResourceType  string `json:"resource_type"`
	DisplayName   string `json:"display_name"`
	Description   string `json:"description"`
	Avatar        string `json:"avatar"`
	OwnerUserId   int    `json:"owner_user_id"`
	OwnerName     string `json:"owner_name"`
	Status        string `json:"status"`
	LatestVersion string `json:"latest_version"`
	TenantId      int    `json:"tenant_id"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type ResourceListResponse struct {
	Items    []ResourceItem `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}
