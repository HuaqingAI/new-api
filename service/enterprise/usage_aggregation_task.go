package enterprise

import (
	"context"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	usageAggregationWindowSeconds  int64 = 3600
	usageAggregationTaskMaxCatchUp       = 24
)

func RunUsageAggregationTaskOnce(ctx context.Context) (int, error) {
	service := NewUsageAggregationService(nil)
	now := common.GetTimestamp()
	latestWindowEnd := now - (now % usageAggregationWindowSeconds)
	if latestWindowEnd < usageAggregationWindowSeconds {
		return 0, nil
	}

	watermark, err := service.getWatermark(0)
	if err != nil {
		return 0, err
	}

	nextWindowStart := latestWindowEnd - usageAggregationWindowSeconds
	if watermark > 0 {
		nextWindowStart = watermark
	} else {
		var firstLog model.Log
		if err := service.logDB.
			Select("created_at").
			Where("type = ? AND created_at < ?", model.LogTypeConsume, latestWindowEnd).
			Order("created_at ASC, id ASC").
			First(&firstLog).Error; err == nil {
			nextWindowStart = firstLog.CreatedAt - (firstLog.CreatedAt % usageAggregationWindowSeconds)
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if nextWindowStart+usageAggregationWindowSeconds > latestWindowEnd {
		return 0, nil
	}

	processedWindows := 0
	for processedWindows < usageAggregationTaskMaxCatchUp && nextWindowStart+usageAggregationWindowSeconds <= latestWindowEnd {
		window := UsageAggregationWindow{
			TenantId:    0,
			WindowStart: nextWindowStart,
			WindowEnd:   nextWindowStart + usageAggregationWindowSeconds,
		}
		logCount, err := service.AggregateWindow(window)
		if err != nil {
			return processedWindows, err
		}
		logger.LogInfo(ctx, fmt.Sprintf("enterprise usage aggregation window processed: start=%d end=%d logs=%d", window.WindowStart, window.WindowEnd, logCount))
		processedWindows++
		nextWindowStart = window.WindowEnd
	}
	return processedWindows, nil
}
