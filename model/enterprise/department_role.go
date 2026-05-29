package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

type DepartmentRole struct {
	Id           int   `json:"id" gorm:"primaryKey"`
	TenantId     int   `json:"tenant_id" gorm:"type:int;default:0;index;uniqueIndex:uq_ent_dept_roles_role"`
	UserId       int   `json:"user_id" gorm:"type:int;not null;index;uniqueIndex:uq_ent_dept_roles_role"`
	DepartmentId int   `json:"department_id" gorm:"type:int;not null;index;uniqueIndex:uq_ent_dept_roles_role"`
	Role         int   `json:"role" gorm:"type:int;not null;default:1;index;uniqueIndex:uq_ent_dept_roles_role"`
	Status       int   `json:"status" gorm:"type:int;not null;default:1;index"`
	CreatedAt    int64 `json:"created_at" gorm:"type:bigint"`
	UpdatedAt    int64 `json:"updated_at" gorm:"type:bigint"`
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
	if r.Status == 0 {
		r.Status = constant.EnterpriseDepartmentRoleStatusActive
	}
	return nil
}

func (r *DepartmentRole) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now().Unix()
	return nil
}
