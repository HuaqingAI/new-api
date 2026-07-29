package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var ErrInvalidSkillDefBody = errors.New("agent platform skill def body invalid")

var allowedSkillInvokeModes = map[string]struct{}{
	"sync":  {},
	"async": {},
	"file":  {},
}

type SkillDef struct {
	Id                int       `json:"id" gorm:"primaryKey"`
	ResourceId        string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_skill_def_version;not null"`
	ResourceVersion   string    `json:"resource_version" gorm:"type:varchar(64);uniqueIndex:idx_ap_skill_def_version;not null"`
	InvokeSchemaJSON  string    `json:"invoke_schema_json" gorm:"column:invoke_schema_json;type:text"`
	OutputSchemaJSON  string    `json:"output_schema_json" gorm:"column:output_schema_json;type:text"`
	InvokeMode        string    `json:"invoke_mode" gorm:"type:varchar(32);not null"`
	TimeoutSeconds    int       `json:"timeout_seconds" gorm:"not null"`
	BindingConfigJSON string    `json:"binding_config_json" gorm:"column:binding_config_json;type:text"`
	FileName          string    `json:"file_name" gorm:"type:varchar(255)"`
	FilePath          string    `json:"file_path" gorm:"type:text"`
	Sha256            string    `json:"sha256" gorm:"type:varchar(64)"`
	SizeBytes         int64     `json:"size_bytes" gorm:"not null;default:0"`
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
	d.FileName = strings.TrimSpace(d.FileName)
	d.FilePath = strings.TrimSpace(d.FilePath)
	d.Sha256 = strings.TrimSpace(d.Sha256)
	if d.InvokeMode == "" {
		d.InvokeMode = "file"
	}
	if d.ResourceId == "" {
		return ErrInvalidSkillDefBody
	}
	if _, ok := allowedSkillInvokeModes[d.InvokeMode]; !ok {
		return ErrInvalidSkillDefBody
	}
	if d.InvokeMode == "file" {
		if d.FilePath == "" || d.Sha256 == "" || d.SizeBytes <= 0 {
			return ErrInvalidSkillDefBody
		}
		return nil
	}
	if d.ResourceVersion == "" || d.InvokeSchemaJSON == "" || d.OutputSchemaJSON == "" || d.BindingConfigJSON == "" || d.TimeoutSeconds <= 0 {
		return ErrInvalidSkillDefBody
	}
	if !validSkillJSONShape(d.InvokeSchemaJSON) || !validSkillJSONShape(d.OutputSchemaJSON) || !validSkillBindingConfig(d.BindingConfigJSON) {
		return ErrInvalidSkillDefBody
	}
	return nil
}

func validSkillJSONShape(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	var payload map[string]any
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return false
	}
	if len(payload) == 0 {
		return false
	}
	return true
}

func validSkillBindingConfig(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	var payload map[string]any
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return false
	}
	if len(payload) == 0 {
		return false
	}
	return true
}
