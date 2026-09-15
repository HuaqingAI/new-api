package enterprise

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

const governanceNotificationDispatchBatchSize = 200

func RunGovernanceNotificationDispatchTaskOnce(ctx context.Context) (GovernanceNotificationDispatchResult, error) {
	result, err := NewGovernanceNotificationDispatchService(nil).DispatchDueDeliveries(ctx, governanceNotificationDispatchBatchSize)
	if err != nil {
		return GovernanceNotificationDispatchResult{}, err
	}
	if result.Processed > 0 || result.FinalFailed > 0 {
		common.SysLog(fmt.Sprintf(
			"enterprise governance notification dispatch task finished: processed=%d sent=%d retried=%d final_failed=%d",
			result.Processed,
			result.Sent,
			result.Retried,
			result.FinalFailed,
		))
	}
	return result, nil
}
