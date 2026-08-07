package aionui

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
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

func TestClientPackageDirectUploadPublishesWindowsPackage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&apmodel.ClientPackage{}))
	store := newFakeClientPackageStore()
	restore := apservice.SetArtifactStoreForTest(store)
	t.Cleanup(restore)

	installer := "windows installer"
	installerSha256 := testClientPackageSha256(installer)
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
	metadataSha256 := testClientPackageSha256(metadata)
	metadataSha512 := testClientPackageSha512(metadata)
	service := NewClientPackageService(db)
	service.now = func() time.Time {
		return time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	}

	initResult, err := service.CreateDirectUpload(ClientPackageDirectUploadInitInput{
		Platform: apmodel.ClientPackagePlatformWindowsX64,
		Version:  "2.1.42",
		Publish:  true,
		Files: []ClientPackageDirectArtifactInput{
			{
				Kind:     "download",
				FileName: "AionUi-2.1.42-win-x64.exe",
				Sha256:   installerSha256,
				Sha512:   installerSha512,
				Size:     int64(len(installer)),
			},
			{
				Kind:     "metadata",
				FileName: "latest.yml",
				Sha256:   metadataSha256,
				Sha512:   metadataSha512,
				Size:     int64(len(metadata)),
			},
		},
		ActorUserId: 7,
	})
	require.NoError(t, err)
	require.Len(t, initResult.Files, 2)
	for _, file := range initResult.Files {
		switch file.Kind {
		case "download":
			store.files[file.ObjectKey] = []byte(installer)
		case "metadata":
			store.files[file.ObjectKey] = []byte(metadata)
		}
	}

	item, err := service.CompleteDirectUpload(ClientPackageDirectUploadCompleteInput{
		Platform: apmodel.ClientPackagePlatformWindowsX64,
		Version:  "2.1.42",
		Publish:  true,
		File: ClientPackageDirectArtifactInput{
			Kind:      initResult.Files[0].Kind,
			FileName:  initResult.Files[0].FileName,
			ObjectURI: initResult.Files[0].ObjectURI,
			Sha256:    initResult.Files[0].Sha256,
			Sha512:    initResult.Files[0].Sha512,
			Size:      initResult.Files[0].Size,
		},
		UpdateMetadataFile: ClientPackageDirectArtifactInput{
			Kind:      initResult.Files[1].Kind,
			FileName:  initResult.Files[1].FileName,
			ObjectURI: initResult.Files[1].ObjectURI,
			Sha256:    initResult.Files[1].Sha256,
			Sha512:    initResult.Files[1].Sha512,
			Size:      initResult.Files[1].Size,
		},
		ActorUserId: 7,
	})

	require.NoError(t, err)
	require.Equal(t, apmodel.ClientPackageStatusPublished, item.Status)
	require.Equal(t, installerSha256, item.FileSha256)
	require.Equal(t, installerSha512, item.FileSha512)
	require.Equal(t, metadataSha256, item.UpdateMetadataSha256)
}

func testClientPackageSha256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
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

func (s *fakeClientPackageStore) PresignPut(_ context.Context, input apservice.PutArtifactInput, expires time.Duration) (apservice.PresignedArtifact, apservice.ArtifactRef, error) {
	key := strings.Trim(input.BucketKey, "/")
	return apservice.PresignedArtifact{
			URL:       "https://oss.test/upload/" + key,
			URLType:   "https",
			ExpiresAt: time.Now().Add(expires).Unix(),
		}, apservice.ArtifactRef{
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
