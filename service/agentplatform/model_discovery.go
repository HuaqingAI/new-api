package agentplatform

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

const (
	OpenCapabilityModelStatusAvailable             = "available"
	OpenCapabilityModelStatusDisabled              = "disabled"
	OpenCapabilityModelStatusProviderOffline       = "provider_offline"
	OpenCapabilityModelStatusUnavailable           = "unavailable"
	OpenCapabilityModelStatusAccountTenantMismatch = "account_tenant_mismatch"

	OpenCapabilityModelDefaultResolved         = "resolved"
	OpenCapabilityModelDefaultNoDefault        = "no_default"
	OpenCapabilityModelDefaultMultipleDefaults = "multiple_defaults"
	OpenCapabilityModelDefaultDisabled         = "default_disabled"
	OpenCapabilityModelDefaultUnavailable      = "default_unavailable"
)

const modelDiscoveryConfigExtensionKey = "model_discovery.config"

type ModelDiscoveryItem struct {
	ModelID          string
	ProviderStableID string
	DisplayName      string
	IsDefault        bool
	Status           string
	DisabledReason   string
	CapabilitiesJSON string
	AccountID        string
	TenantID         string
}

type ModelDiscoveryResult struct {
	ContractVersion string
	DefaultState    string
	Items           []ModelDiscoveryItem
	Total           int
}

type ModelDiscoveryService struct {
	db *gorm.DB
}

type modelDiscoveryConfig struct {
	DefaultModel string                     `json:"default_model"`
	AccountID    string                     `json:"account_id"`
	TenantID     string                     `json:"tenant_id"`
	Models       []modelDiscoveryConfigItem `json:"models"`
}

type modelDiscoveryConfigItem struct {
	ModelID          string         `json:"model_id"`
	ProviderStableID string         `json:"provider_stable_id"`
	DisplayName      string         `json:"display_name"`
	IsDefault        *bool          `json:"is_default,omitempty"`
	Status           string         `json:"status,omitempty"`
	DisabledReason   string         `json:"disabled_reason,omitempty"`
	Capabilities     map[string]any `json:"capabilities,omitempty"`
	AccountID        string         `json:"account_id,omitempty"`
	TenantID         string         `json:"tenant_id,omitempty"`
}

func NewModelDiscoveryService(db *gorm.DB) *ModelDiscoveryService {
	return &ModelDiscoveryService{db: db}
}

func (s *ModelDiscoveryService) List(clientID string) (ModelDiscoveryResult, error) {
	if s == nil || s.db == nil {
		return ModelDiscoveryResult{}, ErrOpenCapabilityContractInvalid
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return ModelDiscoveryResult{}, ErrOpenCapabilityPermissionDenied
	}

	var client apmodel.Client
	if err := s.db.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ModelDiscoveryResult{}, ErrUnauthorizedClient
		}
		return ModelDiscoveryResult{}, err
	}

	config, err := parseModelDiscoveryConfig(client.ExtensionsJSON)
	if err != nil {
		return ModelDiscoveryResult{}, ErrOpenCapabilityContractInvalid
	}

	items := buildModelDiscoveryItems(config)
	defaultState := resolveModelDefaultState(items)
	return ModelDiscoveryResult{
		ContractVersion: strings.TrimSpace(client.ContractVersion),
		DefaultState:    defaultState,
		Items:           items,
		Total:           len(items),
	}, nil
}

func parseModelDiscoveryConfig(raw string) (modelDiscoveryConfig, error) {
	config := modelDiscoveryConfig{}
	if strings.TrimSpace(raw) == "" {
		return config, nil
	}
	var payload map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return config, err
	}
	entry, ok := payload[modelDiscoveryConfigExtensionKey]
	if !ok || len(entry) == 0 {
		return config, nil
	}
	if err := common.Unmarshal(entry, &config); err != nil {
		return config, err
	}
	config.DefaultModel = strings.TrimSpace(config.DefaultModel)
	config.AccountID = strings.TrimSpace(config.AccountID)
	config.TenantID = strings.TrimSpace(config.TenantID)
	return config, nil
}

