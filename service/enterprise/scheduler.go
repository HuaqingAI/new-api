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
	enterpriseWalletTaskTickInterval = 1 * time.Minute
)

var enterpriseSchedulerOnce sync.Once

func StartEnterpriseTasks() {
	enterpriseSchedulerOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			ctx := context.Background()
			logger.LogInfo(ctx, fmt.Sprintf("enterprise wallet tasks started: tick=%s", enterpriseWalletTaskTickInterval))
			ticker := time.NewTicker(enterpriseWalletTaskTickInterval)
			defer ticker.Stop()

			runEnterpriseTasksOnce(ctx)
			for range ticker.C {
				runEnterpriseTasksOnce(ctx)
			}
		})
	})
}

func StartEnterpriseWalletTasks() {
	StartEnterpriseTasks()
}

func runEnterpriseTasksOnce(ctx context.Context) {
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
