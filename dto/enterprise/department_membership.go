package enterprise

type DepartmentMembershipQuery struct {
	TenantId       *int   `form:"tenant_id" json:"tenant_id,omitempty"`
	Status         *int   `form:"status" json:"status,omitempty"`
	ExternalSource string `form:"external_source" json:"external_source,omitempty"`
	Page           *int   `form:"page" json:"page,omitempty"`
	PageSize       *int   `form:"page_size" json:"page_size,omitempty"`
}

type ReplaceUserDepartmentsRequest struct {
	TenantId        *int   `json:"tenant_id,omitempty"`
	DepartmentIds   []int  `json:"department_ids"`
	ExternalUserId  string `json:"external_user_id,omitempty"`
	ExternalSource  string `json:"external_source,omitempty"`
	JoinedAt        *int64 `json:"joined_at,omitempty"`
	DeactivateStale *bool  `json:"deactivate_stale,omitempty"`
}

type AddDepartmentMemberRequest struct {
	TenantId       *int   `json:"tenant_id,omitempty"`
	UserId         int    `json:"user_id"`
	ExternalUserId string `json:"external_user_id,omitempty"`
	ExternalSource string `json:"external_source,omitempty"`
	JoinedAt       *int64 `json:"joined_at,omitempty"`
}

type MembershipStatusRequest struct {
	TenantId       *int   `json:"tenant_id,omitempty"`
	ExternalSource string `json:"external_source,omitempty"`
	ChangedAt      *int64 `json:"changed_at,omitempty"`
}

type RenameDepartmentMemberRequest struct {
	TenantId    *int   `json:"tenant_id,omitempty"`
	NewUsername string `json:"new_username"`
	ChangedAt   *int64 `json:"changed_at,omitempty"`
}

type UserDepartmentItem struct {
	Id             int    `json:"id"`
	TenantId       int    `json:"tenant_id"`
	UserId         int    `json:"user_id"`
	DepartmentId   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
	ExternalUserId string `json:"external_user_id"`
	ExternalSource string `json:"external_source"`
	Status         int    `json:"status"`
	JoinedAt       int64  `json:"joined_at"`
	LeftAt         int64  `json:"left_at"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type DepartmentMemberItem struct {
	Id             int    `json:"id"`
	TenantId       int    `json:"tenant_id"`
	UserId         int    `json:"user_id"`
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`
	DepartmentId   int    `json:"department_id"`
	ExternalUserId string `json:"external_user_id"`
	ExternalSource string `json:"external_source"`
	Status         int    `json:"status"`
	JoinedAt       int64  `json:"joined_at"`
	LeftAt         int64  `json:"left_at"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type UserDepartmentsResponse struct {
	Items        []UserDepartmentItem `json:"items"`
	Total        int                  `json:"total"`
	IsUnassigned bool                 `json:"is_unassigned"`
}

type DepartmentMembersResponse struct {
	Items []DepartmentMemberItem `json:"items"`
	Total int                    `json:"total"`
}
