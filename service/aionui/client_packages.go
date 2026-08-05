package aionui

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const (
	defaultClientPackagePrefix     = "client-packages/aionui"
	defaultClientPackageMaxBytes   = 1 << 30
	clientPackageDisplayTimeLayout = "2006-01-02 15:04:05"
)

var (
	ErrClientPackageInvalidInput       = errors.New("aionui client package input invalid")
	ErrClientPackageNotFound           = errors.New("aionui client package not found")
	ErrClientPackageDuplicateVersion   = errors.New("aionui client package version already exists")
	ErrClientPackageNotPublished       = errors.New("aionui client package is not published")
	ErrClientPackageMetadataRequired   = errors.New("aionui client package update metadata is required")
	ErrClientPackageUpdateFileRequired = errors.New("aionui client package update file is required")
)

type ClientPackageUploadInput struct {
	Platform               string
	Version                string
	ReleaseNote            string
	FileName               string
	File                   io.Reader
	UpdateFileName         string
	UpdateFile             io.Reader
	UpdateMetadataFileName string
	UpdateMetadataFile     io.Reader
	Publish                bool
	ActorUserId            int
}

type ClientPackageQuery struct {
	Platform string
	Status   string
	Page     int
	PageSize int
}

type ClientUpdateFeed struct {
	Version     string
	Path        string
	Sha512      string
	Size        int64
	ReleaseDate time.Time
}

type ClientPackageValidationError struct {
	Message string
	Cause   error
}

func (e ClientPackageValidationError) Error() string {
	return e.Message
}

func (e ClientPackageValidationError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return ErrClientPackageInvalidInput
}

type ClientPackageService struct {
	db    *gorm.DB
	store apservice.ArtifactStore
	now   func() time.Time
}

type storedClientArtifact struct {
	FileName    string
	Path        string
	Sha256      string
	Sha512      string
	Size        int64
	ContentType string
	LocalPath   string
}

type updateMetadataFile struct {
	Version     string               `yaml:"version"`
	Files       []updateMetadataItem `yaml:"files"`
	Path        string               `yaml:"path"`
	Sha512      string               `yaml:"sha512"`
	ReleaseDate string               `yaml:"releaseDate"`
}

type updateMetadataItem struct {
	URL     string `yaml:"url"`
	Sha512  string `yaml:"sha512"`
	Size    int64  `yaml:"size"`
	SizeSet bool   `yaml:"-"`
}

func (item *updateMetadataItem) UnmarshalYAML(node *yaml.Node) error {
	type rawUpdateMetadataItem updateMetadataItem
	var raw rawUpdateMetadataItem
	if err := node.Decode(&raw); err != nil {
		return err
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "size" {
			raw.SizeSet = true
			break
		}
	}
	*item = updateMetadataItem(raw)
	return nil
}

func NewClientPackageService(db *gorm.DB) *ClientPackageService {
	return &ClientPackageService{db: db, now: time.Now}
}

func (s *ClientPackageService) ListLatest() (dtoaionui.ClientPackageLatestResponse, error) {
	items := make([]dtoaionui.ClientPackageLatestItem, 0, 3)
	for _, platform := range clientPackagePlatforms() {
		pkg, ok, err := s.latestPublished(platform, false)
		if err != nil {
			return dtoaionui.ClientPackageLatestResponse{}, err
		}
		item := dtoaionui.ClientPackageLatestItem{Platform: platform, Available: ok}
		if ok {
			mapped := mapClientPackage(pkg)
			item.Release = &mapped
		}
		items = append(items, item)
	}
	return dtoaionui.ClientPackageLatestResponse{Items: items}, nil
}

