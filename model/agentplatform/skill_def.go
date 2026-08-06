package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidSkillDefBody = errors.New("agent platform skill def body invalid")

type SkillDef struct {
	Id         int       `json:"id" gorm:"primaryKey"`
	ResourceId string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_skill_def_resource;not null"`
	FileName   string    `json:"file_name" gorm:"type:varchar(255)"`
	FilePath   string    `json:"file_path" gorm:"type:text"`
	Sha256     string    `json:"sha256" gorm:"type:varchar(64)"`
	SizeBytes  int64     `json:"size_bytes" gorm:"not null;default:0"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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
	d.FileName = strings.TrimSpace(d.FileName)
	d.FilePath = strings.TrimSpace(d.FilePath)
	d.Sha256 = strings.TrimSpace(d.Sha256)
	if d.ResourceId == "" || d.FilePath == "" || d.Sha256 == "" || d.SizeBytes <= 0 {
		return ErrInvalidSkillDefBody
	}
	return nil
}
