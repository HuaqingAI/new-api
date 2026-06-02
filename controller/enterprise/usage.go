package enterprise

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	if !authorizeScopedEnterpriseSummary(c, tenantId, req.DepartmentId) {
		return
	}

	sortConfig := entservice.NormalizeUsageSummarySort(readOptionalString(req.SummarySort), readOptionalString(req.SummaryOrder))
	result, err := entservice.NewUsageAggregationService(model.DB).GetDepartmentSummary(entservice.UsageSummaryQuery{
		TenantId:           tenantId,
		DeptId:             req.DepartmentId,
		From:               req.From,
		To:                 req.To,
		Sort:               sortConfig,
		IncludeDescendants: req.IncludeDescendants != nil && *req.IncludeDescendants,
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

	common.ApiSuccess(c, dtoenterprise.DepartmentUsageSummaryResponse{
		Items: items,
		Scope: dtoenterprise.DepartmentUsageSummaryScope{
			DepartmentId:       result.Scope.DepartmentId,
			DepartmentName:     result.Scope.DepartmentName,
			IncludeDescendants: result.Scope.IncludeDescendants,
			DepartmentIds:      append([]int{}, result.Scope.DepartmentIds...),
			RequestCount:       result.Scope.RequestCount,
			PromptTokens:       result.Scope.PromptTokens,
			CompletionTokens:   result.Scope.CompletionTokens,
			Quota:              result.Scope.Quota,
			UserCount:          result.Scope.UserCount,
		},
	})
}

func ExportDepartmentUsageCSV(c *gin.Context) {
	var req dtoenterprise.DepartmentUsageExportQuery
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
	if !authorizeScopedEnterpriseSummary(c, tenantId, req.DepartmentId) {
		return
	}

	sortConfig := entservice.NormalizeUsageSummarySort(readOptionalString(req.SummarySort), readOptionalString(req.SummaryOrder))
	exportResult, err := entservice.NewUsageExportService(model.DB).ExportDepartmentUsageCSV(entservice.DepartmentUsageExportQuery{
		TenantId:           tenantId,
		DepartmentId:       req.DepartmentId,
		From:               req.From,
		To:                 req.To,
		Sort:               sortConfig,
		IncludeDescendants: req.IncludeDescendants != nil && *req.IncludeDescendants,
	})
	if err != nil {
		writeUsageSummaryError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", exportResult.FileName))
	c.Status(http.StatusOK)
	if err := entservice.NewUsageExportService(model.DB).WriteDepartmentUsageCSV(c.Writer, exportResult); err != nil {
		_ = c.Error(err)
	}
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
			DisplayName:      item.DisplayName,
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

	userOptions := make([]dtoenterprise.DepartmentUsageLogUserOption, 0, len(result.RecentLogsLink.UserOptions))
	for _, item := range result.RecentLogsLink.UserOptions {
		userOptions = append(userOptions, dtoenterprise.DepartmentUsageLogUserOption{
			UserId:      item.UserId,
			Username:    item.Username,
			DisplayName: item.DisplayName,
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
				UserOptions:     userOptions,
			},
		},
	})
}

func GetDepartmentUsageReportConfig(c *gin.Context) {
	tenantId, ok := requestTenantId(c, nil)
	if !ok {
		return
	}
	result, err := entservice.NewUsageReportService(model.DB).GetConfig(tenantId)
	if err != nil {
		writeUsageSummaryError(c, err)
		return
	}
	item, err := mapUsageReportJobDTO(result)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, dtoenterprise.DepartmentUsageReportConfigResponse{
		Item: item,
	})
}

func SaveDepartmentUsageReportConfig(c *gin.Context) {
	var req dtoenterprise.DepartmentUsageReportConfigRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	input := entservice.UsageReportConfigInput{
		TenantId:  tenantId,
		Receivers: req.Receivers,
		Frequency: readOptionalString(req.Frequency),
		RangeType: readOptionalString(req.RangeType),
		Enabled:   req.Enabled,
	}

	var result entservice.UsageReportJobResult
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		service := entservice.NewUsageReportService(tx)
		var err error
		result, err = service.SaveConfig(input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    tenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionUsageReportSet,
			ObjectType:  entservice.AdminObjectUsageReportJob,
			ObjectId:    strconv.Itoa(result.Id),
			DiffSummary: "Saved department usage report configuration",
			Payload: map[string]any{
				"tenant_id":   tenantId,
				"receivers":   result.Receivers,
				"frequency":   result.Frequency,
				"range_type":  result.RangeType,
				"enabled":     result.Enabled,
				"next_run_at": result.NextRunAt,
			},
		})
	})
	if err != nil {
		writeUsageSummaryError(c, err)
		return
	}

	item, err := mapUsageReportJobDTO(result)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, dtoenterprise.DepartmentUsageReportConfigResponse{
		Item: item,
	})
}

func writeUsageSummaryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrInvalidUsageSummaryQuery), errors.Is(err, entservice.ErrInvalidUsageDetailQuery):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrUsageReportInvalidInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrUsageReportInvalidEmail):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseUsageReportInvalidEmail)
	case errors.Is(err, entservice.ErrUsageReportNotConfigured):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseUsageReportNotConfigured)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func readOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func authorizeScopedEnterpriseSummary(c *gin.Context, tenantId int, departmentId *int) bool {
	if c.GetInt("role") >= common.RoleAdminUser {
		return true
	}
	if departmentId == nil || *departmentId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgEnterprisePermissionAdminRequired)
		return false
	}
	allowed, err := entservice.NewPermissionService(model.DB).CanGovernDepartment(c.GetInt("id"), tenantId, *departmentId)
	if err != nil {
		switch {
		case errors.Is(err, entservice.ErrDepartmentOwnerDeniedByLocalRule):
			common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentOwnerDeniedByLocalRule)
		case errors.Is(err, entservice.ErrDepartmentNotFound):
			common.ApiErrorI18n(c, i18n.MsgEnterprisePermissionDeptAdminRequired)
		default:
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		}
		return false
	}
	if !allowed {
		common.ApiErrorI18n(c, i18n.MsgEnterprisePermissionDeptAdminRequired)
		return false
	}
	return true
}

func mapUsageReportJobDTO(item entservice.UsageReportJobResult) (dtoenterprise.DepartmentUsageReportJobItem, error) {
	dto := dtoenterprise.DepartmentUsageReportJobItem{
		Id:              item.Id,
		TenantId:        item.TenantId,
		Receivers:       append([]string{}, item.Receivers...),
		Frequency:       item.Frequency,
		RangeType:       item.RangeType,
		Enabled:         item.Enabled,
		Status:          item.Status,
		LastRunAt:       item.LastRunAt,
		NextRunAt:       item.NextRunAt,
		LastSuccessAt:   item.LastSuccessAt,
		LastWindowStart: item.LastWindowStart,
		LastWindowEnd:   item.LastWindowEnd,
		RunCount:        item.RunCount,
		FailureCount:    item.FailureCount,
		ErrorReason:     item.ErrorReason,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
	if item.LastSnapshot != nil {
		dto.LastSnapshot = &dtoenterprise.DepartmentUsageReportSnapshot{
			WindowStart:         item.LastSnapshot.WindowStart,
			WindowEnd:           item.LastSnapshot.WindowEnd,
			PreviousWindowStart: item.LastSnapshot.PreviousWindowStart,
			PreviousWindowEnd:   item.LastSnapshot.PreviousWindowEnd,
			DepartmentCount:     item.LastSnapshot.DepartmentCount,
			RequestCount:        item.LastSnapshot.RequestCount,
			PromptTokens:        item.LastSnapshot.PromptTokens,
			CompletionTokens:    item.LastSnapshot.CompletionTokens,
			Quota:               item.LastSnapshot.Quota,
			UserCount:           item.LastSnapshot.UserCount,
			TopDepartments:      []dtoenterprise.DepartmentUsageReportTopDepartment{},
			GrowthDepartments:   []dtoenterprise.DepartmentUsageReportGrowthDepartment{},
		}
		for _, top := range item.LastSnapshot.TopDepartments {
			dto.LastSnapshot.TopDepartments = append(dto.LastSnapshot.TopDepartments, dtoenterprise.DepartmentUsageReportTopDepartment{
				DeptId:       top.DeptId,
				DeptName:     top.DeptName,
				RequestCount: top.RequestCount,
				Quota:        top.Quota,
				UserCount:    top.UserCount,
			})
		}
		for _, growth := range item.LastSnapshot.GrowthDepartments {
			dto.LastSnapshot.GrowthDepartments = append(dto.LastSnapshot.GrowthDepartments, dtoenterprise.DepartmentUsageReportGrowthDepartment{
				DeptId:               growth.DeptId,
				DeptName:             growth.DeptName,
				RequestCount:         growth.RequestCount,
				PreviousRequestCount: growth.PreviousRequestCount,
				Quota:                growth.Quota,
				PreviousQuota:        growth.PreviousQuota,
				RequestGrowthRate:    growth.RequestGrowthRate,
				QuotaGrowthRate:      growth.QuotaGrowthRate,
			})
		}
	}
	if dto.Receivers == nil {
		dto.Receivers = []string{}
	}
	dto.Frequency = strings.TrimSpace(dto.Frequency)
	dto.RangeType = strings.TrimSpace(dto.RangeType)
	return dto, nil
}
