package agentplatform

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

type AgentQuery struct {
	OwnerUserId *int
	TenantId    *int
	Page        int
	PageSize    int
}

type AgentCreateInput struct {
	CliType      string
	DisplayName  string
	Description  string
	Avatar       string
	Instructions string
	McpIds       []string
	SkillIds     []string
	KnowledgeIds []string
	OwnerUserId  int
	TenantId     int
}

type AgentUpdateInput struct {
	CliType      string
	DisplayName  string
	Description  string
	Avatar       string
	Instructions string
	McpIds       []string
	SkillIds     []string
	KnowledgeIds []string
}

type AgentItem struct {
	ResourceItem
	CliType      string
	Instructions string
	McpIds       []string
	SkillIds     []string
	KnowledgeIds []string
}

type AgentListResult struct {
	Items    []AgentItem
	Total    int
	Page     int
	PageSize int
}

type AgentService struct {
	resources *ResourceService
	db        *gorm.DB
}

func NewAgentService(db *gorm.DB) *AgentService {
	return &AgentService{
		resources: NewResourceService(db),
		db:        db,
	}
}

func (s *AgentService) List(query AgentQuery) (AgentListResult, error) {
	if s == nil || s.resources == nil {
		return AgentListResult{Items: []AgentItem{}}, ErrInvalidResourceInput
	}
	result, err := s.resources.List(ListResourcesQuery{
		ResourceType: apmodel.ResourceTypeAgent,
		OwnerUserId:  query.OwnerUserId,
		TenantId:     query.TenantId,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		return AgentListResult{Items: []AgentItem{}}, err
	}
	items := make([]AgentItem, 0, len(result.Items))
	for _, item := range result.Items {
		detail, err := s.detailForResource(item)
		if err != nil && !errors.Is(err, ErrResourceNotFound) {
			return AgentListResult{Items: []AgentItem{}}, err
		}
		items = append(items, detail)
	}
	return AgentListResult{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func (s *AgentService) Get(resourceID string) (AgentItem, error) {
	if s == nil || s.resources == nil {
		return AgentItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.GetByResourceID(resourceID)
	if err != nil {
		return AgentItem{}, err
	}
	if item.ResourceType != apmodel.ResourceTypeAgent {
		return AgentItem{}, ErrResourceNotFound
	}
	return s.detailForResource(item)
}

func (s *AgentService) Create(input AgentCreateInput) (AgentItem, error) {
	if s == nil || s.db == nil {
		return AgentItem{}, ErrInvalidResourceInput
	}
	input.normalize()
	if input.DisplayName == "" || input.OwnerUserId <= 0 || !apmodel.ValidAgentCliType(input.CliType) {
		return AgentItem{}, ErrInvalidResourceInput
	}
	var resourceID string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := validateAgentDependencyTargets(tx, input.McpIds, input.SkillIds, input.KnowledgeIds); err != nil {
			return err
		}
		resource := apmodel.Resource{
			ResourceType: apmodel.ResourceTypeAgent,
			DisplayName:  input.DisplayName,
			Description:  input.Description,
			Avatar:       input.Avatar,
			OwnerUserId:  input.OwnerUserId,
			TenantId:     input.TenantId,
			Status:       apmodel.ResourceStatusDraft,
		}
		if err := tx.Create(&resource).Error; err != nil {
			return err
		}
		def := apmodel.AgentDef{
			ResourceId:      resource.ResourceId,
			ResourceVersion: "draft",
			CliType:         input.CliType,
			Name:            input.DisplayName,
			Description:     input.Description,
			Avatar:          input.Avatar,
			Instructions:    input.Instructions,
		}
		if err := tx.Create(&def).Error; err != nil {
			return err
		}
		if err := replaceAgentDependencies(tx, resource.ResourceId, "draft", input.McpIds, input.SkillIds, input.KnowledgeIds, false); err != nil {
			return err
		}
		resourceID = resource.ResourceId
		return nil
	})
	if err != nil {
		return AgentItem{}, err
	}
	return s.Get(resourceID)
}

func (s *AgentService) Update(resourceID string, input AgentUpdateInput) (AgentItem, error) {
	if s == nil || s.db == nil {
		return AgentItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.normalize()
	if resourceID == "" || input.DisplayName == "" || !apmodel.ValidAgentCliType(input.CliType) {
		return AgentItem{}, ErrInvalidResourceInput
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var resource apmodel.Resource
		if err := tx.Where("resource_id = ? AND resource_type = ?", resourceID, apmodel.ResourceTypeAgent).First(&resource).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResourceNotFound
			}
			return err
		}
		if err := validateAgentDependencyTargets(tx, input.McpIds, input.SkillIds, input.KnowledgeIds); err != nil {
			return err
		}
		if err := tx.Model(&apmodel.Resource{}).Where("resource_id = ?", resourceID).Updates(map[string]any{
			"display_name": input.DisplayName,
			"description":  input.Description,
			"avatar":       input.Avatar,
		}).Error; err != nil {
			return err
		}
		defValues := map[string]any{
			"cli_type":     input.CliType,
			"name":         input.DisplayName,
			"description":  input.Description,
			"avatar":       input.Avatar,
			"instructions": input.Instructions,
		}
		result := tx.Model(&apmodel.AgentDef{}).Where("resource_id = ? AND resource_version = ?", resourceID, "draft").Updates(defValues)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			def := apmodel.AgentDef{
				ResourceId:      resourceID,
				ResourceVersion: "draft",
				CliType:         input.CliType,
				Name:            input.DisplayName,
				Description:     input.Description,
				Avatar:          input.Avatar,
				Instructions:    input.Instructions,
			}
			if err := tx.Create(&def).Error; err != nil {
				return err
			}
		}
		return replaceAgentDependencies(tx, resourceID, "draft", input.McpIds, input.SkillIds, input.KnowledgeIds, false)
	})
	if err != nil {
		return AgentItem{}, err
	}
	return s.Get(resourceID)
}

