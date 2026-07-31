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
	ErrSkillContractInvalid        = errors.New("agent platform skill contract invalid")
	ErrKnowledgeContractInvalid    = errors.New("agent platform knowledge contract invalid")
	ErrAgentDependencyInvalid      = errors.New("agent platform agent dependency invalid")
)

type SkillDetailInput struct {
	InvokeSchema   json.RawMessage
	OutputSchema   json.RawMessage
	InvokeMode     string
	TimeoutSeconds *int
	BindingConfig  json.RawMessage
}

type KnowledgeDetailInput struct {
	KnowledgeMode        string
	ProviderType         string
	ProviderAdapterKey   string
	ProviderConfig       json.RawMessage
	QuerySchema          json.RawMessage
	CitationSchema       json.RawMessage
	FreshnessRules       json.RawMessage
	ProviderCapabilities json.RawMessage
}

type AgentDetailInput struct {
	Manifest          json.RawMessage
	Dependencies      json.RawMessage
	PromptMetadata    json.RawMessage
	CompatibilityMeta json.RawMessage
}

type CreateResourceVersionInput struct {
	Version         string
	ContractVersion string
	Summary         string
	Schema          json.RawMessage
	Skill           *SkillDetailInput
	Knowledge       *KnowledgeDetailInput
	Agent           *AgentDetailInput
	CreatedBy       int
}

type ResourceVersionItem struct {
	ResourceId      string
	ResourceType    string
	Version         string
	ContractVersion string
	Summary         string
	SchemaJSON      string
	Status          string
	CreatedBy       int
	PublishedAt     *time.Time
	CreatedAt       int64
	UpdatedAt       int64
	Skill           *SkillDetailOutput
	Knowledge       *KnowledgeDetailOutput
	Agent           *AgentDetailOutput
}

type SkillDetailOutput struct {
	InvokeSchemaJSON  string
	OutputSchemaJSON  string
	InvokeMode        string
	TimeoutSeconds    int
	BindingConfigJSON string
}

type KnowledgeDetailOutput struct {
	KnowledgeMode            string
	ProviderType             string
	ProviderAdapterKey       string
	ProviderConfigJSON       string
	QuerySchemaJSON          string
	CitationSchemaJSON       string
	FreshnessRulesJSON       string
	ProviderCapabilitiesJSON string
}

