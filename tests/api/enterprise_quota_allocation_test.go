package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestEnterpriseQuotaAllocationAPICreatesWalletWithoutAdminActionDoubleWrite(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 2001, Username: "member", Password: "password123", Group: "vip", AffCode: "member-api"}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.UserDepartment{
		TenantId:     0,
		UserId:       2001,
		DepartmentId: 1,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         modelenterprise.DepartmentBudgetTypeBalance,
		Status:       modelenterprise.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	create := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/quota-allocations", cookies, dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		CommittedQuota:     int64PtrValue(300),
	})
	payload := decodeDepartmentMembersAPIResponse(t, create)
	require.True(t, payload.Success, payload.Message)
	require.Contains(t, string(payload.Data), `"committed_quota":300`)

	list := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/quota-allocations?department_budget_id=1&department_id=1", cookies)
	listPayload := decodeDepartmentMembersAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	require.Contains(t, string(listPayload.Data), `"wallet_id":`)

	var wallet model.UserSubscription
	require.NoError(t, fixture.db.Where("user_id = ? AND source_type = ?", 2001, "enterprise_allocation").First(&wallet).Error)
	require.Equal(t, 300, int(wallet.AmountTotal))

	var adminActionCount int64
	require.NoError(t, fixture.db.Model(&modelenterprise.AdminAction{}).Count(&adminActionCount).Error)
	require.Equal(t, int64(0), adminActionCount)
}

func TestEnterpriseQuotaAllocationAPIFailureLeavesNoWalletSideEffects(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 2004, Username: "member-four", Password: "password123", Group: "vip", AffCode: "member-four-api"}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.UserDepartment{
		TenantId:     0,
		UserId:       2004,
		DepartmentId: 1,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentBudget{
		Id:             1,
		TenantId:       0,
		DepartmentId:   1,
		Type:           modelenterprise.DepartmentBudgetTypeSubscription,
		Status:         modelenterprise.DepartmentBudgetStatusActive,
		CycleQuota:     100,
		Remaining:      10,
		AllocatedTotal: 90,
		CycleType:      "monthly",
		CycleStartedAt: 1700000000,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	create := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/quota-allocations", cookies, dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2004,
		CommittedQuota:     int64PtrValue(20),
	})
	payload := decodeDepartmentMembersAPIResponse(t, create)
	require.False(t, payload.Success)
	require.Equal(t, "enterprise.organization.enterprise_budget_insufficient", payload.Message)
	require.Contains(t, string(payload.Data), `"reason":"enterprise.organization.subscription_cycle_allocated_exceeded"`)

	var wallets int64
	require.NoError(t, fixture.db.Model(&model.UserSubscription{}).Count(&wallets).Error)
	require.Equal(t, int64(0), wallets)
	var allocations int64
	require.NoError(t, fixture.db.Model(&modelenterprise.QuotaAllocation{}).Count(&allocations).Error)
	require.Equal(t, int64(0), allocations)
}

func TestEnterpriseQuotaAllocationAPIBalanceFailureReturnsSpecificReason(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 2005, Username: "member-five", Password: "password123", Group: "vip", AffCode: "member-five-api"}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.UserDepartment{
		TenantId:     0,
		UserId:       2005,
		DepartmentId: 1,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         modelenterprise.DepartmentBudgetTypeBalance,
		Status:       modelenterprise.DepartmentBudgetStatusActive,
		TotalQuota:   100,
		Remaining:    10,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	create := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/quota-allocations", cookies, dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2005,
		CommittedQuota:     int64PtrValue(20),
	})
	payload := decodeDepartmentMembersAPIResponse(t, create)
	require.False(t, payload.Success)
	require.Equal(t, "enterprise.organization.enterprise_budget_insufficient", payload.Message)
	require.Contains(t, string(payload.Data), `"reason":"enterprise.organization.balance_remaining_insufficient"`)

	var wallets int64
	require.NoError(t, fixture.db.Model(&model.UserSubscription{}).Count(&wallets).Error)
	require.Equal(t, int64(0), wallets)
	var allocations int64
	require.NoError(t, fixture.db.Model(&modelenterprise.QuotaAllocation{}).Count(&allocations).Error)
	require.Equal(t, int64(0), allocations)
}

func TestEnterpriseQuotaAllocationAPIConcurrentListConsistent(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         modelenterprise.DepartmentBudgetTypeBalance,
		Status:       modelenterprise.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	const workers = 25
	const quotaPerRequest = 50
	for i := 0; i < workers; i++ {
		userID := 2100 + i
		require.NoError(t, fixture.db.Create(&model.User{Id: userID, Username: fmt.Sprintf("member-%d", userID), Password: "password123", Group: "vip", AffCode: fmt.Sprintf("member-%d", userID)}).Error)
		require.NoError(t, fixture.db.Create(&modelenterprise.UserDepartment{
			TenantId:     0,
			UserId:       userID,
			DepartmentId: 1,
			Status:       constant.EnterpriseMembershipStatusActive,
		}).Error)
	}
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	type createResult struct {
		success bool
		message string
		reason  string
	}
	results := make(chan createResult, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		userID := 2100 + i
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			create := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/quota-allocations", cookies, dtoenterprise.CreateQuotaAllocationRequest{
				DepartmentBudgetId: 1,
				DepartmentId:       1,
				TargetUserId:       uid,
				CommittedQuota:     int64PtrValue(quotaPerRequest),
			})
			payload := decodeDepartmentMembersAPIResponse(t, create)
			results <- createResult{
				success: payload.Success,
				message: payload.Message,
				reason:  string(payload.Data),
			}
		}(userID)
	}
	wg.Wait()
	close(results)

	successCount := 0
	failureCount := 0
	for result := range results {
		if result.success {
			successCount++
			continue
		}
		failureCount++
		require.Equal(t, "enterprise.organization.enterprise_budget_insufficient", result.message)
		require.Contains(t, result.reason, `"reason":"enterprise.organization.balance_remaining_insufficient"`)
	}
	require.Equal(t, 20, successCount)
	require.Equal(t, workers-successCount, failureCount)

	list := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/quota-allocations?department_budget_id=1&department_id=1", cookies)
	listPayload := decodeDepartmentMembersAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	require.Equal(t, successCount, strings.Count(string(listPayload.Data), `"wallet_id":`))

	var budget modelenterprise.DepartmentBudget
	require.NoError(t, fixture.db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(0), budget.Remaining)

	var allocations []modelenterprise.QuotaAllocation
	require.NoError(t, fixture.db.Order("id ASC").Find(&allocations).Error)
	require.Len(t, allocations, successCount)

	var wallets []model.UserSubscription
	require.NoError(t, fixture.db.Where("source_type = ?", model.SubscriptionSourceTypeEnterprise).Order("id ASC").Find(&wallets).Error)
	require.Len(t, wallets, successCount)
	allocationBacklinks := make(map[int]struct{}, len(wallets))
	for _, wallet := range wallets {
		require.NotZero(t, wallet.SourceAllocationId)
		_, exists := allocationBacklinks[wallet.SourceAllocationId]
		require.False(t, exists, "duplicate allocation backlink %d", wallet.SourceAllocationId)
		allocationBacklinks[wallet.SourceAllocationId] = struct{}{}
	}
}

func int64PtrValue(value int64) *int64 {
	return &value
}