func buildModelDiscoveryItems(config modelDiscoveryConfig) []ModelDiscoveryItem {
	items := make([]ModelDiscoveryItem, 0, len(config.Models)+1)
	seen := map[string]struct{}{}

	for _, entry := range config.Models {
		modelID := strings.TrimSpace(entry.ModelID)
		if modelID == "" {
			continue
		}
		seen[modelID] = struct{}{}
		displayName := strings.TrimSpace(entry.DisplayName)
		if displayName == "" {
			displayName = modelID
		}
		providerStableID := strings.TrimSpace(entry.ProviderStableID)
		if providerStableID == "" {
			providerStableID = "unknown"
		}
		status := normalizeModelStatus(entry.Status)
		disabledReason := strings.TrimSpace(entry.DisabledReason)
		isDefault := entry.IsDefault != nil && *entry.IsDefault
		if config.DefaultModel != "" && modelID == config.DefaultModel {
			isDefault = true
		}
		accountID := strings.TrimSpace(entry.AccountID)
		if accountID == "" {
			accountID = config.AccountID
		}
		tenantID := strings.TrimSpace(entry.TenantID)
		if tenantID == "" {
			tenantID = config.TenantID
		}
		if config.AccountID != "" && accountID != "" && accountID != config.AccountID {
			status = OpenCapabilityModelStatusAccountTenantMismatch
			disabledReason = "account_tenant_mismatch"
		}
		if config.TenantID != "" && tenantID != "" && tenantID != config.TenantID {
			status = OpenCapabilityModelStatusAccountTenantMismatch
			disabledReason = "account_tenant_mismatch"
		}
		if status == OpenCapabilityModelStatusDisabled && disabledReason == "" {
			disabledReason = "disabled"
		}
		if status == OpenCapabilityModelStatusProviderOffline && disabledReason == "" {
			disabledReason = "provider_offline"
		}
		if status == OpenCapabilityModelStatusUnavailable && disabledReason == "" {
			disabledReason = "model_unavailable"
		}
		capabilitiesJSON := "{}"
		if len(entry.Capabilities) > 0 {
			if body, err := common.Marshal(entry.Capabilities); err == nil {
				capabilitiesJSON = string(body)
			}
		}
		items = append(items, ModelDiscoveryItem{
			ModelID:          modelID,
			ProviderStableID: providerStableID,
			DisplayName:      displayName,
			IsDefault:        isDefault,
			Status:           status,
			DisabledReason:   disabledReason,
			CapabilitiesJSON: capabilitiesJSON,
			AccountID:        accountID,
			TenantID:         tenantID,
		})
	}

	if config.DefaultModel != "" {
		if _, ok := seen[config.DefaultModel]; !ok {
			items = append(items, ModelDiscoveryItem{
				ModelID:          config.DefaultModel,
				ProviderStableID: "unknown",
				DisplayName:      config.DefaultModel,
				IsDefault:        true,
				Status:           OpenCapabilityModelStatusUnavailable,
				DisabledReason:   "model_unavailable",
				CapabilitiesJSON: "{}",
				AccountID:        config.AccountID,
				TenantID:         config.TenantID,
			})
		}
	}

	return items
}

func normalizeModelStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "", OpenCapabilityModelStatusAvailable:
		return OpenCapabilityModelStatusAvailable
	case OpenCapabilityModelStatusDisabled:
		return OpenCapabilityModelStatusDisabled
	case OpenCapabilityModelStatusProviderOffline:
		return OpenCapabilityModelStatusProviderOffline
	case OpenCapabilityModelStatusUnavailable:
		return OpenCapabilityModelStatusUnavailable
	case OpenCapabilityModelStatusAccountTenantMismatch:
		return OpenCapabilityModelStatusAccountTenantMismatch
	default:
		return OpenCapabilityModelStatusUnavailable
	}
}

func resolveModelDefaultState(items []ModelDiscoveryItem) string {
	defaultCount := 0
	disabledDefault := false
	unavailableDefault := false
	for _, item := range items {
		if !item.IsDefault {
			continue
		}
		defaultCount++
		if item.Status == OpenCapabilityModelStatusDisabled {
			disabledDefault = true
		}
		if item.Status == OpenCapabilityModelStatusUnavailable {
			unavailableDefault = true
		}
	}
	switch {
	case defaultCount == 0:
		return OpenCapabilityModelDefaultNoDefault
	case defaultCount > 1:
		return OpenCapabilityModelDefaultMultipleDefaults
	case disabledDefault:
		return OpenCapabilityModelDefaultDisabled
	case unavailableDefault:
		return OpenCapabilityModelDefaultUnavailable
	default:
		return OpenCapabilityModelDefaultResolved
	}
}
