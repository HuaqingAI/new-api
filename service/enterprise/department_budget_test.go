package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDepartmentBudgetTestService(t *testing.T) (*entservice.DepartmentBudgetService, *gorm.DB) {
	t.Helper()

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	require.NoError(t, entmodel.AutoMigrate(db))
	require.NoError(t, db.Create(&entmodel.Department{Id: 1, TenantId: 0, Name: "Engineering", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	return entservice.NewDepartmentBudgetService(db), db
}

func TestCreateDepartmentBudgetBalance(t *testing.T) {
	svc, _ := newDepartmentBudgetTestService(t)
	total := int64(1000)

	item, err := svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetTypeBalance, item.Type)
	require.Equal(t, int64(1000), item.TotalQuota)
	require.Equal(t, int64(1000), item.Remaining)
	require.Equal(t, float64(0), item.UsageRatio)
	require.Equal(t, "healthy", string(item.ThresholdState))
}

func TestCreateDepartmentBudgetSubscription(t *testing.T) {
	svc, _ := newDepartmentBudgetTestService(t)
	quota := int64(500)
	startedAt := int64(1700000000)

	item, err := svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		CycleQuota:     &quota,
		CycleType:      "monthly",
		CycleStartedAt: &startedAt,
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, item.Type)
	require.Equal(t, int64(500), item.CycleQuota)
	require.Equal(t, int64(500), item.Remaining)
	require.Equal(t, "monthly", item.CycleType)
}

func TestPublicBudgetPoolLifecycle(t *testing.T) {
	svc, _ := newDepartmentBudgetTestService(t)
	total := int64(1000)

	created, err := svc.CreatePublic(entservice.CreateDepartmentBudgetInput{
		TenantId:   0,
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	}, "Company wide")
	require.NoError(t, err)
	require.Equal(t, 0, created.DepartmentId)
	require.Equal(t, entmodel.DepartmentBudgetScopePublic, created.ScopeType)
	require.True(t, created.IsPublic)
	require.Equal(t, "Company wide", created.Name)

	activePools, err := svc.ListPublic(0, false)
	require.NoError(t, err)
	require.Len(t, activePools, 1)

	paused, err := svc.UpdatePublicStatus(0, created.Id, entmodel.DepartmentBudgetStatusActive, entmodel.DepartmentBudgetStatusPaused)
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetStatusPaused, paused.Status)
	activePools, err = svc.ListPublic(0, false)
	require.NoError(t, err)
	require.Empty(t, activePools)

	_, err = svc.UpdatePublicStatus(1, created.Id, entmodel.DepartmentBudgetStatusPaused, entmodel.DepartmentBudgetStatusActive)
	require.ErrorIs(t, err, entservice.ErrPublicBudgetNotFound)

	_, err = svc.UpdatePublicStatus(0, created.Id, entmodel.DepartmentBudgetStatusPaused, entmodel.DepartmentBudgetStatusActive)
	require.NoError(t, err)
	resizedTotal := int64(1500)
	resized, err := svc.ResizePublic(0, created.Id, entservice.ResizeDepartmentBudgetInput{TotalQuota: &resizedTotal})
	require.NoError(t, err)
	require.Equal(t, int64(1500), resized.TotalQuota)
	require.Equal(t, int64(1500), resized.Remaining)
}

func TestCreateDepartmentBudgetRejectsInvalidInputs(t *testing.T) {
	svc, _ := newDepartmentBudgetTestService(t)
	zero := int64(0)
	startedAt := int64(1700000000)

	_, err := svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &zero,
	})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetInvalidQuota)

	_, err = svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		CycleQuota:     &zero,
		CycleType:      "daily",
		CycleStartedAt: &startedAt,
	})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetInvalidCycleQuota)

	quota := int64(100)
	_, err = svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		CycleQuota:     &quota,
		CycleType:      "custom",
		CycleStartedAt: &startedAt,
	})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetInvalidCustomSeconds)
}

