package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidSkillDefBody = errors.New("agent platform skill def body invalid")

type SkillDef struct {
	Id                int       `json:"id" gorm:"primaryKey"`
	ResourceId        string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_skill_def_version;not null"`
	ResourceVersion   string    `json:"resource_version" gorm:"type:varchar(64);uniqueIndex:idx_ap_skill_def_version;not null"`
	InvokeSchemaJSON  string    `json:"invoke_schema_json" gorm:"column:invoke_schema_json;type:text"`
	OutputSchemaJSON  string    `json:"output_schema_json" gorm:"column:output_schema_json;type:text"`
	InvokeMode        string    `json:"invoke_mode" gorm:"type:varchar(32);not null"`
	TimeoutSeconds    int       `json:"timeout_seconds" gorm:"not null"`
	BindingConfigJSON string    `json:"binding_config_json" gorm:"column:binding_config_json;type:text"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (SkillDef) TableName() string {
	return "agent_platform_skill_defs"
}

func (d *SkillDef) BeforeCreate(tx *gorm.DB) error {
	return d.applyDefaultsAndValidate()
}

func (d *SkillDef) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return d.applyDefaultsAndValidate()
}

func (d *SkillDef) applyDefaultsAndValidate() error {
	d.ResourceId = strings.TrimSpace(d.ResourceId)
	d.ResourceVersion = strings.TrimSpace(d.ResourceVersion)
	d.InvokeSchemaJSON = strings.TrimSpace(d.InvokeSchemaJSON)
	d.OutputSchemaJSON = strings.TrimSpace(d.OutputSchemaJSON)
	d.InvokeMode = strings.TrimSpace(strings.ToLower(d.InvokeMode))
	d.BindingConfigJSON = strings.TrimSpace(d.BindingConfigJSON)
	if d.InvokeMode == "" {
		d.InvokeMode = "sync"
	}
	if d.ResourceId == "" || d.ResourceVersion == "" {
		return ErrInvalidSkillDefBody
	}
	return nil
}
