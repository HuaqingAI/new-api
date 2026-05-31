package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

const (
	OpenCapabilityFreshnessFresh   = "fresh"
	OpenCapabilityFreshnessStale   = "stale"
	OpenCapabilityFreshnessOffline = "offline"
	OpenCapabilityFreshnessRevoked = "revoked"
)

type DiscoveryQuery struct {
	ClientID     string
	ResourceType string
}

type DiscoveryItem struct {
	ResourceID       string
	ResourceType     string
	DisplayName      string
	ResourceVersion  string
	ContractVersion  string
	VisibilityState  string
	CallableState    string
	FreshnessTTL     int
	Freshness        string
	ETag             string
	ExtensionsJSON   string
	SupportedExtJSON string
}

type DiscoveryResult struct {
	Items []DiscoveryItem
	Total int
}

type CapabilityDetail struct {
	ResourceID          string
	ResourceType        string
	DisplayName         string
	ResourceVersion     string
	ContractVersion     string
	Status              string
	VisibilityState     string
	CallableState       string
	FreshnessTTLSeconds int
	Freshness           string
	ETag                string
	SchemaJSON          string
	DetailJSON          string
	ExtensionsJSON      string
	SupportedExtensions []string
}

type RefreshResult struct {
	ResourceID          string
	ResourceVersion     string
	ContractVersion     string
	FreshnessTTLSeconds int
	Freshness           string
	ETag                string
	VisibilityState     string
	CallableState       string
}

type DiscoveryService struct {
	db *gorm.DB
}

func NewDiscoveryService(db *gorm.DB) *DiscoveryService {
	return &DiscoveryService{db: db}
}

func (s *DiscoveryService) Discovery(query DiscoveryQuery) (DiscoveryResult, error) {
	if s == nil || s.db == nil {
		return DiscoveryResult{Items: []DiscoveryItem{}}, ErrOpenCapabilityContractInvalid
	}
	query.ClientID = strings.TrimSpace(query.ClientID)
	query.ResourceType = strings.TrimSpace(strings.ToLower(query.ResourceType))
	if query.ClientID == "" {
		return DiscoveryResult{Items: []DiscoveryItem{}}, ErrOpenCapabilityPermissionDenied
	}

	db := s.db.Model(&apmodel.Exposure{}).
		Where("client_key = ?", query.ClientID).
		Where("visibility_state = ?", apmodel.ExposureVisibilityVisible)
	if query.ResourceType != "" {
		db = db.Joins("JOIN agent_platform_resources r ON r.resource_id = agent_platform_resource_exposures.resource_id").
			Where("r.resource_type = ?", query.ResourceType)
	}

	var exposures []apmodel.Exposure
	if err := db.Order("id DESC").Find(&exposures).Error; err != nil {
		return DiscoveryResult{Items: []DiscoveryItem{}}, err
	}

	items := make([]DiscoveryItem, 0, len(exposures))
	for _, exposure := range exposures {
		resource, version, err := s.lookupPublishedProjection(exposure, query.ClientID)
		if err != nil {
			if errors.Is(err, ErrOpenCapabilityPermissionDenied) || errors.Is(err, ErrOpenCapabilityResourceRevoked) || errors.Is(err, ErrOpenCapabilityResourceOffline) {
				continue
			}
			return DiscoveryResult{Items: []DiscoveryItem{}}, err
		}
		items = append(items, DiscoveryItem{
			ResourceID:       resource.ResourceId,
			ResourceType:     resource.ResourceType,
			DisplayName:      resource.DisplayName,
			ResourceVersion:  version.Version,
			ContractVersion:  version.ContractVersion,
			VisibilityState:  exposure.VisibilityState,
			CallableState:    exposure.CallableState,
			FreshnessTTL:     exposure.FreshnessTTLSeconds,
			Freshness:        capabilityFreshness(exposure, resource.Status),
			ETag:             exposure.ETag,
			ExtensionsJSON:   exposure.ExtensionsJSON,
			SupportedExtJSON: exposure.ExtensionsJSON,
		})
	}
	return DiscoveryResult{Items: items, Total: len(items)}, nil
}