func (s *AgentService) Delete(resourceID string) error {
	return deleteTypedResource(s.db, resourceID, apmodel.ResourceTypeAgent)
}

func (s *AgentService) detailForResource(item ResourceItem) (AgentItem, error) {
	if item.ResourceType != apmodel.ResourceTypeAgent {
		return AgentItem{}, ErrResourceNotFound
	}
	agent := AgentItem{ResourceItem: item, McpIds: []string{}, SkillIds: []string{}, KnowledgeIds: []string{}}
	var def apmodel.AgentDef
	if err := s.db.Where("resource_id = ? AND resource_version = ?", item.ResourceId, "draft").First(&def).Error; err == nil {
		agent.CliType = def.CliType
		agent.Instructions = def.Instructions
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return AgentItem{}, err
	}
	var deps []apmodel.AgentDependency
	if err := s.db.Where("agent_resource_id = ? AND resource_version = ?", item.ResourceId, "draft").Order("sort_order ASC, id ASC").Find(&deps).Error; err != nil {
		return AgentItem{}, err
	}
	for _, dep := range deps {
		switch dep.TargetType {
		case apmodel.AgentDependencyTypeMCP:
			agent.McpIds = append(agent.McpIds, dep.TargetResourceId)
		case apmodel.AgentDependencyTypeSkill:
			agent.SkillIds = append(agent.SkillIds, dep.TargetResourceId)
		case apmodel.AgentDependencyTypeKnowledge:
			agent.KnowledgeIds = append(agent.KnowledgeIds, dep.TargetResourceId)
		}
	}
	return agent, nil
}

func (input *AgentCreateInput) normalize() {
	input.CliType = strings.TrimSpace(strings.ToLower(input.CliType))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.Avatar = strings.TrimSpace(input.Avatar)
	input.Instructions = strings.TrimSpace(input.Instructions)
	input.McpIds = normalizeIDList(input.McpIds)
	input.SkillIds = normalizeIDList(input.SkillIds)
	input.KnowledgeIds = normalizeIDList(input.KnowledgeIds)
}

func (input *AgentUpdateInput) normalize() {
	input.CliType = strings.TrimSpace(strings.ToLower(input.CliType))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.Avatar = strings.TrimSpace(input.Avatar)
	input.Instructions = strings.TrimSpace(input.Instructions)
	input.McpIds = normalizeIDList(input.McpIds)
	input.SkillIds = normalizeIDList(input.SkillIds)
	input.KnowledgeIds = normalizeIDList(input.KnowledgeIds)
}

