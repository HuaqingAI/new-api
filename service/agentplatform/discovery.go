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
	ContractCompatible  bool
	Diagnostics         CapabilityDiagnostics
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
	ContractCompatible  bool
	Diagnostics         CapabilityDiagnostics
}

type RefreshInput struct {
	ClientID                string
	ResourceID              string
	ObservedETag            string
	ObservedResourceVersion string
	ObservedAtUnix          *int64
}

type CapabilityDiagnostics struct {
	Reason             string
	Converged          bool
	ClientNonCompliant bool
	ObservedETag       string
	ObservedVersion    string
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
		compatible, err := s.contractCompatible(query.ClientID, version.ContractVersion)
		if err != nil {
			return DiscoveryResult{Items: []DiscoveryItem{}}, err
		}
		if !compatible {
			continue
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
	compatible, err := s.contractCompatible(clientID, version.ContractVersion)
	if err != nil {
		return CapabilityDetail{}, err
	}
	if !compatible {
		return CapabilityDetail{}, ErrOpenCapabilityContractInvalid
	}
	supportedExtensions := extractExtensionNamespaces(exposure.ExtensionsJSON)
	diagnostics := computeCapabilityDiagnostics(exposure, resource.Status, "", "", nil)
	detailJSON := ""
	callableState := exposure.CallableState
	if resource.ResourceType == apmodel.ResourceTypeAgent {
		callableState, diagnostics, err = s.resolveAgentCallableState(clientID, resource.ResourceId, version.Version, callableState, diagnostics)
		if err != nil {
			return CapabilityDetail{}, err
		}
		detailJSON, err = s.buildAgentDetailPayload(resource.ResourceId, version.Version)
		if err != nil {
			return CapabilityDetail{}, err
		}
	}

	return CapabilityDetail{
		ResourceID:          resource.ResourceId,
		ResourceType:        resource.ResourceType,
		DisplayName:         resource.DisplayName,
		ResourceVersion:     version.Version,
		ContractVersion:     version.ContractVersion,
		Status:              resource.Status,
		VisibilityState:     exposure.VisibilityState,
		CallableState:       callableState,
		FreshnessTTLSeconds: exposure.FreshnessTTLSeconds,
		Freshness:           capabilityFreshness(exposure, resource.Status),
		ETag:                exposure.ETag,
		SchemaJSON:          "",
		DetailJSON:          detailJSON,
		ExtensionsJSON:      exposure.ExtensionsJSON,
		SupportedExtensions: supportedExtensions,
		ContractCompatible:  compatible,
		Diagnostics:         diagnostics,
	}, nil
}

func (s *DiscoveryService) Refresh(input RefreshInput) (RefreshResult, error) {
	if s == nil || s.db == nil {
		return RefreshResult{}, ErrOpenCapabilityContractInvalid
	}
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.ObservedETag = strings.TrimSpace(input.ObservedETag)
	input.ObservedResourceVersion = strings.TrimSpace(input.ObservedResourceVersion)
	detail, err := s.Detail(input.ClientID, input.ResourceID)
	if err != nil {
		return RefreshResult{}, err
	}
	diagnostics := computeCapabilityDiagnosticsFromDetail(detail, input)
	return RefreshResult{
		ResourceID:          detail.ResourceID,
		ResourceVersion:     detail.ResourceVersion,
		ContractVersion:     detail.ContractVersion,
		FreshnessTTLSeconds: detail.FreshnessTTLSeconds,
		Freshness:           detail.Freshness,
		ETag:                detail.ETag,
		VisibilityState:     detail.VisibilityState,
		CallableState:       detail.CallableState,
		ContractCompatible:  detail.ContractCompatible,
		Diagnostics:         diagnostics,
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

func (s *DiscoveryService) contractCompatible(clientID string, resourceContractVersion string) (bool, error) {
	clientID = strings.TrimSpace(clientID)
	resourceContractVersion = strings.TrimSpace(resourceContractVersion)
	if clientID == "" || resourceContractVersion == "" {
		return false, ErrOpenCapabilityContractInvalid
	}
	var client apmodel.Client
	if err := s.db.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrUnauthorizedClient
		}
		return false, err
	}
	return strings.TrimSpace(client.ContractVersion) == resourceContractVersion, nil
}

func computeCapabilityDiagnostics(exposure apmodel.Exposure, resourceStatus string, observedETag string, observedVersion string, observedAtUnix *int64) CapabilityDiagnostics {
	freshness := capabilityFreshness(exposure, resourceStatus)
	diagnostics := CapabilityDiagnostics{
		Reason:          "in_sync",
		Converged:       true,
		ObservedETag:    observedETag,
		ObservedVersion: observedVersion,
	}
	switch freshness {
	case OpenCapabilityFreshnessRevoked:
		diagnostics.Reason = "resource_revoked"
		diagnostics.Converged = false
	case OpenCapabilityFreshnessOffline:
		diagnostics.Reason = "resource_offline"
		diagnostics.Converged = false
	case OpenCapabilityFreshnessStale:
		diagnostics.Reason = "projection_stale"
		diagnostics.Converged = false
	}
	if observedAtUnix != nil && exposure.PublishedAt != nil && freshness == OpenCapabilityFreshnessStale {
		observedAt := time.Unix(*observedAtUnix, 0).UTC()
		if observedAt.Before(exposure.PublishedAt.Add(-time.Duration(exposure.FreshnessTTLSeconds) * time.Second)) {
			diagnostics.Reason = "client_non_compliant_stale"
			diagnostics.ClientNonCompliant = true
		}
	}
	return diagnostics
}

func computeCapabilityDiagnosticsFromDetail(detail CapabilityDetail, input RefreshInput) CapabilityDiagnostics {
	diagnostics := detail.Diagnostics
	diagnostics.ObservedETag = input.ObservedETag
	diagnostics.ObservedVersion = input.ObservedResourceVersion
	if detail.Freshness == OpenCapabilityFreshnessFresh {
		if input.ObservedETag != "" && input.ObservedETag != detail.ETag {
			diagnostics.Reason = "client_cache_mismatch"
			diagnostics.Converged = false
		}
		if input.ObservedResourceVersion != "" && input.ObservedResourceVersion != detail.ResourceVersion {
			diagnostics.Reason = "client_version_mismatch"
			diagnostics.Converged = false
		}
	}
	if detail.Freshness == OpenCapabilityFreshnessStale && input.ObservedAtUnix != nil {
		diagnostics.ClientNonCompliant = true
		diagnostics.Reason = "client_non_compliant_stale"
		diagnostics.Converged = false
	}
	return diagnostics
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

func (s *DiscoveryService) resolveAgentCallableState(clientID string, resourceID string, version string, current string, diagnostics CapabilityDiagnostics) (string, CapabilityDiagnostics, error) {
	var dependencies []apmodel.AgentDependency
	if err := s.db.Where("agent_resource_id = ? AND resource_version = ?", resourceID, version).Order("sort_order ASC, id ASC").Find(&dependencies).Error; err != nil {
		return current, diagnostics, err
	}
	for _, dependency := range dependencies {
		resourceType := strings.TrimSpace(strings.ToLower(dependency.TargetType))
		dependencyID := strings.TrimSpace(dependency.TargetResourceId)
		if resourceType == "" || dependencyID == "" {
			diagnostics.Reason = "contract_invalid_dependency"
			diagnostics.Converged = false
			return "contract_invalid", diagnostics, nil
		}
		var depExposure apmodel.Exposure
		if err := s.db.Where("resource_id = ? AND client_key = ?", dependencyID, clientID).Order("id DESC").First(&depExposure).Error; err != nil {
			diagnostics.Reason = "dependency_not_published"
			diagnostics.Converged = false
			return "contract_invalid", diagnostics, nil
		}
		if depExposure.VisibilityState != apmodel.ExposureVisibilityVisible || depExposure.CallableState != apmodel.ExposureCallableEnabled {
			diagnostics.Reason = "dependency_not_callable"
			diagnostics.Converged = false
			return "contract_invalid", diagnostics, nil
		}
	}
	return current, diagnostics, nil
}

func (s *DiscoveryService) buildAgentDetailPayload(resourceID string, version string) (string, error) {
	var agentDef apmodel.AgentDef
	if err := s.db.Where("resource_id = ? AND resource_version = ?", resourceID, version).First(&agentDef).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	var deps []apmodel.AgentDependency
	if err := s.db.Where("agent_resource_id = ? AND resource_version = ?", resourceID, version).Order("sort_order ASC, id ASC").Find(&deps).Error; err != nil {
		return "", err
	}
	dependencies := make([]map[string]any, 0, len(deps))
	for _, dep := range deps {
		dependencies = append(dependencies, map[string]any{
			"resource_type": dep.TargetType,
			"resource_id":   dep.TargetResourceId,
			"snapshot":      jsonTextToMap(dep.SnapshotJSON),
		})
	}
	payload := map[string]any{
		"cli_type":         agentDef.CliType,
		"name":             agentDef.Name,
		"description":      agentDef.Description,
		"avatar":           agentDef.Avatar,
		"instructions":     agentDef.Instructions,
		"model_config":     jsonTextToMap(agentDef.ModelConfigJSON),
		"package_path":     agentDef.PackagePath,
		"package_sha256":   agentDef.PackageSha256,
		"package_size":     agentDef.PackageSize,
		"dependencies":     dependencies,
		"resource_version": agentDef.ResourceVersion,
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func jsonTextToMap(raw string) map[string]any {
	result := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return result
	}
	_ = common.UnmarshalJsonStr(raw, &result)
	return result
}
