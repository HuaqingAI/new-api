package enterprise

type DepartmentRoleItem struct {
	Id             int    `json:"id"`
	TenantId       int    `json:"tenant_id"`
	UserId         int    `json:"user_id"`
	DepartmentId   int    `json:"department_id"`
	Role           int    `json:"role"`
	Source         string `json:"source"`
	Effect         string `json:"effect"`
	ExternalSource string `json:"external_source"`
	Status         int    `json:"status"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type DepartmentOwnerMutationRequest struct {
	TenantId *int `json:"tenant_id,omitempty"`
	UserId   int  `json:"user_id"`
}

type DepartmentOwnerFactItem struct {
	Id                        int    `json:"id"`
	TenantId                  int    `json:"tenant_id"`
	UserId                    int    `json:"user_id"`
	DepartmentId              int    `json:"department_id"`
	Role                      int    `json:"role"`
	Source                    string `json:"source"`
	Effect                    string `json:"effect"`
	ExternalSource            string `json:"external_source"`
	Status                    int    `json:"status"`
	InheritedFromDepartmentId int    `json:"inherited_from_department_id"`
	CreatedAt                 int64  `json:"created_at"`
	UpdatedAt                 int64  `json:"updated_at"`
}

type EffectiveDepartmentOwnerItem struct {
	UserId                    int    `json:"user_id"`
	DepartmentId              int    `json:"department_id"`
	Source                    string `json:"source"`
	Effect                    string `json:"effect"`
	InheritedFromDepartmentId int    `json:"inherited_from_department_id"`
	RoleFactId                int    `json:"role_fact_id"`
}

type DepartmentOwnersResponse struct {
	Facts           []DepartmentOwnerFactItem      `json:"facts"`
	EffectiveOwners []EffectiveDepartmentOwnerItem `json:"effective_owners"`
	OwnerCount      int                            `json:"owner_count"`
	Fallback        string                         `json:"fallback"`
}