type AgentDetailOutput struct {
	ManifestJSON          string
	DependenciesJSON      string
	PromptMetadataJSON    string
	CompatibilityMetaJSON string
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

		schemaJSON, err := normalizeJSONText(input.Schema)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		detailJSON, err := buildVersionDetailSnapshot(resource.ResourceType, input)
		if err != nil {
			return err
		}

		version := apmodel.ResourceVersion{
			ResourceId:      resource.ResourceId,
			Version:         input.Version,
			ContractVersion: input.ContractVersion,
			Summary:         input.Summary,
			SchemaJSON:      schemaJSON,
			DetailJSON:      detailJSON,
			Status:          apmodel.ResourceStatusDraft,
			CreatedBy:       input.CreatedBy,
		}
		if err := tx.Create(&version).Error; err != nil {
			if errors.Is(err, apmodel.ErrInvalidResourceVersionBody) || errors.Is(err, apmodel.ErrInvalidResourceStatus) {
				return ErrInvalidResourceVersionInput
			}
			return err
		}

		if err := createTypedDetail(tx, resource.ResourceType, resource.ResourceId, input); err != nil {
			return err
		}
		if err := tx.Model(&apmodel.Resource{}).
			Where("resource_id = ?", resource.ResourceId).
			Updates(map[string]any{"latest_version": input.Version}).Error; err != nil {
			return err
		}

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

	item := ResourceVersionItem{
		ResourceId:      resource.ResourceId,
		ResourceType:    resource.ResourceType,
		Version:         resourceVersion.Version,
		ContractVersion: resourceVersion.ContractVersion,
		Summary:         resourceVersion.Summary,
		SchemaJSON:      resourceVersion.SchemaJSON,
		Status:          resourceVersion.Status,
		CreatedBy:       resourceVersion.CreatedBy,
		PublishedAt:     resourceVersion.PublishedAt,
		CreatedAt:       resourceVersion.CreatedAt.Unix(),
		UpdatedAt:       resourceVersion.UpdatedAt.Unix(),
	}

	switch resource.ResourceType {
	case apmodel.ResourceTypeSkill:
		var def apmodel.SkillDef
		if err := db.Where("resource_id = ? AND resource_version = ?", resourceID, version).First(&def).Error; err != nil {
			return ResourceVersionItem{}, translateDetailLookupError(err)
		}
		item.Skill = &SkillDetailOutput{
			InvokeSchemaJSON:  def.InvokeSchemaJSON,
			OutputSchemaJSON:  def.OutputSchemaJSON,
			InvokeMode:        def.InvokeMode,
			TimeoutSeconds:    def.TimeoutSeconds,
			BindingConfigJSON: def.BindingConfigJSON,
		}
	case apmodel.ResourceTypeKnowledge:
		var def apmodel.KnowledgeDef
		if err := db.Where("resource_id = ? AND resource_version = ?", resourceID, version).First(&def).Error; err != nil {
			return ResourceVersionItem{}, translateDetailLookupError(err)
		}
		item.Knowledge = &KnowledgeDetailOutput{
			KnowledgeMode:            def.KnowledgeMode,
			ProviderType:             def.ProviderType,
			ProviderAdapterKey:       def.ProviderAdapterKey,
			ProviderConfigJSON:       def.ProviderConfigJSON,
			QuerySchemaJSON:          def.QuerySchemaJSON,
			CitationSchemaJSON:       def.CitationSchemaJSON,
			FreshnessRulesJSON:       def.FreshnessRulesJSON,
			ProviderCapabilitiesJSON: def.ProviderCapabilitiesJSON,
		}
	case apmodel.ResourceTypeAgent:
		var def apmodel.AgentDef
		if err := db.Where("resource_id = ? AND resource_version = ?", resourceID, version).First(&def).Error; err != nil {
			return ResourceVersionItem{}, translateDetailLookupError(err)
		}
		item.Agent = &AgentDetailOutput{
			ManifestJSON:          def.ManifestJSON,
			DependenciesJSON:      def.DependenciesJSON,
			PromptMetadataJSON:    def.PromptMetadataJSON,
			CompatibilityMetaJSON: def.CompatibilityMetaJSON,
		}
	default:
		return ResourceVersionItem{}, ErrInvalidResourceVersionInput
	}

	return item, nil
}

func translateDetailLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrResourceVersionNotFound
	}
	return err
}

func buildVersionDetailSnapshot(resourceType string, input CreateResourceVersionInput) (string, error) {
	switch resourceType {
	case apmodel.ResourceTypeSkill:
		if input.Skill == nil || input.Knowledge != nil || input.Agent != nil {
			return "", ErrInvalidResourceVersionInput
		}
		if err := validateSkillDetailInput(*input.Skill); err != nil {
			return "", err
		}
		payload, err := common.Marshal(input.Skill)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	case apmodel.ResourceTypeKnowledge:
		if input.Knowledge == nil || input.Skill != nil || input.Agent != nil {
			return "", ErrInvalidResourceVersionInput
		}
		if err := validateKnowledgeDetailInput(*input.Knowledge); err != nil {
			return "", err
		}
		payload, err := common.Marshal(input.Knowledge)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	case apmodel.ResourceTypeAgent:
		if input.Agent == nil || input.Skill != nil || input.Knowledge != nil {
			return "", ErrInvalidResourceVersionInput
		}
		if err := validateAgentDetailInput(nil, *input.Agent); err != nil {
			return "", err
		}
		payload, err := common.Marshal(input.Agent)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	default:
		return "", ErrInvalidResourceVersionInput
	}
}

