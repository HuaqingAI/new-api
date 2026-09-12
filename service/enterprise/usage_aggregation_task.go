package enterprise

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
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
	tenantIds, err := service.listUsageTenantIDs()
	if err != nil {
		return 0, err
	}

	processedWindows := 0
	for _, tenantId := range tenantIds {
		processed, err := service.runUsageAggregationForTenant(ctx, tenantId, latestWindowEnd)
		if err != nil {
			return processedWindows, err
		}
		processedWindows += processed
	}
	return processedWindows, nil
}

func (s *UsageAggregationService) listUsageTenantIDs() ([]int, error) {
	tenants := map[int]struct{}{0: {}}
	var departmentTenantIDs []int
	if err := s.db.Model(&entmodel.Department{}).Distinct("tenant_id").Pluck("tenant_id", &departmentTenantIDs).Error; err != nil {
		return nil, err
	}
	for _, tenantId := range departmentTenantIDs {
		tenants[tenantId] = struct{}{}
	}
	var membershipTenantIDs []int
	if err := s.db.Model(&entmodel.UserDepartment{}).Distinct("tenant_id").Pluck("tenant_id", &membershipTenantIDs).Error; err != nil {
		return nil, err
	}
	for _, tenantId := range membershipTenantIDs {
		tenants[tenantId] = struct{}{}
	}

	result := make([]int, 0, len(tenants))
	for tenantId := range tenants {
		result = append(result, tenantId)
	}
	sort.Ints(result)
	return result, nil
}

func (s *UsageAggregationService) runUsageAggregationForTenant(ctx context.Context, tenantId int, latestWindowEnd int64) (int, error) {
	watermark, err := s.getWatermark(tenantId)
	if err != nil {
		return 0, err
	}

	nextWindowStart := latestWindowEnd - usageAggregationWindowSeconds
	if watermark > 0 {
		nextWindowStart = watermark
	} else {
		var firstLog model.Log
		if err := s.logDB.
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
			TenantId:    tenantId,
			WindowStart: nextWindowStart,
			WindowEnd:   nextWindowStart + usageAggregationWindowSeconds,
		}
		logCount, err := s.AggregateWindow(window)
		if err != nil {
			return processedWindows, err
		}
		logger.LogInfo(ctx, fmt.Sprintf("enterprise usage aggregation window processed: tenant=%d start=%d end=%d logs=%d", tenantId, window.WindowStart, window.WindowEnd, logCount))
		processedWindows++
		nextWindowStart = window.WindowEnd
	}
	return processedWindows, nil
}
