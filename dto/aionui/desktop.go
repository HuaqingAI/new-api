package aionui

type DesktopTokenRequest struct {
	Code       string `json:"code"`
	DeviceId   string `json:"device_id"`
	AppVersion string `json:"app_version"`
}

type DesktopUser struct {
	Id          int      `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Departments []string `json:"departments,omitempty"`
}

type DesktopTokenResponse struct {
	AccessToken string      `json:"access_token"`
	ExpiresAt   int64       `json:"expires_at"`
	User        DesktopUser `json:"user"`
}

type AgentConfigItem struct {
	Id           string `json:"id,omitempty"`
	CliType      string `json:"cli_type"`
	ArtifactKey  string `json:"artifact_key"`
	Url          string `json:"url"`
	UrlType      string `json:"url_type"`
	UrlExpiresAt int64  `json:"url_expires_at"`
	Version      string `json:"version"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Avatar       string `json:"avatar,omitempty"`
	Sha256       string `json:"sha256,omitempty"`
	Size         int64  `json:"size,omitempty"`
}

type AgentConfigsResponse struct {
	UserEmail string            `json:"user_email"`
	Revision  string            `json:"revision,omitempty"`
	Agents    []AgentConfigItem `json:"agents"`
}
