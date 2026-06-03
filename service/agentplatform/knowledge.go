package agentplatform

import (
	"strings"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

type KnowledgeQuery struct {
	OwnerUserId *int
	TenantId    *int
	Page        int
	PageSize    int
}

type KnowledgeCreateInput struct {
	DisplayName string
	OwnerUserId int
	TenantId    int
}

type KnowledgeUpdateInput struct {
	DisplayName string
}

type KnowledgeItem struct {
	ResourceItem
}

type KnowledgeListResult struct {
	Items    []KnowledgeItem
	Total    int
	Page     int
	PageSize int
}

type KnowledgeService struct {
	resources *ResourceService
	db        *gorm.DB
}

func NewKnowledgeService(db *gorm.DB) *KnowledgeService {
	return &KnowledgeService{
		resources: NewResourceService(db),
		db:        db,
	}
}

func (s *KnowledgeService) List(query KnowledgeQuery) (KnowledgeListResult, error) {
	if s == nil || s.resources == nil {
		return KnowledgeListResult{Items: []KnowledgeItem{}}, ErrInvalidResourceInput
	}
	result, err := s.resources.List(ListResourcesQuery{
		ResourceType: apmodel.ResourceTypeKnowledge,
		OwnerUserId:  query.OwnerUserId,
		TenantId:     query.TenantId,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		return KnowledgeListResult{Items: []KnowledgeItem{}}, err
	}
	items := make([]KnowledgeItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, KnowledgeItem{ResourceItem: item})
	}
	return KnowledgeListResult{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func (s *KnowledgeService) Get(resourceID string) (KnowledgeItem, error) {
	if s == nil || s.resources == nil {
		return KnowledgeItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.GetByResourceID(resourceID)
	if err != nil {
		return KnowledgeItem{}, err
	}
	if item.ResourceType != apmodel.ResourceTypeKnowledge {
		return KnowledgeItem{}, ErrResourceNotFound
	}
	return KnowledgeItem{ResourceItem: item}, nil
}

func (s *KnowledgeService) Create(input KnowledgeCreateInput) (KnowledgeItem, error) {
	if s == nil || s.resources == nil {
		return KnowledgeItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.Create(CreateResourceInput{
		ResourceType: apmodel.ResourceTypeKnowledge,
		DisplayName:  input.DisplayName,
		OwnerUserId:  input.OwnerUserId,
		TenantId:     input.TenantId,
	})
	if err != nil {
		return KnowledgeItem{}, err
	}
	return KnowledgeItem{ResourceItem: item}, nil
}

func (s *KnowledgeService) Update(resourceID string, input KnowledgeUpdateInput) (KnowledgeItem, error) {
	if s == nil || s.db == nil {
		return KnowledgeItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if resourceID == "" || input.DisplayName == "" {
		return KnowledgeItem{}, ErrInvalidResourceInput
	}

	var resource apmodel.Resource
	if err := s.db.Where("resource_id = ? AND resource_type = ?", resourceID, apmodel.ResourceTypeKnowledge).First(&resource).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return KnowledgeItem{}, ErrResourceNotFound
		}
		return KnowledgeItem{}, err
	}
	if err := s.db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", resourceID).
		Update("display_name", input.DisplayName).Error; err != nil {
		return KnowledgeItem{}, err
	}
	return s.Get(resourceID)
}
