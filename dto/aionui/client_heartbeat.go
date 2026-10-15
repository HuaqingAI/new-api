package aionui

type ClientHeartbeatRequest struct {
	ClientVersion string                  `json:"client_version"`
	Platform      string                  `json:"platform"`
	LanIP         string                  `json:"lan_ip"`
	User          ClientHeartbeatUserInfo `json:"user"`
}

type ClientHeartbeatUserInfo struct {
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Departments []string `json:"departments"`
}

type ClientInstallationQuery struct {
	Page       *int    `form:"page" json:"page,omitempty"`
	PageSize   *int    `form:"page_size" json:"page_size,omitempty"`
	Keyword    *string `form:"keyword" json:"keyword,omitempty"`
	Platform   *string `form:"platform" json:"platform,omitempty"`
	Version    *string `form:"version" json:"version,omitempty"`
	ActiveOnly *bool   `form:"active_only" json:"active_only,omitempty"`
}

type ClientHeartbeatResponse struct {
	AcceptedAt           string `json:"accepted_at"`
	NextHeartbeatSeconds int    `json:"next_heartbeat_seconds"`
}

type ClientInstallationItem struct {
	Id               int      `json:"id"`
	ClientVersion    string   `json:"client_version"`
	Platform         string   `json:"platform"`
	LanIP            string   `json:"lan_ip"`
	UserId           int      `json:"user_id"`
	Username         string   `json:"username"`
	Email            string   `json:"email"`
	DisplayName      string   `json:"display_name"`
	Departments      []string `json:"departments"`
	FirstHeartbeatAt string   `json:"first_heartbeat_at"`
	LastHeartbeatAt  string   `json:"last_heartbeat_at"`
	IsActive         bool     `json:"is_active"`
}

type ClientInstallationSummary struct {
	ActiveUsers         int    `json:"active_users"`
	ActiveInstallations int    `json:"active_installations"`
	TotalUsers          int    `json:"total_users"`
	TotalInstallations  int    `json:"total_installations"`
	ActiveSince         string `json:"active_since"`
}

type ClientInstallationListResponse struct {
	Items    []ClientInstallationItem  `json:"items"`
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
	Summary  ClientInstallationSummary `json:"summary"`
}
