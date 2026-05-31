package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrExposureNotFound     = errors.New("agent platform exposure not found")
	ErrInvalidExposureInput = errors.New("agent platform exposure input invalid")
)

type CreateExposureInput struct {
	ResourceVersion     string
	ClientKey           string
	ClientScope         string
	VisibilityState     string
	CallableState       string
	FreshnessTTLSeconds int
	ETag                string
	Extensions          []byte
}

type UpdateExposureInput struct {
	VisibilityState     string
	CallableState       string
	FreshnessTTLSeconds *int
	ETag                string
	Extensions          []byte
}

type ExposureQuery struct {
	ResourceId      string
	ResourceVersion string
	ClientKey       string
	ClientScope     string
}

type ExposureItem struct {
	Id                  int
	ResourceId          string
	ResourceVersion     string
	ClientKey           string
	ClientScope         string
	VisibilityState     string
	CallableState       string
	FreshnessTTLSeconds int
	ETag                string
	ExtensionsJSON      string
	PublishedAt         *time.Time
	RevokedAt           *time.Time
	CreatedAt           int64
	UpdatedAt           int64
}

type ExposureListResult struct {
	Items []ExposureItem
	Total int
}

type ExposureService struct {
	db *gorm.DB
}

func NewExposureService(db *gorm.DB) *ExposureService {
	return &ExposureService{db: db}
}

func (s *ExposureService) Create(resourceID string, input CreateExposureInput) (ExposureItem, error) {
	if s == nil || s.db == nil {
		return ExposureItem{}, ErrInvalidExposureInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.ResourceVersion = strings.TrimSpace(input.ResourceVersion)
	input.ClientKey = strings.TrimSpace(input.ClientKey)
	input.ClientScope = strings.TrimSpace(input.ClientScope)
	if resourceID == "" || input.ResourceVersion == "" || input.ClientKey == "" {
		return ExposureItem{}, ErrInvalidExposureInput
	}

	var version apmodel.ResourceVersion
	if err := s.db.Where("resource_id = ? AND version = ?", resourceID, input.ResourceVersion).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ExposureItem{}, ErrResourceVersionNotFound
		}
		return ExposureItem{}, err
	}

	extensionsJSON, err := normalizeExposureJSON(input.Extensions)
	if err != nil {
		return ExposureItem{}, ErrInvalidExposureInput
	}

	now := time.Now().UTC()
	item := apmodel.Exposure{
		ResourceId:          resourceID,
		ResourceVersion:     input.ResourceVersion,
		ClientKey:           input.ClientKey,
		ClientScope:         input.ClientScope,
		VisibilityState:     input.VisibilityState,
		CallableState:       input.CallableState,
		FreshnessTTLSeconds: input.FreshnessTTLSeconds,
		ETag:                input.ETag,
		ExtensionsJSON:      extensionsJSON,
		PublishedAt:         &now,
	}
	if err := s.db.Create(&item).Error; err != nil {
		if errors.Is(err, apmodel.ErrInvalidExposureBody) || errors.Is(err, apmodel.ErrInvalidExposureState) {
			return ExposureItem{}, ErrInvalidExposureInput
		}
		return ExposureItem{}, err
	}
	return mapExposureItem(item), nil
}

