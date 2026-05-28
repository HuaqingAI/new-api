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
