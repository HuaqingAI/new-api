package agentplatform

type LifecycleActionRequest struct {
	Version   string `json:"version,omitempty"`
	RequestId string `json:"request_id,omitempty"`
}

type LifecycleActionResponse struct {
	ResourceId      string `json:"resource_id"`
	Action          string `json:"action"`
	PreviousStatus  string `json:"previous_status"`
	CurrentStatus   string `json:"current_status"`
	PreviousVersion string `json:"previous_version"`
	CurrentVersion  string `json:"current_version"`
	TargetVersion   string `json:"target_version"`
	RequestId       string `json:"request_id"`
	AuditActionId   int    `json:"audit_action_id"`
}