func TestDepartmentBudgetLifecyclePauseAndResume(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           40,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    700,
	}).Error)

	paused, err := svc.Pause(1, 40, 0)
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetStatusPaused, paused.Status)
	require.Equal(t, int64(1000), paused.TotalQuota)
	require.Equal(t, int64(700), paused.Remaining)

	_, err = svc.Pause(1, 40, 0)
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetStatusTransitionInvalid)

	resumed, err := svc.Resume(1, 40, 0)
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetStatusActive, resumed.Status)

	_, err = svc.Resume(1, 40, 0)
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetStatusTransitionInvalid)
}

func TestDepartmentBudgetResizeBalancePreservesUsedBoundary(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           41,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    700,
	}).Error)

	total := int64(1200)
	expanded, err := svc.Resize(1, 41, 0, entservice.ResizeDepartmentBudgetInput{TotalQuota: &total})
	require.NoError(t, err)
	require.Equal(t, int64(1200), expanded.TotalQuota)
	require.Equal(t, int64(900), expanded.Remaining)

	total = int64(500)
	shrunk, err := svc.Resize(1, 41, 0, entservice.ResizeDepartmentBudgetInput{TotalQuota: &total})
	require.NoError(t, err)
	require.Equal(t, int64(500), shrunk.TotalQuota)
	require.Equal(t, int64(200), shrunk.Remaining)

	total = int64(299)
	_, err = svc.Resize(1, 41, 0, entservice.ResizeDepartmentBudgetInput{TotalQuota: &total})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetResizeBelowCommitted)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 41).First(&budget).Error)
	require.Equal(t, int64(500), budget.TotalQuota)
	require.Equal(t, int64(200), budget.Remaining)
}

func TestDepartmentBudgetResizeSubscriptionPreservesAllocatedBoundary(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:             42,
		TenantId:       0,
		DepartmentId:   1,
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		Status:         entmodel.DepartmentBudgetStatusActive,
		CycleQuota:     1000,
		Remaining:      600,
		AllocatedTotal: 400,
		CycleType:      "monthly",
		CycleStartedAt: 1700000000,
	}).Error)

	cycleQuota := int64(1500)
	expanded, err := svc.Resize(1, 42, 0, entservice.ResizeDepartmentBudgetInput{CycleQuota: &cycleQuota})
	require.NoError(t, err)
	require.Equal(t, int64(1500), expanded.CycleQuota)
	require.Equal(t, int64(1100), expanded.Remaining)

	cycleQuota = int64(450)
	shrunk, err := svc.Resize(1, 42, 0, entservice.ResizeDepartmentBudgetInput{CycleQuota: &cycleQuota})
	require.NoError(t, err)
	require.Equal(t, int64(450), shrunk.CycleQuota)
	require.Equal(t, int64(50), shrunk.Remaining)

	cycleQuota = int64(399)
	_, err = svc.Resize(1, 42, 0, entservice.ResizeDepartmentBudgetInput{CycleQuota: &cycleQuota})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetResizeBelowCommitted)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 42).First(&budget).Error)
	require.Equal(t, int64(450), budget.CycleQuota)
	require.Equal(t, int64(50), budget.Remaining)
}

func TestDepartmentBudgetResizeRejectsTypeMismatchedAndInvalidCapacity(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           43,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)

	cycleQuota := int64(1000)
	_, err := svc.Resize(1, 43, 0, entservice.ResizeDepartmentBudgetInput{CycleQuota: &cycleQuota})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetTypeImmutable)

	zero := int64(0)
	_, err = svc.Resize(1, 43, 0, entservice.ResizeDepartmentBudgetInput{TotalQuota: &zero})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetInvalidQuota)

	_, err = svc.Resize(2, 43, 0, entservice.ResizeDepartmentBudgetInput{TotalQuota: &cycleQuota})
	require.ErrorIs(t, err, entservice.ErrDepartmentNotFound)

	require.NoError(t, db.Create(&entmodel.Department{
		Id:       2,
		TenantId: 0,
		Name:     "Other",
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)
	_, err = svc.Resize(2, 43, 0, entservice.ResizeDepartmentBudgetInput{TotalQuota: &cycleQuota})
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationBudgetNotFound)
}

