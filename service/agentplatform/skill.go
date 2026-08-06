package agentplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

type SkillQuery struct {
	OwnerUserId *int
	TenantId    *int
	Page        int
	PageSize    int
}

type SkillCreateInput struct {
	DisplayName string
	Description string
	OwnerUserId int
	TenantId    int
}

type SkillUpdateInput struct {
	DisplayName string
	Description string
}

type SkillItem struct {
	ResourceItem
	FileName  string
	FilePath  string
	Sha256    string
	SizeBytes int64
}

type SkillListResult struct {
	Items    []SkillItem
	Total    int
	Page     int
	PageSize int
}

type SkillService struct {
	resources *ResourceService
	db        *gorm.DB
}

func NewSkillService(db *gorm.DB) *SkillService {
	return &SkillService{
		resources: NewResourceService(db),
		db:        db,
	}
}

func (s *SkillService) List(query SkillQuery) (SkillListResult, error) {
	if s == nil || s.resources == nil {
		return SkillListResult{Items: []SkillItem{}}, ErrInvalidResourceInput
	}
	result, err := s.resources.List(ListResourcesQuery{
		ResourceType: apmodel.ResourceTypeSkill,
		OwnerUserId:  query.OwnerUserId,
		TenantId:     query.TenantId,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		return SkillListResult{Items: []SkillItem{}}, err
	}
	items := make([]SkillItem, 0, len(result.Items))
	for _, item := range result.Items {
		def, err := s.getDef(item.ResourceId)
		if err != nil && !errors.Is(err, ErrResourceNotFound) {
			return SkillListResult{Items: []SkillItem{}}, err
		}
		items = append(items, SkillItem{
			ResourceItem: item,
			FileName:     def.FileName,
			FilePath:     def.FilePath,
			Sha256:       def.Sha256,
			SizeBytes:    def.SizeBytes,
		})
	}
	return SkillListResult{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func (s *SkillService) Get(resourceID string) (SkillItem, error) {
	if s == nil || s.resources == nil {
		return SkillItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.GetByResourceID(resourceID)
	if err != nil {
		return SkillItem{}, err
	}
	if item.ResourceType != apmodel.ResourceTypeSkill {
		return SkillItem{}, ErrResourceNotFound
	}
	def, err := s.getDef(resourceID)
	if err != nil && !errors.Is(err, ErrResourceNotFound) {
		return SkillItem{}, err
	}
	return SkillItem{
		ResourceItem: item,
		FileName:     def.FileName,
		FilePath:     def.FilePath,
		Sha256:       def.Sha256,
		SizeBytes:    def.SizeBytes,
	}, nil
}

func (s *SkillService) Create(input SkillCreateInput) (SkillItem, error) {
	if s == nil || s.resources == nil {
		return SkillItem{}, ErrInvalidResourceInput
	}
	item, err := s.resources.Create(CreateResourceInput{
		ResourceType: apmodel.ResourceTypeSkill,
		DisplayName:  input.DisplayName,
		Description:  input.Description,
		OwnerUserId:  input.OwnerUserId,
		TenantId:     input.TenantId,
		Status:       apmodel.ResourceStatusPublished,
	})
	if err != nil {
		return SkillItem{}, err
	}
	return SkillItem{ResourceItem: item}, nil
}

func (s *SkillService) Update(resourceID string, input SkillUpdateInput) (SkillItem, error) {
	if s == nil || s.db == nil {
		return SkillItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	if resourceID == "" || input.DisplayName == "" {
		return SkillItem{}, ErrInvalidResourceInput
	}

	var resource apmodel.Resource
	if err := s.db.Where("resource_id = ? AND resource_type = ?", resourceID, apmodel.ResourceTypeSkill).First(&resource).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return SkillItem{}, ErrResourceNotFound
		}
		return SkillItem{}, err
	}
	if err := s.db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", resourceID).
		Updates(map[string]any{"display_name": input.DisplayName, "description": input.Description}).Error; err != nil {
		return SkillItem{}, err
	}
	return s.Get(resourceID)
}

func (s *SkillService) SavePackage(resourceID string, fileName string, reader io.Reader) (SkillItem, error) {
	if s == nil || s.db == nil {
		return SkillItem{}, ErrInvalidResourceInput
	}
	resourceID = strings.TrimSpace(resourceID)
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if resourceID == "" || fileName == "" || !strings.EqualFold(filepath.Ext(fileName), ".zip") || reader == nil {
		return SkillItem{}, ErrInvalidResourceInput
	}
	var resource apmodel.Resource
	if err := s.db.Where("resource_id = ? AND resource_type = ?", resourceID, apmodel.ResourceTypeSkill).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SkillItem{}, ErrResourceNotFound
		}
		return SkillItem{}, err
	}
	packagePath, sum, size, err := storeSkillPackage(resourceID, fileName, reader)
	if err != nil {
		return SkillItem{}, err
	}
	def := apmodel.SkillDef{
		ResourceId: resourceID,
		FileName:   fileName,
		FilePath:   packagePath,
		Sha256:     sum,
		SizeBytes:  size,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var existing apmodel.SkillDef
		if err := tx.Where("resource_id = ?", resourceID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tx.Create(&def).Error
			}
			return err
		}
		return tx.Model(&apmodel.SkillDef{}).Where("resource_id = ?", resourceID).Updates(map[string]any{
			"file_name":  fileName,
			"file_path":  packagePath,
			"sha256":     sum,
			"size_bytes": size,
		}).Error
	})
	if err != nil {
		if errors.Is(err, apmodel.ErrInvalidSkillDefBody) {
			return SkillItem{}, ErrInvalidResourceInput
		}
		return SkillItem{}, err
	}
	return s.Get(resourceID)
}