func (s *DiscoveryService) Detail(clientID string, resourceID string) (CapabilityDetail, error) {
	if s == nil || s.db == nil {
		return CapabilityDetail{}, ErrOpenCapabilityContractInvalid
	}
	clientID = strings.TrimSpace(clientID)
	resourceID = strings.TrimSpace(resourceID)
	if clientID == "" || resourceID == "" {
		return CapabilityDetail{}, ErrOpenCapabilityPermissionDenied
	}

	var exposure apmodel.Exposure
	if err := s.db.Where("resource_id = ? AND client_key = ?", resourceID, clientID).Order("id DESC").First(&exposure).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CapabilityDetail{}, ErrOpenCapabilityResourceNotFound
		}
		return CapabilityDetail{}, err
	}

	resource, version, err := s.lookupPublishedProjection(exposure, clientID)
	if err != nil {
		return CapabilityDetail{}, err
	}
	supportedExtensions := extractExtensionNamespaces(exposure.ExtensionsJSON)

	return CapabilityDetail{
		ResourceID:          resource.ResourceId,
		ResourceType:        resource.ResourceType,
		DisplayName:         resource.DisplayName,
		ResourceVersion:     version.Version,
		ContractVersion:     version.ContractVersion,
		Status:              resource.Status,
		VisibilityState:     exposure.VisibilityState,
		CallableState:       exposure.CallableState,
		FreshnessTTLSeconds: exposure.FreshnessTTLSeconds,
		Freshness:           capabilityFreshness(exposure, resource.Status),
		ETag:                exposure.ETag,
		SchemaJSON:          version.SchemaJSON,
		DetailJSON:          version.DetailJSON,
		ExtensionsJSON:      exposure.ExtensionsJSON,
		SupportedExtensions: supportedExtensions,
	}, nil
}

func (s *DiscoveryService) Refresh(clientID string, resourceID string) (RefreshResult, error) {
	if s == nil || s.db == nil {
		return RefreshResult{}, ErrOpenCapabilityContractInvalid
	}
	detail, err := s.Detail(clientID, resourceID)
	if err != nil {
		return RefreshResult{}, err
	}
	return RefreshResult{
		ResourceID:          detail.ResourceID,
		ResourceVersion:     detail.ResourceVersion,
		ContractVersion:     detail.ContractVersion,
		FreshnessTTLSeconds: detail.FreshnessTTLSeconds,
		Freshness:           detail.Freshness,
		ETag:                detail.ETag,
		VisibilityState:     detail.VisibilityState,
		CallableState:       detail.CallableState,
	}, nil
}

func (s *DiscoveryService) lookupPublishedProjection(exposure apmodel.Exposure, clientID string) (apmodel.Resource, apmodel.ResourceVersion, error) {
	if strings.TrimSpace(exposure.ClientKey) != strings.TrimSpace(clientID) {
		return apmodel.Resource{}, apmodel.ResourceVersion{}, ErrOpenCapabilityPermissionDenied
	}
	if exposure.VisibilityState == apmodel.ExposureVisibilityRevoked || exposure.CallableState == apmodel.ExposureCallableRevoked {
		return apmodel.Resource{}, apmodel.ResourceVersion{}, ErrOpenCapabilityResourceRevoked
	}
	if exposure.VisibilityState != apmodel.ExposureVisibilityVisible {
		return apmodel.Resource{}, apmodel.ResourceVersion{}, ErrOpenCapabilityPermissionDenied
	}

	var resource apmodel.Resource
	if err := s.db.Where("resource_id = ?", exposure.ResourceId).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apmodel.Resource{}, apmodel.ResourceVersion{}, ErrOpenCapabilityResourceNotFound
		}
		return apmodel.Resource{}, apmodel.ResourceVersion{}, err
	}

	switch resource.Status {
	case apmodel.ResourceStatusRevoked:
		return apmodel.Resource{}, apmodel.ResourceVersion{}, ErrOpenCapabilityResourceRevoked
	case apmodel.ResourceStatusOffline, apmodel.ResourceStatusDisabled:
		return apmodel.Resource{}, apmodel.ResourceVersion{}, ErrOpenCapabilityResourceOffline
	}

	var version apmodel.ResourceVersion
	if err := s.db.Where("resource_id = ? AND version = ?", exposure.ResourceId, exposure.ResourceVersion).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apmodel.Resource{}, apmodel.ResourceVersion{}, ErrOpenCapabilityResourceNotFound
		}
		return apmodel.Resource{}, apmodel.ResourceVersion{}, err
	}
	return resource, version, nil
}

func capabilityFreshness(exposure apmodel.Exposure, resourceStatus string) string {
	if exposure.VisibilityState == apmodel.ExposureVisibilityRevoked || exposure.CallableState == apmodel.ExposureCallableRevoked || exposure.RevokedAt != nil {
		return OpenCapabilityFreshnessRevoked
	}
	if resourceStatus == apmodel.ResourceStatusOffline || resourceStatus == apmodel.ResourceStatusDisabled {
		return OpenCapabilityFreshnessOffline
	}
	if exposure.PublishedAt == nil || exposure.FreshnessTTLSeconds <= 0 {
		return OpenCapabilityFreshnessFresh
	}
	if time.Now().UTC().After(exposure.PublishedAt.Add(time.Duration(exposure.FreshnessTTLSeconds) * time.Second)) {
		return OpenCapabilityFreshnessStale
	}
	return OpenCapabilityFreshnessFresh
}

func extractExtensionNamespaces(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var payload map[string]any
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return []string{}
	}
	namespaces := make([]string, 0, len(payload))
	for key := range payload {
		namespaces = append(namespaces, strings.TrimSpace(key))
	}
	return namespaces
}
