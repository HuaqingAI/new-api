package aionui

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestClientPackageUploadPublishesWindowsPackageWithMetadataFeed(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&apmodel.ClientPackage{}))
	store := newFakeClientPackageStore()
	restore := apservice.SetArtifactStoreForTest(store)
	t.Cleanup(restore)

	installer := "windows installer"
	installerSha512 := testClientPackageSha512(installer)
	metadata := fmt.Sprintf(`version: 2.1.42
files:
  - url: AionUi-2.1.42-win-x64.exe
    sha512: %s
    size: %d
path: AionUi-2.1.42-win-x64.exe
sha512: %s
releaseDate: '2026-08-05T10:00:00.000Z'
`, installerSha512, len(installer), installerSha512)

	service := NewClientPackageService(db)
	service.now = func() time.Time {
		return time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	}

	item, err := service.Upload(ClientPackageUploadInput{
		Platform:               apmodel.ClientPackagePlatformWindowsX64,
		Version:                "2.1.42",
		FileName:               "AionUi-2.1.42-win-x64.exe",
		File:                   strings.NewReader(installer),
		UpdateMetadataFileName: "latest.yml",
		UpdateMetadataFile:     strings.NewReader(metadata),
		Publish:                true,
		ActorUserId:            7,
	})

	require.NoError(t, err)
	require.Equal(t, apmodel.ClientPackageStatusPublished, item.Status)
	require.Equal(t, installerSha512, item.FileSha512)
	require.Equal(t, int64(len(installer)), item.FileSize)
	require.NotNil(t, item.PublishedAt)
	require.Equal(t, "2026-08-05 10:00:00", *item.PublishedAt)
	require.Len(t, store.files, 2)

	latest, err := service.ListLatest()
	require.NoError(t, err)
	require.True(t, latest.Items[0].Available)
	require.Equal(t, "2.1.42", latest.Items[0].Release.Version)

	feed, ok, err := service.UpdateFeed("latest.yml")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "AionUi-2.1.42-win-x64.exe", feed.Path)
	require.Equal(t, installerSha512, feed.Sha512)
	require.Equal(t, int64(len(installer)), feed.Size)
}

func TestClientPackageUploadDraftValidatesMetadataImmediately(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&apmodel.ClientPackage{}))
	store := newFakeClientPackageStore()
	restore := apservice.SetArtifactStoreForTest(store)
	t.Cleanup(restore)

	installer := "windows installer"
	draftMetadata := `version: 9.9.9
files:
  - url: unexpected.exe
    sha512: draft-sha512-value
    size: 1
path: unexpected.exe
sha512: draft-sha512-value
releaseDate: '2026-08-05T10:00:00.000Z'
`
	service := NewClientPackageService(db)
	_, err = service.Upload(ClientPackageUploadInput{
		Platform:               apmodel.ClientPackagePlatformWindowsX64,
		Version:                "2.1.42",
		FileName:               "AionUi-2.1.42-win-x64.exe",
		File:                   strings.NewReader(installer),
		UpdateMetadataFileName: "latest.yml",
		UpdateMetadataFile:     strings.NewReader(draftMetadata),
		Publish:                false,
		ActorUserId:            7,
	})

	require.ErrorIs(t, err, ErrClientPackageInvalidInput)
	require.Contains(t, err.Error(), "version 不匹配")
	require.Empty(t, store.files)
}

func TestClientPackagePublishesDraftAfterValidatingStoredMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&apmodel.ClientPackage{}))
	store := newFakeClientPackageStore()
	restore := apservice.SetArtifactStoreForTest(store)
	t.Cleanup(restore)

	installer := "windows installer"
	installerSha512 := testClientPackageSha512(installer)
	metadata := fmt.Sprintf(`version: 2.1.42
files:
  - url: AionUi-2.1.42-win-x64.exe
    sha512: %s
    size: %d
path: AionUi-2.1.42-win-x64.exe
sha512: %s
releaseDate: '2026-08-05T10:00:00.000Z'
`, installerSha512, len(installer), installerSha512)
	service := NewClientPackageService(db)
	service.now = func() time.Time {
		return time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	}
	item, err := service.Upload(ClientPackageUploadInput{
		Platform:               apmodel.ClientPackagePlatformWindowsX64,
		Version:                "2.1.42",
		FileName:               "AionUi-2.1.42-win-x64.exe",
		File:                   strings.NewReader(installer),
		UpdateMetadataFileName: "latest.yml",
		UpdateMetadataFile:     strings.NewReader(metadata),
		Publish:                false,
		ActorUserId:            7,
	})
	require.NoError(t, err)

	published, err := service.UpdateStatus(item.Id, apmodel.ClientPackageStatusPublished)

	require.NoError(t, err)
	require.Equal(t, apmodel.ClientPackageStatusPublished, published.Status)
	require.Equal(t, installerSha512, published.FileSha512)
	require.Equal(t, int64(len(installer)), published.FileSize)
	require.NotNil(t, published.PublishedAt)
}