func (s *ClientPackageService) List(query ClientPackageQuery) (dtoaionui.ClientPackageListResponse, error) {
	if s == nil || s.db == nil {
		return dtoaionui.ClientPackageListResponse{}, ErrClientPackageInvalidInput
	}
	page, pageSize := normalizeClientPackagePage(query.Page, query.PageSize)
	dbQuery := s.db.Model(&apmodel.ClientPackage{})
	if strings.TrimSpace(query.Platform) != "" {
		dbQuery = dbQuery.Where("platform = ?", strings.TrimSpace(query.Platform))
	}
	if strings.TrimSpace(query.Status) != "" {
		dbQuery = dbQuery.Where("status = ?", strings.TrimSpace(query.Status))
	}
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return dtoaionui.ClientPackageListResponse{}, err
	}
	var rows []apmodel.ClientPackage
	if err := dbQuery.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error; err != nil {
		return dtoaionui.ClientPackageListResponse{}, err
	}
	items := make([]dtoaionui.ClientPackageItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapClientPackage(row))
	}
	return dtoaionui.ClientPackageListResponse{
		Items:    items,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *ClientPackageService) Upload(input ClientPackageUploadInput) (dtoaionui.ClientPackageItem, error) {
	input = normalizeClientPackageUploadInput(input)
	if err := validateClientPackageUploadInput(input); err != nil {
		return dtoaionui.ClientPackageItem{}, err
	}
	if s == nil || s.db == nil {
		return dtoaionui.ClientPackageItem{}, ErrClientPackageInvalidInput
	}
	if err := s.ensureStore(); err != nil {
		return dtoaionui.ClientPackageItem{}, err
	}
	var count int64
	if err := s.db.Model(&apmodel.ClientPackage{}).Where("platform = ? AND version = ?", input.Platform, input.Version).Count(&count).Error; err != nil {
		return dtoaionui.ClientPackageItem{}, err
	}
	if count > 0 {
		return dtoaionui.ClientPackageItem{}, ErrClientPackageDuplicateVersion
	}

	uploaded := make([]apservice.ArtifactRef, 0, 3)
	cleanups := []func(){}
	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	fileArtifact, cleanup, err := s.storeUploadArtifact(input.Platform, input.Version, "download", input.FileName, input.File)
	if err != nil {
		return dtoaionui.ClientPackageItem{}, err
	}
	cleanups = append(cleanups, cleanup)
	uploaded = append(uploaded, apservice.ArtifactRef{URI: fileArtifact.Path})

	var updateArtifact storedClientArtifact
	if strings.TrimSpace(input.UpdateFileName) != "" && input.UpdateFile != nil {
		updateArtifact, cleanup, err = s.storeUploadArtifact(input.Platform, input.Version, "update", input.UpdateFileName, input.UpdateFile)
		if err != nil {
			s.deleteUploaded(uploaded)
			return dtoaionui.ClientPackageItem{}, err
		}
		cleanups = append(cleanups, cleanup)
		uploaded = append(uploaded, apservice.ArtifactRef{URI: updateArtifact.Path})
	}

	var metadataArtifact storedClientArtifact
	var metadata updateMetadataFile
	if strings.TrimSpace(input.UpdateMetadataFileName) != "" && input.UpdateMetadataFile != nil {
		metadataArtifact, cleanup, err = s.storeUploadArtifact(input.Platform, input.Version, "metadata", input.UpdateMetadataFileName, input.UpdateMetadataFile)
		if err != nil {
			s.deleteUploaded(uploaded)
			return dtoaionui.ClientPackageItem{}, err
		}
		cleanups = append(cleanups, cleanup)
		uploaded = append(uploaded, apservice.ArtifactRef{URI: metadataArtifact.Path})
		metadata, err = parseUpdateMetadata(metadataArtifact.LocalPath)
		if err != nil {
			s.deleteUploaded(uploaded)
			return dtoaionui.ClientPackageItem{}, err
		}
		if err := applyUpdateMetadata(input.Platform, input.Version, metadata, &fileArtifact, &updateArtifact); err != nil {
			s.deleteUploaded(uploaded)
			return dtoaionui.ClientPackageItem{}, err
		}
	}

	pkg := apmodel.ClientPackage{
		Platform:               input.Platform,
		Version:                input.Version,
		Status:                 apmodel.ClientPackageStatusDraft,
		FileName:               fileArtifact.FileName,
		FilePath:               fileArtifact.Path,
		FileSha256:             fileArtifact.Sha256,
		FileSha512:             fileArtifact.Sha512,
		FileSize:               fileArtifact.Size,
		ContentType:            fileArtifact.ContentType,
		UpdateFileName:         updateArtifact.FileName,
		UpdateFilePath:         updateArtifact.Path,
		UpdateFileSha256:       updateArtifact.Sha256,
		UpdateFileSha512:       updateArtifact.Sha512,
		UpdateFileSize:         updateArtifact.Size,
		UpdateContentType:      updateArtifact.ContentType,
		UpdateMetadataFileName: metadataArtifact.FileName,
		UpdateMetadataFilePath: metadataArtifact.Path,
		UpdateMetadataSha256:   metadataArtifact.Sha256,
		ReleaseNote:            strings.TrimSpace(input.ReleaseNote),
		CreatedBy:              input.ActorUserId,
	}
	if input.Publish {
		if err := validateClientPackagePublishable(pkg); err != nil {
			s.deleteUploaded(uploaded)
			return dtoaionui.ClientPackageItem{}, err
		}
		now := s.now().UTC()
		pkg.Status = apmodel.ClientPackageStatusPublished
		pkg.PublishedAt = &now
	}
	if err := s.db.Create(&pkg).Error; err != nil {
		s.deleteUploaded(uploaded)
		return dtoaionui.ClientPackageItem{}, err
	}
	return mapClientPackage(pkg), nil
}

