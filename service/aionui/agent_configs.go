package aionui

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
)

const (
	CliTypeOpenCode    = "opencode"
	CliTypeCodex       = "codex"
	defaultAgentAvatar = "\U0001f916"
)

type AgentConfigService struct {
	now func() time.Time
}

func NewAgentConfigService() *AgentConfigService {
	return &AgentConfigService{now: time.Now}
}

func NewDefaultAgentConfigService() *AgentConfigService {
	return NewAgentConfigService()
}

func (s *AgentConfigService) List(email string) (dtoaionui.AgentConfigsResponse, error) {
	return s.listPlatform(0, email)
}

func (s *AgentConfigService) ListForUser(userID int, email string) (dtoaionui.AgentConfigsResponse, error) {
	return s.listPlatform(userID, email)
}

func (s *AgentConfigService) listPlatform(userID int, email string) (dtoaionui.AgentConfigsResponse, error) {
	if model.DB == nil || userID <= 0 {
		return dtoaionui.AgentConfigsResponse{
			UserEmail: strings.TrimSpace(strings.ToLower(email)),
			Revision:  s.now().UTC().Format(time.RFC3339),
			Agents:    []dtoaionui.AgentConfigItem{},
		}, nil
	}
	subjects, err := platformGrantSubjects(userID)
	if err != nil {
		return dtoaionui.AgentConfigsResponse{}, err
	}
	if len(subjects) == 0 {
		return dtoaionui.AgentConfigsResponse{
			UserEmail: strings.TrimSpace(strings.ToLower(email)),
			Revision:  s.now().UTC().Format(time.RFC3339),
			Agents:    []dtoaionui.AgentConfigItem{},
		}, nil
	}
	var grants []apmodel.ResourceGrant
	query := model.DB.Where("status = ?", apmodel.GrantStatusActive)
	where, args := platformGrantSubjectWhere(subjects)
	query = query.Where(where, args...)
	if err := query.Find(&grants).Error; err != nil {
		return dtoaionui.AgentConfigsResponse{}, err
	}
	items := make([]dtoaionui.AgentConfigItem, 0, len(grants))
	seen := map[string]struct{}{}
	for _, grant := range grants {
		key := grant.ResourceId + ":" + grant.ResourceVersion
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item, ok, err := platformAgentConfigItem(grant)
		if err != nil {
			return dtoaionui.AgentConfigsResponse{}, err
		}
		if ok {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].Id < items[j].Id
	})
	return dtoaionui.AgentConfigsResponse{
		UserEmail: strings.TrimSpace(strings.ToLower(email)),
		Revision:  s.now().UTC().Format(time.RFC3339),
		Agents:    items,
	}, nil
}

type grantSubject struct {
	Type string
	Id   string
}

func platformGrantSubjects(userID int) ([]grantSubject, error) {
	subjects := []grantSubject{{Type: apmodel.GrantSubjectTypeUser, Id: strconv.Itoa(userID)}}
	var memberships []entmodel.UserDepartment
	if err := model.DB.Where("user_id = ? AND status = ?", userID, constant.EnterpriseMembershipStatusActive).Find(&memberships).Error; err != nil {
		return nil, err
	}
	departmentIDs := make(map[int]struct{}, len(memberships))
	for _, membership := range memberships {
		if membership.DepartmentId > 0 {
			departmentIDs[membership.DepartmentId] = struct{}{}
		}
	}
	parentIDs := make([]int, 0, len(departmentIDs))
	for departmentID := range departmentIDs {
		parentIDs = append(parentIDs, departmentID)
	}
	for len(parentIDs) > 0 {
		var departments []entmodel.Department
		if err := model.DB.Select("id", "parent_id").Where("id IN ?", parentIDs).Find(&departments).Error; err != nil {
			return nil, err
		}
		parentIDs = parentIDs[:0]
		for _, department := range departments {
			if department.ParentId == nil || *department.ParentId <= 0 {
				continue
			}
			if _, exists := departmentIDs[*department.ParentId]; exists {
				continue
			}
			departmentIDs[*department.ParentId] = struct{}{}
			parentIDs = append(parentIDs, *department.ParentId)
		}
	}
	for departmentID := range departmentIDs {
		subjects = append(subjects, grantSubject{Type: apmodel.GrantSubjectTypeDepartment, Id: strconv.Itoa(departmentID)})
	}
	return subjects, nil
}

