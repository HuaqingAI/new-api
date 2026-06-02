package agentplatform

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrClientNotFound      = errors.New("agent platform client not found")
	ErrInvalidClientInput  = errors.New("agent platform client input invalid")
	ErrInvalidIntegration  = errors.New("agent platform client invalid integration")
)

type CreateClientInput struct {
	Slug                   string
	DisplayName            string
	ClientType             string
	Status                 string
	AllowedGrantTypes      []byte
	RedirectURIs           []byte
	AllowedScopes          []byte
	ContractVersion        string
	Capabilities           []byte
	Extensions             []byte
	AllowClientCredentials bool
}

type UpdateClientInput struct {
	DisplayName            string
	Status                 string
	AllowedGrantTypes      []byte
	RedirectURIs           []byte
	AllowedScopes          []byte
	ContractVersion        string
	Capabilities           []byte
	Extensions             []byte
	AllowClientCredentials *bool
}

type ClientQuery struct {
	Status          string
	ContractVersion string
}

type ClientItem struct {
	Id                     int
	ClientId               string
	Slug                   string
	DisplayName            string
	ClientType             string
	Status                 string
	AllowedGrantTypesJSON  string
	RedirectURIsJSON       string
	AllowedScopesJSON      string
	ContractVersion        string
	CapabilitiesJSON       string
	ExtensionsJSON         string
	AllowClientCredentials bool
	CreatedAt              int64
	UpdatedAt              int64
}

type ClientListResult struct {
	Items []ClientItem
	Total int
}

type ClientService struct {
	db *gorm.DB
}

func NewClientService(db *gorm.DB) *ClientService {
	return &ClientService{db: db}
}

func (s *ClientService) Create(input CreateClientInput) (ClientItem, error) {
	if s == nil || s.db == nil {
		return ClientItem{}, ErrInvalidClientInput
	}
	input.Slug = strings.TrimSpace(strings.ToLower(input.Slug))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.ClientType = strings.TrimSpace(strings.ToLower(input.ClientType))
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	if input.Slug == "" || input.DisplayName == "" || input.ClientType == "" {
		return ClientItem{}, ErrInvalidClientInput
	}

	grantJSON, err := normalizeClientJSON(input.AllowedGrantTypes)
	if err != nil {
		return ClientItem{}, ErrInvalidClientInput
	}
	redirectJSON, err := normalizeClientJSON(input.RedirectURIs)
	if err != nil {
		return ClientItem{}, ErrInvalidClientInput
	}
	scopeJSON, err := normalizeClientJSON(input.AllowedScopes)
	if err != nil {
		return ClientItem{}, ErrInvalidClientInput
	}
	capabilitiesJSON, err := normalizeClientJSON(input.Capabilities)
	if err != nil {
		return ClientItem{}, ErrInvalidClientInput
	}
	extensionsJSON, err := normalizeClientJSON(input.Extensions)
	if err != nil {
		return ClientItem{}, ErrInvalidClientInput
	}

	if input.ContractVersion == "" || capabilitiesJSON == "" {
		if input.Status == "" {
			input.Status = "invalid_integration"
		}
	} else if input.Status == "" {
		input.Status = "active"
	}

	if err := validateNamespacedExtensions(extensionsJSON); err != nil {
		return ClientItem{}, ErrInvalidClientInput
	}

	item := apmodel.Client{
		Slug:                   input.Slug,
		DisplayName:            input.DisplayName,
		ClientType:             input.ClientType,
		Status:                 input.Status,
		AllowedGrantTypesJSON:  grantJSON,
		RedirectURIsJSON:       redirectJSON,
		AllowedScopesJSON:      scopeJSON,
		ContractVersion:        input.ContractVersion,
		CapabilitiesJSON:       capabilitiesJSON,
		ExtensionsJSON:         extensionsJSON,
		AllowClientCredentials: input.AllowClientCredentials,
	}
	if err := s.db.Create(&item).Error; err != nil {
		if errors.Is(err, apmodel.ErrInvalidClientBody) {
			return ClientItem{}, ErrInvalidClientInput
		}
		return ClientItem{}, err
	}
	return mapClientItem(item), nil
}

func (s *ClientService) Get(clientID string) (ClientItem, error) {
	if s == nil || s.db == nil {
		return ClientItem{}, ErrInvalidClientInput
	}
	var item apmodel.Client
	if err := s.db.Where("client_id = ?", strings.TrimSpace(clientID)).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ClientItem{}, ErrClientNotFound
		}
		return ClientItem{}, err
	}
	return mapClientItem(item), nil
}

