package enterprise

type DepartmentNameHistoryEntry struct {
	Name      string `json:"name"`
	ChangedAt int64  `json:"changed_at"`
}

type DepartmentTreeNode struct {
	Id          int                          `json:"id"`
	TenantId    int                          `json:"tenant_id"`
	Name        string                       `json:"name"`
	ParentId    *int                         `json:"parent_id"`
	Status      int                          `json:"status"`
	SourceType  int                          `json:"source_type"`
	ExternalId  string                       `json:"external_id"`
	SyncStatus  int                          `json:"sync_status"`
	SyncError   string                       `json:"sync_error"`
	NameHistory []DepartmentNameHistoryEntry `json:"name_history"`
	CreatedAt   int64                        `json:"created_at"`
	UpdatedAt   int64                        `json:"updated_at"`
	DeletedAt   int64                        `json:"deleted_at"`
	Children    []DepartmentTreeNode         `json:"children"`
}
