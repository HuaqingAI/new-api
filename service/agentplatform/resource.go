package agentplatform

import (
	"errors"
	"fmt"
	"strings"

	appmodel "github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrResourceNotFound     = errors.New("agent platform resource not found")
	ErrInvalidResourceInput = errors.New("agent platform resource input invalid")
	ErrResourceInUse        = errors.New("agent platform resource is referenced by agent")
)

type ResourceService struct {
	db *gorm.DB
}

type CreateResourceInput struct {
	ResourceType string
	DisplayName  string
	Description  string
	Avatar       string
	OwnerUserId  int
	TenantId     int
	Status       string
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
	Description   string `json:"description"`
	Avatar        string `json:"avatar"`
	OwnerUserId   int    `json:"owner_user_id"`
	OwnerName     string `json:"owner_name"`
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
	input.Description = strings.TrimSpace(input.Description)
	input.Avatar = strings.TrimSpace(input.Avatar)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	if input.OwnerUserId <= 0 || input.DisplayName == "" || input.ResourceType == "" {
		return ResourceItem{}, ErrInvalidResourceInput
	}
	if input.Status == "" {
		input.Status = apmodel.ResourceStatusDraft
	}

	resource := apmodel.Resource{
		ResourceType: input.ResourceType,
		DisplayName:  input.DisplayName,
		Description:  input.Description,
		Avatar:       input.Avatar,
		OwnerUserId:  input.OwnerUserId,
		TenantId:     input.TenantId,
		Status:       input.Status,
	}
	if err := s.db.Create(&resource).Error; err != nil {
		if errors.Is(err, apmodel.ErrInvalidResourceType) || errors.Is(err, apmodel.ErrInvalidResourceStatus) {
			return ResourceItem{}, ErrInvalidResourceInput
		}
		return ResourceItem{}, err
	}
	item := mapResourceItem(resource)
	if err := s.fillOwnerNames([]*ResourceItem{&item}); err != nil {
		return ResourceItem{}, err
	}
	return item, nil
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
	item := mapResourceItem(resource)
	if err := s.fillOwnerNames([]*ResourceItem{&item}); err != nil {
		return ResourceItem{}, err
	}
	return item, nil
}

