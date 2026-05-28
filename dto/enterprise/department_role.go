package enterprise

type DepartmentRoleItem struct {
	Id           int   `json:"id"`
	TenantId     int   `json:"tenant_id"`
	UserId       int   `json:"user_id"`
	DepartmentId int   `json:"department_id"`
	Role         int   `json:"role"`
	Status       int   `json:"status"`
	CreatedAt    int64 `json:"created_at"`
	UpdatedAt    int64 `json:"updated_at"`
}
