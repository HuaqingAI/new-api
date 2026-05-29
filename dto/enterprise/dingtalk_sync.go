package enterprise

type DingTalkSyncStartRequest struct {
	TenantId *int  `json:"tenant_id,omitempty"`
	Inline   *bool `json:"inline,omitempty"`
}
