package agentplatform

import "encoding/json"

type CreateMcpRequest struct {
	DisplayName string          `json:"display_name" binding:"required"`
	Description string          `json:"description,omitempty"`
	Config      json.RawMessage `json:"config" binding:"required"`
	OwnerUserId int             `json:"owner_user_id,omitempty"`
	TenantId    *int            `json:"tenant_id,omitempty"`
}

type UpdateMcpRequest struct {
	DisplayName string          `json:"display_name" binding:"required"`
	Description string          `json:"description,omitempty"`
	Config      json.RawMessage `json:"config" binding:"required"`
}

type McpQuery struct {
	OwnerUserId *int `form:"owner_user_id" json:"owner_user_id,omitempty"`
	TenantId    *int `form:"tenant_id" json:"tenant_id,omitempty"`
	Page        *int `form:"page" json:"page,omitempty"`
	PageSize    *int `form:"page_size" json:"page_size,omitempty"`
}

type McpItem struct {
	ResourceItem
	Config json.RawMessage `json:"config,omitempty"`
}

type McpListResponse struct {
	Items    []McpItem `json:"items"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}
