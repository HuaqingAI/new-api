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

const EnterpriseExternalSourceManual = "manual"
