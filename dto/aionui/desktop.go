package aionui

type DesktopTokenRequest struct {
	Code       string `json:"code"`
	DeviceId   string `json:"device_id"`
	AppVersion string `json:"app_version"`
	Group      string `json:"group"`
}

type DesktopUser struct {
	Id          int      `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Departments []string `json:"departments,omitempty"`
}

type DesktopTokenResponse struct {
	AccessToken    string                `json:"access_token"`
	ExpiresAt      int64                 `json:"expires_at"`
	User           DesktopUser           `json:"user"`
	PersonalAPIKey DesktopPersonalAPIKey `json:"personal_api_key"`
	QuotaApplyURL  string                `json:"quota_apply_url"`
}

type DesktopPersonalAPIKey struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	MaskedKey string `json:"masked_key"`
}

type AgentConfigItem struct {
	Id                 string   `json:"id,omitempty"`
	CliType            string   `json:"cli_type"`
	ArtifactKey        string   `json:"artifact_key"`
	Url                string   `json:"url"`
	UrlType            string   `json:"url_type"`
	UrlExpiresAt       int64    `json:"url_expires_at"`
	Version            string   `json:"version"`
	Name               string   `json:"name"`
	Description        string   `json:"description,omitempty"`
	Categories         []string `json:"categories"`
	RecommendedPrompts []string `json:"recommended_prompts,omitempty"`
	Avatar             string   `json:"avatar,omitempty"`
	Sha256             string   `json:"sha256,omitempty"`
	Size               int64    `json:"size,omitempty"`
}

type AgentConfigsResponse struct {
	UserEmail string            `json:"user_email"`
	Revision  string            `json:"revision,omitempty"`
	Agents    []AgentConfigItem `json:"agents"`
}

type QuotaWalletSummary struct {
	RemainQuota int    `json:"remain_quota"`
	UsedQuota   int    `json:"used_quota"`
	Display     string `json:"display"`
}

type QuotaSubscriptionItem struct {
	Id                     int    `json:"id"`
	PlanId                 int    `json:"plan_id"`
	AmountTotal            int64  `json:"amount_total"`
	AmountUsed             int64  `json:"amount_used"`
	AmountAvailable        int64  `json:"amount_available"`
	AmountAvailableDisplay string `json:"amount_available_display"`
	EndTime                int64  `json:"end_time"`
}

type QuotaSubscriptionGroup struct {
	GroupKey               string                  `json:"group_key"`
	GroupName              string                  `json:"group_name"`
	AmountTotal            int64                   `json:"amount_total"`
	AmountUsed             int64                   `json:"amount_used"`
	AmountAvailable        int64                   `json:"amount_available"`
	AmountAvailableDisplay string                  `json:"amount_available_display"`
	Items                  []QuotaSubscriptionItem `json:"items"`
}

type QuotaSummaryResponse struct {
	Wallet                QuotaWalletSummary       `json:"wallet"`
	Subscriptions         []QuotaSubscriptionGroup `json:"subscriptions"`
	TotalAvailable        int64                    `json:"total_available"`
	TotalAvailableDisplay string                   `json:"total_available_display"`
	QuotaApplyURL         string                   `json:"quota_apply_url"`
	RefreshedAt           int64                    `json:"refreshed_at"`
}
