package agentplatform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreAgentAvatarUploadsToOSS(t *testing.T) {
	store := newFakeArtifactStore()
	restore := SetArtifactStoreForTest(store)
	t.Cleanup(restore)

	body := []byte("avatar-image")
	avatar, err := StoreAgentAvatar("avatar.png", bytes.NewReader(body))

	require.NoError(t, err)
	sha := sha256.Sum256(body)
	shaText := hex.EncodeToString(sha[:])
	expectedKey := "agent-avatars/" + shaText[:32] + ".png"
	require.Equal(t, "oss://test-bucket/"+expectedKey, avatar.URI)
	require.Equal(t, "https://oss.test/"+expectedKey, avatar.URL)
	require.Equal(t, ArtifactURLTypeHTTPS, avatar.URLType)
	require.Positive(t, avatar.URLExpiresAt)
	require.Equal(t, body, store.files[expectedKey])
}

func TestStoreAgentAvatarRejectsInvalidInput(t *testing.T) {
	store := newFakeArtifactStore()
	restore := SetArtifactStoreForTest(store)
	t.Cleanup(restore)

	_, err := StoreAgentAvatar("avatar.txt", strings.NewReader("avatar"))
	require.ErrorIs(t, err, ErrInvalidResourceInput)

	_, err = StoreAgentAvatar("avatar.png", strings.NewReader(""))
	require.ErrorIs(t, err, ErrInvalidResourceInput)

	_, err = StoreAgentAvatar("avatar.png", strings.NewReader(strings.Repeat("x", maxAvatarBytes+1)))
	require.ErrorIs(t, err, ErrInvalidResourceInput)
}
