package agentplatform

import (
	"strings"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

type AgentQuery struct {
	OwnerUserId *int
	TenantId    *int
	Page        int
	PageSize    int
}

type AgentCreateInput struct {
	DisplayName string
	OwnerUserId int
	TenantId    int
}

type AgentUpdateInput struct {
	DisplayName string
}

type AgentItem struct {
	ResourceItem
}

type AgentListResult struct {
	Items    []AgentItem
	Total    int
	Page     int
	PageSize int
}

type AgentService struct {
	resources *ResourceService
	db        *gorm.DB
}

func NewAgentService(db *gorm.DB) *AgentService {
	return &AgentService{
		resources: NewResourceService(db),
		db:        db,
	}
}

func (s *AgentService) List(query AgentQuery) (AgentListResult, error) {
	if s == nil || s.resources == nil {
		return AgentListResult{Items: []AgentItem{}}, ErrInvalidResourceInput
	}
	result, err := s.resources.List(ListResourcesQuery{
		ResourceType: apmodel.ResourceTypeAgent,
		OwnerUserId:  query.OwnerUserId,
		TenantId:     query.TenantId,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		return AgentListResult{Items: []AgentItem{}}, err
	}
	items := make([]AgentItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, AgentItem{ResourceItem: item})
	}
	return AgentListResult{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func (s *AgentService) Get(resourceID string) (AgentItem, error) {
	if s == nil || s.resources == nil {
		return AgentItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.GetByResourceID(resourceID)
	if err != nil {
		return AgentItem{}, err
	}
	if item.ResourceType != apmodel.ResourceTypeAgent {
		return AgentItem{}, ErrResourceNotFound
	}
	return AgentItem{ResourceItem: item}, nil
}

func (s *AgentService) Create(input AgentCreateInput) (AgentItem, error) {
	if s == nil || s.resources == nil {
		return AgentItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.Create(CreateResourceInput{
		ResourceType: apmodel.ResourceTypeAgent,
		DisplayName:  input.DisplayName,
		OwnerUserId:  input.OwnerUserId,
		TenantId:     input.TenantId,
	})
	if err != nil {
		return AgentItem{}, err
	}
	return AgentItem{ResourceItem: item}, nil
}

func (s *AgentService) Update(resourceID string, input AgentUpdateInput) (AgentItem, error) {
	if s == nil || s.db == nil {
		return AgentItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if resourceID == "" || input.DisplayName == "" {
		return AgentItem{}, ErrInvalidResourceInput
	}

	var resource apmodel.Resource
	if err := s.db.Where("resource_id = ? AND resource_type = ?", resourceID, apmodel.ResourceTypeAgent).First(&resource).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return AgentItem{}, ErrResourceNotFound
		}
		return AgentItem{}, err
	}
	if err := s.db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", resourceID).
		Update("display_name", input.DisplayName).Error; err != nil {
		return AgentItem{}, err
	}
	return s.Get(resourceID)
}
