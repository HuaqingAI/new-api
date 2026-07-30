package agentplatform

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	appmodel "github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAgentPublishGeneratesOpenCodeZip(t *testing.T) {
	db := newAgentPublishTestDB(t)
	root := t.TempDir()
	t.Setenv("AIONUI_AGENT_PACKAGE_DIR", filepath.Join(root, "agents"))
	t.Setenv("AIONUI_SKILL_PACKAGE_DIR", filepath.Join(root, "skills"))
	t.Setenv("AIONUI_SYS_SKILLS_DIR", filepath.Join(root, "sys-skills"))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sys-skills", "cherry-knowledge-search"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sys-skills", "cherry-knowledge-search", "SKILL.md"), []byte("# cherry"), 0o644))
	token := seedAgentModelToken(t, db, 1, "gpt-5.5,gpt-4.1")

	mcp, err := NewMcpService(db).Create(McpCreateInput{
		DisplayName: "Local MCP",
		Config:      []byte(`{"mcpServers":{"demo-local":{"command":"npx","args":["-y","demo"]}}}`),
		OwnerUserId: 1,
	})
	require.NoError(t, err)

	skill, err := NewSkillService(db).Create(SkillCreateInput{DisplayName: "Custom Skill", OwnerUserId: 1})
	require.NoError(t, err)
	skillZip := buildTestSkillZip(t, "custom-skill/SKILL.md", "# custom")
	_, err = NewSkillService(db).SavePackage(skill.ResourceId, "custom-skill.zip", bytes.NewReader(skillZip))
	require.NoError(t, err)

	knowledge, err := NewKnowledgeService(db).Create(KnowledgeCreateInput{
		DisplayName:         "Knowledge",
		ExternalKnowledgeId: "kb-001",
		OwnerUserId:         1,
	})
	require.NoError(t, err)

	agent, err := NewAgentService(db).Create(AgentCreateInput{
		DisplayName:  "Demo Agent",
		Description:  "Agent description",
		Avatar:       "bot",
		Instructions: "Follow team rules.",
		ModelTokenId: token.Id,
		DefaultModel: "gpt-5.5",
		McpIds:       []string{mcp.ResourceId},
		SkillIds:     []string{skill.ResourceId},
		KnowledgeIds: []string{knowledge.ResourceId},
		OwnerUserId:  1,
	})
	require.NoError(t, err)

	result, err := NewAgentPublishService(db).Publish(PublishAgentInput{
		ResourceId:  agent.ResourceId,
		Summary:     "initial",
		ActorUserId: 1,
		Grants:      []PublishAgentGrantInput{{SubjectType: apmodel.GrantSubjectTypeUser, SubjectId: "42"}},
	})
	require.NoError(t, err)
	require.Equal(t, "1.0.0", result.Version)
	require.FileExists(t, filepath.Join(root, "agents", agent.ResourceId, "1.0.0", "opencode.zip"))

	files := readZipFiles(t, filepath.Join(root, "agents", agent.ResourceId, "1.0.0", "opencode.zip"))
	names := readZipEntryNames(t, filepath.Join(root, "agents", agent.ResourceId, "1.0.0", "opencode.zip"))
	require.Contains(t, names, "global/")
	require.NotContains(t, files, "global/opencode.jsonc")
	require.NotContains(t, files, "global/skills/cherry-knowledge-search/SKILL.md")
	require.Contains(t, files, "project/.opencode/skills/cherry-knowledge-search/SKILL.md")
	require.Contains(t, files, "project/.opencode/skills/cherry-knowledge-search/config.json")
	require.Contains(t, files, "project/.opencode/skills/custom-skill/SKILL.md")
	require.Contains(t, files, "project/opencode.jsonc")
	require.Equal(t, "Follow team rules.", string(files["project/instructions.md"]))
	require.Equal(t, openCodeUserContextTemplate, string(files["project/user-context.md"]))

	var projectConfig map[string]any
	require.NoError(t, common.Unmarshal(files["project/opencode.jsonc"], &projectConfig))
	require.Equal(t, "hth/gpt-5.5", projectConfig["model"])
	require.Equal(t, []any{"instructions.md", "user-context.md"}, projectConfig["instructions"])
	provider := projectConfig["provider"].(map[string]any)
	hthProvider := provider["hth"].(map[string]any)
	require.Equal(t, "http://localhost:3000/v1", hthProvider["api"])
	options := hthProvider["options"].(map[string]any)
	require.Equal(t, token.Key, options["apiKey"])
	models := hthProvider["models"].(map[string]any)
	require.Contains(t, models, "gpt-5.5")
	require.Contains(t, models, "gpt-4.1")
	mcpConfig := projectConfig["mcp"].(map[string]any)
	localServer := mcpConfig["demo-local"].(map[string]any)
	require.Equal(t, "local", localServer["type"])
	require.Equal(t, []any{"npx", "-y", "demo"}, localServer["command"])
	require.NotContains(t, localServer, "args")
	require.NotContains(t, localServer, "enabled")

	var knowledgeConfig map[string]any
	require.NoError(t, common.Unmarshal(files["project/.opencode/skills/cherry-knowledge-search/config.json"], &knowledgeConfig))
	require.Equal(t, []any{"kb-001"}, knowledgeConfig["knowledge_base_ids"])
}

