package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	GrantSubjectTypeUser       = "user"
	GrantSubjectTypeDepartment = "department"

	GrantStatusActive  = "active"
	GrantStatusRevoked = "revoked"
)

var ErrInvalidResourceGrantBody = errors.New("agent platform resource grant body invalid")

type ResourceGrant struct {
	Id              int        `json:"id" gorm:"primaryKey"`
	GrantId         string     `json:"grant_id" gorm:"type:varchar(40);uniqueIndex;not null"`
	ResourceId      string     `json:"resource_id" gorm:"type:varchar(40);index:idx_ap_resource_grant_resource;not null"`
	ResourceVersion string     `json:"resource_version" gorm:"type:varchar(64);index:idx_ap_resource_grant_resource;not null"`
	SubjectType     string     `json:"subject_type" gorm:"type:varchar(32);index:idx_ap_resource_grant_subject,priority:1;not null"`
	SubjectId       string     `json:"subject_id" gorm:"type:varchar(128);index:idx_ap_resource_grant_subject,priority:2;not null"`
	Status          string     `json:"status" gorm:"type:varchar(16);index:idx_ap_resource_grant_status;not null"`
	GrantedBy       int        `json:"granted_by" gorm:"index;not null"`
	GrantedAt       *time.Time `json:"granted_at"`
	RevokedAt       *time.Time `json:"revoked_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (ResourceGrant) TableName() string {
	return "agent_platform_resource_grants"
}

func (g *ResourceGrant) BeforeCreate(tx *gorm.DB) error {
	return g.applyDefaultsAndValidate()
}

func (g *ResourceGrant) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return g.applyDefaultsAndValidate()
}

func (g *ResourceGrant) applyDefaultsAndValidate() error {
	g.GrantId = strings.TrimSpace(g.GrantId)
	g.ResourceId = strings.TrimSpace(g.ResourceId)
	g.ResourceVersion = strings.TrimSpace(g.ResourceVersion)
	g.SubjectType = strings.TrimSpace(strings.ToLower(g.SubjectType))
	g.SubjectId = strings.TrimSpace(g.SubjectId)
	g.Status = strings.TrimSpace(strings.ToLower(g.Status))
	if g.GrantId == "" {
		id, err := GenerateResourceGrantID()
		if err != nil {
			return err
		}
		g.GrantId = id
	}
	if g.Status == "" {
		g.Status = GrantStatusActive
	}
	if g.GrantedAt == nil {
		now := time.Now().UTC()
		g.GrantedAt = &now
	}
	if g.ResourceId == "" || g.ResourceVersion == "" || g.SubjectId == "" || g.GrantedBy <= 0 {
		return ErrInvalidResourceGrantBody
	}
	if g.SubjectType != GrantSubjectTypeUser && g.SubjectType != GrantSubjectTypeDepartment {
		return ErrInvalidResourceGrantBody
	}
	if g.Status != GrantStatusActive && g.Status != GrantStatusRevoked {
		return ErrInvalidResourceGrantBody
	}
	return nil
}

func GenerateResourceGrantID() (string, error) {
	key, err := common.GenerateRandomCharsKey(30)
	if err != nil {
		return "", err
	}
	return "grant_" + key, nil
}
