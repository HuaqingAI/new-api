package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidAgentDefBody = errors.New("agent platform agent def body invalid")

type AgentDef struct {
	Id                     int       `json:"id" gorm:"primaryKey"`
	ResourceId             string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_agent_def_version;not null"`
	ResourceVersion        string    `json:"resource_version" gorm:"type:varchar(64);uniqueIndex:idx_ap_agent_def_version;not null"`
	ManifestJSON           string    `json:"manifest_json" gorm:"column:manifest_json;type:text"`
	DependenciesJSON       string    `json:"dependencies_json" gorm:"column:dependencies_json;type:text"`
	PromptMetadataJSON     string    `json:"prompt_metadata_json" gorm:"column:prompt_metadata_json;type:text"`
	CompatibilityMetaJSON  string    `json:"compatibility_meta_json" gorm:"column:compatibility_meta_json;type:text"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (AgentDef) TableName() string {
	return "agent_platform_agent_defs"
}

func (d *AgentDef) BeforeCreate(tx *gorm.DB) error {
	return d.applyDefaultsAndValidate()
}

func (d *AgentDef) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return d.applyDefaultsAndValidate()
}

func (d *AgentDef) applyDefaultsAndValidate() error {
	d.ResourceId = strings.TrimSpace(d.ResourceId)
	d.ResourceVersion = strings.TrimSpace(d.ResourceVersion)
	d.ManifestJSON = strings.TrimSpace(d.ManifestJSON)
	d.DependenciesJSON = strings.TrimSpace(d.DependenciesJSON)
	d.PromptMetadataJSON = strings.TrimSpace(d.PromptMetadataJSON)
	d.CompatibilityMetaJSON = strings.TrimSpace(d.CompatibilityMetaJSON)
	if d.ResourceId == "" || d.ResourceVersion == "" {
		return ErrInvalidAgentDefBody
	}
	return nil
}
