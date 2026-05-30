package enterprise_test

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	servicepkg "github.com/QuantumNous/new-api/service"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	longRunTieredExpr         = `p <= 200000 ? tier("standard", p * 1.5 + c * 7.5) : tier("long_context", p * 3 + c * 11.25)`
	enterpriseLongRunDuration = 10 * time.Minute
)

func TestQuotaAllocationLongRunningMixedBudgetLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skip long-running enterprise allocation load test in short mode")
	}
	if os.Getenv("RUN_ENTERPRISE_BUDGET_LONG_TEST") != "1" {
		t.Skip("set RUN_ENTERPRISE_BUDGET_LONG_TEST=1 to execute the 10-minute enterprise budget load test")
	}
	minRequired := enterpriseLongRunDuration + 2*time.Minute
	if deadline, ok := t.Deadline(); ok {
		if time.Until(deadline) < minRequired {
			t.Skip("run this 10-minute load test with a larger go test timeout, e.g. -timeout 20m")
		}
	}
	if timeoutFlag := flag.Lookup("test.timeout"); timeoutFlag != nil {
		if timeoutValue, err := time.ParseDuration(timeoutFlag.Value.String()); err == nil {
			if timeoutValue > 0 && timeoutValue < minRequired {
				t.Skip("run this 10-minute load test with -timeout 20m or higher")
			}
		}
	}

	svc, db := newQuotaAllocationTestService(t)
	require.NoError(t, db.AutoMigrate(&model.SubscriptionPreConsumeRecord{}))

	const workers = 60
	const allocationQuota int64 = 500000
	cycleStartedAt := time.Now().Add(-2 * time.Minute).Unix()
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 1).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(workers) * allocationQuota,
		"allocated_total":  int64(0),
		"cycle_quota":      int64(workers) * allocationQuota,
		"cycle_type":       model.SubscriptionResetCustom,
		"cycle_started_at": cycleStartedAt,
		"custom_seconds":   int64(60),
	}).Error)

	userIDs := make([]int, 0, workers)
	for i := 0; i < workers; i++ {
		userID := 5000 + i
		seedQuotaAllocationMember(t, db, userID, 1)
		userIDs = append(userIDs, userID)
		_, err := svc.Create(entservice.CreateQuotaAllocationInput{
			TenantId:           0,
			DepartmentBudgetId: 1,
			DepartmentId:       1,
			TargetUserId:       userID,
			ActorId:            1001,
			CommittedQuota:     allocationQuota,
			Reason:             "long-run-budget-seed",
		})
		require.NoError(t, err)
	}

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(workers)*allocationQuota, budget.AllocatedTotal)
	require.Equal(t, budget.CycleQuota-budget.AllocatedTotal, budget.Remaining)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
	})

	var totalOps atomic.Int64
	deadline := time.Now().Add(enterpriseLongRunDuration)
	for time.Now().Before(deadline) {
		var wg sync.WaitGroup
		errCh := make(chan error, workers)
		for i, userID := range userIDs {
			wg.Add(1)
			go func(uid int, idx int) {
				defer wg.Done()
				if idx%2 == 0 {
					quota := fixedQuotaForIteration(idx)
					result, err := preConsumeWithSQLiteRetry(
						fmt.Sprintf("longrun-fixed-%d-%d", uid, totalOps.Add(1)),
						uid,
						"gpt-4o-mini",
						quota,
					)
					if err != nil {
						errCh <- err
						return
					}
					if result.AmountUsedAfter-result.AmountUsedBefore != quota {
						errCh <- fmt.Errorf("fixed pre-consume mismatch for user %d", uid)
					}
					return
				}

				params := tieredParamsForIteration(idx)
				relayInfo := longRunRelayInfo(params)
				ok, quota, result := servicepkg.TryTieredSettle(relayInfo, params)
				if !ok || result == nil {
					errCh <- fmt.Errorf("tiered settle unavailable for user %d", uid)
					return
				}
				pre, err := preConsumeWithSQLiteRetry(
					fmt.Sprintf("longrun-tiered-%d-%d", uid, totalOps.Add(1)),
					uid,
					"claude-3-7-sonnet",
					int64(quota),
				)
				if err != nil {
					errCh <- err
					return
				}
				if pre.AmountUsedAfter-pre.AmountUsedBefore != int64(quota) {
					errCh <- fmt.Errorf("tiered pre-consume mismatch for user %d", uid)
				}
			}(userID, i)
		}
		wg.Wait()
		close(errCh)
		for err := range errCh {
			require.NoError(t, err)
		}

		resetCount, err := resetSubscriptionsWithSQLiteRetry(workers * 2)
		require.NoError(t, err)
		if resetCount > 0 {
			t.Logf("long-run reset cycle applied to %d enterprise wallets", resetCount)
		}
		assertEnterpriseWalletInvariants(t, db)
		time.Sleep(100 * time.Millisecond)
	}

	t.Log("Executed 10-minute mixed fixed/tiered concurrent enterprise wallet load with 60 workers.")
}

