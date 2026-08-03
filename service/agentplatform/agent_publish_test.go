package agentplatform

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	store := newFakeArtifactStore()
	restore := SetArtifactStoreForTest(store)
	t.Cleanup(restore)
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
		CliType:      apmodel.AgentCliTypeOpenCode,
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
	require.Equal(t, ArtifactURLTypeHTTPS, result.Artifact.UrlType)
	require.True(t, strings.HasPrefix(result.Artifact.ArtifactKey, "oss://test-bucket/agent-packages/opencode/"+agent.ResourceId+"/1.0.0/opencode.zip"))

	files := store.readZipFiles(t, result.Artifact.ArtifactKey)
	names := store.readZipEntryNames(t, result.Artifact.ArtifactKey)
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
	require.Equal(t, "sk-"+token.Key, options["apiKey"])
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

func TestAgentPublishGeneratesCodexZip(t *testing.T) {
	db := newAgentPublishTestDB(t)
	root := t.TempDir()
	store := newFakeArtifactStore()
	restore := SetArtifactStoreForTest(store)
	t.Cleanup(restore)
	t.Setenv("AIONUI_SYS_SKILLS_DIR", filepath.Join(root, "sys-skills"))
	t.Setenv("BACKEND_BASE_URL", "https://hth.huaqing.run/v1/")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sys-skills", "cherry-knowledge-search"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sys-skills", "cherry-knowledge-search", "SKILL.md"), []byte("# cherry"), 0o644))
	token := seedAgentModelToken(t, db, 1, "gpt-5.5,gpt-4.1")

	mcp, err := NewMcpService(db).Create(McpCreateInput{
		DisplayName: "Codex MCP",
		Config:      []byte(`{"mcpServers":{"filesystem":{"type":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-filesystem","."]},"docs":{"type":"streamablehttp","url":"https://example.com/mcp","headers":{"X-Client":"codex"}}}}`),
		OwnerUserId: 1,
	})
	require.NoError(t, err)

	knowledge, err := NewKnowledgeService(db).Create(KnowledgeCreateInput{
		DisplayName:         "Knowledge",
		ExternalKnowledgeId: "kb-001",
		OwnerUserId:         1,
	})
	require.NoError(t, err)

	agent, err := NewAgentService(db).Create(AgentCreateInput{
		CliType:      apmodel.AgentCliTypeCodex,
		DisplayName:  "Codex Agent",
		Instructions: "Follow Codex rules.",
		ModelTokenId: token.Id,
		DefaultModel: "gpt-5.5",
		McpIds:       []string{mcp.ResourceId},
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
	require.Equal(t, apmodel.AgentCliTypeCodex, result.Artifact.CliType)
	require.Equal(t, ArtifactURLTypeHTTPS, result.Artifact.UrlType)
	require.True(t, strings.HasPrefix(result.Artifact.ArtifactKey, "oss://test-bucket/agent-packages/codex/"+agent.ResourceId+"/1.0.0/codex.zip"))

	files := store.readZipFiles(t, result.Artifact.ArtifactKey)
	require.Contains(t, files, "global/config.toml")
	require.Contains(t, files, "global/auth.json")
	require.Contains(t, files, "project/.codex/config.toml")
	require.Contains(t, files, "project/.codex/skills/cherry-knowledge-search/SKILL.md")
	require.Contains(t, files, "project/.codex/skills/cherry-knowledge-search/config.json")
	require.NotContains(t, files, "project/AGENTS.md")
	require.NotContains(t, files, "project/user-context.md")

	var authConfig map[string]string
	require.NoError(t, common.Unmarshal(files["global/auth.json"], &authConfig))
	require.Equal(t, "sk-"+token.Key, authConfig["OPENAI_API_KEY"])

	globalConfig := string(files["global/config.toml"])
	require.Contains(t, globalConfig, "model_provider = \"hth\"")
	require.Contains(t, globalConfig, "model = \"gpt-5.5\"")
	require.Contains(t, globalConfig, "base_url = \"https://hth.huaqing.run/v1\"")
	require.NotContains(t, globalConfig, "[projects.")

	projectConfig := string(files["project/.codex/config.toml"])
	require.Contains(t, projectConfig, "model = \"gpt-5.5\"")
	require.Contains(t, projectConfig, "model_reasoning_effort = \"high\"")
	require.Contains(t, projectConfig, "developer_instructions = \"\"\"")
	require.Contains(t, projectConfig, "Follow Codex rules.")
	require.Contains(t, projectConfig, "姓名：<name>")
	require.Contains(t, projectConfig, "[mcp_servers.filesystem]")
	require.Contains(t, projectConfig, "command = \"npx\"")
	require.Contains(t, projectConfig, "args = [\"-y\", \"@modelcontextprotocol/server-filesystem\", \".\"]")
	require.Contains(t, projectConfig, "[mcp_servers.filesystem]\ncommand = \"npx\"\nargs = [\"-y\", \"@modelcontextprotocol/server-filesystem\", \".\"]\ndefault_tools_approval_mode = \"approve\"")
	require.Contains(t, projectConfig, "[mcp_servers.docs]")
	require.Contains(t, projectConfig, "url = \"https://example.com/mcp\"")
	require.Contains(t, projectConfig, "[mcp_servers.docs]\nurl = \"https://example.com/mcp\"\ndefault_tools_approval_mode = \"approve\"")
	require.Contains(t, projectConfig, "[mcp_servers.docs.http_headers]")
	require.Contains(t, projectConfig, "\"X-Client\" = \"codex\"")

	var publishedDef apmodel.AgentDef
	require.NoError(t, db.Where("resource_id = ? AND resource_version = ?", agent.ResourceId, "1.0.0").First(&publishedDef).Error)
	require.Equal(t, apmodel.AgentCliTypeCodex, publishedDef.CliType)
	require.Equal(t, result.Artifact.ArtifactKey, publishedDef.PackagePath)
	require.Contains(t, publishedDef.ModelConfigJSON, `"cli_type":"codex"`)
}

func TestExportedAgentAPIKeyUsesOpenAIStylePrefix(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string
	}{
		{name: "bare key", key: "agent-key", want: "sk-agent-key"},
		{name: "already prefixed", key: "sk-agent-key", want: "sk-agent-key"},
		{name: "trim spaces", key: " agent-key ", want: "sk-agent-key"},
		{name: "empty", key: "", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, exportedAgentAPIKey(tc.key))
		})
	}
}

