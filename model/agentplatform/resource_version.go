package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidResourceVersionBody = errors.New("agent platform resource version body invalid")

type ResourceVersion struct {
	Id              int        `json:"id" gorm:"primaryKey"`
	ResourceId      string     `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_resource_version;not null"`
	Version         string     `json:"version" gorm:"column:version;type:varchar(64);uniqueIndex:idx_ap_resource_version;not null"`
	ContractVersion string     `json:"contract_version" gorm:"type:varchar(64);not null"`
	Summary         string     `json:"summary" gorm:"type:text"`
	SchemaJSON      string     `json:"schema_json" gorm:"column:schema_json;type:text"`
	DetailJSON      string     `json:"detail_json" gorm:"column:detail_json;type:text"`
	Status          string     `json:"status" gorm:"type:varchar(16);index:idx_ap_resource_version_status;not null"`
	CreatedBy       int        `json:"created_by" gorm:"index:idx_ap_resource_version_creator;not null"`
	PublishedAt     *time.Time `json:"published_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Resource        Resource   `json:"-" gorm:"references:ResourceId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (ResourceVersion) TableName() string {
	return "agent_platform_resource_versions"
}

func (r *ResourceVersion) BeforeCreate(tx *gorm.DB) error {
	return r.applyDefaultsAndValidate()
}

func (r *ResourceVersion) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return r.applyDefaultsAndValidate()
}

func (r *ResourceVersion) applyDefaultsAndValidate() error {
	r.ResourceId = strings.TrimSpace(r.ResourceId)
	r.Version = strings.TrimSpace(r.Version)
	r.ContractVersion = strings.TrimSpace(r.ContractVersion)
	r.Summary = strings.TrimSpace(r.Summary)
	r.SchemaJSON = strings.TrimSpace(r.SchemaJSON)
	r.DetailJSON = strings.TrimSpace(r.DetailJSON)
	r.Status = strings.TrimSpace(strings.ToLower(r.Status))

	if r.Status == "" {
		r.Status = ResourceStatusDraft
	}
	if r.ResourceId == "" || r.Version == "" || r.ContractVersion == "" || r.CreatedBy <= 0 {
		return ErrInvalidResourceVersionBody
	}
	if _, ok := allowedResourceStatuses[r.Status]; !ok {
		return ErrInvalidResourceStatus
	}
	return nil
}
