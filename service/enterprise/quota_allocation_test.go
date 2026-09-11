package enterprise_test

import (
	"fmt"
	"strings"
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

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
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
	require.NoError(t, db.Create(&model.User{
		Id:       1001,
		Username: "admin",
		Password: "pwd",
		Group:    "default",
		AffCode:  "admin-aff",
		Role:     common.RoleAdminUser,
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

func TestRevokeBalanceQuotaAllocationRecoversUnspentAndMarksWalletRevoked(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)

	item, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", item.WalletId).Update("amount_used", int64(120)).Error)

	revoked, err := svc.Revoke(entservice.RevokeQuotaAllocationInput{
		TenantId:      0,
		DepartmentId:  1,
		AllocationId:  item.Id,
		ActorId:       1001,
		RevokeReason:  "cleanup",
		TriggeredBy:   entservice.QuotaAllocationProcessTriggerManual,
		TriggeredTime: common.GetTimestamp(),
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.QuotaAllocationStatusRevoked, revoked.Status)
	require.NotZero(t, revoked.ProcessedAt)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(880), budget.Remaining)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", item.WalletId).First(&wallet).Error)
	require.Equal(t, "revoked", wallet.Status)
	require.Equal(t, int64(300), wallet.AmountTotal)
	require.Equal(t, int64(120), wallet.AmountUsed)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusRevoked, allocation.Status)
	require.NotZero(t, allocation.ProcessedAt)
	require.Contains(t, allocation.Reason, "cleanup")
}

func TestRevokeSubscriptionQuotaAllocationReleasesCommitmentWithoutRefundingUsage(t *testing.T) {
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
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", item.WalletId).Updates(map[string]any{
		"amount_used":     int64(80),
		"next_reset_time": time.Now().Add(time.Hour).Unix(),
		"last_reset_time": time.Now().Unix(),
	}).Error)

	revoked, err := svc.Revoke(entservice.RevokeQuotaAllocationInput{
		TenantId:      0,
		DepartmentId:  1,
		AllocationId:  item.Id,
		ActorId:       1001,
		TriggeredBy:   entservice.QuotaAllocationProcessTriggerManual,
		TriggeredTime: common.GetTimestamp(),
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.QuotaAllocationStatusRevoked, revoked.Status)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(0), budget.AllocatedTotal)
	require.Equal(t, int64(600), budget.Remaining)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", item.WalletId).First(&wallet).Error)
	require.Equal(t, "revoked", wallet.Status)
	require.NotZero(t, wallet.NextResetTime)
}

func TestRevokeQuotaAllocationIsIdempotent(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)

	item, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", item.WalletId).Update("amount_used", int64(100)).Error)

	first, err := svc.Revoke(entservice.RevokeQuotaAllocationInput{
		TenantId:      0,
		DepartmentId:  1,
		AllocationId:  item.Id,
		ActorId:       1001,
		TriggeredBy:   entservice.QuotaAllocationProcessTriggerManual,
		TriggeredTime: common.GetTimestamp(),
	})
	require.NoError(t, err)

	second, err := svc.Revoke(entservice.RevokeQuotaAllocationInput{
		TenantId:      0,
		DepartmentId:  1,
		AllocationId:  item.Id,
		ActorId:       1001,
		TriggeredBy:   entservice.QuotaAllocationProcessTriggerManual,
		TriggeredTime: common.GetTimestamp() + 1,
	})
	require.NoError(t, err)
	require.Equal(t, first.ProcessedAt, second.ProcessedAt)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(900), budget.Remaining)
}

