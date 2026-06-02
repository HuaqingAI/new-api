package agentplatform

import (
	"strings"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

type SkillQuery struct {
	OwnerUserId *int
	TenantId    *int
	Page        int
	PageSize    int
}

type SkillCreateInput struct {
	DisplayName string
	OwnerUserId int
	TenantId    int
}

type SkillUpdateInput struct {
	DisplayName string
}

type SkillItem struct {
	ResourceItem
}

type SkillListResult struct {
	Items    []SkillItem
	Total    int
	Page     int
	PageSize int
}

type SkillService struct {
	resources *ResourceService
	db        *gorm.DB
}

func NewSkillService(db *gorm.DB) *SkillService {
	return &SkillService{
		resources: NewResourceService(db),
		db:        db,
	}
}

func (s *SkillService) List(query SkillQuery) (SkillListResult, error) {
	if s == nil || s.resources == nil {
		return SkillListResult{Items: []SkillItem{}}, ErrInvalidResourceInput
	}
	result, err := s.resources.List(ListResourcesQuery{
		ResourceType: apmodel.ResourceTypeSkill,
		OwnerUserId:  query.OwnerUserId,
		TenantId:     query.TenantId,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		return SkillListResult{Items: []SkillItem{}}, err
	}
	items := make([]SkillItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, SkillItem{ResourceItem: item})
	}
	return SkillListResult{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func (s *SkillService) Get(resourceID string) (SkillItem, error) {
	if s == nil || s.resources == nil {
		return SkillItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.GetByResourceID(resourceID)
	if err != nil {
		return SkillItem{}, err
	}
	if item.ResourceType != apmodel.ResourceTypeSkill {
		return SkillItem{}, ErrResourceNotFound
	}
	return SkillItem{ResourceItem: item}, nil
}

func (s *SkillService) Create(input SkillCreateInput) (SkillItem, error) {
	if s == nil || s.resources == nil {
		return SkillItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.Create(CreateResourceInput{
		ResourceType: apmodel.ResourceTypeSkill,
		DisplayName:  input.DisplayName,
		OwnerUserId:  input.OwnerUserId,
		TenantId:     input.TenantId,
	})
	if err != nil {
		return SkillItem{}, err
	}
	return SkillItem{ResourceItem: item}, nil
}

func (s *SkillService) Update(resourceID string, input SkillUpdateInput) (SkillItem, error) {
	if s == nil || s.db == nil {
		return SkillItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if resourceID == "" || input.DisplayName == "" {
		return SkillItem{}, ErrInvalidResourceInput
	}

	var resource apmodel.Resource
	if err := s.db.Where("resource_id = ? AND resource_type = ?", resourceID, apmodel.ResourceTypeSkill).First(&resource).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return SkillItem{}, ErrResourceNotFound
		}
		return SkillItem{}, err
	}
	if err := s.db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", resourceID).
		Update("display_name", input.DisplayName).Error; err != nil {
		return SkillItem{}, err
	}
	return s.Get(resourceID)
}