func TestAgentPublishDefaultsUseLatestVersionGrantsAndKeepHistory(t *testing.T) {
	db := newAgentPublishTestDB(t)
	root := t.TempDir()
	t.Setenv("AIONUI_AGENT_PACKAGE_DIR", filepath.Join(root, "agents"))
	token := seedAgentModelToken(t, db, 1, "gpt-5.5")

	agent, err := NewAgentService(db).Create(AgentCreateInput{
		DisplayName:  "Demo Agent",
		Instructions: "Follow team rules.",
		ModelTokenId: token.Id,
		DefaultModel: "gpt-5.5",
		OwnerUserId:  1,
	})
	require.NoError(t, err)

	service := NewAgentPublishService(db)
	first, err := service.Publish(PublishAgentInput{
		ResourceId:  agent.ResourceId,
		Summary:     "initial",
		ActorUserId: 1,
		Grants:      []PublishAgentGrantInput{{SubjectType: apmodel.GrantSubjectTypeUser, SubjectId: "42"}},
	})
	require.NoError(t, err)
	require.Equal(t, "1.0.0", first.Version)

	defaults, err := service.PublishDefaults(agent.ResourceId)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", defaults.LatestVersion)
	require.Equal(t, "1.0.1", defaults.NextVersion)
	require.Len(t, defaults.Grants, 1)
	require.Equal(t, apmodel.GrantSubjectTypeUser, defaults.Grants[0].SubjectType)
	require.Equal(t, "42", defaults.Grants[0].SubjectId)

	second, err := service.Publish(PublishAgentInput{
		ResourceId:  agent.ResourceId,
		Summary:     "second",
		ActorUserId: 1,
		Grants: []PublishAgentGrantInput{
			{SubjectType: apmodel.GrantSubjectTypeUser, SubjectId: "42"},
			{SubjectType: apmodel.GrantSubjectTypeDepartment, SubjectId: "9"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "1.0.1", second.Version)

	firstGrants, err := service.ListVersionGrants(agent.ResourceId, "1.0.0")
	require.NoError(t, err)
	require.Len(t, firstGrants, 1)

	secondDefaults, err := service.PublishDefaults(agent.ResourceId)
	require.NoError(t, err)
	require.Equal(t, "1.0.1", secondDefaults.LatestVersion)
	require.Equal(t, "1.0.2", secondDefaults.NextVersion)
	require.Len(t, secondDefaults.Grants, 2)
}

func newAgentPublishTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))
	require.NoError(t, db.AutoMigrate(&appmodel.User{}, &appmodel.Token{}))
	return db
}

func seedAgentModelToken(t *testing.T, db *gorm.DB, userID int, models string) appmodel.Token {
	t.Helper()
	token := appmodel.Token{
		UserId:             userID,
		Key:                "sk-agent-publish-test",
		Status:             common.TokenStatusEnabled,
		Name:               "Agent publish key",
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        -1,
		UnlimitedQuota:     true,
		ModelLimitsEnabled: true,
		ModelLimits:        models,
	}
	require.NoError(t, db.Create(&token).Error)
	return token
}

func buildTestSkillZip(t *testing.T, name string, content string) []byte {
	t.Helper()
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)
	file, err := writer.Create(name)
	require.NoError(t, err)
	_, err = file.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func readZipFiles(t *testing.T, path string) map[string][]byte {
	t.Helper()
	reader, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer reader.Close()
	files := map[string][]byte{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		opened, err := file.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(opened)
		require.NoError(t, err)
		require.NoError(t, opened.Close())
		files[file.Name] = data
	}
	return files
}

func readZipEntryNames(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	reader, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer reader.Close()
	names := map[string]struct{}{}
	for _, file := range reader.File {
		names[file.Name] = struct{}{}
	}
	return names
}