func (s *ClientPackageService) UpdateStatus(id int, status string) (dtoaionui.ClientPackageItem, error) {
	if s == nil || s.db == nil || id <= 0 {
		return dtoaionui.ClientPackageItem{}, ErrClientPackageInvalidInput
	}
	status = strings.TrimSpace(strings.ToLower(status))
	if !validClientPackageStatus(status) {
		return dtoaionui.ClientPackageItem{}, ErrClientPackageInvalidInput
	}
	var pkg apmodel.ClientPackage
	if err := s.db.First(&pkg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dtoaionui.ClientPackageItem{}, ErrClientPackageNotFound
		}
		return dtoaionui.ClientPackageItem{}, err
	}
	updates := map[string]any{"status": status}
	if status == apmodel.ClientPackageStatusPublished {
		if err := validateClientPackagePublishable(pkg); err != nil {
			return dtoaionui.ClientPackageItem{}, err
		}
		if err := s.applyStoredUpdateMetadata(&pkg); err != nil {
			return dtoaionui.ClientPackageItem{}, err
		}
		now := s.now().UTC()
		updates["published_at"] = &now
		updates["file_sha512"] = pkg.FileSha512
		updates["file_size"] = pkg.FileSize
		updates["update_file_sha512"] = pkg.UpdateFileSha512
		updates["update_file_size"] = pkg.UpdateFileSize
	}
	if err := s.db.Model(&apmodel.ClientPackage{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return dtoaionui.ClientPackageItem{}, err
	}
	if err := s.db.First(&pkg, id).Error; err != nil {
		return dtoaionui.ClientPackageItem{}, err
	}
	return mapClientPackage(pkg), nil
}

func (s *ClientPackageService) Delete(id int) error {
	if s == nil || s.db == nil || id <= 0 {
		return ErrClientPackageInvalidInput
	}
	var pkg apmodel.ClientPackage
	if err := s.db.First(&pkg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrClientPackageNotFound
		}
		return err
	}
	if pkg.Status == apmodel.ClientPackageStatusPublished {
		return ErrClientPackageNotPublished
	}
	if err := s.ensureStore(); err != nil {
		return err
	}
	refs := []apservice.ArtifactRef{
		{URI: pkg.FilePath},
		{URI: pkg.UpdateFilePath},
		{URI: pkg.UpdateMetadataFilePath},
	}
	for _, ref := range refs {
		if strings.TrimSpace(ref.URI) == "" {
			continue
		}
		if err := s.store.Delete(context.Background(), ref); err != nil {
			return err
		}
	}
	return s.db.Delete(&pkg).Error
}

func (s *ClientPackageService) DownloadURL(id int) (string, error) {
	if s == nil || s.db == nil || id <= 0 {
		return "", ErrClientPackageInvalidInput
	}
	var pkg apmodel.ClientPackage
	if err := s.db.First(&pkg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrClientPackageNotFound
		}
		return "", err
	}
	if pkg.Status != apmodel.ClientPackageStatusPublished {
		return "", ErrClientPackageNotPublished
	}
	return s.presign(pkg.FilePath)
}

func (s *ClientPackageService) UpdateFeed(channel string) (ClientUpdateFeed, bool, error) {
	platform, ok := platformForUpdateChannel(channel)
	if !ok {
		return ClientUpdateFeed{}, false, ErrClientPackageNotFound
	}
	pkg, found, err := s.latestPublished(platform, true)
	if err != nil || !found {
		return ClientUpdateFeed{}, false, err
	}
	path := pkg.FileName
	sha512Value := pkg.FileSha512
	size := pkg.FileSize
	if platform != apmodel.ClientPackagePlatformWindowsX64 {
		path = pkg.UpdateFileName
		sha512Value = pkg.UpdateFileSha512
		size = pkg.UpdateFileSize
	}
	releaseDate := pkg.CreatedAt
	if pkg.PublishedAt != nil {
		releaseDate = *pkg.PublishedAt
	}
	return ClientUpdateFeed{
		Version:     pkg.Version,
		Path:        path,
		Sha512:      sha512Value,
		Size:        size,
		ReleaseDate: releaseDate.UTC(),
	}, true, nil
}

