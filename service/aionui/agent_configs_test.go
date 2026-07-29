package aionui

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAgentConfigServiceRejectsUnsupportedCliType(t *testing.T) {
	_, err := NewAgentConfigService().List("alice@example.com", "codex")

	require.ErrorIs(t, err, ErrUnsupportedCliType)
}

func TestAgentConfigServiceListsPlatformGrantedAgents(t *testing.T) {
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&apmodel.Resource{},
		&apmodel.ResourceVersion{},
		&apmodel.AgentDef{},
		&apmodel.ResourceGrant{},
		&entmodel.UserDepartment{},
	))
	model.DB = db
	t.Cleanup(func() {
		model.DB = oldDB
	})

	dir := t.TempDir()
	t.Chdir(dir)
	packagePath := filepath.Join("agents", "opencode.zip")
	absPackagePath, err := filepath.Abs(packagePath)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(packagePath), 0o700))
	require.NoError(t, os.WriteFile(packagePath, []byte("zip"), 0o600))
	require.NoError(t, db.Create(&apmodel.Resource{
		ResourceId:    "res_agent_1",
		ResourceType:  apmodel.ResourceTypeAgent,
		DisplayName:   "Agent One",
		Description:   "from resource",
		Avatar:        "A",
		OwnerUserId:   1,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      "res_agent_1",
		Version:         "1.0.0",
		ContractVersion: "opencode-agent-platform/v1",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       1,
		PackagePath:     packagePath,
		PackageSha256:   "sha",
		PackageSize:     3,
	}).Error)
	require.NoError(t, db.Create(&apmodel.AgentDef{
		ResourceId:      "res_agent_1",
		ResourceVersion: "1.0.0",
		CliType:         "opencode",
		Name:            "Agent Def One",
		Description:     "from def",
		Avatar:          "D",
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		UserId:       101,
		DepartmentId: 9,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceGrant{
		ResourceId:      "res_agent_1",
		ResourceVersion: "1.0.0",
		SubjectType:     apmodel.GrantSubjectTypeDepartment,
		SubjectId:       "9",
		GrantedBy:       1,
	}).Error)

	service := NewAgentConfigService()
	service.now = func() time.Time {
		return time.Date(2026, 7, 29, 1, 2, 3, 0, time.UTC)
	}
	result, err := service.ListForUser(101, "User@Example.com", CliTypeOpenCode)

	require.NoError(t, err)
	require.Equal(t, "user@example.com", result.UserEmail)
	require.Len(t, result.Agents, 1)
	require.Equal(t, "res_agent_1", result.Agents[0].Id)
	require.Equal(t, "Agent Def One", result.Agents[0].Name)
	require.Equal(t, "from def", result.Agents[0].Description)
	require.Equal(t, "D", result.Agents[0].Avatar)
	require.Equal(t, "1.0.0", result.Agents[0].Version)
	require.Equal(t, "sha", result.Agents[0].Sha256)
	require.Equal(t, "file", result.Agents[0].UrlType)
	require.Equal(t, (&url.URL{Scheme: "file", Path: filepath.ToSlash(absPackagePath)}).String(), result.Agents[0].Url)
}
