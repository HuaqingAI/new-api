package enterprise

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	alertDispatchScanLookback = 24 * time.Hour
	alertDispatchBatchSize    = 200
)

func RunAlertDispatchTaskOnce(ctx context.Context) (AlertDispatchResult, error) {
	alertService := NewAlertService(nil)
	now := time.Now()
	created, err := alertService.EnqueueAlertDeliveriesForPendingEvents(now, alertDispatchScanLookback, alertDispatchBatchSize)
	if err != nil {
		return AlertDispatchResult{}, err
	}

	dispatchResult, err := NewAlertDispatchService(nil).DispatchDueDeliveries(ctx, alertDispatchBatchSize)
	if err != nil {
		return AlertDispatchResult{}, err
	}
	if created > 0 || dispatchResult.Processed > 0 || dispatchResult.FinalFailed > 0 {
		common.SysLog(fmt.Sprintf(
			"enterprise alert dispatch task finished: created=%d processed=%d sent=%d retried=%d final_failed=%d",
			created,
			dispatchResult.Processed,
			dispatchResult.Sent,
			dispatchResult.Retried,
			dispatchResult.FinalFailed,
		))
	}
	return dispatchResult, nil
}
