package agentplatform

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ResourceTypeMCP       = "mcp"
	ResourceTypeSkill     = "skill"
	ResourceTypeKnowledge = "knowledge"
	ResourceTypeAgent     = "agent"

	ResourceStatusDraft      = "draft"
	ResourceStatusPublished  = "published"
	ResourceStatusDisabled   = "disabled"
	ResourceStatusRevoked    = "revoked"
	ResourceStatusOffline    = "offline"
	ResourceStatusDeprecated = "deprecated"
)

var (
	ErrInvalidResourceType   = errors.New("agent platform resource type invalid")
	ErrInvalidResourceStatus = errors.New("agent platform resource status invalid")
	ErrInvalidResourceBody   = errors.New("agent platform resource body invalid")
	ErrImmutableResourceID   = errors.New("agent platform resource id immutable")
	ErrImmutableResourceType = errors.New("agent platform resource type immutable")
)

var allowedResourceTypes = map[string]struct{}{
	ResourceTypeMCP:       {},
	ResourceTypeSkill:     {},
	ResourceTypeKnowledge: {},
	ResourceTypeAgent:     {},
}

var allowedResourceStatuses = map[string]struct{}{
	ResourceStatusDraft:      {},
	ResourceStatusPublished:  {},
	ResourceStatusDisabled:   {},
	ResourceStatusRevoked:    {},
	ResourceStatusOffline:    {},
	ResourceStatusDeprecated: {},
}

type Resource struct {
	Id            int       `json:"id" gorm:"primaryKey"`
	ResourceId    string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_resource_id;not null"`
	ResourceType  string    `json:"resource_type" gorm:"type:varchar(16);index:idx_ap_resource_type;not null"`
	DisplayName   string    `json:"display_name" gorm:"type:varchar(255);not null"`
	Description   string    `json:"description" gorm:"type:text"`
	Avatar        string    `json:"avatar" gorm:"type:text"`
	OwnerUserId   int       `json:"owner_user_id" gorm:"index:idx_ap_resource_owner;not null"`
	Status        string    `json:"status" gorm:"type:varchar(16);index:idx_ap_resource_status;not null"`
	LatestVersion string    `json:"latest_version" gorm:"type:varchar(64);not null"`
	TenantId      int       `json:"tenant_id" gorm:"index:idx_ap_resource_tenant;not null;default:0"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Resource) TableName() string {
	return "agent_platform_resources"
}

func (r *Resource) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(r.ResourceId) == "" {
		resourceID, err := GenerateResourceID()
		if err != nil {
			return err
		}
		r.ResourceId = resourceID
	}
	return r.applyDefaultsAndValidate()
}

func (r *Resource) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	if r.Id == 0 && strings.TrimSpace(r.ResourceId) == "" {
		resourceID, err := GenerateResourceID()
		if err != nil {
			return err
		}
		r.ResourceId = resourceID
	}
	return r.applyDefaultsAndValidate()
}

func (r *Resource) BeforeUpdate(tx *gorm.DB) error {
	if tx == nil || tx.Statement == nil || tx.Statement.Schema == nil {
		return nil
	}
	if isPartialUpdate(tx) {
		return validateIdentityUpdateMap(tx)
	}
	if err := r.applyDefaultsAndValidate(); err != nil {
		return err
	}

	var existing Resource
	switch {
	case r.Id > 0:
		if err := tx.Session(&gorm.Session{}).Select("resource_id", "resource_type").First(&existing, r.Id).Error; err != nil {
			return err
		}
	case strings.TrimSpace(r.ResourceId) != "":
		if err := tx.Session(&gorm.Session{}).Select("resource_id", "resource_type").Where("resource_id = ?", r.ResourceId).First(&existing).Error; err != nil {
			return err
		}
	default:
		return nil
	}

	if existing.ResourceId != "" && r.ResourceId != existing.ResourceId {
		return ErrImmutableResourceID
	}
	if existing.ResourceType != "" && r.ResourceType != existing.ResourceType {
		return ErrImmutableResourceType
	}
	return nil
}

func isPartialUpdate(tx *gorm.DB) bool {
	if tx == nil || tx.Statement == nil || tx.Statement.Dest == nil {
		return false
	}
	destValue := reflect.ValueOf(tx.Statement.Dest)
	if !destValue.IsValid() {
		return false
	}
	for destValue.Kind() == reflect.Ptr {
		if destValue.IsNil() {
			return false
		}
		destValue = destValue.Elem()
	}
	return destValue.Kind() == reflect.Map
}

func validateIdentityUpdateMap(tx *gorm.DB) error {
	if tx == nil || tx.Statement == nil || tx.Statement.Dest == nil {
		return nil
	}
	destValue := reflect.ValueOf(tx.Statement.Dest)
	for destValue.Kind() == reflect.Ptr {
		if destValue.IsNil() {
			return nil
		}
		destValue = destValue.Elem()
	}
	if destValue.Kind() != reflect.Map {
		return nil
	}

	for _, keyValue := range destValue.MapKeys() {
		key, ok := keyValue.Interface().(string)
		if !ok {
			continue
		}
		normalized := strings.TrimSpace(strings.ToLower(key))
		if normalized == "resource_id" || normalized == "resourceid" {
			return ErrImmutableResourceID
		}
		if normalized == "resource_type" || normalized == "resourcetype" {
			return ErrImmutableResourceType
		}
	}
	return nil
}

func (r *Resource) applyDefaultsAndValidate() error {
	r.ResourceType = strings.TrimSpace(strings.ToLower(r.ResourceType))
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.Description = strings.TrimSpace(r.Description)
	r.Avatar = strings.TrimSpace(r.Avatar)
	r.Status = strings.TrimSpace(strings.ToLower(r.Status))
	r.ResourceId = strings.TrimSpace(r.ResourceId)
	r.LatestVersion = strings.TrimSpace(r.LatestVersion)

	if r.Status == "" {
		r.Status = ResourceStatusDraft
	}
	if r.LatestVersion == "" {
		r.LatestVersion = ""
	}
	if r.ResourceId == "" {
		return errors.New("agent platform resource id required")
	}
	if r.DisplayName == "" || r.OwnerUserId <= 0 {
		return ErrInvalidResourceBody
	}
	if _, ok := allowedResourceTypes[r.ResourceType]; !ok {
		return ErrInvalidResourceType
	}
	if _, ok := allowedResourceStatuses[r.Status]; !ok {
		return ErrInvalidResourceStatus
	}
	return nil
}

func GenerateResourceID() (string, error) {
	key, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return "", err
	}
	return "res_" + key, nil
}