func (s *ClientService) List(query ClientQuery) (ClientListResult, error) {
	if s == nil || s.db == nil {
		return ClientListResult{Items: []ClientItem{}}, ErrInvalidClientInput
	}
	db := s.db.Model(&apmodel.Client{})
	if query.Status != "" {
		db = db.Where("status = ?", strings.TrimSpace(strings.ToLower(query.Status)))
	}
	if query.ContractVersion != "" {
		db = db.Where("contract_version = ?", strings.TrimSpace(query.ContractVersion))
	}
	var items []apmodel.Client
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		return ClientListResult{Items: []ClientItem{}}, err
	}
	result := make([]ClientItem, 0, len(items))
	for _, item := range items {
		result = append(result, mapClientItem(item))
	}
	return ClientListResult{Items: result, Total: len(result)}, nil
}

func (s *ClientService) Update(clientID string, input UpdateClientInput) (ClientItem, error) {
	if s == nil || s.db == nil {
		return ClientItem{}, ErrInvalidClientInput
	}
	var item apmodel.Client
	if err := s.db.Where("client_id = ?", strings.TrimSpace(clientID)).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ClientItem{}, ErrClientNotFound
		}
		return ClientItem{}, err
	}

	if strings.TrimSpace(input.DisplayName) != "" {
		item.DisplayName = strings.TrimSpace(input.DisplayName)
	}
	if strings.TrimSpace(input.Status) != "" {
		item.Status = strings.TrimSpace(strings.ToLower(input.Status))
	}
	if strings.TrimSpace(input.ContractVersion) != "" {
		item.ContractVersion = strings.TrimSpace(input.ContractVersion)
	}
	if len(input.AllowedGrantTypes) > 0 {
		normalized, err := normalizeClientJSON(input.AllowedGrantTypes)
		if err != nil {
			return ClientItem{}, ErrInvalidClientInput
		}
		item.AllowedGrantTypesJSON = normalized
	}
	if len(input.RedirectURIs) > 0 {
		normalized, err := normalizeClientJSON(input.RedirectURIs)
		if err != nil {
			return ClientItem{}, ErrInvalidClientInput
		}
		item.RedirectURIsJSON = normalized
	}
	if len(input.AllowedScopes) > 0 {
		normalized, err := normalizeClientJSON(input.AllowedScopes)
		if err != nil {
			return ClientItem{}, ErrInvalidClientInput
		}
		item.AllowedScopesJSON = normalized
	}
	if len(input.Capabilities) > 0 {
		normalized, err := normalizeClientJSON(input.Capabilities)
		if err != nil {
			return ClientItem{}, ErrInvalidClientInput
		}
		item.CapabilitiesJSON = normalized
	}
	if len(input.Extensions) > 0 {
		normalized, err := normalizeClientJSON(input.Extensions)
		if err != nil {
			return ClientItem{}, ErrInvalidClientInput
		}
		if err := validateNamespacedExtensions(normalized); err != nil {
			return ClientItem{}, ErrInvalidClientInput
		}
		item.ExtensionsJSON = normalized
	}
	if input.AllowClientCredentials != nil {
		item.AllowClientCredentials = *input.AllowClientCredentials
	}

	if item.ContractVersion == "" || item.CapabilitiesJSON == "" {
		item.Status = "invalid_integration"
	}

	if err := s.db.Save(&item).Error; err != nil {
		if errors.Is(err, apmodel.ErrInvalidClientBody) {
			return ClientItem{}, ErrInvalidClientInput
		}
		return ClientItem{}, err
	}
	return mapClientItem(item), nil
}

func normalizeClientJSON(raw []byte) (string, error) {
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

func validateNamespacedExtensions(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var payload map[string]any
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return err
	}
	for key := range payload {
		if !strings.Contains(key, ".") {
			return ErrInvalidClientInput
		}
	}
	return nil
}

func mapClientItem(item apmodel.Client) ClientItem {
	return ClientItem{
		Id:                     item.Id,
		ClientId:               item.ClientId,
		Slug:                   item.Slug,
		DisplayName:            item.DisplayName,
		ClientType:             item.ClientType,
		Status:                 item.Status,
		AllowedGrantTypesJSON:  item.AllowedGrantTypesJSON,
		RedirectURIsJSON:       item.RedirectURIsJSON,
		AllowedScopesJSON:      item.AllowedScopesJSON,
		ContractVersion:        item.ContractVersion,
		CapabilitiesJSON:       item.CapabilitiesJSON,
		ExtensionsJSON:         item.ExtensionsJSON,
		AllowClientCredentials: item.AllowClientCredentials,
		CreatedAt:              item.CreatedAt.Unix(),
		UpdatedAt:              item.UpdatedAt.Unix(),
	}
}