func platformGrantSubjectWhere(subjects []grantSubject) (string, []any) {
	clauses := make([]string, 0, len(subjects))
	args := make([]any, 0, len(subjects)*2)
	for _, subject := range subjects {
		clauses = append(clauses, "(subject_type = ? AND subject_id = ?)")
		args = append(args, subject.Type, subject.Id)
	}
	return strings.Join(clauses, " OR "), args
}

func platformAgentConfigItem(grant apmodel.ResourceGrant) (dtoaionui.AgentConfigItem, bool, error) {
	var resource apmodel.Resource
	if err := model.DB.Where("resource_id = ? AND resource_type = ? AND status = ?", grant.ResourceId, apmodel.ResourceTypeAgent, apmodel.ResourceStatusPublished).First(&resource).Error; err != nil {
		return dtoaionui.AgentConfigItem{}, false, nil
	}
	var version apmodel.ResourceVersion
	if err := model.DB.Where("resource_id = ? AND version = ? AND status = ?", grant.ResourceId, grant.ResourceVersion, apmodel.ResourceStatusPublished).First(&version).Error; err != nil {
		return dtoaionui.AgentConfigItem{}, false, nil
	}
	packagePath := strings.TrimSpace(version.PackagePath)
	sha := strings.TrimSpace(version.PackageSha256)
	var def apmodel.AgentDef
	if err := model.DB.Where("resource_id = ? AND resource_version = ?", grant.ResourceId, grant.ResourceVersion).First(&def).Error; err != nil {
		return dtoaionui.AgentConfigItem{}, false, nil
	}
	cliType := strings.TrimSpace(strings.ToLower(def.CliType))
	if cliType != CliTypeOpenCode && cliType != CliTypeCodex {
		return dtoaionui.AgentConfigItem{}, false, nil
	}
	if packagePath == "" {
		packagePath = strings.TrimSpace(def.PackagePath)
	}
	if sha == "" {
		sha = strings.TrimSpace(def.PackageSha256)
	}
	if packagePath == "" {
		return dtoaionui.AgentConfigItem{}, false, nil
	}
	ref, err := apservice.ParseArtifactURI(packagePath)
	if err != nil {
		return dtoaionui.AgentConfigItem{}, false, err
	}
	store, err := apservice.DefaultArtifactStore()
	if err != nil {
		return dtoaionui.AgentConfigItem{}, false, err
	}
	signed, err := store.PresignGet(context.Background(), ref, apservice.ArtifactPresignExpiresForAionUI())
	if err != nil {
		return dtoaionui.AgentConfigItem{}, false, err
	}
	name := strings.TrimSpace(def.Name)
	if name == "" {
		name = resource.DisplayName
	}
	description := strings.TrimSpace(def.Description)
	if description == "" {
		description = resource.Description
	}
	avatar := strings.TrimSpace(def.Avatar)
	if avatar == "" {
		avatar = resource.Avatar
	}
	if resolvedAvatar, err := apservice.ResolveAgentAvatarURL(context.Background(), avatar, apservice.ArtifactPresignExpiresForAionUI()); err == nil && strings.TrimSpace(resolvedAvatar) != "" {
		avatar = resolvedAvatar
	}
	if avatar == "" {
		avatar = defaultAgentAvatar
	}
	return dtoaionui.AgentConfigItem{
		Id:                 resource.ResourceId,
		CliType:            cliType,
		ArtifactKey:        ref.URI,
		Url:                signed.URL,
		UrlType:            signed.URLType,
		UrlExpiresAt:       signed.ExpiresAt,
		Version:            version.Version,
		Name:               name,
		Description:        description,
		Categories:         def.Categories(),
		RecommendedPrompts: def.RecommendedPrompts(),
		Avatar:             avatar,
		Sha256:             sha,
		Size:               version.PackageSize,
	}, true, nil
}