func TestDepartmentBudgetResizeRejectsInactiveBudget(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           44,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusPaused,
		TotalQuota:   1000,
		Remaining:    700,
	}).Error)

	total := int64(1200)
	_, err := svc.Resize(1, 44, 0, entservice.ResizeDepartmentBudgetInput{TotalQuota: &total})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetStatusTransitionInvalid)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 44).First(&budget).Error)
	require.Equal(t, entmodel.DepartmentBudgetStatusPaused, budget.Status)
	require.Equal(t, int64(1000), budget.TotalQuota)
	require.Equal(t, int64(700), budget.Remaining)
}

func TestCreateDepartmentBudgetAllowsMixedTypesInSameDepartment(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	total := int64(1000)
	balance, err := svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetTypeBalance, balance.Type)

	cycleQuota := int64(200)
	startedAt := int64(1700000000)
	subscription, err := svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "monthly",
		CycleStartedAt: &startedAt,
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, subscription.Type)
	require.NotEqual(t, balance.Id, subscription.Id)

	var budgets []entmodel.DepartmentBudget
	require.NoError(t, db.Where("department_id = ?", 1).Order("id ASC").Find(&budgets).Error)
	require.Len(t, budgets, 2)
	require.Equal(t, entmodel.DepartmentBudgetTypeBalance, budgets[0].Type)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, budgets[1].Type)
}

func TestCreateDepartmentBudgetAllowsSubscriptionThenBalanceInSameDepartment(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	cycleQuota := int64(200)
	startedAt := int64(1700000000)
	subscription, err := svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "monthly",
		CycleStartedAt: &startedAt,
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, subscription.Type)

	total := int64(1000)
	balance, err := svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetTypeBalance, balance.Type)
	require.NotEqual(t, subscription.Id, balance.Id)

	var budgets []entmodel.DepartmentBudget
	require.NoError(t, db.Where("department_id = ?", 1).Order("id ASC").Find(&budgets).Error)
	require.Len(t, budgets, 2)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, budgets[0].Type)
	require.Equal(t, entmodel.DepartmentBudgetTypeBalance, budgets[1].Type)
}

func TestGetDepartmentBudgetReturnsLatest(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   100,
		Remaining:    90,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		TenantId:       0,
		DepartmentId:   1,
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		Status:         entmodel.DepartmentBudgetStatusActive,
		CycleQuota:     200,
		Remaining:      50,
		AllocatedTotal: 150,
		CycleType:      "weekly",
		CycleStartedAt: 1700000000,
	}).Error)

	item, err := svc.GetByDepartment(1, 0)
	require.NoError(t, err)
	require.NotNil(t, item)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, item.Type)
	require.Equal(t, float64(75), item.UsageRatio)
}

func TestListDepartmentBudgetsSortsAndCalculatesThresholds(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	warning := operation_setting.GetQuotaSetting().EnterpriseBudgetWarningThreshold
	critical := operation_setting.GetQuotaSetting().EnterpriseBudgetCriticalThreshold
	t.Cleanup(func() {
		operation_setting.GetQuotaSetting().EnterpriseBudgetWarningThreshold = warning
		operation_setting.GetQuotaSetting().EnterpriseBudgetCriticalThreshold = critical
	})
	operation_setting.GetQuotaSetting().EnterpriseBudgetWarningThreshold = 40
	operation_setting.GetQuotaSetting().EnterpriseBudgetCriticalThreshold = 70

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           10,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusPaused,
		TotalQuota:   1000,
		Remaining:    900,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:             11,
		TenantId:       0,
		DepartmentId:   1,
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		Status:         entmodel.DepartmentBudgetStatusActive,
		CycleQuota:     500,
		Remaining:      150,
		AllocatedTotal: 350,
		CycleType:      "monthly",
		CycleStartedAt: 1700000000,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           12,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   200,
		Remaining:    10,
	}).Error)

	result, err := svc.ListByDepartment(1, 0, entservice.DepartmentBudgetListQuery{
		SortBy:    "usage_ratio",
		SortOrder: "desc",
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 3)
	require.Equal(t, 12, result.Items[0].Id)
	require.Equal(t, float64(95), result.Items[0].UsageRatio)
	require.Equal(t, "critical", string(result.Items[0].ThresholdState))
	require.Equal(t, 11, result.Items[1].Id)
	require.Equal(t, float64(70), result.Items[1].UsageRatio)
	require.Equal(t, "critical", string(result.Items[1].ThresholdState))
	require.Equal(t, 10, result.Items[2].Id)
	require.Equal(t, float64(10), result.Items[2].UsageRatio)
	require.Equal(t, "healthy", string(result.Items[2].ThresholdState))
	require.Equal(t, 40, result.Thresholds.Warning)
	require.Equal(t, 70, result.Thresholds.Critical)
}

