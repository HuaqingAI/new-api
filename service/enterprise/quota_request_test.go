package enterprise_test

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestQuotaRequestSubmitRequiresExplicitDepartmentBudgetContext(t *testing.T) {
	svc, _ := newQuotaAllocationTestService(t)

	_, err := entservice.NewQuotaRequestService(model.DB).Submit(entservice.SubmitQuotaRequestInput{
		TenantId:        0,
		DepartmentId:    1,
		BudgetMode:      entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId: 2001,
		RequestedQuota:  100,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaRequestInvalidInput)

	_, err = entservice.NewQuotaRequestService(model.DB).Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		RequesterUserId:    2001,
		RequestedQuota:     100,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaRequestBudgetModeRequired)

	_ = svc
}

func TestQuotaRequestApproveFallsBackToAdminWhenNoOwner(t *testing.T) {
	_, db := newQuotaAllocationTestService(t)
	reqSvc := entservice.NewQuotaRequestService(db)

	item, err := reqSvc.Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2001,
		RequestedQuota:     200,
		RequestReason:      "need more quota",
	})
	require.NoError(t, err)
	require.Equal(t, 0, item.OwnerCountSnapshot)
	require.Equal(t, "admin", item.Fallback)
	require.NotZero(t, item.SubmittedAt)

	_, err = reqSvc.Approve(entservice.DecideQuotaRequestInput{
		TenantId:       0,
		RequestId:      item.Id,
		ActorId:        2001,
		ApprovedQuota:  quotaRequestInt64Ptr(100),
		ApprovalReason: "self-approve should fail",
	})
	require.ErrorIs(t, err, entservice.ErrQuotaRequestApprovalNotAllowed)

	result, err := reqSvc.Approve(entservice.DecideQuotaRequestInput{
		TenantId:       0,
		RequestId:      item.Id,
		ActorId:        1001,
		ApprovedQuota:  quotaRequestInt64Ptr(150),
		ApprovalReason: "approved by admin fallback",
	})
	require.NoError(t, err)
	require.Equal(t, "fulfilled", result.Request.Status)
	require.Equal(t, int64(150), result.Request.ApprovedQuota)
	require.NotNil(t, result.Allocation)
	require.NotZero(t, result.Request.AllocationId)
	require.Equal(t, result.Request.AllocationId, result.Allocation.Id)
	require.NotZero(t, result.Request.ApprovedAt)
	require.NotZero(t, result.Request.ProcessedAt)
}

func TestQuotaRequestApproveKeepsFulfilledWhenNotificationChannelMissing(t *testing.T) {
	_, db := newQuotaAllocationTestService(t)
	reqSvc := entservice.NewQuotaRequestService(db)

	item, err := reqSvc.Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2001,
		RequestedQuota:     200,
		RequestReason:      "need more quota",
	})
	require.NoError(t, err)

	result, err := reqSvc.Approve(entservice.DecideQuotaRequestInput{
		TenantId:       0,
		RequestId:      item.Id,
		ActorId:        1001,
		ApprovedQuota:  quotaRequestInt64Ptr(150),
		ApprovalReason: "approved by admin fallback",
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.QuotaRequestStatusFulfilled, result.Request.Status)
	require.NotNil(t, result.Allocation)
	require.Equal(t, result.Request.AllocationId, result.Allocation.Id)

	dispatchSvc := entservice.NewGovernanceNotificationDispatchServiceForTest(
		db,
		func() time.Time { return time.Unix(time.Now().Unix()+10, 0) },
		func(string, string, dto.Notify) error {
			t.Fatal("webhook sender should not run without a configured governance channel")
			return nil
		},
	)
	_, err = dispatchSvc.DispatchDueDeliveries(context.Background(), 10)
	require.NoError(t, err)

	var storedRequest entmodel.QuotaRequest
	require.NoError(t, db.Where("id = ?", item.Id).First(&storedRequest).Error)
	require.Equal(t, entmodel.QuotaRequestStatusFulfilled, storedRequest.Status)
	require.Equal(t, result.Request.AllocationId, storedRequest.AllocationId)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", result.Request.AllocationId).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusActive, allocation.Status)

	var delivery entmodel.GovernanceNotificationDelivery
	require.NoError(t, db.Where(
		"source_type = ? AND source_id = ? AND action_type = ?",
		entservice.GovernanceSourceQuotaRequest,
		item.Id,
		entservice.GovernanceActionQuotaRequestApproved,
	).First(&delivery).Error)
	require.Equal(t, entmodel.GovernanceNotificationStatusUnconfigured, delivery.Status)
	require.Equal(t, int64(0), delivery.NextRetryAt)
	trace, err := delivery.ParsedTracePayload()
	require.NoError(t, err)
	require.NotNil(t, trace)
	require.Equal(t, entmodel.QuotaRequestStatusFulfilled, trace.Status)
}