func (s *ExposureService) Update(resourceID string, targetKey string, input UpdateExposureInput) (ExposureItem, error) {
	if s == nil || s.db == nil {
		return ExposureItem{}, ErrInvalidExposureInput
	}
	resourceID = strings.TrimSpace(resourceID)
	targetKey = strings.TrimSpace(targetKey)
	if resourceID == "" || targetKey == "" {
		return ExposureItem{}, ErrInvalidExposureInput
	}

	var exposure apmodel.Exposure
	if err := s.db.Where("resource_id = ? AND client_key = ?", resourceID, targetKey).First(&exposure).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ExposureItem{}, ErrExposureNotFound
		}
		return ExposureItem{}, err
	}

	if strings.TrimSpace(input.VisibilityState) != "" {
		exposure.VisibilityState = input.VisibilityState
	}
	if strings.TrimSpace(input.CallableState) != "" {
		exposure.CallableState = input.CallableState
	}
	if input.FreshnessTTLSeconds != nil {
		exposure.FreshnessTTLSeconds = *input.FreshnessTTLSeconds
	}
	if strings.TrimSpace(input.ETag) != "" {
		exposure.ETag = input.ETag
	}
	if len(input.Extensions) > 0 {
		extensionsJSON, err := normalizeExposureJSON(input.Extensions)
		if err != nil {
			return ExposureItem{}, ErrInvalidExposureInput
		}
		exposure.ExtensionsJSON = extensionsJSON
	}
	if exposure.VisibilityState == apmodel.ExposureVisibilityHidden {
		now := time.Now().UTC()
		exposure.RevokedAt = &now
	}
	if err := s.db.Save(&exposure).Error; err != nil {
		if errors.Is(err, apmodel.ErrInvalidExposureBody) || errors.Is(err, apmodel.ErrInvalidExposureState) {
			return ExposureItem{}, ErrInvalidExposureInput
		}
		return ExposureItem{}, err
	}
	return mapExposureItem(exposure), nil
}

func (s *ExposureService) Revoke(resourceID string, targetKey string) (ExposureItem, error) {
	return s.Update(resourceID, targetKey, UpdateExposureInput{
		VisibilityState: apmodel.ExposureVisibilityRevoked,
		CallableState:   apmodel.ExposureCallableRevoked,
	})
}

func (s *ExposureService) Get(resourceID string, targetKey string) (ExposureItem, error) {
	if s == nil || s.db == nil {
		return ExposureItem{}, ErrInvalidExposureInput
	}
	var exposure apmodel.Exposure
	if err := s.db.Where("resource_id = ? AND client_key = ?", strings.TrimSpace(resourceID), strings.TrimSpace(targetKey)).First(&exposure).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ExposureItem{}, ErrExposureNotFound
		}
		return ExposureItem{}, err
	}
	return mapExposureItem(exposure), nil
}

func (s *ExposureService) List(query ExposureQuery) (ExposureListResult, error) {
	if s == nil || s.db == nil {
		return ExposureListResult{Items: []ExposureItem{}}, ErrInvalidExposureInput
	}
	db := s.db.Model(&apmodel.Exposure{})
	if query.ResourceId != "" {
		db = db.Where("resource_id = ?", strings.TrimSpace(query.ResourceId))
	}
	if query.ResourceVersion != "" {
		db = db.Where("resource_version = ?", strings.TrimSpace(query.ResourceVersion))
	}
	if query.ClientScope != "" {
		db = db.Where("client_scope = ?", strings.TrimSpace(query.ClientScope))
	}
	if query.ClientKey != "" {
		db = db.Where("client_key = ?", strings.TrimSpace(query.ClientKey))
	}

	var exposures []apmodel.Exposure
	if err := db.Order("id DESC").Find(&exposures).Error; err != nil {
		return ExposureListResult{Items: []ExposureItem{}}, err
	}
	items := make([]ExposureItem, 0, len(exposures))
	for _, exposure := range exposures {
		items = append(items, mapExposureItem(exposure))
	}
	return ExposureListResult{Items: items, Total: len(items)}, nil
}

func normalizeExposureJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
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

func mapExposureItem(exposure apmodel.Exposure) ExposureItem {
	return ExposureItem{
		Id:                  exposure.Id,
		ResourceId:          exposure.ResourceId,
		ResourceVersion:     exposure.ResourceVersion,
		ClientKey:           exposure.ClientKey,
		ClientScope:         exposure.ClientScope,
		VisibilityState:     exposure.VisibilityState,
		CallableState:       exposure.CallableState,
		FreshnessTTLSeconds: exposure.FreshnessTTLSeconds,
		ETag:                exposure.ETag,
		ExtensionsJSON:      exposure.ExtensionsJSON,
		PublishedAt:         exposure.PublishedAt,
		RevokedAt:           exposure.RevokedAt,
		CreatedAt:           exposure.CreatedAt.Unix(),
		UpdatedAt:           exposure.UpdatedAt.Unix(),
	}
}
