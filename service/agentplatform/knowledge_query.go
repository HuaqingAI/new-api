package agentplatform

import (
	"encoding/json"
	"errors"
	"strings"

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
	db *gorm.DB
}

func NewKnowledgeQueryService(db *gorm.DB) *KnowledgeQueryService {
	return &KnowledgeQueryService{db: db}
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

	result := KnowledgeQueryResult{
		ResourceID:      detail.ResourceID,
		ResourceVersion: detail.ResourceVersion,
		ContractVersion: detail.ContractVersion,
		Items: []KnowledgeResultItem{
			{
				ID:      "doc-1",
				Score:   0.98,
				Snippet: "Retrieval-ready Knowledge result for query: " + strings.TrimSpace(request.Query),
				Metadata: map[string]any{
					"provider_type": knowledgeDef.ProviderType,
				},
			},
		},
		Citations: []KnowledgeCitation{
			{
				SourceID: "doc-1",
				Title:    "Knowledge Source",
				URL:      "https://example.com/knowledge/doc-1",
				Metadata: map[string]any{
					"provider_adapter_key": knowledgeDef.ProviderAdapterKey,
				},
			},
		},
	}
	return result, nil
}
