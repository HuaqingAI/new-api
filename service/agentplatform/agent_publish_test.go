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
	t.Setenv("BACKEND_BASE_URL", "https://hth.huaqing.run/")
	t.Setenv("HTH_AGENT_ALLOWED_MODELS", "gpt-4.1,custom-model")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sys-skills", "cherry-knowledge-search"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sys-skills", "cherry-knowledge-search", "SKILL.md"), []byte("# cherry"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sys-skills", "ziniao-store"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sys-skills", "ziniao-store", "SKILL.md"), []byte("# ziniao store"), 0o644))
	mcp, err := NewMcpService(db).Create(McpCreateInput{
		DisplayName: "Local MCP",
		Config:      []byte(`{"mcpServers":{"demo-local":{"command":"npx","args":["-y","demo"]},"meta-ads":{"type":"streamableHttp","url":"https://mcp.facebook.com/ads","oauth":{"callback_port":51786,"callback_url":"https://glove-throttle-galley.ngrok-free.dev/callback/p3oJPar44nR2","client_id":"920282887328556"}}}}`),
		OwnerUserId: 1,
	})
	require.NoError(t, err)

	skill, err := NewSkillService(db).Create(SkillCreateInput{DisplayName: "Custom Skill", OwnerUserId: 1})
	require.NoError(t, err)
	skillZip := buildTestSkillZipWithFiles(t, map[string]string{
		"custom-skill/SKILL.md":            "# custom",
		"custom-order-reconciler/SKILL.md": "# reconciler",
	})
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
		Categories:   []string{apmodel.AgentCategoryAmazonOperations, apmodel.AgentCategoryCustomerService},
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
	require.Contains(t, files, "project/.opencode/skills/cherry-knowledge-search/SKILL.md")
	require.Contains(t, files, "project/.opencode/skills/ziniao-store/SKILL.md")
	require.Contains(t, files, "project/.opencode/skills/custom-skill/SKILL.md")
	require.Contains(t, files, "project/.opencode/skills/custom-order-reconciler/SKILL.md")
	require.Contains(t, files, "project/opencode.jsonc")
	require.Equal(t, "Follow team rules.", string(files["project/instructions.md"]))
	require.Equal(t, openCodeUserContextTemplate, string(files["project/user-context.md"]))
	require.Contains(t, string(files["project/opencode.jsonc"]), aionUIPersonalAPIKeyPlaceholder)
	require.NotContains(t, string(files["project/opencode.jsonc"]), `\u003chth-personal-apikey\u003e`)
	var projectConfig map[string]any
	require.NoError(t, common.Unmarshal(files["project/opencode.jsonc"], &projectConfig))
	require.Equal(t, "hth/gpt-5.6-terra", projectConfig["model"])
	require.Equal(t, []any{"instructions.md", "user-context.md"}, projectConfig["instructions"])
	permission := projectConfig["permission"].(map[string]any)
	require.Equal(t, "allow", permission["external_directory"])
	provider := projectConfig["provider"].(map[string]any)
	hthProvider := provider["hth"].(map[string]any)
	require.Equal(t, "https://hth.huaqing.run/v1", hthProvider["api"])
	options := hthProvider["options"].(map[string]any)
	require.Equal(t, aionUIPersonalAPIKeyPlaceholder, options["apiKey"])
	models := hthProvider["models"].(map[string]any)
	require.Empty(t, models)
	var publishedDef apmodel.AgentDef
	require.NoError(t, db.Where("resource_id = ? AND resource_version = ?", agent.ResourceId, "1.0.0").First(&publishedDef).Error)
	var modelConfigSnapshot map[string]any
	require.NoError(t, common.UnmarshalJsonStr(publishedDef.ModelConfigJSON, &modelConfigSnapshot))
	require.Equal(t, "aionui_runtime_v1", modelConfigSnapshot["model_source"])
	require.NotContains(t, modelConfigSnapshot, "models")
	mcpConfig := projectConfig["mcp"].(map[string]any)
	localServer := mcpConfig["demo-local"].(map[string]any)
	require.Equal(t, "local", localServer["type"])
	require.Equal(t, []any{"npx", "-y", "demo"}, localServer["command"])
	require.NotContains(t, localServer, "args")
	require.NotContains(t, localServer, "enabled")
	metaAds := mcpConfig["meta-ads"].(map[string]any)
	require.Equal(t, "remote", metaAds["type"])
	require.Equal(t, "https://mcp.facebook.com/ads", metaAds["url"])
	oauth := metaAds["oauth"].(map[string]any)
	require.Equal(t, "920282887328556", oauth["clientId"])
	require.NotContains(t, oauth, "callback_url")
	require.NotContains(t, oauth, "callback_port")
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
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sys-skills", "ziniao-store"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sys-skills", "ziniao-store", "SKILL.md"), []byte("# ziniao store"), 0o644))
	mcp, err := NewMcpService(db).Create(McpCreateInput{
		DisplayName: "Codex MCP",
		Config:      []byte(`{"mcpServers":{"filesystem":{"type":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-filesystem","."]},"docs":{"type":"streamablehttp","url":"https://example.com/mcp","headers":{"X-Client":"codex"},"oauth":{"callback_port":51786,"callback_url":"https://glove-throttle-galley.ngrok-free.dev/callback/p3oJPar44nR2","client_id":"920282887328556"}}}}`),
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
		CliType:            apmodel.AgentCliTypeCodex,
		DisplayName:        "Codex Agent",
		Instructions:       "Follow Codex rules.",
		Categories:         []string{apmodel.AgentCategoryGeneral},
		RecommendedPrompts: []string{"整理资料", "生成摘要"},
		McpIds:             []string{mcp.ResourceId},
		KnowledgeIds:       []string{knowledge.ResourceId},
		OwnerUserId:        1,
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
	require.Contains(t, files, "project/.codex/skills/ziniao-store/SKILL.md")
	require.NotContains(t, files, "project/AGENTS.md")
	require.NotContains(t, files, "project/user-context.md")

	var authConfig map[string]string
	require.NoError(t, common.Unmarshal(files["global/auth.json"], &authConfig))
	require.Equal(t, aionUIPersonalAPIKeyPlaceholder, authConfig["OPENAI_API_KEY"])
	require.Contains(t, string(files["global/auth.json"]), aionUIPersonalAPIKeyPlaceholder)
	require.NotContains(t, string(files["global/auth.json"]), `\u003chth-personal-apikey\u003e`)

	globalConfig := string(files["global/config.toml"])
	require.Contains(t, globalConfig, "model_provider = \"hth\"")
	require.Contains(t, globalConfig, "model = \"gpt-5.6-terra\"")
	require.Contains(t, globalConfig, "base_url = \"https://hth.huaqing.run/v1\"")
	require.NotContains(t, globalConfig, "[projects.")

	projectConfig := string(files["project/.codex/config.toml"])
	require.Contains(t, projectConfig, "model = \"gpt-5.6-terra\"")
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
	require.Contains(t, projectConfig, "[mcp_servers.docs]\nurl = \"https://example.com/mcp\"\nauth = \"oauth\"\ndefault_tools_approval_mode = \"approve\"")
	require.Contains(t, projectConfig, "[mcp_servers.docs.http_headers]")
	require.Contains(t, projectConfig, "\"X-Client\" = \"codex\"")
	require.Contains(t, projectConfig, "auth = \"oauth\"")
	require.Contains(t, projectConfig, "[mcp_servers.docs.oauth]")
	require.Contains(t, projectConfig, "client_id = \"920282887328556\"")
	require.Contains(t, projectConfig, "callback_url = \"https://glove-throttle-galley.ngrok-free.dev/callback/p3oJPar44nR2\"")
	require.Contains(t, projectConfig, "callback_port = 51786")

	var publishedDef apmodel.AgentDef
	require.NoError(t, db.Where("resource_id = ? AND resource_version = ?", agent.ResourceId, "1.0.0").First(&publishedDef).Error)
	require.Equal(t, apmodel.AgentCliTypeCodex, publishedDef.CliType)
	require.Equal(t, []string{apmodel.AgentCategoryGeneral}, publishedDef.Categories())
	require.Equal(t, []string{"整理资料", "生成摘要"}, publishedDef.RecommendedPrompts())
	require.Equal(t, result.Artifact.ArtifactKey, publishedDef.PackagePath)
	require.Contains(t, publishedDef.ModelConfigJSON, `"cli_type":"codex"`)
}

func TestCodexMcpRejectsSSEWithActionableError(t *testing.T) {
	_, err := toCodexMcpServer("events", map[string]any{
		"type": "sse",
		"url":  "https://example.com/mcp",
	})

	require.ErrorIs(t, err, ErrCodexMcpSSEUnsupported)
	require.EqualError(t, err, ErrCodexMcpSSEUnsupported.Error())
}

func TestAgentPublishDefaultsUseLatestVersionGrantsAndKeepHistory(t *testing.T) {
	db := newAgentPublishTestDB(t)
	restore := SetArtifactStoreForTest(newFakeArtifactStore())
	t.Cleanup(restore)

	agent, err := NewAgentService(db).Create(AgentCreateInput{
		CliType:      apmodel.AgentCliTypeOpenCode,
		DisplayName:  "Demo Agent",
		Instructions: "Follow team rules.",
		Categories:   []string{apmodel.AgentCategoryGeneral},
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

func TestSafeExtractSkillZipUsesSingleRootDirectoryWhenZipHasDirectoryEntry(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "refund-order-reconciler.zip")
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)
	_, err := writer.Create("refund-order-reconciler/")
	require.NoError(t, err)
	file, err := writer.Create("refund-order-reconciler/SKILL.md")
	require.NoError(t, err)
	_, err = file.Write([]byte("# refund"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, os.WriteFile(zipPath, buffer.Bytes(), 0o644))

	skillsDir := filepath.Join(root, "skills")
	require.NoError(t, safeExtractSkillZip(zipPath, skillsDir, "1111"))

	require.FileExists(t, filepath.Join(skillsDir, "refund-order-reconciler", "SKILL.md"))
	require.NoFileExists(t, filepath.Join(skillsDir, "1111", "refund-order-reconciler", "SKILL.md"))
}

func TestSafeExtractSkillZipExtractsMultipleRootDirectories(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "shopify-skill.zip")
	zipData := buildTestSkillZipWithFiles(t, map[string]string{
		"shopify-admin/SKILL.md":    "# admin",
		"shopify-customer/SKILL.md": "# customer",
	})
	require.NoError(t, os.WriteFile(zipPath, zipData, 0o644))

	skillsDir := filepath.Join(root, "skills")
	require.NoError(t, safeExtractSkillZip(zipPath, skillsDir, "Shopify Skills"))

	require.FileExists(t, filepath.Join(skillsDir, "shopify-admin", "SKILL.md"))
	require.FileExists(t, filepath.Join(skillsDir, "shopify-customer", "SKILL.md"))
	require.NoFileExists(t, filepath.Join(skillsDir, "shopify-skills", "shopify-admin", "SKILL.md"))
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

func (s *fakeArtifactStore) PresignPut(_ context.Context, input PutArtifactInput, expires time.Duration) (PresignedArtifact, ArtifactRef, error) {
	key := strings.Trim(input.BucketKey, "/")
	return PresignedArtifact{
			URL:       "https://oss.test/upload/" + key,
			URLType:   ArtifactURLTypeHTTPS,
			ExpiresAt: time.Now().Add(expires).Unix(),
		}, ArtifactRef{
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
