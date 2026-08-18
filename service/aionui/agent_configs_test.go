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

func TestAgentConfigServiceListsAgentsGrantedToParentDepartment(t *testing.T) {
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&apmodel.Resource{},
		&apmodel.ResourceVersion{},
		&apmodel.AgentDef{},
		&apmodel.ResourceGrant{},
		&entmodel.Department{},
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
	avatarPath := "oss://test-bucket/agent-avatars/avatar.png"
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
	firstDef := apmodel.AgentDef{
		ResourceId:      "res_agent_1",
		ResourceVersion: "1.0.0",
		CliType:         "opencode",
		Name:            "Agent Def One",
		Description:     "from def",
		Avatar:          avatarPath,
	}
	require.NoError(t, firstDef.SetCategories([]string{apmodel.AgentCategoryGeneral}))
	require.NoError(t, db.Create(&firstDef).Error)
	parentDepartmentID := 8
	require.NoError(t, db.Create([]entmodel.Department{
		{Id: parentDepartmentID, Name: "信息部"},
		{Id: 9, Name: "AI 部", ParentId: &parentDepartmentID},
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
		SubjectId:       "8",
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
	secondDef := apmodel.AgentDef{
		ResourceId:      "res_agent_2",
		ResourceVersion: "1.0.0",
		CliType:         "codex",
		Name:            "Agent Def Two",
	}
	require.NoError(t, secondDef.SetCategories([]string{apmodel.AgentCategoryGeneral}))
	require.NoError(t, db.Create(&secondDef).Error)
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
	require.Equal(t, "https://oss.test/agent-avatars/avatar.png", result.Agents[0].Avatar)
	require.Equal(t, "1.0.0", result.Agents[0].Version)
	require.Equal(t, "sha", result.Agents[0].Sha256)
	require.Equal(t, "https", result.Agents[0].UrlType)
	require.Equal(t, packagePath, result.Agents[0].ArtifactKey)
	require.Equal(t, "https://oss.test/agent-packages/opencode/res_agent_1/1.0.0/opencode.zip", result.Agents[0].Url)
	require.Equal(t, int64(1780000000), result.Agents[0].UrlExpiresAt)
	require.Equal(t, int64(3), result.Agents[0].Size)
	require.Equal(t, []string{apmodel.AgentCategoryGeneral}, result.Agents[0].Categories)
	require.Empty(t, result.Agents[0].RecommendedPrompts)
	require.Equal(t, "res_agent_2", result.Agents[1].Id)
	require.Equal(t, "codex", result.Agents[1].CliType)
	require.Equal(t, "Agent Def Two", result.Agents[1].Name)
	require.Equal(t, "codex-sha", result.Agents[1].Sha256)
	require.Equal(t, "https", result.Agents[1].UrlType)
	require.Equal(t, codexPackagePath, result.Agents[1].ArtifactKey)
	require.Equal(t, "https://oss.test/agent-packages/codex/res_agent_2/1.0.0/codex.zip", result.Agents[1].Url)
}

func TestAgentConfigServiceUsesIndependentRecommendedPrompts(t *testing.T) {
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&apmodel.Resource{},
		&apmodel.ResourceVersion{},
		&apmodel.AgentDef{},
		&apmodel.ResourceGrant{},
		&entmodel.Department{},
		&entmodel.UserDepartment{},
	))
	model.DB = db
	t.Cleanup(func() {
		model.DB = oldDB
	})
	restore := apservice.SetArtifactStoreForTest(fakeAionUIArtifactStore{})
	t.Cleanup(restore)

	require.NoError(t, db.Create(&apmodel.Resource{
		ResourceId:    "res_agent_3",
		ResourceType:  apmodel.ResourceTypeAgent,
		DisplayName:   "Agent Three",
		OwnerUserId:   1,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "2.0.0",
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      "res_agent_3",
		Version:         "2.0.0",
		ContractVersion: "opencode-agent-platform/v1",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       1,
		PackagePath:     "oss://test-bucket/agent-packages/opencode/res_agent_3/2.0.0/opencode.zip",
		PackageSha256:   "sha3",
		PackageSize:     4,
	}).Error)
	def := apmodel.AgentDef{
		ResourceId:      "res_agent_3",
		ResourceVersion: "2.0.0",
		CliType:         "opencode",
		Name:            "Agent Def Three",
		Description:     "负责售后。\n<open-remark>\n旧的推荐问题\n</open-remark>",
	}
	require.NoError(t, def.SetCategories([]string{apmodel.AgentCategoryOperations, apmodel.AgentCategoryCustomerService}))
	require.NoError(t, def.SetRecommendedPrompts([]string{"独立推荐问题一", "独立推荐问题二"}))
	require.NoError(t, db.Create(&def).Error)
	require.NoError(t, db.Create(&apmodel.ResourceGrant{
		ResourceId:      "res_agent_3",
		ResourceVersion: "2.0.0",
		SubjectType:     apmodel.GrantSubjectTypeUser,
		SubjectId:       "101",
		GrantedBy:       1,
	}).Error)

	service := NewAgentConfigService()
	result, err := service.ListForUser(101, "user@example.com")
	require.NoError(t, err)
	require.Len(t, result.Agents, 1)
	require.Equal(t, []string{apmodel.AgentCategoryOperations, apmodel.AgentCategoryCustomerService}, result.Agents[0].Categories)
	require.Equal(t, []string{"独立推荐问题一", "独立推荐问题二"}, result.Agents[0].RecommendedPrompts)
	require.Contains(t, result.Agents[0].Description, "<open-remark>")
}

type fakeAionUIArtifactStore struct{}

func (fakeAionUIArtifactStore) PutFile(context.Context, apservice.PutArtifactInput) (apservice.ArtifactRef, error) {
	return apservice.ArtifactRef{}, nil
}

func (fakeAionUIArtifactStore) PresignPut(_ context.Context, input apservice.PutArtifactInput, _ time.Duration) (apservice.PresignedArtifact, apservice.ArtifactRef, error) {
	return apservice.PresignedArtifact{
			URL:       "https://oss.test/upload/" + input.BucketKey,
			URLType:   apservice.ArtifactURLTypeHTTPS,
			ExpiresAt: 1780000000,
		}, apservice.ArtifactRef{
			URI:    "oss://test-bucket/" + input.BucketKey,
			Bucket: "test-bucket",
			Key:    input.BucketKey,
			Sha256: input.Sha256,
			Size:   input.SizeBytes,
		}, nil
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
