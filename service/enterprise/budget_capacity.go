package enterprise

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

func withBudgetMutationRetry(run func() error) error {
	const maxAttempts = 30
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := run()
		if err == nil {
			return nil
		}
		lastErr = err
		if !shouldRetryBudgetMutationTx(err) || attempt == maxAttempts-1 {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return lastErr
}

func shouldRetryBudgetMutationTx(err error) bool {
	if err == nil || !common.UsingMainDatabase(common.DatabaseTypeSQLite) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database is deadlocked")
}

func reserveDepartmentBudgetQuota(tx *gorm.DB, budget entmodel.DepartmentBudget, committedQuota int64) (*quotaAllocationBudgetReservation, error) {
	if committedQuota <= 0 {
		return nil, ErrQuotaAllocationQuotaInvalid
	}
	beforeSnapshot, err := marshalDepartmentBudgetSnapshot(budget)
	if err != nil {
		return nil, err
	}

	updateResult, insufficiencyReason := buildBudgetReservationUpdate(tx, budget, committedQuota)
	if updateResult.Error != nil {
		return nil, updateResult.Error
	}
	if updateResult.RowsAffected == 0 {
		return nil, newQuotaAllocationBudgetError(insufficiencyReason)
	}

	var refreshedBudget entmodel.DepartmentBudget
	if err := tx.Where("id = ?", budget.Id).First(&refreshedBudget).Error; err != nil {
		return nil, err
	}
	afterSnapshot, err := marshalDepartmentBudgetSnapshot(refreshedBudget)
	if err != nil {
		return nil, err
	}
	return &quotaAllocationBudgetReservation{
		beforeSnapshot: beforeSnapshot,
		afterSnapshot:  afterSnapshot,
	}, nil
}

func buildBudgetReservationUpdate(tx *gorm.DB, budget entmodel.DepartmentBudget, committedQuota int64) (*gorm.DB, error) {
	now := common.GetTimestamp()
	switch budget.Type {
	case entmodel.DepartmentBudgetTypeSubscription:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ? AND allocated_total + ? <= cycle_quota", budget.Id, committedQuota).
			Updates(map[string]any{
				"allocated_total": gorm.Expr("allocated_total + ?", committedQuota),
				"remaining":       gorm.Expr("cycle_quota - (allocated_total + ?)", committedQuota),
				"updated_at":      now,
			}), ErrQuotaAllocationSubscriptionCycleAllocatedExceeded
	default:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ? AND remaining >= ?", budget.Id, committedQuota).
			Updates(map[string]any{
				"remaining":  gorm.Expr("remaining - ?", committedQuota),
				"updated_at": now,
			}), ErrQuotaAllocationBalanceRemainingInsufficient
	}
}

func buildBudgetCreditUpdate(tx *gorm.DB, budget entmodel.DepartmentBudget, committedQuota int64) *gorm.DB {
	now := common.GetTimestamp()
	switch budget.Type {
	case entmodel.DepartmentBudgetTypeSubscription:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ?", budget.Id).
			Updates(map[string]any{
				"cycle_quota": gorm.Expr("cycle_quota + ?", committedQuota),
				"remaining":   gorm.Expr("remaining + ?", committedQuota),
				"updated_at":  now,
			})
	default:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ?", budget.Id).
			Updates(map[string]any{
				"total_quota": gorm.Expr("total_quota + ?", committedQuota),
				"remaining":   gorm.Expr("remaining + ?", committedQuota),
				"updated_at":  now,
			})
	}
}

func buildBudgetDelegationReverseTargetUpdate(tx *gorm.DB, budget entmodel.DepartmentBudget, committedQuota int64) (*gorm.DB, error) {
	now := common.GetTimestamp()
	switch budget.Type {
	case entmodel.DepartmentBudgetTypeSubscription:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ? AND remaining >= ? AND cycle_quota >= ?", budget.Id, committedQuota, committedQuota).
			Updates(map[string]any{
				"cycle_quota": gorm.Expr("cycle_quota - ?", committedQuota),
				"remaining":   gorm.Expr("remaining - ?", committedQuota),
				"updated_at":  now,
			}), ErrBudgetDelegationTargetBudgetCapacityLocked
	default:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ? AND remaining >= ? AND total_quota >= ?", budget.Id, committedQuota, committedQuota).
			Updates(map[string]any{
				"total_quota": gorm.Expr("total_quota - ?", committedQuota),
				"remaining":   gorm.Expr("remaining - ?", committedQuota),
				"updated_at":  now,
			}), ErrBudgetDelegationTargetBudgetCapacityLocked
	}
}

func applyBudgetReleaseUpdate(tx *gorm.DB, budget entmodel.DepartmentBudget, committedQuota int64) error {
	now := common.GetTimestamp()
	switch budget.Type {
	case entmodel.DepartmentBudgetTypeSubscription:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ?", budget.Id).
			Updates(map[string]any{
				"allocated_total": gorm.Expr("CASE WHEN allocated_total >= ? THEN allocated_total - ? ELSE 0 END", committedQuota, committedQuota),
				"remaining":       gorm.Expr("CASE WHEN cycle_quota >= (CASE WHEN allocated_total >= ? THEN allocated_total - ? ELSE 0 END) THEN cycle_quota - (CASE WHEN allocated_total >= ? THEN allocated_total - ? ELSE 0 END) ELSE 0 END", committedQuota, committedQuota, committedQuota, committedQuota),
				"updated_at":      now,
			}).Error
	default:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ?", budget.Id).
			Updates(map[string]any{
				"remaining":  gorm.Expr("remaining + ?", committedQuota),
				"updated_at": now,
			}).Error
	}
}

func marshalDepartmentBudgetSnapshot(budget entmodel.DepartmentBudget) (string, error) {
	snapshot, err := common.Marshal(map[string]any{
		"id":              budget.Id,
		"type":            budget.Type,
		"remaining":       budget.Remaining,
		"allocated_total": budget.AllocatedTotal,
		"total_quota":     budget.TotalQuota,
		"cycle_quota":     budget.CycleQuota,
		"cycle_type":      budget.CycleType,
		"cycle_started_at": budget.CycleStartedAt,
		"custom_seconds":  budget.CustomSeconds,
		"expires_at":      budget.ExpiresAt,
		"department_id":   budget.DepartmentId,
	})
	if err != nil {
		return "", err
	}
	return string(snapshot), nil
}
