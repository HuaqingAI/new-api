package enterprise

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	enterpriseMaintenanceTaskTickInterval        = 1 * time.Minute
	enterpriseAlertDispatchTaskTickInterval      = 30 * time.Second
	enterpriseGovernanceNotificationTickInterval = 30 * time.Second
)

var enterpriseSchedulerOnce sync.Once

func StartEnterpriseTasks() {
	enterpriseSchedulerOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			ctx := context.Background()
			logger.LogInfo(ctx, fmt.Sprintf("enterprise maintenance tasks started: tick=%s", enterpriseMaintenanceTaskTickInterval))
			ticker := time.NewTicker(enterpriseMaintenanceTaskTickInterval)
			defer ticker.Stop()

			runEnterpriseMaintenanceTasksOnce(ctx)
			for range ticker.C {
				runEnterpriseMaintenanceTasksOnce(ctx)
			}
		})
		gopool.Go(func() {
			ctx := context.Background()
			logger.LogInfo(ctx, fmt.Sprintf("enterprise alert dispatch tasks started: tick=%s", enterpriseAlertDispatchTaskTickInterval))
			ticker := time.NewTicker(enterpriseAlertDispatchTaskTickInterval)
			defer ticker.Stop()

			runEnterpriseAlertDispatchTaskOnce(ctx)
			for range ticker.C {
				runEnterpriseAlertDispatchTaskOnce(ctx)
			}
		})
		gopool.Go(func() {
			ctx := context.Background()
			logger.LogInfo(ctx, fmt.Sprintf("enterprise governance notification dispatch tasks started: tick=%s", enterpriseGovernanceNotificationTickInterval))
			ticker := time.NewTicker(enterpriseGovernanceNotificationTickInterval)
			defer ticker.Stop()

			runEnterpriseGovernanceNotificationDispatchTaskOnce(ctx)
			for range ticker.C {
				runEnterpriseGovernanceNotificationDispatchTaskOnce(ctx)
			}
		})
	})
}

func StartEnterpriseWalletTasks() {
	StartEnterpriseTasks()
}

func runEnterpriseMaintenanceTasksOnce(ctx context.Context) {
	if _, err := ExpireBalanceAllocations(nil, 200, common.GetTimestamp()); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("enterprise balance expiry task failed: %v", err))
	}
	if _, err := SyncWalletStates(nil, 200); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("enterprise wallet state sync task failed: %v", err))
	}
	if _, err := RunUsageAggregationTaskOnce(ctx); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("enterprise usage aggregation task failed: %v", err))
	}
	if result, err := NewUsageReportService(nil).RunDueReports(ctx); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("enterprise usage report task failed: %v", err))
	} else if result.Processed > 0 || result.Failed > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("enterprise usage report task finished: processed=%d failed=%d", result.Processed, result.Failed))
	}
}

func runEnterpriseAlertDispatchTaskOnce(ctx context.Context) {
	if result, err := RunAlertDispatchTaskOnce(ctx); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("enterprise alert dispatch task failed: %v", err))
	} else if result.Processed > 0 || result.FinalFailed > 0 {
		logger.LogInfo(ctx, fmt.Sprintf(
			"enterprise alert dispatch task finished: processed=%d sent=%d retried=%d final_failed=%d",
			result.Processed,
			result.Sent,
			result.Retried,
			result.FinalFailed,
		))
	}
}

func runEnterpriseGovernanceNotificationDispatchTaskOnce(ctx context.Context) {
	if result, err := RunGovernanceNotificationDispatchTaskOnce(ctx); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("enterprise governance notification dispatch task failed: %v", err))
	} else if result.Processed > 0 || result.FinalFailed > 0 {
		logger.LogInfo(ctx, fmt.Sprintf(
			"enterprise governance notification dispatch task finished: processed=%d sent=%d retried=%d final_failed=%d",
			result.Processed,
			result.Sent,
			result.Retried,
			result.FinalFailed,
		))
	}
}
