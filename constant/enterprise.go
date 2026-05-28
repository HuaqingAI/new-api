package constant

const (
	DepartmentStatusEnabled = iota + 1
	DepartmentStatusDisabled
	DepartmentStatusDeleted
)

const (
	DepartmentSourceTypeManual = iota + 1
	DepartmentSourceTypeDingTalk
)

const (
	DepartmentSyncStatusNone = iota
	DepartmentSyncStatusOK
	DepartmentSyncStatusWarning
	DepartmentSyncStatusFailed
)

const (
	EnterpriseDepartmentStatusActive   = DepartmentStatusEnabled
	EnterpriseDepartmentStatusInactive = DepartmentStatusDisabled
	EnterpriseDepartmentStatusDeleted  = DepartmentStatusDeleted
)

const (
	EnterpriseMembershipStatusActive   = 1
	EnterpriseMembershipStatusInactive = 2
	EnterpriseMembershipStatusLeft     = 3
	EnterpriseMembershipStatusPending  = 4
)

const (
	EnterpriseExternalSourceManual   = "manual"
	EnterpriseExternalSourceDingTalk = "dingtalk"
)

const (
	EnterpriseDepartmentRoleDeptAdmin = 1
)

const (
	EnterpriseDepartmentRoleStatusActive = iota + 1
	EnterpriseDepartmentRoleStatusInactive
)

const (
	DingTalkSyncTaskModeFull = "full"

	DingTalkSyncTaskStatusPending   = "pending"
	DingTalkSyncTaskStatusRunning   = "running"
	DingTalkSyncTaskStatusSucceeded = "succeeded"
	DingTalkSyncTaskStatusFailed    = "failed"

	DingTalkSyncObjectDepartment = "department"
	DingTalkSyncObjectUser       = "user"
	DingTalkSyncObjectMembership = "membership"
	DingTalkSyncObjectConflict   = "conflict"

	DingTalkSyncLogActionCreated         = "created"
	DingTalkSyncLogActionUpdated         = "updated"
	DingTalkSyncLogActionDisabled        = "disabled"
	DingTalkSyncLogActionConflictPending = "conflict_pending"
	DingTalkSyncLogActionSkipped         = "skipped"
	DingTalkSyncLogActionFailed          = "failed"

	DingTalkSyncLogStatusSuccess = "success"
	DingTalkSyncLogStatusFailed  = "failed"
	DingTalkSyncLogStatusSkipped = "skipped"
	DingTalkSyncLogStatusWarning = "warning"
)

const (
	DingTalkSyncConflictStatusPending  = "pending"
	DingTalkSyncConflictStatusResolved = "resolved"
	DingTalkSyncConflictStatusIgnored  = "ignored"
)