func TestClientPackageUploadAcceptsElectronBuilderMetadataWithoutSize(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&apmodel.ClientPackage{}))
	store := newFakeClientPackageStore()
	restore := apservice.SetArtifactStoreForTest(store)
	t.Cleanup(restore)

	installer := "windows installer"
	installerSha512 := testClientPackageSha512(installer)
	metadata := fmt.Sprintf(`version: 2.1.42
files:
  - url: AionUi-2.1.42-win-x64.exe
    sha512: %s
path: AionUi-2.1.42-win-x64.exe
sha512: %s
releaseDate: '2026-08-05T10:00:00.000Z'
`, installerSha512, installerSha512)

	service := NewClientPackageService(db)
	item, err := service.Upload(ClientPackageUploadInput{
		Platform:               apmodel.ClientPackagePlatformWindowsX64,
		Version:                "2.1.42",
		FileName:               "AionUi-2.1.42-win-x64.exe",
		File:                   strings.NewReader(installer),
		UpdateMetadataFileName: "latest.yml",
		UpdateMetadataFile:     strings.NewReader(metadata),
		Publish:                false,
		ActorUserId:            7,
	})

	require.NoError(t, err)
	require.Equal(t, installerSha512, item.FileSha512)
	require.Equal(t, int64(len(installer)), item.FileSize)
}

func TestClientPackageUploadReturnsSpecificMetadataMismatchError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&apmodel.ClientPackage{}))
	store := newFakeClientPackageStore()
	restore := apservice.SetArtifactStoreForTest(store)
	t.Cleanup(restore)

	installer := "windows installer"
	metadata := fmt.Sprintf(`version: 2.1.42
files:
  - url: AionUi-2.1.42-win-x64.exe
    sha512: wrong-sha512-value
    size: %d
path: AionUi-2.1.42-win-x64.exe
sha512: wrong-sha512-value
releaseDate: '2026-08-05T10:00:00.000Z'
`, len(installer))

	service := NewClientPackageService(db)
	_, err = service.Upload(ClientPackageUploadInput{
		Platform:               apmodel.ClientPackagePlatformWindowsX64,
		Version:                "2.1.42",
		FileName:               "AionUi-2.1.42-win-x64.exe",
		File:                   strings.NewReader(installer),
		UpdateMetadataFileName: "latest.yml",
		UpdateMetadataFile:     strings.NewReader(metadata),
		Publish:                false,
		ActorUserId:            7,
	})

	var validationErr ClientPackageValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Contains(t, validationErr.Message, "sha512 不匹配")
	require.Empty(t, store.files)
}

func testClientPackageSha512(content string) string {
	sum := sha512.Sum512([]byte(content))
	return base64.StdEncoding.EncodeToString(sum[:])
}

type fakeClientPackageStore struct {
	bucket string
	files  map[string][]byte
}

func newFakeClientPackageStore() *fakeClientPackageStore {
	return &fakeClientPackageStore{bucket: "test-bucket", files: map[string][]byte{}}
}

func (s *fakeClientPackageStore) PutFile(_ context.Context, input apservice.PutArtifactInput) (apservice.ArtifactRef, error) {
	data, err := os.ReadFile(input.LocalPath)
	if err != nil {
		return apservice.ArtifactRef{}, err
	}
	key := strings.Trim(input.BucketKey, "/")
	s.files[key] = append([]byte(nil), data...)
	return apservice.ArtifactRef{
		URI:    "oss://" + s.bucket + "/" + key,
		Bucket: s.bucket,
		Key:    key,
		Sha256: input.Sha256,
		Size:   input.SizeBytes,
	}, nil
}

func (s *fakeClientPackageStore) PresignGet(_ context.Context, ref apservice.ArtifactRef, expires time.Duration) (apservice.PresignedArtifact, error) {
	if ref.Key == "" {
		parsed, err := apservice.ParseArtifactURI(ref.URI)
		if err != nil {
			return apservice.PresignedArtifact{}, err
		}
		ref = parsed
	}
	return apservice.PresignedArtifact{
		URL:       "https://oss.test/" + ref.Key,
		URLType:   "https",
		ExpiresAt: time.Now().Add(expires).Unix(),
	}, nil
}

func (s *fakeClientPackageStore) DownloadToFile(_ context.Context, ref apservice.ArtifactRef, localPath string) error {
	if ref.Key == "" {
		parsed, err := apservice.ParseArtifactURI(ref.URI)
		if err != nil {
			return err
		}
		ref = parsed
	}
	data, ok := s.files[ref.Key]
	if !ok {
		return os.ErrNotExist
	}
	return os.WriteFile(localPath, data, 0o600)
}

func (s *fakeClientPackageStore) Delete(_ context.Context, ref apservice.ArtifactRef) error {
	if ref.Key == "" {
		parsed, err := apservice.ParseArtifactURI(ref.URI)
		if err != nil {
			return err
		}
		ref = parsed
	}
	delete(s.files, ref.Key)
	return nil
}
