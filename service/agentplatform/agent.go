package agentplatform

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	appmodel "github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	coreservice "github.com/QuantumNous/new-api/service"
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
	ModelTokenId int
	DefaultModel string
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
	ModelTokenId int
	DefaultModel string
	McpIds       []string
	SkillIds     []string
	KnowledgeIds []string
}

type AgentItem struct {
	ResourceItem
	CliType             string
	Instructions        string
	ModelTokenId        int
	DefaultModel        string
	ModelTokenUserId    int
	ModelTokenUserName  string
	ModelTokenName      string
	ModelTokenMaskedKey string
	McpIds              []string
	SkillIds            []string
	KnowledgeIds        []string
}

type AgentListResult struct {
	Items    []AgentItem
	Total    int
	Page     int
	PageSize int
}

type AgentModelKeyItem struct {
	Id                 int
	UserId             int
	UserName           string
	Name               string
	MaskedKey          string
	Status             int
	ExpiredTime        int64
	RemainQuota        int
	UnlimitedQuota     bool
	Group              string
	ModelLimitsEnabled bool
	ModelCount         int
	Available          bool
	DisabledReason     string
}

type AgentModelItem struct {
	Model        string
	DisplayName  string
	Status       string
	Capabilities map[string]any
}

type AgentTokenModelResult struct {
	Token  appmodel.Token
	Models []AgentModelItem
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
		if input.ModelTokenId > 0 || input.DefaultModel != "" {
			if _, err := resolveTokenModels(tx, input.ModelTokenId, input.DefaultModel, true); err != nil {
				return err
			}
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
			ModelTokenId:    input.ModelTokenId,
			DefaultModel:    input.DefaultModel,
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
		if input.ModelTokenId > 0 || input.DefaultModel != "" {
			if _, err := resolveTokenModels(tx, input.ModelTokenId, input.DefaultModel, true); err != nil {
				return err
			}
		}
		if err := tx.Model(&apmodel.Resource{}).Where("resource_id = ?", resourceID).Updates(map[string]any{
			"display_name": input.DisplayName,
			"description":  input.Description,
			"avatar":       input.Avatar,
		}).Error; err != nil {
			return err
		}
		defValues := map[string]any{
			"cli_type":       input.CliType,
			"name":           input.DisplayName,
			"description":    input.Description,
			"avatar":         input.Avatar,
			"instructions":   input.Instructions,
			"model_token_id": input.ModelTokenId,
			"default_model":  input.DefaultModel,
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
				ModelTokenId:    input.ModelTokenId,
				DefaultModel:    input.DefaultModel,
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
	return setTypedResourceStatus(s.db, resourceID, apmodel.ResourceTypeAgent, apmodel.ResourceStatusRevoked)
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
		agent.ModelTokenId = def.ModelTokenId
		agent.DefaultModel = def.DefaultModel
		if def.ModelTokenId > 0 {
			if token, err := tokenByID(s.db, def.ModelTokenId); err == nil {
				agent.ModelTokenUserId = token.UserId
				agent.ModelTokenUserName = tokenOwnerName(s.db, token.UserId)
				agent.ModelTokenName = strings.TrimSpace(token.Name)
				agent.ModelTokenMaskedKey = token.GetMaskedKey()
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return AgentItem{}, err
			}
		}
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
	input.DefaultModel = strings.TrimSpace(input.DefaultModel)
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
	input.DefaultModel = strings.TrimSpace(input.DefaultModel)
	input.McpIds = normalizeIDList(input.McpIds)
	input.SkillIds = normalizeIDList(input.SkillIds)
	input.KnowledgeIds = normalizeIDList(input.KnowledgeIds)
}

func (s *AgentService) ListModelKeys(keyword string) ([]AgentModelKeyItem, error) {
	if s == nil || s.db == nil {
		return nil, ErrInvalidResourceInput
	}
	keyword = strings.TrimSpace(keyword)
	var tokens []appmodel.Token
	query := s.db.Model(&appmodel.Token{}).Joins("LEFT JOIN users ON users.id = tokens.user_id")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"tokens.name LIKE ? OR tokens.key LIKE ? OR users.username LIKE ? OR users.display_name LIKE ? OR users.email LIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}
	if err := query.Order("tokens.id DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	userNames, err := tokenOwnerNames(s.db, tokens)
	if err != nil {
		return nil, err
	}
	items := make([]AgentModelKeyItem, 0, len(tokens))
	for _, token := range tokens {
		result, err := resolveTokenModels(s.db, token.Id, "", false)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		modelCount := 0
		if err == nil {
			modelCount = len(result.Models)
		}
		available, disabledReason := tokenAvailability(token)
		items = append(items, AgentModelKeyItem{
			Id:                 token.Id,
			UserId:             token.UserId,
			UserName:           userNames[token.UserId],
			Name:               strings.TrimSpace(token.Name),
			MaskedKey:          token.GetMaskedKey(),
			Status:             token.Status,
			ExpiredTime:        token.ExpiredTime,
			RemainQuota:        token.RemainQuota,
			UnlimitedQuota:     token.UnlimitedQuota,
			Group:              strings.TrimSpace(token.Group),
			ModelLimitsEnabled: token.ModelLimitsEnabled,
			ModelCount:         modelCount,
			Available:          available && modelCount > 0,
			DisabledReason:     disabledReason,
		})
	}
	return items, nil
}

func (s *AgentService) ListTokenModels(tokenID int) (AgentTokenModelResult, error) {
	if s == nil || s.db == nil || tokenID <= 0 {
		return AgentTokenModelResult{}, ErrInvalidResourceInput
	}
	return resolveTokenModels(s.db, tokenID, "", false)
}

func resolveTokenModels(db *gorm.DB, tokenID int, defaultModel string, requireDefault bool) (AgentTokenModelResult, error) {
	if db == nil || tokenID <= 0 {
		return AgentTokenModelResult{}, ErrInvalidResourceInput
	}
	defaultModel = strings.TrimSpace(defaultModel)
	if requireDefault && defaultModel == "" {
		return AgentTokenModelResult{}, ErrInvalidResourceInput
	}
	token, err := tokenByID(db, tokenID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AgentTokenModelResult{}, ErrInvalidResourceInput
		}
		return AgentTokenModelResult{}, err
	}
	if requireDefault {
		if available, _ := tokenAvailability(token); !available {
			return AgentTokenModelResult{}, ErrInvalidResourceInput
		}
	}
	modelNames, err := tokenModelNames(db, token)
	if err != nil {
		return AgentTokenModelResult{}, err
	}
	items := make([]AgentModelItem, 0, len(modelNames))
	defaultFound := defaultModel == ""
	for _, modelName := range modelNames {
		if modelName == defaultModel {
			defaultFound = true
		}
		items = append(items, AgentModelItem{
			Model:        modelName,
			DisplayName:  openCodeModelDisplayName(db, modelName),
			Status:       "available",
			Capabilities: buildOpenCodeModelConfig(modelName),
		})
	}
	if !defaultFound {
		return AgentTokenModelResult{}, ErrInvalidResourceInput
	}
	return AgentTokenModelResult{Token: token, Models: items}, nil
}

func tokenByID(db *gorm.DB, tokenID int) (appmodel.Token, error) {
	var token appmodel.Token
	err := db.Where("id = ?", tokenID).First(&token).Error
	return token, err
}

func tokenModelNames(db *gorm.DB, token appmodel.Token) ([]string, error) {
	if token.ModelLimitsEnabled {
		return normalizeModelNames(token.GetModelLimits()), nil
	}
	groups, err := tokenModelGroups(db, token)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	modelNames := make([]string, 0)
	for _, group := range groups {
		var groupModels []string
		if err := db.Model(&appmodel.Ability{}).
			Where(&appmodel.Ability{Group: group, Enabled: true}).
			Distinct("model").
			Pluck("model", &groupModels).Error; err != nil {
			return nil, err
		}
		for _, modelName := range groupModels {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}
			if _, ok := seen[modelName]; ok {
				continue
			}
			seen[modelName] = struct{}{}
			modelNames = append(modelNames, modelName)
		}
	}
	sort.Strings(modelNames)
	return modelNames, nil
}

func tokenModelGroups(db *gorm.DB, token appmodel.Token) ([]string, error) {
	tokenGroup := strings.TrimSpace(token.Group)
	if tokenGroup != "" && tokenGroup != "auto" {
		return []string{tokenGroup}, nil
	}
	var user appmodel.User
	if err := db.Select("id", "group").Where("id = ?", token.UserId).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []string{"default"}, nil
		}
		return nil, err
	}
	userGroup := strings.TrimSpace(user.Group)
	if userGroup == "" {
		userGroup = "default"
	}
	if tokenGroup == "auto" {
		groups := coreservice.GetUserAutoGroup(userGroup)
		if len(groups) == 0 {
			return []string{userGroup}, nil
		}
		return normalizeModelNames(groups), nil
	}
	return []string{userGroup}, nil
}