func TestSupersedeQuotaAllocationCreatesNewWalletAndClosesOldChain(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)

	created, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
		Reason:             "initial",
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", created.WalletId).Update("amount_used", int64(120)).Error)

	superseded, err := svc.Supersede(entservice.SupersedeQuotaAllocationInput{
		TenantId:          0,
		DepartmentId:      1,
		AllocationId:      created.Id,
		ActorId:           1002,
		NewCommittedQuota: 250,
		Reason:            "downsize",
	})
	require.NoError(t, err)
	require.NotEqual(t, created.Id, superseded.Id)
	require.NotEqual(t, created.WalletId, superseded.WalletId)
	require.Equal(t, entmodel.QuotaAllocationStatusActive, superseded.Status)
	require.Equal(t, created.Id, superseded.SupersedesAllocationId)

	var oldAllocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", created.Id).First(&oldAllocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusSuperseded, oldAllocation.Status)
	require.Equal(t, superseded.Id, oldAllocation.SupersededById)
	require.NotZero(t, oldAllocation.ProcessedAt)

	var newAllocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", superseded.Id).First(&newAllocation).Error)
	require.Equal(t, created.Id, newAllocation.SupersedesAllocationId)

	var oldWallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", created.WalletId).First(&oldWallet).Error)
	require.Equal(t, "cancelled", oldWallet.Status)

	var newWallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", superseded.WalletId).First(&newWallet).Error)
	require.Equal(t, "active", newWallet.Status)
	require.Equal(t, superseded.Id, newWallet.SourceAllocationId)
	require.Equal(t, int64(250), newWallet.AmountTotal)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(630), budget.Remaining)
}

func TestDirectPublicBudgetAllocationIsRejected(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           4,
		TenantId:     0,
		DepartmentId: 0,
		ScopeType:    entmodel.DepartmentBudgetScopePublic,
		Name:         "Company wide",
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)

	_, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 4,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
	})
	require.ErrorIs(t, err, entservice.ErrPublicBudgetManualAllocationDenied)
}

func TestCancelQuotaAllocationRefundsOnlyRecoverableBalance(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)

	created, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", created.WalletId).Update("amount_used", int64(180)).Error)

	cancelled, err := svc.Cancel(entservice.CancelQuotaAllocationInput{
		TenantId:      0,
		DepartmentId:  1,
		AllocationId:  created.Id,
		ActorId:       1003,
		Reason:        "member left",
		ProcessedTime: common.GetTimestamp(),
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.QuotaAllocationStatusRevoked, cancelled.Status)
	require.NotZero(t, cancelled.ProcessedAt)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(820), budget.Remaining)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", created.Id).First(&allocation).Error)
	require.Contains(t, allocation.RevokeReason, "member left")

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", created.WalletId).First(&wallet).Error)
	require.Equal(t, "revoked", wallet.Status)
}

func TestReclaimQuotaAllocationRecordsReclaimedQuotaAndIsIdempotent(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)

	created, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", created.WalletId).Update("amount_used", int64(125)).Error)

	now := common.GetTimestamp()
	first, err := svc.Reclaim(entservice.ReclaimQuotaAllocationInput{
		TenantId:      0,
		DepartmentId:  1,
		AllocationId:  created.Id,
		ActorId:       1004,
		Reason:        "manual reclaim",
		ProcessedTime: now,
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.QuotaAllocationStatusClosed, first.Status)
	require.Equal(t, int64(175), first.ReclaimedQuota)
	require.NotZero(t, first.ProcessedAt)

	second, err := svc.Reclaim(entservice.ReclaimQuotaAllocationInput{
		TenantId:      0,
		DepartmentId:  1,
		AllocationId:  created.Id,
		ActorId:       1005,
		Reason:        "retry",
		ProcessedTime: now + 60,
	})
	require.NoError(t, err)
	require.Equal(t, first.ProcessedAt, second.ProcessedAt)
	require.Equal(t, first.ReclaimedQuota, second.ReclaimedQuota)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(875), budget.Remaining)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", created.Id).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusClosed, allocation.Status)
	require.Equal(t, int64(175), allocation.ReclaimedQuota)
	require.Equal(t, "manual_reclaim", allocation.ProcessedSource)
	require.Contains(t, allocation.RevokeReason, "manual reclaim")

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", created.WalletId).First(&wallet).Error)
	require.Equal(t, "cancelled", wallet.Status)
}
