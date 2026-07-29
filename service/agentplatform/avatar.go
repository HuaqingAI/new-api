package agentplatform

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxAvatarBytes = 2 * 1024 * 1024

func StoreAgentAvatar(fileName string, reader io.Reader) (string, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if reader == nil || fileName == "" {
		return "", ErrInvalidResourceInput
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
	default:
		return "", ErrInvalidResourceInput
	}
	dir := defaultAgentAvatarDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmpFile, err := os.CreateTemp(dir, "avatar-*.tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	hash := sha256.New()
	size, err := io.Copy(tmpFile, io.TeeReader(io.LimitReader(reader, maxAvatarBytes+1), hash))
	closeErr := tmpFile.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if size <= 0 || size > maxAvatarBytes {
		return "", ErrInvalidResourceInput
	}
	targetName := hex.EncodeToString(hash.Sum(nil))[:32] + ext
	targetPath := filepath.Join(dir, targetName)
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return "", err
	}
	return "/api/agent-platform/assets/avatars/" + targetName, nil
}

func AgentAvatarPath(fileName string) (string, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if fileName == "" || strings.Contains(fileName, "..") {
		return "", ErrInvalidResourceInput
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
	default:
		return "", ErrInvalidResourceInput
	}
	return filepath.Join(defaultAgentAvatarDir(), fileName), nil
}

func defaultAgentAvatarDir() string {
	if configured := strings.TrimSpace(os.Getenv("AIONUI_AGENT_AVATAR_DIR")); configured != "" {
		return configured
	}
	return filepath.Join(defaultAgentPackageDir(), "avatars")
}
