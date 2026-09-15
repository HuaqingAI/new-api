package enterprise

type UserDepartment struct {
	Id             int    `json:"id"`
	TenantId       int    `json:"tenant_id" gorm:"type:int;default:0;index;uniqueIndex:uq_enterprise_user_departments_membership"`
	UserId         int    `json:"user_id" gorm:"type:int;not null;index;uniqueIndex:uq_enterprise_user_departments_membership"`
	DepartmentId   int    `json:"department_id" gorm:"type:int;not null;index;uniqueIndex:uq_enterprise_user_departments_membership"`
	ExternalUserId string `json:"external_user_id,omitempty" gorm:"type:varchar(128);default:'';index"`
	ExternalSource string `json:"external_source,omitempty" gorm:"type:varchar(32);default:'manual';index;uniqueIndex:uq_enterprise_user_departments_membership"`
	Status         int    `json:"status" gorm:"type:int;default:1;index"`
	JoinedAt       int64  `json:"joined_at" gorm:"type:bigint;default:0"`
	LeftAt         int64  `json:"left_at" gorm:"type:bigint;default:0"`
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt      int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (UserDepartment) TableName() string {
	return "enterprise_user_departments"
}
