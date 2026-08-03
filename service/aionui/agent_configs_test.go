package aionui

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

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
	restore := apservice.SetArtifactStoreForTest(fakeAionUIArtifactStore{})
	t.Cleanup(restore)

	packagePath := "oss://test-bucket/agent-packages/opencode/res_agent_1/1.0.0/opencode.zip"
	codexPackagePath := "oss://test-bucket/agent-packages/codex/res_agent_2/1.0.0/codex.zip"
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
	require.NoError(t, db.Create(&apmodel.Resource{
		ResourceId:    "res_agent_2",
		ResourceType:  apmodel.ResourceTypeAgent,
		DisplayName:   "Agent Two",
		OwnerUserId:   1,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      "res_agent_2",
		Version:         "1.0.0",
		ContractVersion: "codex-agent-platform/v1",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       1,
		PackagePath:     codexPackagePath,
		PackageSha256:   "codex-sha",
		PackageSize:     9,
	}).Error)
	require.NoError(t, db.Create(&apmodel.AgentDef{
		ResourceId:      "res_agent_2",
		ResourceVersion: "1.0.0",
		CliType:         "codex",
		Name:            "Agent Def Two",
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceGrant{
		ResourceId:      "res_agent_2",
		ResourceVersion: "1.0.0",
		SubjectType:     apmodel.GrantSubjectTypeUser,
		SubjectId:       "101",
		GrantedBy:       1,
	}).Error)

	service := NewAgentConfigService()
	service.now = func() time.Time {
		return time.Date(2026, 7, 29, 1, 2, 3, 0, time.UTC)
	}
	result, err := service.ListForUser(101, "User@Example.com")

	require.NoError(t, err)
	require.Equal(t, "user@example.com", result.UserEmail)
	require.Len(t, result.Agents, 2)
	require.Equal(t, "res_agent_1", result.Agents[0].Id)
	require.Equal(t, "opencode", result.Agents[0].CliType)
	require.Equal(t, "Agent Def One", result.Agents[0].Name)
	require.Equal(t, "from def", result.Agents[0].Description)
	require.Equal(t, "D", result.Agents[0].Avatar)
	require.Equal(t, "1.0.0", result.Agents[0].Version)
	require.Equal(t, "sha", result.Agents[0].Sha256)
	require.Equal(t, "https", result.Agents[0].UrlType)
	require.Equal(t, packagePath, result.Agents[0].ArtifactKey)
	require.Equal(t, "https://oss.test/agent-packages/opencode/res_agent_1/1.0.0/opencode.zip", result.Agents[0].Url)
	require.Equal(t, int64(1780000000), result.Agents[0].UrlExpiresAt)
	require.Equal(t, int64(3), result.Agents[0].Size)
	require.Equal(t, "res_agent_2", result.Agents[1].Id)
	require.Equal(t, "codex", result.Agents[1].CliType)
	require.Equal(t, "Agent Def Two", result.Agents[1].Name)
	require.Equal(t, "codex-sha", result.Agents[1].Sha256)
	require.Equal(t, "https", result.Agents[1].UrlType)
	require.Equal(t, codexPackagePath, result.Agents[1].ArtifactKey)
	require.Equal(t, "https://oss.test/agent-packages/codex/res_agent_2/1.0.0/codex.zip", result.Agents[1].Url)
}

type fakeAionUIArtifactStore struct{}

func (fakeAionUIArtifactStore) PutFile(context.Context, apservice.PutArtifactInput) (apservice.ArtifactRef, error) {
	return apservice.ArtifactRef{}, nil
}

func (fakeAionUIArtifactStore) PresignGet(_ context.Context, ref apservice.ArtifactRef, _ time.Duration) (apservice.PresignedArtifact, error) {
	return apservice.PresignedArtifact{
		URL:       "https://oss.test/" + ref.Key,
		URLType:   apservice.ArtifactURLTypeHTTPS,
		ExpiresAt: 1780000000,
	}, nil
}

func (fakeAionUIArtifactStore) DownloadToFile(context.Context, apservice.ArtifactRef, string) error {
	return nil
}

func (fakeAionUIArtifactStore) Delete(context.Context, apservice.ArtifactRef) error {
	return nil
}
