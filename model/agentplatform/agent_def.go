package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var ErrInvalidAgentDefBody = errors.New("agent platform agent def body invalid")

type AgentDef struct {
	Id                    int       `json:"id" gorm:"primaryKey"`
	ResourceId            string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_agent_def_version;not null"`
	ResourceVersion       string    `json:"resource_version" gorm:"type:varchar(64);uniqueIndex:idx_ap_agent_def_version;not null"`
	CliType               string    `json:"cli_type" gorm:"type:varchar(32);not null;default:'opencode'"`
	Name                  string    `json:"name" gorm:"type:varchar(255)"`
	Description           string    `json:"description" gorm:"type:text"`
	Avatar                string    `json:"avatar" gorm:"type:varchar(64)"`
	Instructions          string    `json:"instructions" gorm:"type:text"`
	ModelTokenId          int       `json:"model_token_id" gorm:"not null;default:0"`
	DefaultModel          string    `json:"default_model" gorm:"type:varchar(128);not null;default:''"`
	ModelConfigJSON       string    `json:"model_config_json" gorm:"type:text"`
	PackagePath           string    `json:"package_path" gorm:"type:text"`
	PackageSha256         string    `json:"package_sha256" gorm:"type:varchar(64)"`
	PackageSize           int64     `json:"package_size" gorm:"not null;default:0"`
	ManifestJSON          string    `json:"manifest_json" gorm:"column:manifest_json;type:text"`
	DependenciesJSON      string    `json:"dependencies_json" gorm:"column:dependencies_json;type:text"`
	PromptMetadataJSON    string    `json:"prompt_metadata_json" gorm:"column:prompt_metadata_json;type:text"`
	CompatibilityMetaJSON string    `json:"compatibility_meta_json" gorm:"column:compatibility_meta_json;type:text"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
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
	d.CliType = strings.TrimSpace(strings.ToLower(d.CliType))
	d.Name = strings.TrimSpace(d.Name)
	d.Description = strings.TrimSpace(d.Description)
	d.Avatar = strings.TrimSpace(d.Avatar)
	d.Instructions = strings.TrimSpace(d.Instructions)
	d.DefaultModel = strings.TrimSpace(d.DefaultModel)
	d.ModelConfigJSON = strings.TrimSpace(d.ModelConfigJSON)
	d.PackagePath = strings.TrimSpace(d.PackagePath)
	d.PackageSha256 = strings.TrimSpace(d.PackageSha256)
	d.ManifestJSON = strings.TrimSpace(d.ManifestJSON)
	d.DependenciesJSON = strings.TrimSpace(d.DependenciesJSON)
	d.PromptMetadataJSON = strings.TrimSpace(d.PromptMetadataJSON)
	d.CompatibilityMetaJSON = strings.TrimSpace(d.CompatibilityMetaJSON)
	if d.CliType == "" {
		d.CliType = "opencode"
	}
	if d.ResourceId == "" || d.ResourceVersion == "" || d.CliType != "opencode" {
		return ErrInvalidAgentDefBody
	}
	if d.ManifestJSON == "" && d.DependenciesJSON == "" && d.CompatibilityMetaJSON == "" {
		if d.Name == "" {
			return ErrInvalidAgentDefBody
		}
		return nil
	}
	if d.ManifestJSON == "" || d.DependenciesJSON == "" || d.CompatibilityMetaJSON == "" {
		return ErrInvalidAgentDefBody
	}
	if !validAgentJSONObject(d.ManifestJSON) || !validAgentJSONArray(d.DependenciesJSON) || !validAgentJSONObject(d.CompatibilityMetaJSON) {
		return ErrInvalidAgentDefBody
	}
	return nil
}

func validAgentJSONObject(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	var payload map[string]any
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return false
	}
	return len(payload) > 0
}

func validAgentJSONArray(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	var payload []any
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return false
	}
	return len(payload) > 0
}