func TestQuotaRequestApproveByDepartmentOwnerCreatesAllocationOnce(t *testing.T) {
	_, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Create(&model.User{
		Id:       3001,
		Username: "dept-owner",
		Password: "pwd",
		Group:    "default",
		AffCode:  "dept-owner-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       3001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualGrant,
		Effect:       constant.EnterpriseDepartmentRoleEffectAllow,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	reqSvc := entservice.NewQuotaRequestService(db)
	item, err := reqSvc.Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2001,
		RequestedQuota:     240,
		RequestReason:      "need project quota",
	})
	require.NoError(t, err)

	first, err := reqSvc.Approve(entservice.DecideQuotaRequestInput{
		TenantId:       0,
		RequestId:      item.Id,
		ActorId:        3001,
		ApprovedQuota:  quotaRequestInt64Ptr(180),
		ApprovalReason: "approve smaller amount",
	})
	require.NoError(t, err)
	require.NotNil(t, first.Allocation)
	require.Equal(t, int64(180), first.Request.ApprovedQuota)

	second, err := reqSvc.Approve(entservice.DecideQuotaRequestInput{
		TenantId:       0,
		RequestId:      item.Id,
		ActorId:        3001,
		ApprovedQuota:  quotaRequestInt64Ptr(180),
		ApprovalReason: "retry approval",
	})
	require.NoError(t, err)
	require.NotNil(t, second.Allocation)
	require.Equal(t, first.Request.AllocationId, second.Request.AllocationId)

	var allocations []entmodel.QuotaAllocation
	require.NoError(t, db.Find(&allocations).Error)
	require.Len(t, allocations, 1)
}

func TestQuotaRequestRejectRequiresReason(t *testing.T) {
	_, db := newQuotaAllocationTestService(t)
	reqSvc := entservice.NewQuotaRequestService(db)

	item, err := reqSvc.Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2001,
		RequestedQuota:     100,
		RequestReason:      "need quota",
	})
	require.NoError(t, err)

	_, err = reqSvc.Reject(entservice.DecideQuotaRequestInput{
		TenantId:  0,
		RequestId: item.Id,
		ActorId:   1001,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaRequestRejectedReasonRequired)

	result, err := reqSvc.Reject(entservice.DecideQuotaRequestInput{
		TenantId:       0,
		RequestId:      item.Id,
		ActorId:        1001,
		RejectedReason: "budget unavailable",
	})
	require.NoError(t, err)
	require.Equal(t, "rejected", result.Request.Status)
	require.Nil(t, result.Allocation)
	require.NotZero(t, result.Request.RejectedAt)
	require.NotZero(t, result.Request.ProcessedAt)
}

func quotaRequestInt64Ptr(value int64) *int64 {
	return &value
}

func TestQuotaRequestSubmitRejectsRequesterOutsideDepartment(t *testing.T) {
	_, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Delete(&entmodel.UserDepartment{}, "user_id = ?", 2001).Error)

	_, err := entservice.NewQuotaRequestService(db).Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2001,
		RequestedQuota:     100,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaRequestDepartmentMembershipRequired)
}

func TestQuotaRequestSubmitRejectsInactiveBudget(t *testing.T) {
	_, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Update("status", entmodel.DepartmentBudgetStatusPaused).Error)

	_, err := entservice.NewQuotaRequestService(db).Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2001,
		RequestedQuota:     100,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationBudgetInactive)
}

func TestQuotaRequestIdempotencyScopedByTenantRequesterAndKey(t *testing.T) {
	_, db := newQuotaAllocationTestService(t)
	seedQuotaAllocationMember(t, db, 2002, 1)
	reqSvc := entservice.NewQuotaRequestService(db)

	first, err := reqSvc.Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2001,
		RequestedQuota:     100,
		IdempotencyKey:     "same-key",
	})
	require.NoError(t, err)
	replay, err := reqSvc.Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2001,
		RequestedQuota:     120,
		IdempotencyKey:     "same-key",
	})
	require.NoError(t, err)
	require.Equal(t, first.Id, replay.Id)

	secondUser, err := reqSvc.Submit(entservice.SubmitQuotaRequestInput{
		TenantId:           0,
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         entservice.QuotaRequestBudgetModeDepartment,
		RequesterUserId:    2002,
		RequestedQuota:     120,
		IdempotencyKey:     "same-key",
	})
	require.NoError(t, err)
	require.NotEqual(t, first.Id, secondUser.Id)
}

func TestQuotaRequestCapabilityReturnsMixedActiveBudgetPools(t *testing.T) {
	_, db := newQuotaAllocationTestService(t)
	startedAt := int64(1700000000)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:             2,
		TenantId:       0,
		DepartmentId:   1,
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		Status:         entmodel.DepartmentBudgetStatusActive,
		Remaining:      300,
		AllocatedTotal: 0,
		CycleQuota:     300,
		CycleType:      "monthly",
		CycleStartedAt: startedAt,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           3,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusPaused,
		TotalQuota:   100,
		Remaining:    100,
	}).Error)

	capability, err := entservice.NewQuotaRequestService(db).GetCapability(0, 1, 2001)
	require.NoError(t, err)
	require.True(t, capability.CanSubmit)
	require.Len(t, capability.Budgets, 2)

	byID := map[int]string{}
	for _, budget := range capability.Budgets {
		byID[budget.Id] = budget.Type
	}
	require.Equal(t, entmodel.DepartmentBudgetTypeBalance, byID[1])
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, byID[2])
	require.NotContains(t, byID, 3)

	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 3).Update("status", entmodel.DepartmentBudgetStatusActive).Error)
	resumedCapability, err := entservice.NewQuotaRequestService(db).GetCapability(0, 1, 2001)
	require.NoError(t, err)
	resumedByID := map[int]string{}
	for _, budget := range resumedCapability.Budgets {
		resumedByID[budget.Id] = budget.Status
	}
	require.Equal(t, entmodel.DepartmentBudgetStatusActive, resumedByID[3])
}

func init() {
	common.RedisEnabled = false
}
