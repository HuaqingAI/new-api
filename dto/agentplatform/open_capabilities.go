package agentplatform

import "encoding/json"

type OpenCapabilityDiscoveryQuery struct {
	ResourceType string `form:"resource_type" json:"resource_type,omitempty"`
}

type OpenCapabilityDiscoveryItem struct {
	ResourceId          string          `json:"resource_id"`
	ResourceType        string          `json:"resource_type"`
	DisplayName         string          `json:"display_name"`
	ResourceVersion     string          `json:"resource_version"`
	ContractVersion     string          `json:"contract_version"`
	VisibilityState     string          `json:"visibility_state"`
	CallableState       string          `json:"callable_state"`
	FreshnessTTLSeconds int             `json:"freshness_ttl_seconds"`
	Freshness           string          `json:"freshness"`
	ETag                string          `json:"etag"`
	Extensions          json.RawMessage `json:"extensions,omitempty"`
}

type OpenCapabilityDiscoveryResponse struct {
	Items []OpenCapabilityDiscoveryItem `json:"items"`
	Total int                           `json:"total"`
}

type OpenCapabilityDetailResponse struct {
	ResourceId          string          `json:"resource_id"`
	ResourceType        string          `json:"resource_type"`
	DisplayName         string          `json:"display_name"`
	ResourceVersion     string          `json:"resource_version"`
	ContractVersion     string          `json:"contract_version"`
	Status              string          `json:"status"`
	VisibilityState     string          `json:"visibility_state"`
	CallableState       string          `json:"callable_state"`
	FreshnessTTLSeconds int             `json:"freshness_ttl_seconds"`
	Freshness           string          `json:"freshness"`
	ETag                string          `json:"etag"`
	Schema              json.RawMessage `json:"schema,omitempty"`
	Detail              json.RawMessage `json:"detail,omitempty"`
	Extensions          json.RawMessage `json:"extensions,omitempty"`
	SupportedExtensions []string        `json:"supported_extensions,omitempty"`
}

type OpenCapabilityRefreshRequest struct {
	ResourceId string `json:"resource_id" binding:"required"`
}

type OpenCapabilityRefreshResponse struct {
	ResourceId          string `json:"resource_id"`
	ResourceVersion     string `json:"resource_version"`
	ContractVersion     string `json:"contract_version"`
	FreshnessTTLSeconds int    `json:"freshness_ttl_seconds"`
	Freshness           string `json:"freshness"`
	ETag                string `json:"etag"`
	VisibilityState     string `json:"visibility_state"`
	CallableState       string `json:"callable_state"`
}

type OpenCapabilityInvokePlaceholderResponse struct {
	ResourceId string `json:"resource_id"`
	Status     string `json:"status"`
}
