package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	AgentDependencyTypeMCP       = "mcp"
	AgentDependencyTypeSkill     = "skill"
	AgentDependencyTypeKnowledge = "knowledge"
)

var ErrInvalidAgentDependencyBody = errors.New("agent platform agent dependency body invalid")

type AgentDependency struct {
	Id               int       `json:"id" gorm:"primaryKey"`
	AgentResourceId  string    `json:"agent_resource_id" gorm:"type:varchar(40);index:idx_ap_agent_dependency,priority:1;not null"`
	ResourceVersion  string    `json:"resource_version" gorm:"type:varchar(64);index:idx_ap_agent_dependency,priority:2;not null"`
	TargetType       string    `json:"target_type" gorm:"type:varchar(32);index;not null"`
	TargetResourceId string    `json:"target_resource_id" gorm:"type:varchar(40);index;not null"`
	SortOrder        int       `json:"sort_order" gorm:"not null;default:0"`
	SnapshotJSON     string    `json:"snapshot_json" gorm:"column:snapshot_json;type:text"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (AgentDependency) TableName() string {
	return "agent_platform_agent_dependencies"
}

func (d *AgentDependency) BeforeCreate(tx *gorm.DB) error {
	return d.applyDefaultsAndValidate()
}

func (d *AgentDependency) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return d.applyDefaultsAndValidate()
}

func (d *AgentDependency) applyDefaultsAndValidate() error {
	d.AgentResourceId = strings.TrimSpace(d.AgentResourceId)
	d.ResourceVersion = strings.TrimSpace(d.ResourceVersion)
	d.TargetType = strings.TrimSpace(strings.ToLower(d.TargetType))
	d.TargetResourceId = strings.TrimSpace(d.TargetResourceId)
	d.SnapshotJSON = strings.TrimSpace(d.SnapshotJSON)
	if d.AgentResourceId == "" || d.ResourceVersion == "" || d.TargetResourceId == "" {
		return ErrInvalidAgentDependencyBody
	}
	if d.TargetType != AgentDependencyTypeMCP && d.TargetType != AgentDependencyTypeSkill && d.TargetType != AgentDependencyTypeKnowledge {
		return ErrInvalidAgentDependencyBody
	}
	return nil
}
