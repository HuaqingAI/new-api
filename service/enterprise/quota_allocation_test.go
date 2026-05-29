package enterprise_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

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
		TenantId:       0,
		UserId:         2001,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
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

func seedQuotaAllocationMember(t *testing.T, db *gorm.DB, userId int, departmentId int) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:       userId,
		Username: fmt.Sprintf("member-%d", userId),
		Password: "pwd",
		Group:    "default",
		AffCode:  fmt.Sprintf("member-%d-aff", userId),
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         userId,
		DepartmentId:   departmentId,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
	}).Error)
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

func TestCreateQuotaAllocationSubscriptionBudgetTracksAllocatedTotal(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(600),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(600),
		"cycle_type":       "monthly",
		"cycle_started_at": time.Now().Unix(),
	}).Error)

	item, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     250,
	})
	require.NoError(t, err)
	require.NotZero(t, item.Id)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(250), budget.AllocatedTotal)
	require.Equal(t, int64(350), budget.Remaining)
}

func TestCreateQuotaAllocationReturnsSpecificBudgetReasons(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)

	_, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     1500,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationBudgetInsufficient)
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationBalanceRemainingInsufficient)

	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(100),
		"allocated_total":  int64(500),
		"cycle_quota":      int64(600),
		"cycle_type":       "monthly",
		"cycle_started_at": time.Now().Unix(),
	}).Error)
	_, err = svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     200,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationBudgetInsufficient)
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationSubscriptionCycleAllocatedExceeded)
}

func TestCreateQuotaAllocationConcurrentBalanceBudget(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)
	const workers = 50
	const quotaPerRequest int64 = 30
	for i := 0; i < workers-1; i++ {
		seedQuotaAllocationMember(t, db, 3000+i, 1)
	}

	type result struct {
		err error
	}
	results := make(chan result, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		userID := 2001
		if i > 0 {
			userID = 3000 + (i - 1)
		}
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			_, err := svc.Create(entservice.CreateQuotaAllocationInput{
				TenantId:           0,
				DepartmentBudgetId: 1,
				DepartmentId:       1,
				TargetUserId:       uid,
				ActorId:            1001,
				CommittedQuota:     quotaPerRequest,
			})
			results <- result{err: err}
		}(userID)
	}
	wg.Wait()
	close(results)

	successCount := 0
	failureCount := 0
	for item := range results {
		if item.err == nil {
			successCount++
			continue
		}
		failureCount++
		require.ErrorIs(t, item.err, entservice.ErrQuotaAllocationBudgetInsufficient)
		require.ErrorIs(t, item.err, entservice.ErrQuotaAllocationBalanceRemainingInsufficient)
	}
	require.Equal(t, 33, successCount)
	require.Equal(t, workers-successCount, failureCount)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(10), budget.Remaining)

	var allocations []entmodel.QuotaAllocation
	require.NoError(t, db.Order("id ASC").Find(&allocations).Error)
	require.Len(t, allocations, successCount)

	var wallets []model.UserSubscription
	require.NoError(t, db.Where("source_type = ?", model.SubscriptionSourceTypeEnterprise).Find(&wallets).Error)
	require.Len(t, wallets, successCount)
	walletIDs := make(map[int]struct{}, len(wallets))
	for _, wallet := range wallets {
		require.NotZero(t, wallet.SourceAllocationId)
		_, exists := walletIDs[wallet.Id]
		require.False(t, exists, "duplicate wallet id %d", wallet.Id)
		walletIDs[wallet.Id] = struct{}{}
	}
}

func TestCreateQuotaAllocationConcurrentSubscriptionBudget(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(600),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(600),
		"cycle_type":       "weekly",
		"cycle_started_at": time.Now().Unix(),
	}).Error)
	const workers = 50
	const quotaPerRequest int64 = 20
	for i := 0; i < workers-1; i++ {
		seedQuotaAllocationMember(t, db, 4000+i, 1)
	}

	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		userID := 2001
		if i > 0 {
			userID = 4000 + (i - 1)
		}
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			_, err := svc.Create(entservice.CreateQuotaAllocationInput{
				TenantId:           0,
				DepartmentBudgetId: 1,
				DepartmentId:       1,
				TargetUserId:       uid,
				ActorId:            1001,
				CommittedQuota:     quotaPerRequest,
			})
			results <- err
		}(userID)
	}
	wg.Wait()
	close(results)

	successCount := 0
	for err := range results {
		if err == nil {
			successCount++
			continue
		}
		require.ErrorIs(t, err, entservice.ErrQuotaAllocationBudgetInsufficient)
		require.ErrorIs(t, err, entservice.ErrQuotaAllocationSubscriptionCycleAllocatedExceeded)
	}
	require.Equal(t, 30, successCount)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(600), budget.AllocatedTotal)
	require.Equal(t, int64(0), budget.Remaining)

	var committedTotal int64
	require.NoError(t, db.Model(&entmodel.QuotaAllocation{}).Select("COALESCE(SUM(committed_quota), 0)").Scan(&committedTotal).Error)
	require.Equal(t, budget.AllocatedTotal, committedTotal)

	var wallets []model.UserSubscription
	require.NoError(t, db.Where("source_type = ?", model.SubscriptionSourceTypeEnterprise).Find(&wallets).Error)
	require.Len(t, wallets, successCount)
	allocationIDs := make(map[int]struct{}, len(wallets))
	for _, wallet := range wallets {
		require.NotZero(t, wallet.SourceAllocationId)
		_, exists := allocationIDs[wallet.SourceAllocationId]
		require.False(t, exists, "duplicate allocation backlink %d", wallet.SourceAllocationId)
		allocationIDs[wallet.SourceAllocationId] = struct{}{}
	}
}
