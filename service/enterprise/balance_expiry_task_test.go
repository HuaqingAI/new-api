package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestExpireBalanceAllocationsMarksAllocationExpiredAndRefundsOnce(t *testing.T) {
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
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", item.WalletId).Updates(map[string]any{
		"amount_used":  int64(100),
		"end_time":     common.GetTimestamp() - 10,
		"updated_at":   common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Model(&entmodel.QuotaAllocation{}).Where("id = ?", item.Id).Update("expires_at_snapshot", common.GetTimestamp()-10).Error)

	count, err := entservice.ExpireBalanceAllocations(db, 50, common.GetTimestamp())
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusExpired, allocation.Status)
	require.NotZero(t, allocation.ProcessedAt)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(900), budget.Remaining)

	count, err = entservice.ExpireBalanceAllocations(db, 50, common.GetTimestamp()+1)
	require.NoError(t, err)
	require.Equal(t, 0, count)
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(900), budget.Remaining)
}

func TestExpireBalanceAllocationsProcessesPausedAllocation(t *testing.T) {
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
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("id = ?", item.WalletId).Updates(map[string]any{
		"amount_used": int64(120),
		"status":      "paused",
		"updated_at":  common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Model(&entmodel.QuotaAllocation{}).Where("id = ?", item.Id).Updates(map[string]any{
		"status":              entmodel.QuotaAllocationStatusPaused,
		"expires_at_snapshot": common.GetTimestamp() - 10,
	}).Error)

	count, err := entservice.ExpireBalanceAllocations(db, 50, common.GetTimestamp())
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", item.Id).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusExpired, allocation.Status)
	require.NotZero(t, allocation.ProcessedAt)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", item.WalletId).First(&wallet).Error)
	require.Equal(t, "expired", wallet.Status)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(880), budget.Remaining)
}
