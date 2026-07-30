package agentplatform

import "time"

type CreateAgentRequest struct {
	DisplayName  string   `json:"display_name" binding:"required"`
	Description  string   `json:"description,omitempty"`
	Avatar       string   `json:"avatar,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	ModelTokenId int      `json:"model_token_id,omitempty"`
	DefaultModel string   `json:"default_model,omitempty"`
	McpIds       []string `json:"mcp_ids,omitempty"`
	SkillIds     []string `json:"skill_ids,omitempty"`
	KnowledgeIds []string `json:"knowledge_ids,omitempty"`
	OwnerUserId  int      `json:"owner_user_id,omitempty"`
	TenantId     *int     `json:"tenant_id,omitempty"`
}

type UpdateAgentRequest struct {
	DisplayName  string   `json:"display_name" binding:"required"`
	Description  string   `json:"description,omitempty"`
	Avatar       string   `json:"avatar,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	ModelTokenId int      `json:"model_token_id,omitempty"`
	DefaultModel string   `json:"default_model,omitempty"`
	McpIds       []string `json:"mcp_ids,omitempty"`
	SkillIds     []string `json:"skill_ids,omitempty"`
	KnowledgeIds []string `json:"knowledge_ids,omitempty"`
}

type AgentQuery struct {
	OwnerUserId *int `form:"owner_user_id" json:"owner_user_id,omitempty"`
	TenantId    *int `form:"tenant_id" json:"tenant_id,omitempty"`
	Page        *int `form:"page" json:"page,omitempty"`
	PageSize    *int `form:"page_size" json:"page_size,omitempty"`
}

type AgentItem = ResourceItem

type AgentDetailItem struct {
	ResourceItem
	CliType             string   `json:"cli_type"`
	Instructions        string   `json:"instructions"`
	ModelTokenId        int      `json:"model_token_id"`
	DefaultModel        string   `json:"default_model"`
	ModelTokenUserId    int      `json:"model_token_user_id,omitempty"`
	ModelTokenUserName  string   `json:"model_token_user_name,omitempty"`
	ModelTokenName      string   `json:"model_token_name,omitempty"`
	ModelTokenMaskedKey string   `json:"model_token_masked_key,omitempty"`
	McpIds              []string `json:"mcp_ids"`
	SkillIds            []string `json:"skill_ids"`
	KnowledgeIds        []string `json:"knowledge_ids"`
}

type AgentListResponse struct {
	Items    []AgentItem `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

type AgentGrantRequest struct {
	SubjectType string `json:"subject_type" binding:"required"`
	SubjectId   string `json:"subject_id" binding:"required"`
}

type AgentGrantSetRequest struct {
	Users       []string `json:"users,omitempty"`
	Departments []string `json:"departments,omitempty"`
}

type PublishAgentRequest struct {
	Summary string               `json:"summary" binding:"required"`
	Grants  AgentGrantSetRequest `json:"grants" binding:"required"`
}

type AgentArtifactResponse struct {
	CliType string `json:"cli_type"`
	Url     string `json:"url"`
	Sha256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

type AgentGrantResponse struct {
	GrantId     string `json:"grant_id"`
	SubjectType string `json:"subject_type"`
	SubjectId   string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
}

type AgentVersionResponse struct {
	ResourceId    string     `json:"resource_id"`
	Version       string     `json:"version"`
	Summary       string     `json:"summary"`
	Status        string     `json:"status"`
	PackagePath   string     `json:"package_path"`
	PackageSha256 string     `json:"package_sha256"`
	PackageSize   int64      `json:"package_size"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type PublishAgentDefaultsResponse struct {
	ResourceId    string               `json:"resource_id"`
	LatestVersion string               `json:"latest_version"`
	NextVersion   string               `json:"next_version"`
	Grants        []AgentGrantResponse `json:"grants"`
}

type PublishAgentResponse struct {
	ResourceId string                `json:"resource_id"`
	Version    string                `json:"version"`
	Status     string                `json:"status"`
	Artifact   AgentArtifactResponse `json:"artifact"`
	Grants     []AgentGrantResponse  `json:"grants"`
}

type AgentModelKeyResponse struct {
	Id                 int    `json:"id"`
	UserId             int    `json:"user_id"`
	UserName           string `json:"user_name"`
	Name               string `json:"name"`
	MaskedKey          string `json:"masked_key"`
	Status             int    `json:"status"`
	ExpiredTime        int64  `json:"expired_time"`
	RemainQuota        int    `json:"remain_quota"`
	UnlimitedQuota     bool   `json:"unlimited_quota"`
	Group              string `json:"group"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled"`
	ModelCount         int    `json:"model_count"`
	Available          bool   `json:"available"`
	DisabledReason     string `json:"disabled_reason,omitempty"`
}

type AgentModelResponse struct {
	Model        string         `json:"model"`
	DisplayName  string         `json:"display_name"`
	Status       string         `json:"status"`
	Capabilities map[string]any `json:"capabilities"`
}
