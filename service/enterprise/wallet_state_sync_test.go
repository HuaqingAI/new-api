package enterprise_test

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestSyncWalletStatesRevokesActiveChildrenForRevokedBudget(t *testing.T) {
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
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Updates(map[string]any{
		"status": entmodel.DepartmentBudgetStatusRevoked,
	}).Error)

	count, err := entservice.SyncWalletStates(db, 50)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusRevoked, allocation.Status)
	require.NotZero(t, allocation.ProcessedAt)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", item.WalletId).First(&wallet).Error)
	require.Equal(t, "revoked", wallet.Status)
}

func TestSyncWalletStatesPausesChildrenForPausedBudget(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(600),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(600),
		"cycle_type":       "weekly",
		"cycle_started_at": time.Now().Unix(),
	}).Error)
	item, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Update("status", entmodel.DepartmentBudgetStatusPaused).Error)

	count, err := entservice.SyncWalletStates(db, 50)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusPaused, allocation.Status)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", item.WalletId).First(&wallet).Error)
	require.Equal(t, "paused", wallet.Status)
}

func TestSyncWalletStatesRevokedBudgetIsIdempotentAcrossRuns(t *testing.T) {
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
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Update("status", entmodel.DepartmentBudgetStatusRevoked).Error)

	firstCount, err := entservice.SyncWalletStates(db, 50)
	require.NoError(t, err)
	require.Equal(t, 1, firstCount)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(880), budget.Remaining)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	firstProcessedAt := allocation.ProcessedAt
	require.NotZero(t, firstProcessedAt)

	secondCount, err := entservice.SyncWalletStates(db, 50)
	require.NoError(t, err)
	require.Equal(t, 0, secondCount)

	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(880), budget.Remaining)
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	require.Equal(t, firstProcessedAt, allocation.ProcessedAt)
}

func TestSyncWalletStatesRevokesChildrenForExpiredBudget(t *testing.T) {
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
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Update("status", entmodel.DepartmentBudgetStatusExpired).Error)

	count, err := entservice.SyncWalletStates(db, 50)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusRevoked, allocation.Status)
	require.NotZero(t, allocation.ProcessedAt)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", item.WalletId).First(&wallet).Error)
	require.Equal(t, "revoked", wallet.Status)
}

func TestSyncWalletStatesRevokesPreviouslyPausedChildrenForRevokedBudget(t *testing.T) {
	svc, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(600),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(600),
		"cycle_type":       "weekly",
		"cycle_started_at": time.Now().Unix(),
	}).Error)
	item, err := svc.Create(entservice.CreateQuotaAllocationInput{
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		ActorId:            1001,
		CommittedQuota:     300,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Update("status", entmodel.DepartmentBudgetStatusPaused).Error)

	count, err := entservice.SyncWalletStates(db, 50)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Update("status", entmodel.DepartmentBudgetStatusRevoked).Error)
	count, err = entservice.SyncWalletStates(db, 50)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusRevoked, allocation.Status)
	require.NotZero(t, allocation.ProcessedAt)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", item.WalletId).First(&wallet).Error)
	require.Equal(t, "revoked", wallet.Status)
}