func createTypedDetail(tx *gorm.DB, resourceType string, resourceID string, input CreateResourceVersionInput) error {
	switch resourceType {
	case apmodel.ResourceTypeSkill:
		if err := validateSkillDetailInput(*input.Skill); err != nil {
			return err
		}
		timeout := 0
		if input.Skill.TimeoutSeconds != nil {
			timeout = *input.Skill.TimeoutSeconds
		}
		invokeSchemaJSON, err := normalizeJSONText(input.Skill.InvokeSchema)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		outputSchemaJSON, err := normalizeJSONText(input.Skill.OutputSchema)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		bindingConfigJSON, err := normalizeJSONText(input.Skill.BindingConfig)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		err = tx.Create(&apmodel.SkillDef{
			ResourceId:        resourceID,
			ResourceVersion:   input.Version,
			InvokeSchemaJSON:  invokeSchemaJSON,
			OutputSchemaJSON:  outputSchemaJSON,
			InvokeMode:        input.Skill.InvokeMode,
			TimeoutSeconds:    timeout,
			BindingConfigJSON: bindingConfigJSON,
		}).Error
		return normalizeTypedDetailWriteError(err)
	case apmodel.ResourceTypeKnowledge:
		if err := validateKnowledgeDetailInput(*input.Knowledge); err != nil {
			return err
		}
		providerConfigJSON, err := normalizeJSONText(input.Knowledge.ProviderConfig)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		querySchemaJSON, err := normalizeJSONText(input.Knowledge.QuerySchema)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		citationSchemaJSON, err := normalizeJSONText(input.Knowledge.CitationSchema)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		freshnessRulesJSON, err := normalizeJSONText(input.Knowledge.FreshnessRules)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		providerCapabilitiesJSON, err := normalizeJSONText(input.Knowledge.ProviderCapabilities)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		err = tx.Create(&apmodel.KnowledgeDef{
			ResourceId:               resourceID,
			ResourceVersion:          input.Version,
			KnowledgeMode:            input.Knowledge.KnowledgeMode,
			ProviderType:             input.Knowledge.ProviderType,
			ProviderAdapterKey:       input.Knowledge.ProviderAdapterKey,
			ProviderConfigJSON:       providerConfigJSON,
			QuerySchemaJSON:          querySchemaJSON,
			CitationSchemaJSON:       citationSchemaJSON,
			FreshnessRulesJSON:       freshnessRulesJSON,
			ProviderCapabilitiesJSON: providerCapabilitiesJSON,
		}).Error
		return normalizeTypedDetailWriteError(err)
	case apmodel.ResourceTypeAgent:
		if err := validateAgentDetailInput(tx, *input.Agent); err != nil {
			return err
		}
		manifestJSON, err := normalizeJSONText(input.Agent.Manifest)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		dependenciesJSON, err := normalizeJSONText(input.Agent.Dependencies)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		promptMetadataJSON, err := normalizeJSONText(input.Agent.PromptMetadata)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		compatibilityMetaJSON, err := normalizeJSONText(input.Agent.CompatibilityMeta)
		if err != nil {
			return ErrInvalidResourceVersionInput
		}
		err = tx.Create(&apmodel.AgentDef{
			ResourceId:            resourceID,
			ResourceVersion:       input.Version,
			CliType:               apmodel.AgentCliTypeOpenCode,
			ManifestJSON:          manifestJSON,
			DependenciesJSON:      dependenciesJSON,
			PromptMetadataJSON:    promptMetadataJSON,
			CompatibilityMetaJSON: compatibilityMetaJSON,
		}).Error
		return normalizeTypedDetailWriteError(err)
	default:
		return ErrInvalidResourceVersionInput
	}
}

