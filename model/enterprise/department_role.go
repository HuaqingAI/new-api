package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

type DepartmentRole struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	TenantId       int    `json:"tenant_id" gorm:"type:int;default:0;index;uniqueIndex:uq_ent_dept_roles_fact"`
	UserId         int    `json:"user_id" gorm:"type:int;not null;index;uniqueIndex:uq_ent_dept_roles_fact"`
	DepartmentId   int    `json:"department_id" gorm:"type:int;not null;index;uniqueIndex:uq_ent_dept_roles_fact"`
	Role           int    `json:"role" gorm:"type:int;not null;default:1;index;uniqueIndex:uq_ent_dept_roles_fact"`
	Source         string `json:"source" gorm:"type:varchar(32);not null;default:'manual_grant';index;uniqueIndex:uq_ent_dept_roles_fact"`
	Effect         string `json:"effect" gorm:"type:varchar(16);not null;default:'allow';index"`
	ExternalSource string `json:"external_source" gorm:"type:varchar(32);not null;default:'';index"`
	Status         int    `json:"status" gorm:"type:int;not null;default:1;index"`
	CreatedAt      int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt      int64  `json:"updated_at" gorm:"type:bigint"`
}

func (DepartmentRole) TableName() string {
	return "enterprise_department_roles"
}

func (r *DepartmentRole) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	if r.UpdatedAt == 0 {
		r.UpdatedAt = now
	}
	if r.Role == 0 {
		r.Role = constant.EnterpriseDepartmentRoleDeptAdmin
	}
	normalizeDepartmentRoleFact(r)
	return nil
}

func (r *DepartmentRole) BeforeSave(tx *gorm.DB) error {
	normalizeDepartmentRoleFact(r)
	return nil
}

func normalizeDepartmentRoleFact(r *DepartmentRole) {
	if r.Source == "" {
		r.Source = constant.EnterpriseDepartmentRoleSourceManualGrant
	}
	if r.Effect == "" {
		if r.Source == constant.EnterpriseDepartmentRoleSourceManualDenyOverride {
			r.Effect = constant.EnterpriseDepartmentRoleEffectDeny
		} else {
			r.Effect = constant.EnterpriseDepartmentRoleEffectAllow
		}
	}
	if r.ExternalSource == "" && r.Source == constant.EnterpriseDepartmentRoleSourceDingTalkOwner {
		r.ExternalSource = constant.EnterpriseExternalSourceDingTalk
	}
	if r.Status == 0 {
		r.Status = constant.EnterpriseDepartmentRoleStatusActive
	}
}

func (r *DepartmentRole) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now().Unix()
	return nil
}
