package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidAgentDefBody = errors.New("agent platform agent def body invalid")

const (
	AgentCliTypeOpenCode = "opencode"
	AgentCliTypeCodex    = "codex"
)

type AgentDef struct {
	Id              int       `json:"id" gorm:"primaryKey"`
	ResourceId      string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_agent_def_version;not null"`
	ResourceVersion string    `json:"resource_version" gorm:"type:varchar(64);uniqueIndex:idx_ap_agent_def_version;not null"`
	CliType         string    `json:"cli_type" gorm:"type:varchar(32);not null;default:'opencode'"`
	Name            string    `json:"name" gorm:"type:varchar(255)"`
	Description     string    `json:"description" gorm:"type:text"`
	Avatar          string    `json:"avatar" gorm:"type:text"`
	Instructions    string    `json:"instructions" gorm:"type:text"`
	ModelConfigJSON string    `json:"model_config_json" gorm:"type:text"`
	PackagePath     string    `json:"package_path" gorm:"type:text"`
	PackageSha256   string    `json:"package_sha256" gorm:"type:varchar(64)"`
	PackageSize     int64     `json:"package_size" gorm:"not null;default:0"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
	d.ModelConfigJSON = strings.TrimSpace(d.ModelConfigJSON)
	d.PackagePath = strings.TrimSpace(d.PackagePath)
	d.PackageSha256 = strings.TrimSpace(d.PackageSha256)
	if d.ResourceId == "" || d.ResourceVersion == "" || d.Name == "" || !ValidAgentCliType(d.CliType) {
		return ErrInvalidAgentDefBody
	}
	return nil
}

func ValidAgentCliType(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == AgentCliTypeOpenCode || value == AgentCliTypeCodex
}