func validateSkillDetailInput(input SkillDetailInput) error {
	input.InvokeMode = strings.TrimSpace(strings.ToLower(input.InvokeMode))
	if len(input.InvokeSchema) == 0 || len(input.OutputSchema) == 0 || len(input.BindingConfig) == 0 {
		return ErrSkillContractInvalid
	}
	if input.TimeoutSeconds == nil || *input.TimeoutSeconds <= 0 {
		return ErrSkillContractInvalid
	}
	if input.InvokeMode != "sync" && input.InvokeMode != "async" {
		return ErrSkillContractInvalid
	}
	invokeSchemaJSON, err := normalizeJSONText(input.InvokeSchema)
	if err != nil || strings.TrimSpace(invokeSchemaJSON) == "" {
		return ErrSkillContractInvalid
	}
	outputSchemaJSON, err := normalizeJSONText(input.OutputSchema)
	if err != nil || strings.TrimSpace(outputSchemaJSON) == "" {
		return ErrSkillContractInvalid
	}
	bindingConfigJSON, err := normalizeJSONText(input.BindingConfig)
	if err != nil || strings.TrimSpace(bindingConfigJSON) == "" {
		return ErrSkillContractInvalid
	}
	return nil
}

func validateKnowledgeDetailInput(input KnowledgeDetailInput) error {
	input.KnowledgeMode = strings.TrimSpace(strings.ToLower(input.KnowledgeMode))
	input.ProviderType = strings.TrimSpace(strings.ToLower(input.ProviderType))
	input.ProviderAdapterKey = strings.TrimSpace(input.ProviderAdapterKey)
	if input.KnowledgeMode != "retrieval" {
		return ErrKnowledgeContractInvalid
	}
	if input.ProviderType != "native" && input.ProviderType != "http_retrieval" {
		return ErrKnowledgeContractInvalid
	}
	if input.ProviderAdapterKey == "" {
		return ErrKnowledgeContractInvalid
	}
	providerConfigJSON, err := normalizeJSONText(input.ProviderConfig)
	if err != nil || strings.TrimSpace(providerConfigJSON) == "" {
		return ErrKnowledgeContractInvalid
	}
	querySchemaJSON, err := normalizeJSONText(input.QuerySchema)
	if err != nil || strings.TrimSpace(querySchemaJSON) == "" {
		return ErrKnowledgeContractInvalid
	}
	citationSchemaJSON, err := normalizeJSONText(input.CitationSchema)
	if err != nil || strings.TrimSpace(citationSchemaJSON) == "" {
		return ErrKnowledgeContractInvalid
	}
	return nil
}

func validateAgentDetailInput(db *gorm.DB, input AgentDetailInput) error {
	manifestJSON, err := normalizeJSONText(input.Manifest)
	if err != nil || strings.TrimSpace(manifestJSON) == "" {
		return ErrAgentDependencyInvalid
	}
	dependenciesJSON, err := normalizeJSONText(input.Dependencies)
	if err != nil || strings.TrimSpace(dependenciesJSON) == "" {
		return ErrAgentDependencyInvalid
	}
	compatibilityJSON, err := normalizeJSONText(input.CompatibilityMeta)
	if err != nil || strings.TrimSpace(compatibilityJSON) == "" {
		return ErrAgentDependencyInvalid
	}

	var dependencies []map[string]any
	if err := common.UnmarshalJsonStr(dependenciesJSON, &dependencies); err != nil || len(dependencies) == 0 {
		return ErrAgentDependencyInvalid
	}
	for _, dependency := range dependencies {
		resourceType := strings.TrimSpace(strings.ToLower(common.Interface2String(dependency["resource_type"])))
		resourceID := strings.TrimSpace(common.Interface2String(dependency["resource_id"]))
		if resourceID == "" || (resourceType != apmodel.ResourceTypeSkill && resourceType != apmodel.ResourceTypeKnowledge) {
			return ErrAgentDependencyInvalid
		}
		if db != nil {
			var resource apmodel.Resource
			if err := db.Where("resource_id = ? AND resource_type = ?", resourceID, resourceType).First(&resource).Error; err != nil {
				return ErrAgentDependencyInvalid
			}
		}
	}
	return nil
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

func normalizeTypedDetailWriteError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, apmodel.ErrInvalidSkillDefBody),
		errors.Is(err, apmodel.ErrInvalidKnowledgeDefBody),
		errors.Is(err, apmodel.ErrInvalidAgentDefBody):
		return ErrInvalidResourceVersionInput
	default:
		return err
	}
}
