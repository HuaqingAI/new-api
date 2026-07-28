package aionui

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAgentConfigServiceScansZipPackages(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "Demo Agent.zip")
	content := []byte("zip-bytes")
	require.NoError(t, os.WriteFile(zipPath, content, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{
		"agents": [
			{
				"file": "Demo Agent.zip",
				"id": "demo-from-manifest",
				"name": "测试助手1",
				"description": "manifest description",
				"avatar": "🤖",
				"version": "2.0.0"
			}
		]
	}`), 0o600))

	service := NewAgentConfigService(dir)
	service.now = func() time.Time {
		return time.Date(2026, 7, 28, 13, 0, 0, 0, time.UTC)
	}

	result, err := service.List("Alice@Example.COM", CliTypeOpenCode)

	require.NoError(t, err)
	require.Equal(t, "alice@example.com", result.UserEmail)
	require.Equal(t, CliTypeOpenCode, result.CliType)
	require.Equal(t, "2026-07-28T13:00:00Z", result.Revision)
	require.Len(t, result.Agents, 1)
	require.Equal(t, "demo-from-manifest", result.Agents[0].Id)
	require.Equal(t, "测试助手1", result.Agents[0].Name)
	require.Equal(t, "manifest description", result.Agents[0].Description)
	require.Equal(t, "file", result.Agents[0].UrlType)
	require.Equal(t, "2.0.0", result.Agents[0].Version)
	require.Equal(t, "🤖", result.Agents[0].Avatar)
	require.Contains(t, result.Agents[0].Url, "Demo%20Agent.zip")

	sum := sha256.Sum256(content)
	require.Equal(t, hex.EncodeToString(sum[:]), result.Agents[0].Sha256)
}

func TestAgentConfigServiceRejectsUnsupportedCliType(t *testing.T) {
	_, err := NewAgentConfigService(t.TempDir()).List("alice@example.com", "codex")

	require.ErrorIs(t, err, ErrUnsupportedCliType)
}