func (s *ClientPackageService) UpdateArtifactURL(version string, fileName string) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrClientPackageInvalidInput
	}
	version = strings.TrimSpace(version)
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if version == "" || fileName == "" || strings.Contains(fileName, "..") {
		return "", ErrClientPackageInvalidInput
	}
	var rows []apmodel.ClientPackage
	if err := s.db.Where("version = ? AND status = ?", version, apmodel.ClientPackageStatusPublished).Find(&rows).Error; err != nil {
		return "", err
	}
	for _, row := range rows {
		switch fileName {
		case row.FileName:
			return s.presign(row.FilePath)
		case row.UpdateFileName:
			return s.presign(row.UpdateFilePath)
		}
	}
	return "", ErrClientPackageNotFound
}

func (s *ClientPackageService) ensureStore() error {
	if s.store != nil {
		return nil
	}
	store, err := apservice.DefaultArtifactStore()
	if err != nil {
		return err
	}
	s.store = store
	return nil
}

func (s *ClientPackageService) storeUploadArtifact(platform string, version string, kind string, fileName string, reader io.Reader) (storedClientArtifact, func(), error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if fileName == "" || reader == nil {
		return storedClientArtifact{}, func() {}, ErrClientPackageInvalidInput
	}
	localPath, cleanup, err := materializeClientPackageFile(reader)
	if err != nil {
		return storedClientArtifact{}, cleanup, err
	}
	sha256Value, sha512Value, size, err := hashClientPackageFile(localPath)
	if err != nil {
		cleanup()
		return storedClientArtifact{}, func() {}, err
	}
	maxBytes := clientPackageMaxBytes()
	if size <= 0 || size > maxBytes {
		cleanup()
		return storedClientArtifact{}, func() {}, ErrClientPackageInvalidInput
	}
	ref, err := s.store.PutFile(context.Background(), apservice.PutArtifactInput{
		Kind:        "aionui-client",
		BucketKey:   buildClientPackageObjectKey(platform, version, kind, sha256Value, fileName),
		LocalPath:   localPath,
		ContentType: clientPackageContentType(fileName),
		Sha256:      sha256Value,
		SizeBytes:   size,
		Metadata: map[string]string{
			"platform": platform,
			"version":  version,
			"sha256":   sha256Value,
			"kind":     kind,
		},
	})
	if err != nil {
		cleanup()
		return storedClientArtifact{}, func() {}, err
	}
	return storedClientArtifact{
		FileName:    fileName,
		Path:        ref.URI,
		Sha256:      sha256Value,
		Sha512:      sha512Value,
		Size:        size,
		ContentType: clientPackageContentType(fileName),
		LocalPath:   localPath,
	}, cleanup, nil
}

func (s *ClientPackageService) presign(uri string) (string, error) {
	if err := s.ensureStore(); err != nil {
		return "", err
	}
	ref, err := apservice.ParseArtifactURI(uri)
	if err != nil {
		return "", err
	}
	signed, err := s.store.PresignGet(context.Background(), ref, apservice.ArtifactPresignExpiresForAionUI())
	if err != nil {
		return "", err
	}
	return signed.URL, nil
}

func (s *ClientPackageService) applyStoredUpdateMetadata(pkg *apmodel.ClientPackage) error {
	if pkg == nil {
		return ErrClientPackageInvalidInput
	}
	if err := s.ensureStore(); err != nil {
		return err
	}
	metadataPath, cleanup, err := s.downloadStoredClientPackageFile(pkg.UpdateMetadataFilePath)
	if err != nil {
		return err
	}
	defer cleanup()
	metadata, err := parseUpdateMetadata(metadataPath)
	if err != nil {
		return err
	}
	fileArtifact := storedClientArtifact{
		FileName: pkg.FileName,
		Sha512:   pkg.FileSha512,
		Size:     pkg.FileSize,
	}
	updateArtifact := storedClientArtifact{
		FileName: pkg.UpdateFileName,
		Sha512:   pkg.UpdateFileSha512,
		Size:     pkg.UpdateFileSize,
	}
	if err := applyUpdateMetadata(pkg.Platform, pkg.Version, metadata, &fileArtifact, &updateArtifact); err != nil {
		return err
	}
	pkg.FileSha512 = fileArtifact.Sha512
	pkg.FileSize = fileArtifact.Size
	pkg.UpdateFileSha512 = updateArtifact.Sha512
	pkg.UpdateFileSize = updateArtifact.Size
	return nil
}

