package agentplatform

import (
	"errors"
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
	DisplayName         string
	Description         string
	ExternalKnowledgeId string
	OwnerUserId         int
	TenantId            int
}

type KnowledgeUpdateInput struct {
	DisplayName         string
	Description         string
	ExternalKnowledgeId string
}

type KnowledgeItem struct {
	ResourceItem
	ExternalKnowledgeId string
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
		def, err := s.getDef(item.ResourceId)
		if err != nil && !errors.Is(err, ErrResourceNotFound) {
			return KnowledgeListResult{Items: []KnowledgeItem{}}, err
		}
		items = append(items, KnowledgeItem{ResourceItem: item, ExternalKnowledgeId: def.ExternalKnowledgeId})
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
	def, err := s.getDef(resourceID)
	if err != nil && !errors.Is(err, ErrResourceNotFound) {
		return KnowledgeItem{}, err
	}
	return KnowledgeItem{ResourceItem: item, ExternalKnowledgeId: def.ExternalKnowledgeId}, nil
}

func (s *KnowledgeService) Create(input KnowledgeCreateInput) (KnowledgeItem, error) {
	if s == nil || s.db == nil {
		return KnowledgeItem{}, ErrInvalidResourceInput
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.ExternalKnowledgeId = strings.TrimSpace(input.ExternalKnowledgeId)
	if input.DisplayName == "" || input.OwnerUserId <= 0 {
		return KnowledgeItem{}, ErrInvalidResourceInput
	}
	if input.ExternalKnowledgeId == "" {
		item, err := s.resources.Create(CreateResourceInput{
			ResourceType: apmodel.ResourceTypeKnowledge,
			DisplayName:  input.DisplayName,
			Description:  input.Description,
			OwnerUserId:  input.OwnerUserId,
			TenantId:     input.TenantId,
			Status:       apmodel.ResourceStatusPublished,
		})
		if err != nil {
			return KnowledgeItem{}, err
		}
		return KnowledgeItem{ResourceItem: item}, nil
	}
	var resourceID string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		resource := apmodel.Resource{
			ResourceType: apmodel.ResourceTypeKnowledge,
			DisplayName:  input.DisplayName,
			Description:  input.Description,
			OwnerUserId:  input.OwnerUserId,
			TenantId:     input.TenantId,
			Status:       apmodel.ResourceStatusPublished,
		}
		if err := tx.Create(&resource).Error; err != nil {
			return err
		}
		def := apmodel.KnowledgeDef{
			ResourceId:          resource.ResourceId,
			ResourceVersion:     "",
			KnowledgeMode:       "reference",
			ProviderType:        "external_id",
			ProviderAdapterKey:  "new-api-knowledge",
			ExternalKnowledgeId: input.ExternalKnowledgeId,
		}
		if err := tx.Create(&def).Error; err != nil {
			return err
		}
		resourceID = resource.ResourceId
		return nil
	})
	if err != nil {
		if errors.Is(err, apmodel.ErrInvalidKnowledgeDefBody) {
			return KnowledgeItem{}, ErrInvalidResourceInput
		}
		return KnowledgeItem{}, err
	}
	return s.Get(resourceID)
}

func (s *KnowledgeService) Update(resourceID string, input KnowledgeUpdateInput) (KnowledgeItem, error) {
	if s == nil || s.db == nil {
		return KnowledgeItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.ExternalKnowledgeId = strings.TrimSpace(input.ExternalKnowledgeId)
	if resourceID == "" || input.DisplayName == "" {
		return KnowledgeItem{}, ErrInvalidResourceInput
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var resource apmodel.Resource
		if err := tx.Where("resource_id = ? AND resource_type = ?", resourceID, apmodel.ResourceTypeKnowledge).First(&resource).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResourceNotFound
			}
			return err
		}
		if err := tx.Model(&apmodel.Resource{}).Where("resource_id = ?", resourceID).Updates(map[string]any{
			"display_name": input.DisplayName,
			"description":  input.Description,
		}).Error; err != nil {
			return err
		}
		if input.ExternalKnowledgeId == "" {
			return nil
		}
		result := tx.Model(&apmodel.KnowledgeDef{}).Where("resource_id = ?", resourceID).Update("external_knowledge_id", input.ExternalKnowledgeId)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return tx.Create(&apmodel.KnowledgeDef{
				ResourceId:          resourceID,
				ResourceVersion:     "",
				KnowledgeMode:       "reference",
				ProviderType:        "external_id",
				ProviderAdapterKey:  "new-api-knowledge",
				ExternalKnowledgeId: input.ExternalKnowledgeId,
			}).Error
		}
		return nil
	})
	if err != nil {
		return KnowledgeItem{}, err
	}
	return s.Get(resourceID)
}

func (s *KnowledgeService) SetStatus(resourceID string, status string) (KnowledgeItem, error) {
	if err := setTypedResourceStatus(s.db, resourceID, apmodel.ResourceTypeKnowledge, status); err != nil {
		return KnowledgeItem{}, err
	}
	return s.Get(resourceID)
}

func (s *KnowledgeService) Delete(resourceID string) error {
	return setTypedResourceStatus(s.db, resourceID, apmodel.ResourceTypeKnowledge, apmodel.ResourceStatusRevoked)
}

func (s *KnowledgeService) getDef(resourceID string) (apmodel.KnowledgeDef, error) {
	var def apmodel.KnowledgeDef
	if err := s.db.Where("resource_id = ?", resourceID).First(&def).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apmodel.KnowledgeDef{}, ErrResourceNotFound
		}
		return apmodel.KnowledgeDef{}, err
	}
	return def, nil
}
