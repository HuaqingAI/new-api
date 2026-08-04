package agentplatform

import (
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

func (s *KnowledgeQueryService) WithProvider(_ KnowledgeProvider) *KnowledgeQueryService {
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
	return KnowledgeQueryResult{}, ErrOpenCapabilityContractInvalid
}
