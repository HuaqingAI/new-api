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
	if err := db.Where("status IN ?", []string{
		entmodel.DepartmentBudgetStatusPaused,
		entmodel.DepartmentBudgetStatusRevoked,
		entmodel.DepartmentBudgetStatusExpired,
	}).
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
			status := budget.Status
			if status == entmodel.DepartmentBudgetStatusPaused {
				if err := tx.Model(&entmodel.QuotaAllocation{}).
					Where("department_budget_id = ? AND status = ?", budget.Id, entmodel.QuotaAllocationStatusActive).
					Updates(map[string]any{
						"status":     entmodel.QuotaAllocationStatusPaused,
						"updated_at": common.GetTimestamp(),
					}).Error; err != nil {
					return err
				}
				return tx.Model(&model.UserSubscription{}).
					Where("source_type = ? AND source_allocation_id IN (?) AND status = ?", model.SubscriptionSourceTypeEnterprise,
						tx.Model(&entmodel.QuotaAllocation{}).Select("id").Where("department_budget_id = ?", budget.Id),
						"active").
					Updates(map[string]any{
						"status":     "paused",
						"updated_at": common.GetTimestamp(),
					}).Error
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
