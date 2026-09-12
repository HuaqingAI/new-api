package enterprise

type DingTalkSyncStartRequest struct {
	TenantId *int  `json:"tenant_id,omitempty"`
	Inline   *bool `json:"inline,omitempty"`
}

type DingTalkSyncResolveConflictRequest struct {
	TenantId        *int `json:"tenant_id,omitempty"`
	CandidateUserId *int `json:"candidate_user_id,omitempty"`
}