func (s *ClientPackageService) downloadStoredClientPackageFile(uri string) (string, func(), error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", func() {}, ErrClientPackageInvalidInput
	}
	ref, err := apservice.ParseArtifactURI(uri)
	if err != nil {
		return "", func() {}, err
	}
	tmp, err := os.CreateTemp("", "aionui-client-package-metadata-*")
	if err != nil {
		return "", func() {}, err
	}
	path := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return "", func() {}, err
	}
	cleanup := func() {
		_ = os.Remove(path)
	}
	if err := s.store.DownloadToFile(context.Background(), ref, path); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return path, cleanup, nil
}

func (s *ClientPackageService) deleteUploaded(refs []apservice.ArtifactRef) {
	if s == nil || s.store == nil {
		return
	}
	for _, ref := range refs {
		if strings.TrimSpace(ref.URI) == "" {
			continue
		}
		if err := s.store.Delete(context.Background(), ref); err != nil {
			common.SysError("delete uploaded aionui client package failed: " + err.Error())
		}
	}
}

func (s *ClientPackageService) latestPublished(platform string, requireUpdate bool) (apmodel.ClientPackage, bool, error) {
	if s == nil || s.db == nil {
		return apmodel.ClientPackage{}, false, ErrClientPackageInvalidInput
	}
	var rows []apmodel.ClientPackage
	query := s.db.Where("platform = ? AND status = ?", platform, apmodel.ClientPackageStatusPublished)
	if requireUpdate {
		query = query.Where("update_metadata_file_path <> ?", "")
		if platform != apmodel.ClientPackagePlatformWindowsX64 {
			query = query.Where("update_file_path <> ?", "")
		}
	}
	if err := query.Find(&rows).Error; err != nil {
		return apmodel.ClientPackage{}, false, err
	}
	if len(rows) == 0 {
		return apmodel.ClientPackage{}, false, nil
	}
	sort.Slice(rows, func(i int, j int) bool {
		cmp := compareClientPackageVersion(rows[i].Version, rows[j].Version)
		if cmp != 0 {
			return cmp > 0
		}
		if rows[i].PublishedAt != nil && rows[j].PublishedAt != nil && !rows[i].PublishedAt.Equal(*rows[j].PublishedAt) {
			return rows[i].PublishedAt.After(*rows[j].PublishedAt)
		}
		return rows[i].Id > rows[j].Id
	})
	return rows[0], true, nil
}

func validateClientPackageUploadInput(input ClientPackageUploadInput) error {
	if !validClientPackagePlatform(input.Platform) {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "客户端平台无效，请选择 Windows x64、macOS Apple Silicon 或 macOS Intel")
	}
	if !validClientPackageVersion(input.Version) {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "版本号格式不正确，请使用 x.y.z 格式，例如 2.1.42")
	}
	if input.ActorUserId <= 0 {
		return ErrClientPackageInvalidInput
	}
	if input.File == nil {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "请上传客户端下载用安装包")
	}
	if !validClientPackageFileExt(input.Platform, input.FileName, false) {
		return clientPackageValidationError(
			ErrClientPackageInvalidInput,
			"安装包文件类型不正确，%s 需要上传 %s 文件",
			clientPackagePlatformLabel(input.Platform),
			expectedClientPackageFileExt(input.Platform, false),
		)
	}
	if strings.TrimSpace(input.UpdateFileName) != "" && !validClientPackageFileExt(input.Platform, input.UpdateFileName, true) {
		return clientPackageValidationError(
			ErrClientPackageInvalidInput,
			"自动更新安装包文件类型不正确，%s 需要上传 %s 文件",
			clientPackagePlatformLabel(input.Platform),
			expectedClientPackageFileExt(input.Platform, true),
		)
	}
	if input.Publish {
		if strings.TrimSpace(input.UpdateMetadataFileName) == "" || input.UpdateMetadataFile == nil {
			return clientPackageValidationError(ErrClientPackageMetadataRequired, "发布时必须上传 electron-builder 产出的 latest*.yml 更新元数据文件")
		}
		if input.Platform != apmodel.ClientPackagePlatformWindowsX64 && (strings.TrimSpace(input.UpdateFileName) == "" || input.UpdateFile == nil) {
			return clientPackageValidationError(ErrClientPackageUpdateFileRequired, "发布 macOS 客户端时必须同时上传 electron-builder 产出的 .zip 自动更新包")
		}
	}
	if strings.TrimSpace(input.UpdateMetadataFileName) != "" && !validUpdateMetadataFileName(input.Platform, input.UpdateMetadataFileName) {
		return clientPackageValidationError(
			ErrClientPackageInvalidInput,
			"更新元数据文件名不正确，%s 需要上传 %s",
			clientPackagePlatformLabel(input.Platform),
			expectedUpdateMetadataFileName(input.Platform),
		)
	}
	return nil
}

