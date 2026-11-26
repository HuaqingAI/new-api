package enterprise

import "errors"

var (
	ErrDepartmentNotFound                                = errors.New("department not found")
	ErrDepartmentNameHistoryInvalid                      = errors.New("department name history invalid")
	ErrUserNotFound                                      = errors.New("enterprise user not found")
	ErrMembershipNotFound                                = errors.New("enterprise membership not found")
	ErrMembershipAlreadyExists                           = errors.New("enterprise membership already exists")
	ErrDuplicateDepartment                               = errors.New("duplicate department id")
	ErrInvalidMembershipInput                            = errors.New("invalid enterprise membership input")
	ErrEnterpriseUsernameInvalid                         = errors.New("enterprise username invalid")
	ErrEnterpriseUsernameExists                          = errors.New("enterprise username already exists")
	ErrDepartmentAdminRequired                           = errors.New("enterprise department admin permission required")
	ErrDepartmentOwnerDeniedByLocalRule                  = errors.New("enterprise department owner denied by local override")
	ErrDepartmentOwnerNotFound                           = errors.New("enterprise department owner not found")
	ErrEnterpriseAdminRequired                           = errors.New("enterprise admin permission required")
	ErrInvalidAdminActionInput                           = errors.New("invalid enterprise admin action input")
	ErrAdminActionNotFound                               = errors.New("enterprise admin action not found")
	ErrDepartmentRoleNotFound                            = errors.New("enterprise department role not found")
	ErrInvalidDepartmentBudgetInput                      = errors.New("invalid enterprise department budget input")
	ErrDepartmentBudgetInvalidType                       = errors.New("enterprise department budget type invalid")
	ErrDepartmentBudgetInvalidQuota                      = errors.New("enterprise department budget quota invalid")
	ErrDepartmentBudgetInvalidCycleQuota                 = errors.New("enterprise department budget cycle quota invalid")
	ErrDepartmentBudgetInvalidCycleType                  = errors.New("enterprise department budget cycle type invalid")
	ErrDepartmentBudgetInvalidCustomSeconds              = errors.New("enterprise department budget custom seconds invalid")
	ErrDepartmentBudgetInvalidCycleStartedAt             = errors.New("enterprise department budget cycle started at invalid")
	ErrDepartmentBudgetTypeImmutable                     = errors.New("enterprise department budget type immutable")
	ErrDepartmentBudgetThresholdInvalid                  = errors.New("enterprise department budget threshold invalid")
	ErrDepartmentBudgetStatusTransitionInvalid           = errors.New("enterprise department budget status transition invalid")
	ErrDepartmentBudgetResizeBelowCommitted              = errors.New("enterprise department budget resize below committed")
	ErrPublicBudgetInvalidInput                          = errors.New("enterprise public budget input invalid")
	ErrPublicBudgetNotFound                              = errors.New("enterprise public budget not found")
	ErrQuotaAllocationInvalidInput                       = errors.New("enterprise quota allocation input invalid")
	ErrQuotaAllocationBudgetNotFound                     = errors.New("enterprise quota allocation budget not found")
	ErrQuotaAllocationBudgetInactive                     = errors.New("enterprise quota allocation budget inactive")
	ErrQuotaAllocationQuotaInvalid                       = errors.New("enterprise quota allocation quota invalid")
	ErrQuotaAllocationBudgetInsufficient                 = errors.New("enterprise budget insufficient")
	ErrPublicBudgetManualAllocationDenied                = errors.New("enterprise public budget manual assignment not allowed")
	ErrQuotaAllocationBalanceRemainingInsufficient       = errors.New("enterprise budget balance remaining insufficient")
	ErrQuotaAllocationSubscriptionCycleAllocatedExceeded = errors.New("enterprise budget subscription cycle allocated exceeded")
	ErrQuotaAllocationUserOutOfDepartment                = errors.New("enterprise quota allocation user out of department")
	ErrQuotaAllocationWalletProtected                    = errors.New("enterprise quota allocation wallet protected")
	ErrQuotaAllocationWalletNotFound                     = errors.New("enterprise quota allocation wallet not found")
	ErrQuotaAllocationWalletOrderInvalid                 = errors.New("enterprise quota allocation wallet order invalid")
	ErrQuotaAllocationAlreadyProcessed                   = errors.New("enterprise quota allocation already processed")
	ErrQuotaAllocationProtected                          = errors.New("enterprise quota allocation protected")
	ErrQuotaAllocationNotFound                           = errors.New("enterprise quota allocation not found")
	ErrQuotaRequestInvalidInput                          = errors.New("enterprise quota request input invalid")
	ErrQuotaRequestNotFound                              = errors.New("enterprise quota request not found")
	ErrQuotaRequestBudgetModeRequired                    = errors.New("enterprise quota request budget mode required")
	ErrQuotaRequestBudgetPoolRequired                    = errors.New("enterprise quota request department budget required")
	ErrQuotaRequestBudgetScopeMismatch                   = errors.New("enterprise quota request budget scope mismatch")
	ErrQuotaRequestDepartmentMembershipRequired          = errors.New("enterprise quota request requester membership required")
	ErrQuotaRequestApprovalNotAllowed                    = errors.New("enterprise quota request approval not allowed")
	ErrQuotaRequestAlreadyProcessed                      = errors.New("enterprise quota request already processed")
	ErrQuotaRequestApprovalReasonRequired                = errors.New("enterprise quota request approval reason required")
	ErrQuotaRequestApprovedQuotaInvalid                  = errors.New("enterprise quota request approved quota invalid")
	ErrQuotaRequestRejectedReasonRequired                = errors.New("enterprise quota request rejected reason required")
	ErrBudgetDelegationInvalidInput                      = errors.New("enterprise budget delegation input invalid")
	ErrBudgetDelegationNotFound                          = errors.New("enterprise budget delegation not found")
	ErrBudgetDelegationQuotaInvalid                      = errors.New("enterprise budget delegation quota invalid")
	ErrBudgetDelegationBudgetInactive                    = errors.New("enterprise budget delegation budget inactive")
	ErrBudgetDelegationTargetNotDescendant               = errors.New("enterprise budget delegation target department is not a descendant")
	ErrBudgetDelegationPermissionDenied                  = errors.New("enterprise budget delegation permission denied")
	ErrBudgetDelegationBudgetTypeMismatch                = errors.New("enterprise budget delegation budget type mismatch")
	ErrBudgetDelegationTargetBudgetCapacityLocked        = errors.New("enterprise budget delegation target budget capacity locked")
	ErrDingTalkMissingCredentials                        = errors.New("enterprise dingtalk credentials missing")
	ErrDingTalkInvalidCallbackURL                        = errors.New("enterprise dingtalk callback url invalid")
	ErrDingTalkConfigNotFound                            = errors.New("enterprise dingtalk config not found")
	ErrDingTalkOAuthNotEnabled                           = errors.New("enterprise dingtalk oauth not enabled")
	ErrDingTalkOAuthCodeMissing                          = errors.New("enterprise dingtalk oauth code missing")
	ErrDingTalkOAuthProviderFailed                       = errors.New("enterprise dingtalk oauth provider failed")
	ErrDingTalkOAuthIdentityMissing                      = errors.New("enterprise dingtalk oauth identity missing")
	ErrDingTalkOAuthEmployeeNotFound                     = errors.New("enterprise dingtalk oauth employee could not be verified")
	ErrDingTalkOAuthUserDisabled                         = errors.New("enterprise dingtalk oauth user disabled")
	ErrDingTalkOAuthOutOfScope                           = errors.New("enterprise dingtalk oauth user out of sync scope")
	ErrDingTalkOAuthBindingConflict                      = errors.New("enterprise dingtalk oauth binding conflict")
	ErrDingTalkOAuthRegistrationDisabled                 = errors.New("enterprise dingtalk oauth registration disabled")
	ErrDingTalkAutoSyncDisabled                          = errors.New("enterprise dingtalk auto sync on login disabled")
	ErrDingTalkSyncNotEnabled                            = errors.New("enterprise dingtalk sync not enabled")
	ErrDingTalkScheduleCronInvalid                       = errors.New("enterprise dingtalk schedule cron invalid")
	ErrDingTalkScheduleTimezoneInvalid                   = errors.New("enterprise dingtalk schedule timezone invalid")
	ErrDingTalkSyncTaskNotFound                          = errors.New("enterprise dingtalk sync task not found")
	ErrDingTalkSyncConflictNotFound                      = errors.New("enterprise dingtalk sync conflict not found")
	ErrDingTalkSyncConflictNotPending                    = errors.New("enterprise dingtalk sync conflict not pending")
	ErrDingTalkSyncConflictNoCandidate                   = errors.New("enterprise dingtalk sync conflict has no single candidate")
	ErrUsageReportInvalidInput                           = errors.New("enterprise usage report input invalid")
	ErrUsageReportInvalidEmail                           = errors.New("enterprise usage report email invalid")
	ErrUsageReportNotConfigured                          = errors.New("enterprise usage report not configured")
	ErrInvalidAlertEventQuery                            = errors.New("enterprise alert event query invalid")
	ErrInvalidDepartmentRiskSummaryQuery                 = errors.New("enterprise department risk summary query invalid")
	ErrInvalidAlertDeliveryQuery                         = errors.New("enterprise alert delivery query invalid")
	ErrAlertRuleNotFound                                 = errors.New("enterprise alert rule not found")
	ErrAlertRuleInvalidInput                             = errors.New("enterprise alert rule input invalid")
	ErrAlertRuleInvalidEmail                             = errors.New("enterprise alert rule email invalid")
	ErrAlertRuleInvalidWebhookURL                        = errors.New("enterprise alert rule webhook url invalid")
	ErrAlertRuleChannelRequired                          = errors.New("enterprise alert rule channel required")
	ErrAlertDeliveryNotFound                             = errors.New("enterprise alert delivery not found")
	ErrAlertDeliveryResendNotAllowed                     = errors.New("enterprise alert delivery resend not allowed")
	ErrInvalidGovernanceTimelineQuery                    = errors.New("enterprise governance timeline query invalid")
	ErrInvalidGovernanceNotificationQuery                = errors.New("enterprise governance notification query invalid")
	ErrGovernanceNotificationDeliveryNotFound            = errors.New("enterprise governance notification delivery not found")
	ErrGovernanceNotificationResendNotAllowed            = errors.New("enterprise governance notification resend not allowed")
)

type QuotaAllocationBudgetError struct {
	Reason error
}

func (e *QuotaAllocationBudgetError) Error() string {
	return ErrQuotaAllocationBudgetInsufficient.Error()
}

func (e *QuotaAllocationBudgetError) Is(target error) bool {
	if target == nil {
		return false
	}
	if target == ErrQuotaAllocationBudgetInsufficient {
		return true
	}
	return e.Reason != nil && target == e.Reason
}

func newQuotaAllocationBudgetError(reason error) error {
	return &QuotaAllocationBudgetError{Reason: reason}
}

func QuotaAllocationBudgetReason(err error) error {
	var budgetErr *QuotaAllocationBudgetError
	if errors.As(err, &budgetErr) {
		return budgetErr.Reason
	}
	return nil
}