func tokenOwnerNames(db *gorm.DB, tokens []appmodel.Token) (map[int]string, error) {
	userIDs := make([]int, 0, len(tokens))
	seen := map[int]struct{}{}
	for _, token := range tokens {
		if token.UserId <= 0 {
			continue
		}
		if _, ok := seen[token.UserId]; ok {
			continue
		}
		seen[token.UserId] = struct{}{}
		userIDs = append(userIDs, token.UserId)
	}
	if len(userIDs) == 0 {
		return map[int]string{}, nil
	}
	var users []appmodel.User
	if err := db.Select("id", "username", "display_name", "email").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	names := make(map[int]string, len(users))
	for _, user := range users {
		names[user.Id] = displayUserName(user)
	}
	for _, userID := range userIDs {
		if names[userID] == "" {
			names[userID] = "#" + strconv.Itoa(userID)
		}
	}
	return names, nil
}

func tokenOwnerName(db *gorm.DB, userID int) string {
	if db == nil || userID <= 0 {
		return ""
	}
	var user appmodel.User
	if err := db.Select("id", "username", "display_name", "email").Where("id = ?", userID).First(&user).Error; err != nil {
		return "#" + strconv.Itoa(userID)
	}
	return displayUserName(user)
}

func displayUserName(user appmodel.User) string {
	for _, value := range []string{user.DisplayName, user.Username, user.Email} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	if user.Id > 0 {
		return "#" + strconv.Itoa(user.Id)
	}
	return ""
}

