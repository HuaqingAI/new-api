package agentplatform

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrResourceVersionNotFound     = errors.New("agent platform resource version not found")
	ErrInvalidResourceVersionInput = errors.New("agent platform resource version input invalid")
	ErrAgentDependencyInvalid      = errors.New("agent platform agent dependency invalid")
)

type CreateResourceVersionInput struct {
	Version         string
	ContractVersion string
	Summary         string
	CreatedBy       int
}

type ResourceVersionItem struct {
	ResourceId      string
	ResourceType    string
	Version         string
	ContractVersion string
	Summary         string
	Status          string
	CreatedBy       int
	PublishedAt     *time.Time
	CreatedAt       int64
	UpdatedAt       int64
}

type ResourceVersionService struct {
	db *gorm.DB
}

func NewResourceVersionService(db *gorm.DB) *ResourceVersionService {
	return &ResourceVersionService{db: db}
}

func (s *ResourceVersionService) Create(resourceID string, input CreateResourceVersionInput) (ResourceVersionItem, error) {
	if s == nil || s.db == nil {
		return ResourceVersionItem{}, ErrInvalidResourceVersionInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.Version = strings.TrimSpace(input.Version)
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Summary = strings.TrimSpace(input.Summary)
	if resourceID == "" || input.Version == "" || input.ContractVersion == "" || input.CreatedBy <= 0 {
		return ResourceVersionItem{}, ErrInvalidResourceVersionInput
	}

	var created ResourceVersionItem
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var resource apmodel.Resource
		if err := tx.Where("resource_id = ?", resourceID).First(&resource).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResourceNotFound
			}
			return err
		}

		version := apmodel.ResourceVersion{
			ResourceId:      resource.ResourceId,
			Version:         input.Version,
			ContractVersion: input.ContractVersion,
			Summary:         input.Summary,
			Status:          apmodel.ResourceStatusDraft,
			CreatedBy:       input.CreatedBy,
		}
		if err := tx.Create(&version).Error; err != nil {
			if errors.Is(err, apmodel.ErrInvalidResourceVersionBody) || errors.Is(err, apmodel.ErrInvalidResourceStatus) {
				return ErrInvalidResourceVersionInput
			}
			return err
		}
		if err := tx.Model(&apmodel.Resource{}).
			Where("resource_id = ?", resource.ResourceId).
			Updates(map[string]any{"latest_version": input.Version}).Error; err != nil {
			return err
		}

		var err error
		created, err = s.getWithinTx(tx, resource.ResourceId, input.Version)
		return err
	})
	return created, err
}

func (s *ResourceVersionService) Get(resourceID string, version string) (ResourceVersionItem, error) {
	if s == nil || s.db == nil {
		return ResourceVersionItem{}, ErrInvalidResourceVersionInput
	}
	resourceID = strings.TrimSpace(resourceID)
	version = strings.TrimSpace(version)
	if resourceID == "" || version == "" {
		return ResourceVersionItem{}, ErrInvalidResourceVersionInput
	}
	return s.getWithinTx(s.db, resourceID, version)
}

func (s *ResourceVersionService) getWithinTx(db *gorm.DB, resourceID string, version string) (ResourceVersionItem, error) {
	var resource apmodel.Resource
	if err := db.Where("resource_id = ?", resourceID).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ResourceVersionItem{}, ErrResourceNotFound
		}
		return ResourceVersionItem{}, err
	}

	var resourceVersion apmodel.ResourceVersion
	if err := db.Where("resource_id = ? AND version = ?", resourceID, version).First(&resourceVersion).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ResourceVersionItem{}, ErrResourceVersionNotFound
		}
		return ResourceVersionItem{}, err
	}

	return ResourceVersionItem{
		ResourceId:      resource.ResourceId,
		ResourceType:    resource.ResourceType,
		Version:         resourceVersion.Version,
		ContractVersion: resourceVersion.ContractVersion,
		Summary:         resourceVersion.Summary,
		Status:          resourceVersion.Status,
		CreatedBy:       resourceVersion.CreatedBy,
		PublishedAt:     resourceVersion.PublishedAt,
		CreatedAt:       resourceVersion.CreatedAt.Unix(),
		UpdatedAt:       resourceVersion.UpdatedAt.Unix(),
	}, nil
}

func normalizeJSONText(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "", nil
	}
	var payload any
	if err := common.Unmarshal(raw, &payload); err != nil {
		return "", err
	}
	normalized, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}
