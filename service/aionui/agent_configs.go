package aionui

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
)

const (
	CliTypeOpenCode        = "opencode"
	AgentConfigUrlTypeFile = "file"
	defaultAgentAvatar     = "\U0001f916"
)

var ErrUnsupportedCliType = errors.New("unsupported cli_type")

type AgentConfigService struct {
	now func() time.Time
}

func NewAgentConfigService(_ ...string) *AgentConfigService {
	return &AgentConfigService{now: time.Now}
}

func NewDefaultAgentConfigService() *AgentConfigService {
	return NewAgentConfigService()
}

func ValidateAgentConfigRequest(cliType string) error {
	if strings.TrimSpace(cliType) == "" || cliType == CliTypeOpenCode {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedCliType, cliType)
}

func (s *AgentConfigService) List(email string, cliType string) (dtoaionui.AgentConfigsResponse, error) {
	return s.listPlatform(0, email, cliType)
}

func (s *AgentConfigService) ListForUser(userID int, email string, cliType string) (dtoaionui.AgentConfigsResponse, error) {
	return s.listPlatform(userID, email, cliType)
}

func (s *AgentConfigService) listPlatform(userID int, email string, cliType string) (dtoaionui.AgentConfigsResponse, error) {
	if strings.TrimSpace(cliType) == "" {
		cliType = CliTypeOpenCode
	}
	if cliType != CliTypeOpenCode {
		return dtoaionui.AgentConfigsResponse{}, ErrUnsupportedCliType
	}
	if model.DB == nil || userID <= 0 {
		return dtoaionui.AgentConfigsResponse{
			UserEmail: strings.TrimSpace(strings.ToLower(email)),
			CliType:   CliTypeOpenCode,
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
			CliType:   CliTypeOpenCode,
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
		CliType:   CliTypeOpenCode,
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
	for _, membership := range memberships {
		subjects = append(subjects, grantSubject{Type: apmodel.GrantSubjectTypeDepartment, Id: strconv.Itoa(membership.DepartmentId)})
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
	if packagePath == "" {
		packagePath = strings.TrimSpace(def.PackagePath)
	}
	if sha == "" {
		sha = strings.TrimSpace(def.PackageSha256)
	}
	if packagePath == "" {
		return dtoaionui.AgentConfigItem{}, false, nil
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
	if avatar == "" {
		avatar = defaultAgentAvatar
	}
	return dtoaionui.AgentConfigItem{
		Id:          resource.ResourceId,
		Url:         fileURL(packagePath),
		UrlType:     AgentConfigUrlTypeFile,
		Version:     version.Version,
		Name:        name,
		Description: description,
		Avatar:      avatar,
		Sha256:      sha,
	}, true, nil
}

func fileURL(filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if strings.HasPrefix(strings.ToLower(filePath), "file://") {
		return filePath
	}
	if abs, err := filepath.Abs(filePath); err == nil {
		filePath = abs
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(filePath)}
	return u.String()
}