func TestAgentPublishDefaultsUseLatestVersionGrantsAndKeepHistory(t *testing.T) {
	db := newAgentPublishTestDB(t)
	restore := SetArtifactStoreForTest(newFakeArtifactStore())
	t.Cleanup(restore)
	token := seedAgentModelToken(t, db, 1, "gpt-5.5")

	agent, err := NewAgentService(db).Create(AgentCreateInput{
		CliType:      apmodel.AgentCliTypeOpenCode,
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
		Key:                "agent-publish-test",
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

type fakeArtifactStore struct {
	bucket string
	files  map[string][]byte
}

func newFakeArtifactStore() *fakeArtifactStore {
	return &fakeArtifactStore{bucket: "test-bucket", files: map[string][]byte{}}
}

func (s *fakeArtifactStore) PutFile(_ context.Context, input PutArtifactInput) (ArtifactRef, error) {
	data, err := os.ReadFile(input.LocalPath)
	if err != nil {
		return ArtifactRef{}, err
	}
	key := strings.Trim(input.BucketKey, "/")
	s.files[key] = append([]byte(nil), data...)
	return ArtifactRef{
		URI:    buildOSSURI(s.bucket, key),
		Bucket: s.bucket,
		Key:    key,
		Sha256: input.Sha256,
		Size:   input.SizeBytes,
	}, nil
}

func (s *fakeArtifactStore) PresignGet(_ context.Context, ref ArtifactRef, expires time.Duration) (PresignedArtifact, error) {
	return PresignedArtifact{
		URL:       "https://oss.test/" + ref.Key,
		URLType:   ArtifactURLTypeHTTPS,
		ExpiresAt: time.Now().Add(expires).Unix(),
	}, nil
}

func (s *fakeArtifactStore) DownloadToFile(_ context.Context, ref ArtifactRef, targetPath string) error {
	key := ref.Key
	if key == "" {
		parsed, err := ParseArtifactURI(ref.URI)
		if err != nil {
			return err
		}
		key = parsed.Key
	}
	data, ok := s.files[key]
	if !ok {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0o644)
}

func (s *fakeArtifactStore) Delete(_ context.Context, ref ArtifactRef) error {
	delete(s.files, ref.Key)
	return nil
}

func (s *fakeArtifactStore) readZipFiles(t *testing.T, uri string) map[string][]byte {
	t.Helper()
	ref, err := ParseArtifactURI(uri)
	require.NoError(t, err)
	data, ok := s.files[ref.Key]
	require.True(t, ok)
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	files := map[string][]byte{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		opened, err := file.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(opened)
		require.NoError(t, err)
		require.NoError(t, opened.Close())
		files[file.Name] = content
	}
	return files
}

func (s *fakeArtifactStore) readZipEntryNames(t *testing.T, uri string) map[string]struct{} {
	t.Helper()
	ref, err := ParseArtifactURI(uri)
	require.NoError(t, err)
	data, ok := s.files[ref.Key]
	require.True(t, ok)
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	names := map[string]struct{}{}
	for _, file := range reader.File {
		names[file.Name] = struct{}{}
	}
	return names
}