func fixedQuotaForIteration(index int) int64 {
	return int64(5 + (index % 7))
}

func tieredParamsForIteration(index int) billingexpr.TokenParams {
	prompt := float64(6 + (index % 5))
	completion := float64(4 + (index % 3))
	return billingexpr.TokenParams{
		P: prompt,
		C: completion,
	}
}

func longRunRelayInfo(params billingexpr.TokenParams) *relaycommon.RelayInfo {
	cost, trace, err := billingexpr.RunExpr(longRunTieredExpr, params)
	if err != nil {
		panic(err)
	}
	beforeGroup := cost / 1_000_000 * common.QuotaPerUnit
	return &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			ExprString:                longRunTieredExpr,
			ExprHash:                  billingexpr.ExprHashString(longRunTieredExpr),
			GroupRatio:                1.0,
			EstimatedPromptTokens:     int(params.P),
			EstimatedCompletionTokens: int(params.C),
			EstimatedQuotaBeforeGroup: beforeGroup,
			EstimatedQuotaAfterGroup:  billingexpr.QuotaRound(beforeGroup),
			EstimatedTier:             trace.MatchedTier,
			QuotaPerUnit:              common.QuotaPerUnit,
		},
		FinalPreConsumedQuota: billingexpr.QuotaRound(beforeGroup),
	}
}

func assertEnterpriseWalletInvariants(t *testing.T, db *gorm.DB) {
	t.Helper()

	var wallets []model.UserSubscription
	require.NoError(t, db.Where("source_type = ?", model.SubscriptionSourceTypeEnterprise).Find(&wallets).Error)
	require.NotEmpty(t, wallets)
	var walletTotal int64
	for _, wallet := range wallets {
		require.NotZero(t, wallet.SourceAllocationId)
		require.GreaterOrEqual(t, wallet.AmountUsed, int64(0))
		require.GreaterOrEqual(t, wallet.AmountTotal, wallet.AmountUsed)
		walletTotal += wallet.AmountTotal
	}

	var allocations []entmodel.QuotaAllocation
	require.NoError(t, db.Where("status = ?", entmodel.QuotaAllocationStatusActive).Find(&allocations).Error)
	require.Len(t, allocations, len(wallets))
	var allocationTotal int64
	for _, allocation := range allocations {
		require.NotZero(t, allocation.WalletId)
		require.Contains(t, allocation.BeforeBudgetSnapshot, `"allocated_total":`)
		require.Contains(t, allocation.AfterBudgetSnapshot, `"allocated_total":`)
		allocationTotal += allocation.CommittedQuota
	}

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.GreaterOrEqual(t, budget.Remaining, int64(0))
	require.LessOrEqual(t, budget.AllocatedTotal, budget.CycleQuota)
	require.Equal(t, budget.CycleQuota-budget.AllocatedTotal, budget.Remaining)
	require.Equal(t, budget.AllocatedTotal, walletTotal)
	require.Equal(t, budget.AllocatedTotal, allocationTotal)
}

func preConsumeWithSQLiteRetry(requestID string, userID int, modelName string, quota int64) (*model.SubscriptionPreConsumeResult, error) {
	const maxAttempts = 50
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := model.PreConsumeUserSubscription(requestID, userID, modelName, 0, quota)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !shouldRetrySQLiteLongRunErr(err) || attempt == maxAttempts-1 {
			return nil, err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil, lastErr
}

func resetSubscriptionsWithSQLiteRetry(limit int) (int, error) {
	const maxAttempts = 50
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		count, err := model.ResetDueSubscriptions(limit)
		if err == nil {
			return count, nil
		}
		lastErr = err
		if !shouldRetrySQLiteLongRunErr(err) || attempt == maxAttempts-1 {
			return 0, err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return 0, lastErr
}

func shouldRetrySQLiteLongRunErr(err error) bool {
	if err == nil || !common.UsingSQLite {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database is deadlocked")
}