func normalizeClientPackageUploadInput(input ClientPackageUploadInput) ClientPackageUploadInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.Version = strings.TrimSpace(input.Version)
	input.ReleaseNote = strings.TrimSpace(input.ReleaseNote)
	input.FileName = safeClientPackageBaseName(input.FileName)
	input.UpdateFileName = safeClientPackageBaseName(input.UpdateFileName)
	input.UpdateMetadataFileName = safeClientPackageBaseName(input.UpdateMetadataFileName)
	return input
}

func safeClientPackageBaseName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return filepath.Base(value)
}

func validateClientPackagePublishable(pkg apmodel.ClientPackage) error {
	if strings.TrimSpace(pkg.UpdateMetadataFilePath) == "" || strings.TrimSpace(pkg.UpdateMetadataFileName) == "" {
		return clientPackageValidationError(ErrClientPackageMetadataRequired, "发布时必须上传 electron-builder 产出的 latest*.yml 更新元数据文件")
	}
	if pkg.Platform != apmodel.ClientPackagePlatformWindowsX64 && strings.TrimSpace(pkg.UpdateFilePath) == "" {
		return clientPackageValidationError(ErrClientPackageUpdateFileRequired, "发布 macOS 客户端时必须同时上传 electron-builder 产出的 .zip 自动更新包")
	}
	return nil
}

func validClientPackagePlatform(platform string) bool {
	switch platform {
	case apmodel.ClientPackagePlatformWindowsX64, apmodel.ClientPackagePlatformMacArm64, apmodel.ClientPackagePlatformMacX64:
		return true
	default:
		return false
	}
}

func validClientPackageStatus(status string) bool {
	switch status {
	case apmodel.ClientPackageStatusDraft, apmodel.ClientPackageStatusPublished, apmodel.ClientPackageStatusDisabled:
		return true
	default:
		return false
	}
}

func validClientPackageVersion(version string) bool {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func validClientPackageFileExt(platform string, fileName string, update bool) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	if update && platform != apmodel.ClientPackagePlatformWindowsX64 {
		return ext == ".zip"
	}
	if platform == apmodel.ClientPackagePlatformWindowsX64 {
		return ext == ".exe"
	}
	return ext == ".dmg"
}

func expectedClientPackageFileExt(platform string, update bool) string {
	if update && platform != apmodel.ClientPackagePlatformWindowsX64 {
		return ".zip"
	}
	if platform == apmodel.ClientPackagePlatformWindowsX64 {
		return ".exe"
	}
	return ".dmg"
}

func validUpdateMetadataFileName(platform string, fileName string) bool {
	base := filepath.Base(strings.TrimSpace(fileName))
	return base == expectedUpdateMetadataFileName(platform)
}

func expectedUpdateMetadataFileName(platform string) string {
	switch platform {
	case apmodel.ClientPackagePlatformWindowsX64:
		return "latest.yml"
	case apmodel.ClientPackagePlatformMacX64:
		return "latest-mac.yml"
	case apmodel.ClientPackagePlatformMacArm64:
		return "latest-arm64-mac.yml"
	default:
		return "latest*.yml"
	}
}

func clientPackagePlatformLabel(platform string) string {
	switch platform {
	case apmodel.ClientPackagePlatformWindowsX64:
		return "Windows x64"
	case apmodel.ClientPackagePlatformMacArm64:
		return "macOS Apple Silicon"
	case apmodel.ClientPackagePlatformMacX64:
		return "macOS Intel"
	default:
		return platform
	}
}

