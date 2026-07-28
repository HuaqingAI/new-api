package aionui

type DesktopTokenRequest struct {
	Code       string `json:"code"`
	DeviceId   string `json:"device_id"`
	AppVersion string `json:"app_version"`
}

type DesktopUser struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type DesktopTokenResponse struct {
	AccessToken string      `json:"access_token"`
	ExpiresAt   int64       `json:"expires_at"`
	User        DesktopUser `json:"user"`
}

type AgentConfigItem struct {
	Id          string `json:"id,omitempty"`
	Url         string `json:"url"`
	UrlType     string `json:"url_type"`
	Version     string `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
	Sha256      string `json:"sha256,omitempty"`
}

type AgentConfigsResponse struct {
	UserEmail string            `json:"user_email"`
	CliType   string            `json:"cli_type"`
	Revision  string            `json:"revision,omitempty"`
	Agents    []AgentConfigItem `json:"agents"`
}
