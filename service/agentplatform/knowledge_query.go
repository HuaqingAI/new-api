package agentplatform

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

type KnowledgeQueryInput struct {
	ClientID   string
	ResourceID string
	Payload    []byte
}

type KnowledgeQueryResult struct {
	ResourceID      string                `json:"resource_id"`
	ResourceVersion string                `json:"resource_version"`
	ContractVersion string                `json:"contract_version"`
	Items           []KnowledgeResultItem `json:"items"`
	Citations       []KnowledgeCitation   `json:"citations"`
}

type KnowledgeResultItem struct {
	ID       string         `json:"id"`
	Score    float64        `json:"score"`
	Snippet  string         `json:"snippet"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type KnowledgeCitation struct {
	SourceID string         `json:"source_id"`
	Title    string         `json:"title,omitempty"`
	URL      string         `json:"url,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type KnowledgeQueryService struct {
	db       *gorm.DB
	provider KnowledgeProvider
}

func NewKnowledgeQueryService(db *gorm.DB) *KnowledgeQueryService {
	return &KnowledgeQueryService{
		db:       db,
		provider: NewHTTPRetrievalProvider(),
	}
}

func (s *KnowledgeQueryService) WithProvider(provider KnowledgeProvider) *KnowledgeQueryService {
	if provider != nil {
		s.provider = provider
	}
	return s
}

func (s *KnowledgeQueryService) Query(input KnowledgeQueryInput) (KnowledgeQueryResult, error) {
	if s == nil || s.db == nil {
		return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
	}
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	if input.ClientID == "" || input.ResourceID == "" || len(input.Payload) == 0 {
		return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
	}

	discovery := NewDiscoveryService(s.db)
	detail, err := discovery.Detail(input.ClientID, input.ResourceID)
	if err != nil {
		return KnowledgeQueryResult{}, err
	}
	if detail.ResourceType != apmodel.ResourceTypeKnowledge || detail.CallableState != apmodel.ExposureCallableEnabled {
		return KnowledgeQueryResult{}, ErrOpenCapabilityPermissionDenied
	}

	var knowledgeDef apmodel.KnowledgeDef
	if err := s.db.Where("resource_id = ? AND resource_version = ?", input.ResourceID, detail.ResourceVersion).First(&knowledgeDef).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
		}
		return KnowledgeQueryResult{}, err
	}
	if knowledgeDef.KnowledgeMode != "retrieval" {
		return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
	}

	var request struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input.Payload, &request); err != nil {
		return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
	}
	if strings.TrimSpace(request.Query) == "" {
		return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
	}
	bindingConfig := map[string]any{}
	if err := common.UnmarshalJsonStr(knowledgeDef.ProviderConfigJSON, &bindingConfig); err != nil {
		return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
	}
	binding := KnowledgeProviderBinding{
		ProviderType:       knowledgeDef.ProviderType,
		ProviderAdapterKey: knowledgeDef.ProviderAdapterKey,
		Config:             bindingConfig,
	}
	if err := s.provider.ValidateBinding(context.Background(), binding); err != nil {
		return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
	}
	providerResponse, err := s.provider.Query(context.Background(), binding, KnowledgeProviderQueryRequest{
		Query: strings.TrimSpace(request.Query),
	})
	if err != nil {
		return KnowledgeQueryResult{}, err
	}

	result := KnowledgeQueryResult{
		ResourceID:      detail.ResourceID,
		ResourceVersion: detail.ResourceVersion,
		ContractVersion: detail.ContractVersion,
		Items:           providerResponse.Items,
		Citations:       providerResponse.Citations,
	}
	return result, nil
}
