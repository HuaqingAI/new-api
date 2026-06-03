package agentplatform

import "context"

type KnowledgeProvider interface {
	ValidateBinding(ctx context.Context, binding KnowledgeProviderBinding) error
	Query(ctx context.Context, binding KnowledgeProviderBinding, req KnowledgeProviderQueryRequest) (KnowledgeProviderQueryResponse, error)
	Refresh(ctx context.Context, binding KnowledgeProviderBinding) (KnowledgeProviderRefreshState, error)
	Health(ctx context.Context, binding KnowledgeProviderBinding) (KnowledgeProviderHealthState, error)
}

type KnowledgeProviderBinding struct {
	ProviderType       string
	ProviderAdapterKey string
	Config             map[string]any
}

type KnowledgeProviderQueryRequest struct {
	Query string
}

type KnowledgeProviderQueryResponse struct {
	Items     []KnowledgeResultItem
	Citations []KnowledgeCitation
}

type KnowledgeProviderRefreshState struct {
	Status string
}

type KnowledgeProviderHealthState struct {
	Status string
}