func (s *SkillService) PackagePath(resourceID string) (string, error) {
	def, err := s.getDef(resourceID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(def.FilePath) == "" {
		return "", ErrResourceNotFound
	}
	return def.FilePath, nil
}

func (s *SkillService) PackageDownloadURL(resourceID string) (string, error) {
	def, err := s.getDef(resourceID)
	if err != nil {
		return "", err
	}
	ref, err := ParseArtifactURI(def.FilePath)
	if err != nil {
		return "", err
	}
	store, err := DefaultArtifactStore()
	if err != nil {
		return "", err
	}
	signed, err := store.PresignGet(context.Background(), ref, artifactPresignExpires())
	if err != nil {
		return "", err
	}
	return signed.URL, nil
}

func (s *SkillService) SetStatus(resourceID string, status string) (SkillItem, error) {
	if err := setTypedResourceStatus(s.db, resourceID, apmodel.ResourceTypeSkill, status); err != nil {
		return SkillItem{}, err
	}
	return s.Get(resourceID)
}

func (s *SkillService) Delete(resourceID string) error {
	return deleteTypedResource(s.db, resourceID, apmodel.ResourceTypeSkill)
}

func (s *SkillService) getDef(resourceID string) (apmodel.SkillDef, error) {
	var def apmodel.SkillDef
	if err := s.db.Where("resource_id = ?", resourceID).First(&def).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apmodel.SkillDef{}, ErrResourceNotFound
		}
		return apmodel.SkillDef{}, err
	}
	return def, nil
}

func storeSkillPackage(resourceID string, fileName string, reader io.Reader) (string, string, int64, error) {
	tmpDir, err := os.MkdirTemp("", "new-api-skill-upload-*")
	if err != nil {
		return "", "", 0, err
	}
	defer os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", "", 0, err
	}
	tmpFile, err := os.CreateTemp(tmpDir, "skill-*.zip")
	if err != nil {
		return "", "", 0, err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	hash := sha256.New()
	size, err := io.Copy(tmpFile, io.TeeReader(reader, hash))
	closeErr := tmpFile.Close()
	if err != nil {
		return "", "", 0, err
	}
	if closeErr != nil {
		return "", "", 0, closeErr
	}
	if size <= 0 {
		return "", "", 0, ErrInvalidResourceInput
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	store, err := DefaultArtifactStore()
	if err != nil {
		return "", "", 0, err
	}
	ref, err := store.PutFile(context.Background(), PutArtifactInput{
		Kind:        ArtifactKindSkill,
		BucketKey:   buildSkillObjectKey(resourceID, sum, fileName),
		LocalPath:   tmpPath,
		ContentType: "application/zip",
		Sha256:      sum,
		SizeBytes:   size,
		Metadata: map[string]string{
			"resource-id":   resourceID,
			"sha256":        sum,
			"artifact-kind": ArtifactKindSkill,
		},
	})
	if err != nil {
		return "", "", 0, err
	}
	return ref.URI, sum, size, nil
}
