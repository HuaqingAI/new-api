package agentplatform

import (
	"encoding/json"
	"errors"
	"strings"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

type McpQuery struct {
	OwnerUserId *int
	TenantId    *int
	Page        int
	PageSize    int
}

type McpCreateInput struct {
	DisplayName string
	Description string
	Config      json.RawMessage
	OwnerUserId int
	TenantId    int
}

type McpUpdateInput struct {
	DisplayName string
	Description string
	Config      json.RawMessage
}

type McpItem struct {
	ResourceItem
	ConfigJSON string
}

type McpListResult struct {
	Items    []McpItem
	Total    int
	Page     int
	PageSize int
}

type McpService struct {
	resources *ResourceService
	db        *gorm.DB
}

func NewMcpService(db *gorm.DB) *McpService {
	return &McpService{resources: NewResourceService(db), db: db}
}

func (s *McpService) List(query McpQuery) (McpListResult, error) {
	if s == nil || s.resources == nil {
		return McpListResult{Items: []McpItem{}}, ErrInvalidResourceInput
	}
	result, err := s.resources.List(ListResourcesQuery{
		ResourceType: apmodel.ResourceTypeMCP,
		OwnerUserId:  query.OwnerUserId,
		TenantId:     query.TenantId,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		return McpListResult{Items: []McpItem{}}, err
	}
	items := make([]McpItem, 0, len(result.Items))
	for _, item := range result.Items {
		config, err := s.getConfig(item.ResourceId)
		if err != nil {
			return McpListResult{Items: []McpItem{}}, err
		}
		items = append(items, McpItem{ResourceItem: item, ConfigJSON: config})
	}
	return McpListResult{Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize}, nil
}

func (s *McpService) Get(resourceID string) (McpItem, error) {
	item, err := s.resources.GetByResourceID(resourceID)
	if err != nil {
		return McpItem{}, err
	}
	if item.ResourceType != apmodel.ResourceTypeMCP {
		return McpItem{}, ErrResourceNotFound
	}
	config, err := s.getConfig(item.ResourceId)
	if err != nil {
		return McpItem{}, err
	}
	return McpItem{ResourceItem: item, ConfigJSON: config}, nil
}

func (s *McpService) Create(input McpCreateInput) (McpItem, error) {
	if s == nil || s.db == nil {
		return McpItem{}, ErrInvalidResourceInput
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	configJSON, err := normalizeJSONText(input.Config)
	if err != nil || !apmodel.McpDefConfigValidForService(configJSON) {
		return McpItem{}, ErrInvalidResourceInput
	}
	if input.DisplayName == "" || input.OwnerUserId <= 0 {
		return McpItem{}, ErrInvalidResourceInput
	}
	var createdID string
	err = s.db.Transaction(func(tx *gorm.DB) error {
		resource := apmodel.Resource{
			ResourceType: apmodel.ResourceTypeMCP,
			DisplayName:  input.DisplayName,
			Description:  input.Description,
			OwnerUserId:  input.OwnerUserId,
			TenantId:     input.TenantId,
			Status:       apmodel.ResourceStatusPublished,
		}
		if err := tx.Create(&resource).Error; err != nil {
			return err
		}
		if err := tx.Create(&apmodel.McpDef{ResourceId: resource.ResourceId, ConfigJSON: configJSON}).Error; err != nil {
			return err
		}
		createdID = resource.ResourceId
		return nil
	})
	if err != nil {
		if errors.Is(err, apmodel.ErrInvalidMcpDefBody) {
			return McpItem{}, ErrInvalidResourceInput
		}
		return McpItem{}, err
	}
	return s.Get(createdID)
}

func (s *McpService) Update(resourceID string, input McpUpdateInput) (McpItem, error) {
	if s == nil || s.db == nil {
		return McpItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	configJSON, err := normalizeJSONText(input.Config)
	if err != nil || !apmodel.McpDefConfigValidForService(configJSON) || resourceID == "" || input.DisplayName == "" {
		return McpItem{}, ErrInvalidResourceInput
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var resource apmodel.Resource
		if err := tx.Where("resource_id = ? AND resource_type = ?", resourceID, apmodel.ResourceTypeMCP).First(&resource).Error; err != nil {
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
		return tx.Model(&apmodel.McpDef{}).Where("resource_id = ?", resourceID).Update("config_json", configJSON).Error
	})
	if err != nil {
		return McpItem{}, err
	}
	return s.Get(resourceID)
}

func (s *McpService) SetStatus(resourceID string, status string) (McpItem, error) {
	if err := setTypedResourceStatus(s.db, resourceID, apmodel.ResourceTypeMCP, status); err != nil {
		return McpItem{}, err
	}
	return s.Get(resourceID)
}

func (s *McpService) Delete(resourceID string) error {
	return deleteTypedResource(s.db, resourceID, apmodel.ResourceTypeMCP)
}

func (s *McpService) getConfig(resourceID string) (string, error) {
	var def apmodel.McpDef
	if err := s.db.Where("resource_id = ?", resourceID).First(&def).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrResourceNotFound
		}
		return "", err
	}
	return def.ConfigJSON, nil
}

func setTypedResourceStatus(db *gorm.DB, resourceID string, resourceType string, status string) error {
	if db == nil || strings.TrimSpace(resourceID) == "" {
		return ErrInvalidResourceInput
	}
	result := db.Model(&apmodel.Resource{}).
		Where("resource_id = ? AND resource_type = ?", strings.TrimSpace(resourceID), resourceType).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResourceNotFound
	}
	return nil
}