func clientPackagePlatforms() []string {
	return []string{
		apmodel.ClientPackagePlatformWindowsX64,
		apmodel.ClientPackagePlatformMacArm64,
		apmodel.ClientPackagePlatformMacX64,
	}
}

func platformForUpdateChannel(channel string) (string, bool) {
	switch strings.TrimSpace(channel) {
	case "latest.yml":
		return apmodel.ClientPackagePlatformWindowsX64, true
	case "latest-mac.yml":
		return apmodel.ClientPackagePlatformMacX64, true
	case "latest-arm64-mac.yml":
		return apmodel.ClientPackagePlatformMacArm64, true
	default:
		return "", false
	}
}

func materializeClientPackageFile(reader io.Reader) (string, func(), error) {
	tmp, err := os.CreateTemp("", "aionui-client-package-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		_ = os.Remove(tmp.Name())
	}
	if _, err := io.Copy(tmp, reader); err != nil {
		_ = tmp.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tmp.Name(), cleanup, nil
}

func hashClientPackageFile(path string) (string, string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", 0, err
	}
	defer file.Close()
	sha256Hash := sha256.New()
	sha512Hash := sha512.New()
	size, err := io.Copy(io.MultiWriter(sha256Hash, sha512Hash), file)
	if err != nil {
		return "", "", 0, err
	}
	return hex.EncodeToString(sha256Hash.Sum(nil)), base64.StdEncoding.EncodeToString(sha512Hash.Sum(nil)), size, nil
}

func buildClientPackageObjectKey(platform string, version string, kind string, sha256Value string, fileName string) string {
	prefix := strings.Trim(strings.TrimSpace(os.Getenv("OSS_AIONUI_CLIENT_PREFIX")), "/")
	if prefix == "" {
		prefix = defaultClientPackagePrefix
	}
	hashPrefix := sha256Value
	if len(hashPrefix) > 12 {
		hashPrefix = hashPrefix[:12]
	}
	return strings.Join([]string{
		prefix,
		strings.TrimSpace(platform),
		strings.TrimSpace(version),
		strings.TrimSpace(kind),
		hashPrefix + "-" + filepath.Base(fileName),
	}, "/")
}

func clientPackageContentType(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".exe":
		return "application/vnd.microsoft.portable-executable"
	case ".dmg":
		return "application/x-apple-diskimage"
	case ".zip":
		return "application/zip"
	case ".yml", ".yaml":
		return "text/yaml; charset=utf-8"
	default:
		if value := mime.TypeByExtension(filepath.Ext(fileName)); value != "" {
			return value
		}
		return "application/octet-stream"
	}
}

func clientPackageMaxBytes() int64 {
	value := strings.TrimSpace(os.Getenv("AIONUI_CLIENT_PACKAGE_MAX_BYTES"))
	if value == "" {
		return defaultClientPackageMaxBytes
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return defaultClientPackageMaxBytes
	}
	return parsed
}

func clientPackageValidationError(cause error, format string, args ...any) error {
	return ClientPackageValidationError{
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

func parseUpdateMetadata(path string) (updateMetadataFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return updateMetadataFile{}, err
	}
	var metadata updateMetadataFile
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return updateMetadataFile{}, clientPackageValidationError(ErrClientPackageInvalidInput, "更新元数据 YAML 解析失败，请上传 electron-builder 产出的 latest*.yml 文件")
	}
	if strings.TrimSpace(metadata.Version) == "" {
		return updateMetadataFile{}, clientPackageValidationError(ErrClientPackageInvalidInput, "更新元数据缺少 version 字段")
	}
	return metadata, nil
}

