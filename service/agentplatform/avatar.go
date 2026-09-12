package agentplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxAvatarBytes = 2 * 1024 * 1024

type StoredAgentAvatar struct {
	URI          string
	URL          string
	URLType      string
	URLExpiresAt int64
}

func StoreAgentAvatar(fileName string, reader io.Reader) (StoredAgentAvatar, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if reader == nil || fileName == "" {
		return StoredAgentAvatar{}, ErrInvalidResourceInput
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
	default:
		return StoredAgentAvatar{}, ErrInvalidResourceInput
	}
	tmpFile, err := os.CreateTemp("", "agent-avatar-*.tmp")
	if err != nil {
		return StoredAgentAvatar{}, err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	hash := sha256.New()
	size, err := io.Copy(tmpFile, io.TeeReader(io.LimitReader(reader, maxAvatarBytes+1), hash))
	closeErr := tmpFile.Close()
	if err != nil {
		return StoredAgentAvatar{}, err
	}
	if closeErr != nil {
		return StoredAgentAvatar{}, closeErr
	}
	if size <= 0 || size > maxAvatarBytes {
		return StoredAgentAvatar{}, ErrInvalidResourceInput
	}
	sha := hex.EncodeToString(hash.Sum(nil))
	targetName := sha[:32] + ext
	store, err := DefaultArtifactStore()
	if err != nil {
		return StoredAgentAvatar{}, err
	}
	ref, err := store.PutFile(context.Background(), PutArtifactInput{
		Kind:        ArtifactKindAvatar,
		BucketKey:   buildAvatarObjectKey(targetName),
		LocalPath:   tmpPath,
		ContentType: avatarContentType(ext),
		Sha256:      sha,
		SizeBytes:   size,
		Metadata: map[string]string{
			"artifact-kind": ArtifactKindAvatar,
			"sha256":        sha,
		},
	})
	if err != nil {
		return StoredAgentAvatar{}, err
	}
	signed, err := store.PresignGet(context.Background(), ref, artifactPresignExpires())
	if err != nil {
		return StoredAgentAvatar{}, err
	}
	return StoredAgentAvatar{
		URI:          ref.URI,
		URL:          signed.URL,
		URLType:      signed.URLType,
		URLExpiresAt: signed.ExpiresAt,
	}, nil
}

func ResolveAgentAvatarURL(ctx context.Context, avatar string, expires time.Duration) (string, error) {
	avatar = strings.TrimSpace(avatar)
	if avatar == "" || !strings.HasPrefix(avatar, "oss://") {
		return avatar, nil
	}
	ref, err := ParseArtifactURI(avatar)
	if err != nil {
		return "", err
	}
	store, err := DefaultArtifactStore()
	if err != nil {
		return "", err
	}
	signed, err := store.PresignGet(ctx, ref, expires)
	if err != nil {
		return "", err
	}
	return signed.URL, nil
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
	return filepath.Join(".", "agent-avatars")
}

func buildAvatarObjectKey(fileName string) string {
	return strings.Trim(defaultOSSAvatarObjectPrefix(), "/") + "/" + filepath.Base(fileName)
}

func defaultOSSAvatarObjectPrefix() string {
	if configured := strings.TrimSpace(os.Getenv("OSS_AVATAR_PREFIX")); configured != "" {
		return configured
	}
	return defaultOSSAvatarPrefix
}

func avatarContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
