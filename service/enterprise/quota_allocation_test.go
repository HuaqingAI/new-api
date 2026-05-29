package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newQuotaAllocationTestService(t *testing.T) (*entservice.QuotaAllocationService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:quota-allocation-test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	oldDB := model.DB
	model.DB = db
	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	t.Cleanup(func() {
		model.DB = oldDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		_ = sqlDB.Close()
	})

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.SubscriptionPlan{}, &model.UserSubscription{}))
	require.NoError(t, entmodel.AutoMigrate(db))
	require.NoError(t, db.Create(&model.User{
		Id:       2001,
		Username: "member",
		Password: "pwd",
		Group:    "default",
		AffCode:  "member-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.Department{
		Id:       1,
		TenantId: 0,
		Name:     "Engineering",
		Status:   constant.DepartmentStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       2001,
		DepartmentId: 1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	return entservice.NewQuotaAllocationService(db), db
}

func TestCreateQuotaAllocationCreatesWalletAndLedger(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)

	item, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
		Reason:             "team budget",
	})
	require.NoError(t, err)
	require.Equal(t, int64(300), item.CommittedQuota)
	require.NotZero(t, item.WalletId)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", item.WalletId).First(&wallet).Error)
	require.Equal(t, model.SubscriptionSourceTypeEnterprise, wallet.SourceType)
	require.Equal(t, item.Id, wallet.SourceAllocationId)
	require.Equal(t, int64(300), wallet.AmountTotal)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(700), budget.Remaining)

	var ledger entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&ledger).Error)
	require.Contains(t, ledger.BeforeBudgetSnapshot, `"remaining":1000`)
	require.Contains(t, ledger.AfterBudgetSnapshot, `"remaining":700`)
}

func TestCreateQuotaAllocationRejectsUserOutsideDepartment(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Delete(&entmodel.UserDepartment{}, "user_id = ?", 2001).Error)

	_, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     100,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationUserOutOfDepartment)
}

func TestCreateQuotaAllocationRollsBackOnLedgerFailure(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)
	invalidJSON := string([]byte{0xff, 0xfe, 0xfd})
	originalMarshal := common.Marshal
	_ = invalidJSON
	_ = originalMarshal

	require.NoError(t, db.Exec("DROP TABLE enterprise_quota_allocations").Error)

	_, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     200,
	})
	require.Error(t, err)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(1000), budget.Remaining)

	var count int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestGetBudgetDepartmentReturnsBudgetOwnerDepartment(t *testing.T) {
	svc, _ := newQuotaAllocationTestService(t)

	departmentId, err := svc.GetBudgetDepartment(0, 1)
	require.NoError(t, err)
	require.Equal(t, 1, departmentId)
}
