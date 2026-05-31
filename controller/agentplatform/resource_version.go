package agentplatform

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func CreateResourceVersion(c *gin.Context) {
	var req dtoagentplatform.CreateResourceVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}

	item, err := resourceVersionService().Create(c.Param("id"), apservice.CreateResourceVersionInput{
		Version:         req.Version,
		ContractVersion: req.ContractVersion,
		Summary:         req.Summary,
		Schema:          req.Schema,
		Skill:           mapSkillDetailInput(req.Skill),
		Knowledge:       mapKnowledgeDetailInput(req.Knowledge),
		Agent:           mapAgentDetailInput(req.Agent),
		CreatedBy:       c.GetInt("id"),
	})
	if err != nil {
		writeResourceVersionError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceVersionItem(item))
}

func GetResourceVersion(c *gin.Context) {
	item, err := resourceVersionService().Get(c.Param("id"), c.Param("version"))
	if err != nil {
		writeResourceVersionError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceVersionItem(item))
}

func resourceVersionService() *apservice.ResourceVersionService {
	return apservice.NewResourceVersionService(model.DB)
}

func writeResourceVersionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apservice.ErrInvalidResourceVersionInput), errors.Is(err, apservice.ErrSkillContractInvalid):
		common.ApiErrorMsg(c, "invalid request params")
	case errors.Is(err, apservice.ErrResourceNotFound), errors.Is(err, apservice.ErrResourceVersionNotFound):
		common.ApiErrorMsg(c, "resource not found")
	default:
		common.ApiError(c, err)
	}
}

func mapResourceVersionItem(item apservice.ResourceVersionItem) dtoagentplatform.ResourceVersionItem {
	response := dtoagentplatform.ResourceVersionItem{
		ResourceId:      item.ResourceId,
		ResourceType:    item.ResourceType,
		Version:         item.Version,
		ContractVersion: item.ContractVersion,
		Summary:         item.Summary,
		Schema:          jsonTextToRawMessage(item.SchemaJSON),
		Status:          item.Status,
		CreatedBy:       item.CreatedBy,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		Skill:           mapSkillDetailOutput(item.Skill),
		Knowledge:       mapKnowledgeDetailOutput(item.Knowledge),
		Agent:           mapAgentDetailOutput(item.Agent),
	}
	if item.PublishedAt != nil {
		publishedAt := item.PublishedAt.Unix()
		response.PublishedAt = &publishedAt
	}
	return response
}

func mapSkillDetailInput(detail *dtoagentplatform.SkillDetailRequest) *apservice.SkillDetailInput {
	if detail == nil {
		return nil
	}
	return &apservice.SkillDetailInput{
		InvokeSchema:   detail.InvokeSchema,
		OutputSchema:   detail.OutputSchema,
		InvokeMode:     detail.InvokeMode,
		TimeoutSeconds: detail.TimeoutSeconds,
		BindingConfig:  detail.BindingConfig,
	}
}

func mapKnowledgeDetailInput(detail *dtoagentplatform.KnowledgeDetailRequest) *apservice.KnowledgeDetailInput {
	if detail == nil {
		return nil
	}
	return &apservice.KnowledgeDetailInput{
		KnowledgeMode:        detail.KnowledgeMode,
		ProviderType:         detail.ProviderType,
		ProviderAdapterKey:   detail.ProviderAdapterKey,
		ProviderConfig:       detail.ProviderConfig,
		QuerySchema:          detail.QuerySchema,
		CitationSchema:       detail.CitationSchema,
		FreshnessRules:       detail.FreshnessRules,
		ProviderCapabilities: detail.ProviderCapabilities,
	}
}

func mapAgentDetailInput(detail *dtoagentplatform.AgentDetailRequest) *apservice.AgentDetailInput {
	if detail == nil {
		return nil
	}
	return &apservice.AgentDetailInput{
		Manifest:          detail.Manifest,
		Dependencies:      detail.Dependencies,
		PromptMetadata:    detail.PromptMetadata,
		CompatibilityMeta: detail.CompatibilityMeta,
	}
}

func mapSkillDetailOutput(detail *apservice.SkillDetailOutput) *dtoagentplatform.SkillDetailResponse {
	if detail == nil {
		return nil
	}
	return &dtoagentplatform.SkillDetailResponse{
		InvokeSchema:   jsonTextToRawMessage(detail.InvokeSchemaJSON),
		OutputSchema:   jsonTextToRawMessage(detail.OutputSchemaJSON),
		InvokeMode:     detail.InvokeMode,
		TimeoutSeconds: detail.TimeoutSeconds,
		BindingConfig:  jsonTextToRawMessage(detail.BindingConfigJSON),
	}
}

func mapKnowledgeDetailOutput(detail *apservice.KnowledgeDetailOutput) *dtoagentplatform.KnowledgeDetailResponse {
	if detail == nil {
		return nil
	}
	return &dtoagentplatform.KnowledgeDetailResponse{
		KnowledgeMode:        detail.KnowledgeMode,
		ProviderType:         detail.ProviderType,
		ProviderAdapterKey:   detail.ProviderAdapterKey,
		ProviderConfig:       jsonTextToRawMessage(detail.ProviderConfigJSON),
		QuerySchema:          jsonTextToRawMessage(detail.QuerySchemaJSON),
		CitationSchema:       jsonTextToRawMessage(detail.CitationSchemaJSON),
		FreshnessRules:       jsonTextToRawMessage(detail.FreshnessRulesJSON),
		ProviderCapabilities: jsonTextToRawMessage(detail.ProviderCapabilitiesJSON),
	}
}

func mapAgentDetailOutput(detail *apservice.AgentDetailOutput) *dtoagentplatform.AgentDetailResponse {
	if detail == nil {
		return nil
	}
	return &dtoagentplatform.AgentDetailResponse{
		Manifest:          jsonTextToRawMessage(detail.ManifestJSON),
		Dependencies:      jsonTextToRawMessage(detail.DependenciesJSON),
		PromptMetadata:    jsonTextToRawMessage(detail.PromptMetadataJSON),
		CompatibilityMeta: jsonTextToRawMessage(detail.CompatibilityMetaJSON),
	}
}

func jsonTextToRawMessage(value string) []byte {
	if value == "" {
		return nil
	}
	return []byte(value)
}
