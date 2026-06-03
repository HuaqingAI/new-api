package agentplatform

import (
	"errors"
	"strings"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrResourceNotFound     = errors.New("agent platform resource not found")
	ErrInvalidResourceInput = errors.New("agent platform resource input invalid")
)

type ResourceService struct {
	db *gorm.DB
}

type CreateResourceInput struct {
	ResourceType string
	DisplayName  string
	OwnerUserId  int
	TenantId     int
}

type ListResourcesQuery struct {
	ResourceType string
	OwnerUserId  *int
	TenantId     *int
	Page         int
	PageSize     int
}

type ResourceItem struct {
	Id            int    `json:"id"`
	ResourceId    string `json:"resource_id"`
	ResourceType  string `json:"resource_type"`
	DisplayName   string `json:"display_name"`
	OwnerUserId   int    `json:"owner_user_id"`
	Status        string `json:"status"`
	LatestVersion string `json:"latest_version"`
	TenantId      int    `json:"tenant_id"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type ResourceListResult struct {
	Items    []ResourceItem `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

func NewResourceService(db *gorm.DB) *ResourceService {
	return &ResourceService{db: db}
}

func (s *ResourceService) Create(input CreateResourceInput) (ResourceItem, error) {
	if s == nil || s.db == nil {
		return ResourceItem{}, ErrInvalidResourceInput
	}
	input.ResourceType = strings.TrimSpace(strings.ToLower(input.ResourceType))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.OwnerUserId <= 0 || input.DisplayName == "" || input.ResourceType == "" {
		return ResourceItem{}, ErrInvalidResourceInput
	}

	resource := apmodel.Resource{
		ResourceType: input.ResourceType,
		DisplayName:  input.DisplayName,
		OwnerUserId:  input.OwnerUserId,
		TenantId:     input.TenantId,
		Status:       apmodel.ResourceStatusDraft,
	}
	if err := s.db.Create(&resource).Error; err != nil {
		if errors.Is(err, apmodel.ErrInvalidResourceType) || errors.Is(err, apmodel.ErrInvalidResourceStatus) {
			return ResourceItem{}, ErrInvalidResourceInput
		}
		return ResourceItem{}, err
	}
	return mapResourceItem(resource), nil
}

func (s *ResourceService) GetByResourceID(resourceID string) (ResourceItem, error) {
	if s == nil || s.db == nil {
		return ResourceItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return ResourceItem{}, ErrInvalidResourceInput
	}

	var resource apmodel.Resource
	if err := s.db.Where("resource_id = ?", resourceID).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ResourceItem{}, ErrResourceNotFound
		}
		return ResourceItem{}, err
	}
	return mapResourceItem(resource), nil
}

func (s *ResourceService) List(query ListResourcesQuery) (ResourceListResult, error) {
	if s == nil || s.db == nil {
		return ResourceListResult{Items: []ResourceItem{}}, ErrInvalidResourceInput
	}
	page, pageSize := normalizeResourcePage(query.Page, query.PageSize)
	query.ResourceType = strings.TrimSpace(strings.ToLower(query.ResourceType))
	if query.ResourceType != "" {
		switch query.ResourceType {
		case apmodel.ResourceTypeSkill, apmodel.ResourceTypeKnowledge, apmodel.ResourceTypeAgent:
		default:
			return ResourceListResult{Items: []ResourceItem{}}, ErrInvalidResourceInput
		}
	}

	db := s.db.Model(&apmodel.Resource{})
	if query.ResourceType != "" {
		db = db.Where("resource_type = ?", query.ResourceType)
	}
	if query.OwnerUserId != nil {
		db = db.Where("owner_user_id = ?", *query.OwnerUserId)
	}
	if query.TenantId != nil {
		db = db.Where("tenant_id = ?", *query.TenantId)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ResourceListResult{Items: []ResourceItem{}}, err
	}

	var resources []apmodel.Resource
	if err := db.Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&resources).Error; err != nil {
		return ResourceListResult{Items: []ResourceItem{}}, err
	}

	items := make([]ResourceItem, 0, len(resources))
	for _, resource := range resources {
		items = append(items, mapResourceItem(resource))
	}

	return ResourceListResult{
		Items:    items,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func normalizeResourcePage(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func mapResourceItem(resource apmodel.Resource) ResourceItem {
	return ResourceItem{
		Id:            resource.Id,
		ResourceId:    resource.ResourceId,
		ResourceType:  resource.ResourceType,
		DisplayName:   resource.DisplayName,
		OwnerUserId:   resource.OwnerUserId,
		Status:        resource.Status,
		LatestVersion: resource.LatestVersion,
		TenantId:      resource.TenantId,
		CreatedAt:     resource.CreatedAt.Unix(),
		UpdatedAt:     resource.UpdatedAt.Unix(),
	}
}
