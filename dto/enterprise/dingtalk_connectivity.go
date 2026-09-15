package enterprise

const (
	DingTalkConnectivityAuthSuccess            = "auth_success"
	DingTalkConnectivityAuthInvalidCredentials = "auth_invalid_credentials"
	DingTalkConnectivityPermissionInsufficient = "auth_permission_insufficient"
	DingTalkConnectivityNetworkUnreachable     = "network_unreachable"
	DingTalkConnectivityCallbackMisconfigured  = "callback_misconfigured"
)

type DingTalkConnectivityResponse struct {
	TenantId   int    `json:"tenant_id"`
	Code       string `json:"code"`
	Stage      string `json:"stage"`
	Summary    string `json:"summary"`
	HTTPStatus int    `json:"http_status,omitempty"`
	CheckedAt  int64  `json:"checked_at"`
}
