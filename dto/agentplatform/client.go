package agentplatform

import "encoding/json"

type CreateClientRequest struct {
	Slug                   string          `json:"slug" binding:"required"`
	DisplayName            string          `json:"display_name" binding:"required"`
	ClientType             string          `json:"client_type" binding:"required"`
	Status                 string          `json:"status,omitempty"`
	AllowedGrantTypes      json.RawMessage `json:"allowed_grant_types,omitempty"`
	RedirectURIs           json.RawMessage `json:"redirect_uris,omitempty"`
	AllowedScopes          json.RawMessage `json:"allowed_scopes,omitempty"`
	ContractVersion        string          `json:"contract_version,omitempty"`
	Capabilities           json.RawMessage `json:"capabilities,omitempty"`
	Extensions             json.RawMessage `json:"extensions,omitempty"`
	AllowClientCredentials bool            `json:"allow_client_credentials,omitempty"`
}

type UpdateClientRequest struct {
	DisplayName            string          `json:"display_name,omitempty"`
	Status                 string          `json:"status,omitempty"`
	AllowedGrantTypes      json.RawMessage `json:"allowed_grant_types,omitempty"`
	RedirectURIs           json.RawMessage `json:"redirect_uris,omitempty"`
	AllowedScopes          json.RawMessage `json:"allowed_scopes,omitempty"`
	ContractVersion        string          `json:"contract_version,omitempty"`
	Capabilities           json.RawMessage `json:"capabilities,omitempty"`
	Extensions             json.RawMessage `json:"extensions,omitempty"`
	AllowClientCredentials *bool           `json:"allow_client_credentials,omitempty"`
}

type ClientQuery struct {
	Status          string `form:"status" json:"status,omitempty"`
	ContractVersion string `form:"contract_version" json:"contract_version,omitempty"`
}

type ClientItem struct {
	Id                     int             `json:"id"`
	ClientId               string          `json:"client_id"`
	Slug                   string          `json:"slug"`
	DisplayName            string          `json:"display_name"`
	ClientType             string          `json:"client_type"`
	Status                 string          `json:"status"`
	AllowedGrantTypes      json.RawMessage `json:"allowed_grant_types,omitempty"`
	RedirectURIs           json.RawMessage `json:"redirect_uris,omitempty"`
	AllowedScopes          json.RawMessage `json:"allowed_scopes,omitempty"`
	ContractVersion        string          `json:"contract_version"`
	Capabilities           json.RawMessage `json:"capabilities,omitempty"`
	Extensions             json.RawMessage `json:"extensions,omitempty"`
	AllowClientCredentials bool            `json:"allow_client_credentials"`
	CreatedAt              int64           `json:"created_at"`
	UpdatedAt              int64           `json:"updated_at"`
}

type ClientListResponse struct {
	Items []ClientItem `json:"items"`
	Total int          `json:"total"`
}
