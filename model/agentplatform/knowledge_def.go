package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidKnowledgeDefBody = errors.New("agent platform knowledge def body invalid")

type KnowledgeDef struct {
	Id                       int       `json:"id" gorm:"primaryKey"`
	ResourceId               string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_knowledge_def_version;not null"`
	ResourceVersion          string    `json:"resource_version" gorm:"type:varchar(64);uniqueIndex:idx_ap_knowledge_def_version;not null"`
	KnowledgeMode            string    `json:"knowledge_mode" gorm:"type:varchar(32);not null"`
	ProviderType             string    `json:"provider_type" gorm:"type:varchar(32);not null"`
	ProviderAdapterKey       string    `json:"provider_adapter_key" gorm:"type:varchar(64);not null"`
	ProviderConfigJSON       string    `json:"provider_config_json" gorm:"column:provider_config_json;type:text"`
	QuerySchemaJSON          string    `json:"query_schema_json" gorm:"column:query_schema_json;type:text"`
	CitationSchemaJSON       string    `json:"citation_schema_json" gorm:"column:citation_schema_json;type:text"`
	FreshnessRulesJSON       string    `json:"freshness_rules_json" gorm:"column:freshness_rules_json;type:text"`
	ProviderCapabilitiesJSON string    `json:"provider_capabilities_json" gorm:"column:provider_capabilities_json;type:text"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
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
	d.ResourceVersion = strings.TrimSpace(d.ResourceVersion)
	d.KnowledgeMode = strings.TrimSpace(strings.ToLower(d.KnowledgeMode))
	d.ProviderType = strings.TrimSpace(strings.ToLower(d.ProviderType))
	d.ProviderAdapterKey = strings.TrimSpace(d.ProviderAdapterKey)
	d.ProviderConfigJSON = strings.TrimSpace(d.ProviderConfigJSON)
	d.QuerySchemaJSON = strings.TrimSpace(d.QuerySchemaJSON)
	d.CitationSchemaJSON = strings.TrimSpace(d.CitationSchemaJSON)
	d.FreshnessRulesJSON = strings.TrimSpace(d.FreshnessRulesJSON)
	d.ProviderCapabilitiesJSON = strings.TrimSpace(d.ProviderCapabilitiesJSON)
	if d.KnowledgeMode == "" {
		d.KnowledgeMode = "retrieval"
	}
	if d.ResourceId == "" || d.ResourceVersion == "" || d.ProviderType == "" || d.ProviderAdapterKey == "" {
		return ErrInvalidKnowledgeDefBody
	}
	return nil
}
