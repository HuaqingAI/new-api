package enterprise

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
)

func GetDepartmentUsageSummary(c *gin.Context) {
	var req dtoenterprise.DepartmentUsageSummaryQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}
	if req.From <= 0 || req.To <= 0 || req.From >= req.To {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	result, err := entservice.NewUsageAggregationService(model.DB).GetDepartmentSummary(entservice.UsageSummaryQuery{
		TenantId: tenantId,
		From:     req.From,
		To:       req.To,
	})
	if err != nil {
		writeUsageSummaryError(c, err)
		return
	}

	items := make([]dtoenterprise.DepartmentUsageSummaryItem, 0, len(result.Items))
	for _, item := range result.Items {
		modelDistribution := make([]dtoenterprise.UsageModelDistributionItem, 0, len(item.ModelDistribution))
		for _, stat := range item.ModelDistribution {
			modelDistribution = append(modelDistribution, dtoenterprise.UsageModelDistributionItem{
				ModelName:        stat.ModelName,
				RequestCount:     stat.RequestCount,
				PromptTokens:     stat.PromptTokens,
				CompletionTokens: stat.CompletionTokens,
				Quota:            stat.Quota,
			})
		}
		items = append(items, dtoenterprise.DepartmentUsageSummaryItem{
			DeptId:            item.DeptId,
			DeptName:          item.DeptName,
			WindowStart:       item.WindowStart,
			WindowEnd:         item.WindowEnd,
			RequestCount:      item.RequestCount,
			PromptTokens:      item.PromptTokens,
			CompletionTokens:  item.CompletionTokens,
			Quota:             item.Quota,
			UserCount:         item.UserCount,
			ModelDistribution: modelDistribution,
		})
	}
	if items == nil {
		items = []dtoenterprise.DepartmentUsageSummaryItem{}
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentUsageSummaryResponse{Items: items})
}

func GetDepartmentUsageDetail(c *gin.Context) {
	var req dtoenterprise.DepartmentUsageDetailQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}
	if req.DeptId == nil || req.From <= 0 || req.To <= 0 || req.From >= req.To {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	result, err := entservice.NewUsageAggregationService(model.DB).GetDepartmentDetail(entservice.UsageDetailQuery{
		TenantId: tenantId,
		DeptId:   req.DeptId,
		From:     req.From,
		To:       req.To,
	})
	if err != nil {
		writeUsageSummaryError(c, err)
		return
	}

	userRanking := make([]dtoenterprise.DepartmentUsageUserRankItem, 0, len(result.UserRanking))
	for _, item := range result.UserRanking {
		userRanking = append(userRanking, dtoenterprise.DepartmentUsageUserRankItem{
			UserId:           item.UserId,
			Username:         item.Username,
			RequestCount:     item.RequestCount,
			PromptTokens:     item.PromptTokens,
			CompletionTokens: item.CompletionTokens,
			TokenCount:       item.TokenCount,
			Quota:            item.Quota,
		})
	}

	modelDistribution := make([]dtoenterprise.UsageModelDistributionItem, 0, len(result.ModelDistribution))
	for _, stat := range result.ModelDistribution {
		modelDistribution = append(modelDistribution, dtoenterprise.UsageModelDistributionItem{
			ModelName:        stat.ModelName,
			RequestCount:     stat.RequestCount,
			PromptTokens:     stat.PromptTokens,
			CompletionTokens: stat.CompletionTokens,
			Quota:            stat.Quota,
		})
	}

	trend := make([]dtoenterprise.DepartmentUsageTrendPoint, 0, len(result.Trend))
	for _, point := range result.Trend {
		trend = append(trend, dtoenterprise.DepartmentUsageTrendPoint{
			WindowStart:      point.WindowStart,
			WindowEnd:        point.WindowEnd,
			RequestCount:     point.RequestCount,
			PromptTokens:     point.PromptTokens,
			CompletionTokens: point.CompletionTokens,
			TokenCount:       point.TokenCount,
			Quota:            point.Quota,
			UserCount:        point.UserCount,
		})
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentUsageDetailResponse{
		DeptId:            result.DeptId,
		DeptName:          result.DeptName,
		WindowStart:       result.WindowStart,
		WindowEnd:         result.WindowEnd,
		RequestCount:      result.RequestCount,
		PromptTokens:      result.PromptTokens,
		CompletionTokens:  result.CompletionTokens,
		TokenCount:        result.TokenCount,
		Quota:             result.Quota,
		UserCount:         result.UserCount,
		UserRanking:       userRanking,
		ModelDistribution: modelDistribution,
		Trend:             trend,
		RecentLogsEntry: dtoenterprise.DepartmentUsageLogEntryLink{
			Path:    result.RecentLogsLink.Path,
			Section: result.RecentLogsLink.Section,
			Filters: dtoenterprise.DepartmentUsageLogFilters{
				DepartmentId:    result.RecentLogsLink.DepartmentId,
				DepartmentName:  result.RecentLogsLink.DepartmentName,
				StartTimestamp:  result.RecentLogsLink.StartTimestamp,
				EndTimestamp:    result.RecentLogsLink.EndTimestamp,
				UsernameOptions: append([]string{}, result.RecentLogsLink.Usernames...),
			},
		},
	})
}

func writeUsageSummaryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrInvalidUsageSummaryQuery), errors.Is(err, entservice.ErrInvalidUsageDetailQuery):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}
