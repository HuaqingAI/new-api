package enterprise

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

func ExpireBalanceAllocations(db *gorm.DB, batchSize int, now int64) (int, error) {
	if db == nil {
		db = model.DB
	}
	if batchSize <= 0 {
		batchSize = 200
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}

	var rows []entmodel.QuotaAllocation
	if err := db.Where("budget_type_snapshot = ? AND status IN ? AND expires_at_snapshot > 0 AND expires_at_snapshot <= ?",
		entmodel.DepartmentBudgetTypeBalance,
		[]string{
			entmodel.QuotaAllocationStatusActive,
			entmodel.QuotaAllocationStatusPaused,
		},
		now,
	).
		Order("expires_at_snapshot ASC, id ASC").
		Limit(batchSize).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}

	processed := 0
	service := NewQuotaAllocationService(db)
	for _, row := range rows {
		if _, err := service.Revoke(RevokeQuotaAllocationInput{
			TenantId:      row.TenantId,
			DepartmentId:  row.DepartmentId,
			AllocationId:  row.Id,
			ActorId:       0,
			TriggeredBy:   QuotaAllocationProcessTriggerExpiry,
			TriggeredTime: now,
		}); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}
