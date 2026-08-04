package agentplatform

type CreateResourceVersionRequest struct {
	Version         string `json:"version" binding:"required"`
	ContractVersion string `json:"contract_version" binding:"required"`
	Summary         string `json:"summary,omitempty"`
}

type ResourceVersionItem struct {
	ResourceId      string `json:"resource_id"`
	ResourceType    string `json:"resource_type"`
	Version         string `json:"version"`
	ContractVersion string `json:"contract_version"`
	Summary         string `json:"summary"`
	Status          string `json:"status"`
	CreatedBy       int    `json:"created_by"`
	PublishedAt     *int64 `json:"published_at,omitempty"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}