func (s *ResourceService) List(query ListResourcesQuery) (ResourceListResult, error) {
	if s == nil || s.db == nil {
		return ResourceListResult{Items: []ResourceItem{}}, ErrInvalidResourceInput
	}
	page, pageSize := normalizeResourcePage(query.Page, query.PageSize)
	query.ResourceType = strings.TrimSpace(strings.ToLower(query.ResourceType))
	if query.ResourceType != "" {
		switch query.ResourceType {
		case apmodel.ResourceTypeMCP, apmodel.ResourceTypeSkill, apmodel.ResourceTypeKnowledge, apmodel.ResourceTypeAgent:
		default:
			return ResourceListResult{Items: []ResourceItem{}}, ErrInvalidResourceInput
		}
	}

	db := s.db.Model(&apmodel.Resource{})
	db = db.Where("status <> ?", apmodel.ResourceStatusRevoked)
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
	itemPointers := make([]*ResourceItem, 0, len(items))
	for i := range items {
		itemPointers = append(itemPointers, &items[i])
	}
	if err := s.fillOwnerNames(itemPointers); err != nil {
		return ResourceListResult{Items: []ResourceItem{}}, err
	}

	return ResourceListResult{
		Items:    items,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func deleteTypedResource(db *gorm.DB, resourceID string, resourceType string) error {
	if db == nil {
		return ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	resourceType = strings.TrimSpace(strings.ToLower(resourceType))
	if resourceID == "" || resourceType == "" {
		return ErrInvalidResourceInput
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var resource apmodel.Resource
		if err := tx.Where("resource_id = ? AND resource_type = ?", resourceID, resourceType).First(&resource).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResourceNotFound
			}
			return err
		}
		if resourceType != apmodel.ResourceTypeAgent {
			var count int64
			dependencyTable := apmodel.AgentDependency{}.TableName()
			resourceTable := apmodel.Resource{}.TableName()
			if err := tx.Model(&apmodel.AgentDependency{}).
				Joins("JOIN "+resourceTable+" ON "+resourceTable+".resource_id = "+dependencyTable+".agent_resource_id").
				Where(dependencyTable+".target_resource_id = ? AND "+dependencyTable+".target_type = ?", resourceID, resourceType).
				Where(dependencyTable+".resource_version = "+resourceTable+".latest_version").
				Where(resourceTable+".resource_type = ? AND "+resourceTable+".latest_version <> ? AND "+resourceTable+".status <> ?", apmodel.ResourceTypeAgent, "", apmodel.ResourceStatusRevoked).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return ErrResourceInUse
			}
			if err := tx.Where("target_resource_id = ? AND target_type = ?", resourceID, resourceType).Delete(&apmodel.AgentDependency{}).Error; err != nil {
				return err
			}
		}
		switch resourceType {
		case apmodel.ResourceTypeMCP:
			if err := tx.Where("resource_id = ?", resourceID).Delete(&apmodel.McpDef{}).Error; err != nil {
				return err
			}
		case apmodel.ResourceTypeSkill:
			if err := tx.Where("resource_id = ?", resourceID).Delete(&apmodel.SkillDef{}).Error; err != nil {
				return err
			}
		case apmodel.ResourceTypeKnowledge:
			if err := tx.Where("resource_id = ?", resourceID).Delete(&apmodel.KnowledgeDef{}).Error; err != nil {
				return err
			}
		case apmodel.ResourceTypeAgent:
			if err := tx.Where("resource_id = ?", resourceID).Delete(&apmodel.AgentDef{}).Error; err != nil {
				return err
			}
			if err := tx.Where("agent_resource_id = ?", resourceID).Delete(&apmodel.AgentDependency{}).Error; err != nil {
				return err
			}
			if err := tx.Where("resource_id = ?", resourceID).Delete(&apmodel.ResourceVersion{}).Error; err != nil {
				return err
			}
			if err := tx.Where("resource_id = ?", resourceID).Delete(&apmodel.ResourceGrant{}).Error; err != nil {
				return err
			}
		default:
			return ErrInvalidResourceInput
		}
		result := tx.Where("resource_id = ? AND resource_type = ?", resourceID, resourceType).Delete(&apmodel.Resource{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrResourceNotFound
		}
		return nil
	})
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

func (s *ResourceService) fillOwnerNames(items []*ResourceItem) error {
	ownerIds := make([]int, 0, len(items))
	seen := map[int]struct{}{}
	for _, item := range items {
		if item == nil || item.OwnerUserId <= 0 {
			continue
		}
		if _, ok := seen[item.OwnerUserId]; ok {
			continue
		}
		seen[item.OwnerUserId] = struct{}{}
		ownerIds = append(ownerIds, item.OwnerUserId)
	}
	if len(ownerIds) == 0 {
		return nil
	}
	var users []appmodel.User
	if err := s.db.Select("id", "username", "display_name", "email").Where("id IN ?", ownerIds).Find(&users).Error; err != nil {
		if !isOptionalNameLookupError(err) {
			return err
		}
	}
	names := map[int]string{}
	for _, user := range users {
		names[user.Id] = userDisplayName(user)
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if name := names[item.OwnerUserId]; name != "" {
			item.OwnerName = name
			continue
		}
		item.OwnerName = fmt.Sprintf("#%d", item.OwnerUserId)
	}
	return nil
}

func isOptionalNameLookupError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table") ||
		strings.Contains(message, "doesn't exist") ||
		strings.Contains(message, "does not exist") ||
		strings.Contains(message, "undefined table") ||
		strings.Contains(message, "42p01")
}

func userDisplayName(user appmodel.User) string {
	if name := strings.TrimSpace(user.DisplayName); name != "" {
		return name
	}
	if name := strings.TrimSpace(user.Username); name != "" {
		return name
	}
	if email := strings.TrimSpace(user.Email); email != "" {
		return email
	}
	return fmt.Sprintf("#%d", user.Id)
}

func mapResourceItem(resource apmodel.Resource) ResourceItem {
	return ResourceItem{
		Id:            resource.Id,
		ResourceId:    resource.ResourceId,
		ResourceType:  resource.ResourceType,
		DisplayName:   resource.DisplayName,
		Description:   resource.Description,
		Avatar:        resource.Avatar,
		OwnerUserId:   resource.OwnerUserId,
		Status:        resource.Status,
		LatestVersion: resource.LatestVersion,
		TenantId:      resource.TenantId,
		CreatedAt:     resource.CreatedAt.Unix(),
		UpdatedAt:     resource.UpdatedAt.Unix(),
	}
}
