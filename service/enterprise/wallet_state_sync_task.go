package enterprise

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

func SyncWalletStates(db *gorm.DB, batchSize int) (int, error) {
	if db == nil {
		db = model.DB
	}
	if batchSize <= 0 {
		batchSize = 200
	}

	var budgets []entmodel.DepartmentBudget
	pausedAllocationBudgetIds := db.Model(&entmodel.QuotaAllocation{}).
		Select("department_budget_id").
		Where("status = ?", entmodel.QuotaAllocationStatusPaused)
	if err := db.Where("status IN ? OR (status = ? AND id IN (?))", []string{
		entmodel.DepartmentBudgetStatusPaused,
		entmodel.DepartmentBudgetStatusRevoked,
		entmodel.DepartmentBudgetStatusExpired,
	}, entmodel.DepartmentBudgetStatusActive, pausedAllocationBudgetIds).
		Order("id ASC").
		Limit(batchSize).
		Find(&budgets).Error; err != nil {
		return 0, err
	}
	if len(budgets) == 0 {
		return 0, nil
	}

	updated := 0
	for _, budget := range budgets {
		err := db.Transaction(func(tx *gorm.DB) error {
			if budget.Status == entmodel.DepartmentBudgetStatusPaused || budget.Status == entmodel.DepartmentBudgetStatusActive {
				return syncBudgetChildrenForStatusTx(tx, budget)
			}

			var allocationIds []int
			if err := tx.Model(&entmodel.QuotaAllocation{}).
				Where("department_budget_id = ? AND status IN ?", budget.Id, []string{
					entmodel.QuotaAllocationStatusActive,
					entmodel.QuotaAllocationStatusPaused,
				}).
				Pluck("id", &allocationIds).Error; err != nil {
				return err
			}
			for _, allocationId := range allocationIds {
				allocationService := NewQuotaAllocationService(tx)
				if _, err := allocationService.Revoke(RevokeQuotaAllocationInput{
					TenantId:      budget.TenantId,
					DepartmentId:  budget.DepartmentId,
					AllocationId:  allocationId,
					ActorId:       0,
					TriggeredBy:   QuotaAllocationProcessTriggerSync,
					TriggeredTime: common.GetTimestamp(),
				}); err != nil {
					return err
				}
				updated++
			}
			return nil
		})
		if err != nil {
			return updated, err
		}
	}
	return updated, nil
}

func syncBudgetChildrenForStatusTx(tx *gorm.DB, budget entmodel.DepartmentBudget) error {
	now := common.GetTimestamp()
	switch budget.Status {
	case entmodel.DepartmentBudgetStatusPaused:
		if err := tx.Model(&entmodel.QuotaAllocation{}).
			Where("department_budget_id = ? AND status = ?", budget.Id, entmodel.QuotaAllocationStatusActive).
			Updates(map[string]any{
				"status":     entmodel.QuotaAllocationStatusPaused,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserSubscription{}).
			Where("source_type = ? AND source_allocation_id IN (?) AND status = ?", model.SubscriptionSourceTypeEnterprise,
				tx.Model(&entmodel.QuotaAllocation{}).Select("id").Where("department_budget_id = ?", budget.Id),
				"active").
			Updates(map[string]any{
				"status":     "paused",
				"updated_at": now,
			}).Error
	case entmodel.DepartmentBudgetStatusActive:
		var restorableAllocationIds []int
		if err := tx.Model(&entmodel.QuotaAllocation{}).
			Joins("INNER JOIN user_subscriptions AS wallets ON wallets.id = enterprise_quota_allocations.wallet_id AND wallets.source_allocation_id = enterprise_quota_allocations.id").
			Where("enterprise_quota_allocations.department_budget_id = ? AND enterprise_quota_allocations.status = ?", budget.Id, entmodel.QuotaAllocationStatusPaused).
			Where("wallets.source_type = ? AND wallets.status = ?", model.SubscriptionSourceTypeEnterprise, "paused").
			Pluck("enterprise_quota_allocations.id", &restorableAllocationIds).Error; err != nil {
			return err
		}
		if len(restorableAllocationIds) == 0 {
			return nil
		}
		if err := tx.Model(&model.UserSubscription{}).
			Where("source_type = ? AND source_allocation_id IN (?) AND status = ?", model.SubscriptionSourceTypeEnterprise, restorableAllocationIds, "paused").
			Updates(map[string]any{
				"status":     "active",
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&entmodel.QuotaAllocation{}).
			Where("id IN (?) AND status = ?", restorableAllocationIds, entmodel.QuotaAllocationStatusPaused).
			Updates(map[string]any{
				"status":     entmodel.QuotaAllocationStatusActive,
				"updated_at": now,
			}).Error
	default:
		return nil
	}
}