func TestListDepartmentBudgetsSupportsIncludeDescendantsAndDepartmentNames(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	parentID := 1
	require.NoError(t, db.Create(&entmodel.Department{
		Id:       2,
		TenantId: 0,
		Name:     "Platform",
		ParentId: &parentID,
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           30,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   100,
		Remaining:    70,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           31,
		TenantId:     0,
		DepartmentId: 2,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   50,
		Remaining:    10,
	}).Error)

	currentOnly, err := svc.ListByDepartment(1, 0, entservice.DepartmentBudgetListQuery{})
	require.NoError(t, err)
	require.Len(t, currentOnly.Items, 1)
	require.Equal(t, "Engineering", currentOnly.Items[0].DepartmentName)

	withDescendants, err := svc.ListByDepartment(1, 0, entservice.DepartmentBudgetListQuery{
		IncludeDescendants: true,
	})
	require.NoError(t, err)
	require.Len(t, withDescendants.Items, 2)
	require.Equal(t, []int{1, 2}, withDescendants.ScopeDepartmentIds)
	names := []string{withDescendants.Items[0].DepartmentName, withDescendants.Items[1].DepartmentName}
	require.ElementsMatch(t, []string{"Engineering", "Platform"}, names)
}

func TestGetDepartmentBudgetDetailAggregatesAllocationWalletsWithoutLogs(t *testing.T) {
	svc, db := newDepartmentBudgetTestService(t)
	require.NoError(t, db.Create(&model.User{
		Id:          2001,
		Username:    "alice",
		DisplayName: "Alice",
		Password:    "password123",
		AffCode:     "alice-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           21,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		ParentStatus: entmodel.DepartmentBudgetStatusPaused,
		TotalQuota:   1000,
		Remaining:    700,
	}).Error)
	require.NoError(t, db.Create(&entmodel.QuotaAllocation{
		Id:                 31,
		TenantId:           0,
		DepartmentBudgetId: 21,
		DepartmentId:       1,
		TargetUserId:       2001,
		WalletId:           41,
		CommittedQuota:     300,
		BudgetTypeSnapshot: entmodel.DepartmentBudgetTypeBalance,
		CycleTypeSnapshot:  "monthly",
		Status:             entmodel.QuotaAllocationStatusRevoked,
		ProcessedAt:        1700000900,
		Reason:             "manual revoke",
		CreatedAt:          1700000000,
		UpdatedAt:          1700001000,
	}).Error)
	require.NoError(t, db.Create(&model.UserSubscription{
		Id:                 41,
		UserId:             2001,
		AmountTotal:        300,
		AmountUsed:         120,
		Status:             "expired",
		SourceType:         model.SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 31,
		NextResetTime:      1700002000,
		EndTime:            1700003000,
	}).Error)

	detail, err := svc.GetDetail(1, 21, 0)
	require.NoError(t, err)
	require.NotNil(t, detail)
	require.Equal(t, 21, detail.Budget.Id)
	require.Len(t, detail.Wallets, 1)
	require.Equal(t, 31, detail.Wallets[0].AllocationId)
	require.Equal(t, 2001, detail.Wallets[0].TargetUserId)
	require.Equal(t, "alice", detail.Wallets[0].TargetUsername)
	require.Equal(t, "Alice", detail.Wallets[0].TargetDisplayName)
	require.Equal(t, int64(300), detail.Wallets[0].Quota)
	require.Equal(t, int64(180), detail.Wallets[0].RemainQuota)
	require.Equal(t, "expired", detail.Wallets[0].WalletStatus)
	require.Equal(t, 31, detail.Wallets[0].SourceAllocationId)
	require.Equal(t, 21, detail.Wallets[0].SourceParentBudgetId)
	require.Equal(t, entmodel.DepartmentBudgetStatusActive, detail.Wallets[0].SourceParentBudgetStatus)
}
