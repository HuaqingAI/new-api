package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDepartmentBudgetTestService(t *testing.T) (*entservice.DepartmentBudgetService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
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
		Remaining:      200,
		CycleType:      "weekly",
		CycleStartedAt: 1700000000,
	}).Error)

	item, err := svc.GetByDepartment(1, 0)
	require.NoError(t, err)
	require.NotNil(t, item)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, item.Type)
}
