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
	ErrDepartmentAdminRequired                           = errors.New("enterprise department admin permission required")
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
	ErrQuotaAllocationInvalidInput                       = errors.New("enterprise quota allocation input invalid")
	ErrQuotaAllocationBudgetNotFound                     = errors.New("enterprise quota allocation budget not found")
	ErrQuotaAllocationBudgetInactive                     = errors.New("enterprise quota allocation budget inactive")
	ErrQuotaAllocationQuotaInvalid                       = errors.New("enterprise quota allocation quota invalid")
	ErrQuotaAllocationBudgetInsufficient                 = errors.New("enterprise budget insufficient")
	ErrQuotaAllocationBalanceRemainingInsufficient       = errors.New("enterprise budget balance remaining insufficient")
	ErrQuotaAllocationSubscriptionCycleAllocatedExceeded = errors.New("enterprise budget subscription cycle allocated exceeded")
	ErrQuotaAllocationUserOutOfDepartment                = errors.New("enterprise quota allocation user out of department")
	ErrQuotaAllocationWalletProtected                    = errors.New("enterprise quota allocation wallet protected")
	ErrQuotaAllocationWalletNotFound                     = errors.New("enterprise quota allocation wallet not found")
	ErrQuotaAllocationWalletOrderInvalid                 = errors.New("enterprise quota allocation wallet order invalid")
	ErrQuotaAllocationAlreadyProcessed                   = errors.New("enterprise quota allocation already processed")
	ErrDingTalkMissingCredentials                        = errors.New("enterprise dingtalk credentials missing")
	ErrDingTalkInvalidCallbackURL                        = errors.New("enterprise dingtalk callback url invalid")
	ErrDingTalkConfigNotFound                            = errors.New("enterprise dingtalk config not found")
	ErrDingTalkOAuthNotEnabled                           = errors.New("enterprise dingtalk oauth not enabled")
	ErrDingTalkOAuthCodeMissing                          = errors.New("enterprise dingtalk oauth code missing")
	ErrDingTalkOAuthProviderFailed                       = errors.New("enterprise dingtalk oauth provider failed")
	ErrDingTalkOAuthIdentityMissing                      = errors.New("enterprise dingtalk oauth identity missing")
	ErrDingTalkOAuthUserDisabled                         = errors.New("enterprise dingtalk oauth user disabled")
	ErrDingTalkOAuthOutOfScope                           = errors.New("enterprise dingtalk oauth user out of sync scope")
	ErrDingTalkOAuthBindingConflict                      = errors.New("enterprise dingtalk oauth binding conflict")
	ErrDingTalkOAuthRegistrationDisabled                 = errors.New("enterprise dingtalk oauth registration disabled")
	ErrDingTalkSyncNotEnabled                            = errors.New("enterprise dingtalk sync not enabled")
	ErrDingTalkSyncTaskNotFound                          = errors.New("enterprise dingtalk sync task not found")
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
