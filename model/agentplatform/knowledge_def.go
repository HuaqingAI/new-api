package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidKnowledgeDefBody = errors.New("agent platform knowledge def body invalid")

type KnowledgeDef struct {
	Id                  int       `json:"id" gorm:"primaryKey"`
	ResourceId          string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_knowledge_def_resource;not null"`
	ExternalKnowledgeId string    `json:"external_knowledge_id" gorm:"type:varchar(128);index"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (KnowledgeDef) TableName() string {
	return "agent_platform_knowledge_defs"
}

func (d *KnowledgeDef) BeforeCreate(tx *gorm.DB) error {
	return d.applyDefaultsAndValidate()
}

func (d *KnowledgeDef) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return d.applyDefaultsAndValidate()
}

func (d *KnowledgeDef) applyDefaultsAndValidate() error {
	d.ResourceId = strings.TrimSpace(d.ResourceId)
	d.ExternalKnowledgeId = strings.TrimSpace(d.ExternalKnowledgeId)
	if d.ResourceId == "" || d.ExternalKnowledgeId == "" {
		return ErrInvalidKnowledgeDefBody
	}
	return nil
}
