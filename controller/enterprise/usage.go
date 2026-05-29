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

func writeUsageSummaryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrInvalidUsageSummaryQuery):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}