func normalizeModelNames(values []string) []string {
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
	sort.Strings(out)
	return out
}

func tokenAvailability(token appmodel.Token) (bool, string) {
	switch {
	case token.Status != common.TokenStatusEnabled:
		return false, "disabled"
	case token.ExpiredTime != -1 && token.ExpiredTime < common.GetTimestamp():
		return false, "expired"
	case !token.UnlimitedQuota && token.RemainQuota <= 0:
		return false, "exhausted"
	default:
		return true, ""
	}
}

func openCodeModelDisplayName(db *gorm.DB, modelName string) string {
	var meta appmodel.Model
	if err := db.Select("model_name").Where("model_name = ?", modelName).First(&meta).Error; err == nil && strings.TrimSpace(meta.ModelName) != "" {
		return strings.TrimSpace(meta.ModelName)
	}
	return titleModelName(modelName)
}

func titleModelName(modelName string) string {
	parts := strings.FieldsFunc(modelName, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	if len(parts) == 0 {
		return modelName
	}
	for index := range parts {
		parts[index] = strings.ToUpper(parts[index])
	}
	return strings.Join(parts, "-")
}

func buildOpenCodeModelConfig(modelName string) map[string]any {
	return map[string]any{
		"name":        titleModelName(modelName),
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

func buildOpenCodeModels(items []AgentModelItem) map[string]any {
	models := make(map[string]any, len(items))
	for _, item := range items {
		config := item.Capabilities
		config["name"] = item.DisplayName
		models[item.Model] = config
	}
	return models
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
