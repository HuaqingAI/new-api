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

func TestCreateDepartmentBudgetKeepsExistingTypeImmutable(t *testing.T) {
	svc, _ := newDepartmentBudgetTestService(t)
	total := int64(1000)
	_, err := svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	})
	require.NoError(t, err)

	cycleQuota := int64(200)
	startedAt := int64(1700000000)
	_, err = svc.Create(1, entservice.CreateDepartmentBudgetInput{
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "monthly",
		CycleStartedAt: &startedAt,
	})
	require.ErrorIs(t, err, entservice.ErrDepartmentBudgetTypeImmutable)
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