func applyUpdateMetadata(platform string, version string, metadata updateMetadataFile, fileArtifact *storedClientArtifact, updateArtifact *storedClientArtifact) error {
	metadataVersion := strings.TrimSpace(metadata.Version)
	if metadataVersion != version {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "更新元数据 version 不匹配：页面填写 %s，latest.yml 中为 %s", version, metadataVersion)
	}
	target := fileArtifact
	if platform != apmodel.ClientPackagePlatformWindowsX64 {
		if updateArtifact == nil || strings.TrimSpace(updateArtifact.FileName) == "" {
			return clientPackageValidationError(ErrClientPackageUpdateFileRequired, "校验 macOS 更新元数据需要上传对应的 .zip 自动更新包")
		}
		target = updateArtifact
	}
	item, ok := findMetadataItem(metadata, target.FileName)
	if !ok {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "更新元数据 files/path 中未找到上传文件 %s，请确认 latest*.yml 和安装包来自同一次打包", target.FileName)
	}
	item.Sha512 = strings.TrimSpace(item.Sha512)
	if item.Sha512 == "" {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "更新元数据中 %s 的 sha512 为空", target.FileName)
	}
	if item.Sha512 != target.Sha512 {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "更新元数据中 %s 的 sha512 不匹配，请确认 latest*.yml 和安装包来自同一次打包", target.FileName)
	}
	if !item.SizeSet {
		item.Size = target.Size
	}
	if item.SizeSet && item.Size <= 0 {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "更新元数据中 %s 的 size 无效：%d", target.FileName, item.Size)
	}
	if item.SizeSet && item.Size != target.Size {
		return clientPackageValidationError(ErrClientPackageInvalidInput, "更新元数据中 %s 的 size 不匹配：yml 为 %d，上传文件为 %d", target.FileName, item.Size, target.Size)
	}
	target.Sha512 = item.Sha512
	target.Size = item.Size
	return nil
}

func findMetadataItem(metadata updateMetadataFile, fileName string) (updateMetadataItem, bool) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	for _, item := range metadata.Files {
		if filepath.Base(strings.TrimSpace(item.URL)) == fileName {
			return item, true
		}
	}
	if filepath.Base(strings.TrimSpace(metadata.Path)) == fileName {
		return updateMetadataItem{
			URL:    metadata.Path,
			Sha512: metadata.Sha512,
			Size:   metadataTopLevelSize(metadata),
		}, true
	}
	return updateMetadataItem{}, false
}

func metadataTopLevelSize(metadata updateMetadataFile) int64 {
	if len(metadata.Files) == 1 {
		return metadata.Files[0].Size
	}
	return 0
}

func normalizeClientPackagePage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func compareClientPackageVersion(left string, right string) int {
	leftParts := parseClientPackageVersion(left)
	rightParts := parseClientPackageVersion(right)
	for i := 0; i < 3; i++ {
		if leftParts[i] > rightParts[i] {
			return 1
		}
		if leftParts[i] < rightParts[i] {
			return -1
		}
	}
	return 0
}

func parseClientPackageVersion(version string) [3]int {
	parts := strings.Split(strings.TrimSpace(version), ".")
	out := [3]int{}
	for i := 0; i < len(parts) && i < 3; i++ {
		value, _ := strconv.Atoi(parts[i])
		out[i] = value
	}
	return out
}

func mapClientPackage(pkg apmodel.ClientPackage) dtoaionui.ClientPackageItem {
	createdAt := pkg.CreatedAt.UTC().Format(time.RFC3339)
	updatedAt := pkg.UpdatedAt.UTC().Format(time.RFC3339)
	var publishedAt *string
	if pkg.PublishedAt != nil {
		value := pkg.PublishedAt.UTC().Format(clientPackageDisplayTimeLayout)
		publishedAt = &value
	}
	return dtoaionui.ClientPackageItem{
		Id:                     pkg.Id,
		Platform:               pkg.Platform,
		Version:                pkg.Version,
		Status:                 pkg.Status,
		FileName:               pkg.FileName,
		FileSha256:             pkg.FileSha256,
		FileSha512:             pkg.FileSha512,
		FileSize:               pkg.FileSize,
		UpdateFileName:         pkg.UpdateFileName,
		UpdateFileSha256:       pkg.UpdateFileSha256,
		UpdateFileSha512:       pkg.UpdateFileSha512,
		UpdateFileSize:         pkg.UpdateFileSize,
		UpdateMetadataFileName: pkg.UpdateMetadataFileName,
		UpdateMetadataSha256:   pkg.UpdateMetadataSha256,
		ReleaseNote:            pkg.ReleaseNote,
		CreatedBy:              pkg.CreatedBy,
		PublishedAt:            publishedAt,
		CreatedAt:              createdAt,
		UpdatedAt:              updatedAt,
	}
}

func BuildClientUpdateFeedYAML(feed ClientUpdateFeed) string {
	releaseDate := feed.ReleaseDate.UTC().Format(time.RFC3339Nano)
	return fmt.Sprintf("version: %s\nfiles:\n  - url: %s\n    sha512: %s\n    size: %d\npath: %s\nsha512: %s\nreleaseDate: '%s'\n",
		feed.Version,
		feed.Path,
		feed.Sha512,
		feed.Size,
		feed.Path,
		feed.Sha512,
		releaseDate,
	)
}