func buildOpenCodeModelConfig(modelName string) map[string]any {
	return map[string]any{
		"name":        modelName,
		"temperature": false,
		"reasoning":   true,
		"tool_call":   true,
		"modalities": map[string]any{
			"input":  []string{"text", "image"},
			"output": []string{"text", "image"},
		},
		"limit": map[string]any{
			"context": 200000,
			"input":   200000,
			"output":  32000,
		},
		"variants": map[string]any{
			"low":    map[string]any{"reasoningEffort": "low"},
			"medium": map[string]any{"reasoningEffort": "medium"},
			"high":   map[string]any{"reasoningEffort": "high"},
			"xhigh":  map[string]any{"reasoningEffort": "xhigh"},
		},
	}
}

func normalizeIDList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validateAgentDependencyTargets(tx *gorm.DB, mcpIds []string, skillIds []string, knowledgeIds []string) error {
	for _, item := range []struct {
		typ string
		ids []string
	}{
		{apmodel.ResourceTypeMCP, mcpIds},
		{apmodel.ResourceTypeSkill, skillIds},
		{apmodel.ResourceTypeKnowledge, knowledgeIds},
	} {
		for _, id := range item.ids {
			var count int64
			if err := tx.Model(&apmodel.Resource{}).Where("resource_id = ? AND resource_type = ? AND status <> ?", id, item.typ, apmodel.ResourceStatusRevoked).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrAgentDependencyInvalid
			}
		}
	}
	return nil
}

func replaceAgentDependencies(tx *gorm.DB, agentResourceID string, version string, mcpIds []string, skillIds []string, knowledgeIds []string, snapshot bool) error {
	if err := tx.Where("agent_resource_id = ? AND resource_version = ?", agentResourceID, version).Delete(&apmodel.AgentDependency{}).Error; err != nil {
		return err
	}
	order := 0
	for _, item := range []struct {
		typ string
		ids []string
	}{
		{apmodel.AgentDependencyTypeMCP, mcpIds},
		{apmodel.AgentDependencyTypeSkill, skillIds},
		{apmodel.AgentDependencyTypeKnowledge, knowledgeIds},
	} {
		for _, id := range item.ids {
			dep := apmodel.AgentDependency{
				AgentResourceId:  agentResourceID,
				ResourceVersion:  version,
				TargetType:       item.typ,
				TargetResourceId: id,
				SortOrder:        order,
			}
			if snapshotJSON, err := dependencySnapshot(tx, item.typ, id, snapshot); err != nil {
				return err
			} else {
				dep.SnapshotJSON = snapshotJSON
			}
			if err := tx.Create(&dep).Error; err != nil {
				return err
			}
			order++
		}
	}
	return nil
}

func dependencySnapshot(tx *gorm.DB, resourceType string, resourceID string, snapshot bool) (string, error) {
	if !snapshot {
		return "", nil
	}
	var resource apmodel.Resource
	if err := tx.Where("resource_id = ? AND resource_type = ?", resourceID, resourceType).First(&resource).Error; err != nil {
		return "", err
	}
	payload := map[string]any{
		"resource_type": resource.ResourceType,
		"resource_id":   resource.ResourceId,
		"display_name":  resource.DisplayName,
		"description":   resource.Description,
	}
	switch resourceType {
	case apmodel.ResourceTypeMCP:
		var def apmodel.McpDef
		if err := tx.Where("resource_id = ?", resourceID).First(&def).Error; err != nil {
			return "", err
		}
		payload["config_json"] = def.ConfigJSON
	case apmodel.ResourceTypeSkill:
		var def apmodel.SkillDef
		if err := tx.Where("resource_id = ?", resourceID).First(&def).Error; err != nil {
			return "", err
		}
		payload["file_path"] = def.FilePath
		payload["sha256"] = def.Sha256
	case apmodel.ResourceTypeKnowledge:
		var def apmodel.KnowledgeDef
		if err := tx.Where("resource_id = ?", resourceID).First(&def).Error; err != nil {
			return "", err
		}
		payload["external_knowledge_id"] = def.ExternalKnowledgeId
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
