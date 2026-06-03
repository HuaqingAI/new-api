package agentplatform

import "encoding/json"

type SkillDetailRequest struct {
	InvokeSchema   json.RawMessage `json:"invoke_schema,omitempty"`
	OutputSchema   json.RawMessage `json:"output_schema,omitempty"`
	InvokeMode     string          `json:"invoke_mode,omitempty"`
	TimeoutSeconds *int            `json:"timeout_seconds,omitempty"`
	BindingConfig  json.RawMessage `json:"binding_config,omitempty"`
}

type KnowledgeDetailRequest struct {
	KnowledgeMode        string          `json:"knowledge_mode,omitempty"`
	ProviderType         string          `json:"provider_type,omitempty"`
	ProviderAdapterKey   string          `json:"provider_adapter_key,omitempty"`
	ProviderConfig       json.RawMessage `json:"provider_config,omitempty"`
	QuerySchema          json.RawMessage `json:"query_schema,omitempty"`
	CitationSchema       json.RawMessage `json:"citation_schema,omitempty"`
	FreshnessRules       json.RawMessage `json:"freshness_rules,omitempty"`
	ProviderCapabilities json.RawMessage `json:"provider_capabilities,omitempty"`
}

type AgentDetailRequest struct {
	Manifest            json.RawMessage `json:"manifest,omitempty"`
	Dependencies        json.RawMessage `json:"dependencies,omitempty"`
	PromptMetadata      json.RawMessage `json:"prompt_metadata,omitempty"`
	CompatibilityMeta   json.RawMessage `json:"compatibility_metadata,omitempty"`
}

type CreateResourceVersionRequest struct {
	Version         string                  `json:"version" binding:"required"`
	ContractVersion string                  `json:"contract_version" binding:"required"`
	Summary         string                  `json:"summary,omitempty"`
	Schema          json.RawMessage         `json:"schema,omitempty"`
	Skill           *SkillDetailRequest     `json:"skill,omitempty"`
	Knowledge       *KnowledgeDetailRequest `json:"knowledge,omitempty"`
	Agent           *AgentDetailRequest     `json:"agent,omitempty"`
}

type SkillDetailResponse struct {
	InvokeSchema   json.RawMessage `json:"invoke_schema,omitempty"`
	OutputSchema   json.RawMessage `json:"output_schema,omitempty"`
	InvokeMode     string          `json:"invoke_mode"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	BindingConfig  json.RawMessage `json:"binding_config,omitempty"`
}

type KnowledgeDetailResponse struct {
	KnowledgeMode        string          `json:"knowledge_mode"`
	ProviderType         string          `json:"provider_type"`
	ProviderAdapterKey   string          `json:"provider_adapter_key"`
	ProviderConfig       json.RawMessage `json:"provider_config,omitempty"`
	QuerySchema          json.RawMessage `json:"query_schema,omitempty"`
	CitationSchema       json.RawMessage `json:"citation_schema,omitempty"`
	FreshnessRules       json.RawMessage `json:"freshness_rules,omitempty"`
	ProviderCapabilities json.RawMessage `json:"provider_capabilities,omitempty"`
}

type AgentDetailResponse struct {
	Manifest          json.RawMessage `json:"manifest,omitempty"`
	Dependencies      json.RawMessage `json:"dependencies,omitempty"`
	PromptMetadata    json.RawMessage `json:"prompt_metadata,omitempty"`
	CompatibilityMeta json.RawMessage `json:"compatibility_metadata,omitempty"`
}

type ResourceVersionItem struct {
	ResourceId      string                   `json:"resource_id"`
	ResourceType    string                   `json:"resource_type"`
	Version         string                   `json:"version"`
	ContractVersion string                   `json:"contract_version"`
	Summary         string                   `json:"summary"`
	Schema          json.RawMessage          `json:"schema,omitempty"`
	Status          string                   `json:"status"`
	CreatedBy       int                      `json:"created_by"`
	PublishedAt     *int64                   `json:"published_at,omitempty"`
	CreatedAt       int64                    `json:"created_at"`
	UpdatedAt       int64                    `json:"updated_at"`
	Skill           *SkillDetailResponse     `json:"skill,omitempty"`
	Knowledge       *KnowledgeDetailResponse `json:"knowledge,omitempty"`
	Agent           *AgentDetailResponse     `json:"agent,omitempty"`
}
