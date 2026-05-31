package agentplatform

import "encoding/json"

type CreateExposureRequest struct {
	ResourceVersion     string          `json:"resource_version" binding:"required"`
	ClientKey           string          `json:"client_key" binding:"required"`
	ClientScope         string          `json:"client_scope,omitempty"`
	VisibilityState     string          `json:"visibility_state,omitempty"`
	CallableState       string          `json:"callable_state,omitempty"`
	FreshnessTTLSeconds *int            `json:"freshness_ttl_seconds,omitempty"`
	ETag                string          `json:"etag,omitempty"`
	Extensions          json.RawMessage `json:"extensions,omitempty"`
}

type UpdateExposureRequest struct {
	VisibilityState     string          `json:"visibility_state,omitempty"`
	CallableState       string          `json:"callable_state,omitempty"`
	FreshnessTTLSeconds *int            `json:"freshness_ttl_seconds,omitempty"`
	ETag                string          `json:"etag,omitempty"`
	Extensions          json.RawMessage `json:"extensions,omitempty"`
}

type ExposureQuery struct {
	ResourceId      string `form:"resource_id" json:"resource_id,omitempty"`
	ResourceVersion string `form:"resource_version" json:"resource_version,omitempty"`
	ClientKey       string `form:"client_key" json:"client_key,omitempty"`
	ClientScope     string `form:"client_scope" json:"client_scope,omitempty"`
}

type ExposureItem struct {
	Id                  int             `json:"id"`
	ResourceId          string          `json:"resource_id"`
	ResourceVersion     string          `json:"resource_version"`
	ClientKey           string          `json:"client_key"`
	ClientScope         string          `json:"client_scope"`
	VisibilityState     string          `json:"visibility_state"`
	CallableState       string          `json:"callable_state"`
	FreshnessTTLSeconds int             `json:"freshness_ttl_seconds"`
	ETag                string          `json:"etag"`
	Extensions          json.RawMessage `json:"extensions,omitempty"`
	PublishedAt         *int64          `json:"published_at,omitempty"`
	RevokedAt           *int64          `json:"revoked_at,omitempty"`
	CreatedAt           int64           `json:"created_at"`
	UpdatedAt           int64           `json:"updated_at"`
}

type ExposureListResponse struct {
	Items []ExposureItem `json:"items"`
	Total int            `json:"total"`
}
