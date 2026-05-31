package enterprise

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListAlertEvents(c *gin.Context) {
	var query dtoenterprise.AlertEventQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}

	result, err := entservice.NewAlertService(model.DB).ListAlertEvents(entservice.AlertEventQuery{
		TenantId:       tenantId,
		EventId:        query.EventId,
		DepartmentId:   query.DepartmentId,
		UnassignedOnly: query.UnassignedOnly,
		UserId:         query.UserId,
		Username:       readOptionalString(query.Username),
		ModelName:      readOptionalString(query.ModelName),
		RiskType:       readOptionalString(query.RiskType),
		From:           query.From,
		To:             query.To,
		Page:           valueOrZero(query.Page),
		PageSize:       valueOrZero(query.PageSize),
	})
	if err != nil {
		writeAlertEventError(c, err)
		return
	}

	items := make([]dtoenterprise.AlertEventItem, 0, len(result.Items))
	for _, item := range result.Items {
		snapshot := make([]dtoenterprise.AlertEventDepartmentSnapshot, 0, len(item.DepartmentSnapshot))
		for _, department := range item.DepartmentSnapshot {
			snapshot = append(snapshot, dtoenterprise.AlertEventDepartmentSnapshot{
				DepartmentId:   department.DepartmentId,
				DepartmentName: department.DepartmentName,
				ExternalSource: department.ExternalSource,
				Status:         department.Status,
			})
		}
		if snapshot == nil {
			snapshot = []dtoenterprise.AlertEventDepartmentSnapshot{}
		}
		items = append(items, dtoenterprise.AlertEventItem{
			Id:                 item.Id,
			TenantId:           item.TenantId,
			UserId:             item.UserId,
			Username:           item.Username,
			DisplayName:        item.DisplayName,
			UsernameSnapshot:   item.UsernameSnapshot,
			RequestId:          item.RequestId,
			ModelName:          item.ModelName,
			RiskType:           item.RiskType,
			ActionResult:       item.ActionResult,
			CreatedAt:          item.CreatedAt,
			DepartmentSnapshot: snapshot,
			Summary:            item.Summary,
		})
	}
	if items == nil {
		items = []dtoenterprise.AlertEventItem{}
	}

	common.ApiSuccess(c, dtoenterprise.AlertEventsResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func GetDepartmentRiskSummary(c *gin.Context) {
	var req dtoenterprise.DepartmentRiskSummaryQuery
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

	sortConfig := entservice.NormalizeUsageSummarySort(readOptionalString(req.SummarySort), readOptionalString(req.SummaryOrder))
	result, err := entservice.NewAlertService(model.DB).GetDepartmentRiskSummary(entservice.DepartmentRiskSummaryQuery{
		TenantId: tenantId,
		From:     req.From,
		To:       req.To,
		Sort:     sortConfig,
	})
	if err != nil {
		writeAlertEventError(c, err)
		return
	}

	items := make([]dtoenterprise.DepartmentRiskSummaryItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapDepartmentRiskSummaryItem(item))
	}
	if items == nil {
		items = []dtoenterprise.DepartmentRiskSummaryItem{}
	}

	topDepartments := make([]dtoenterprise.DepartmentRiskSummaryItem, 0, len(result.TopDepartments))
	for _, item := range result.TopDepartments {
		topDepartments = append(topDepartments, mapDepartmentRiskSummaryItem(item))
	}
	if topDepartments == nil {
		topDepartments = []dtoenterprise.DepartmentRiskSummaryItem{}
	}

	trend := make([]dtoenterprise.DepartmentRiskTrendPoint, 0, len(result.Trend))
	for _, point := range result.Trend {
		trend = append(trend, dtoenterprise.DepartmentRiskTrendPoint{
			WindowStart:                 point.WindowStart,
			WindowEnd:                   point.WindowEnd,
			RiskEventCount:              point.RiskEventCount,
			TotalRequestCount:           point.TotalRequestCount,
			RiskRate:                    point.RiskRate,
			UnassignedRiskEventCount:    point.UnassignedRiskEventCount,
			UnassignedTotalRequestCount: point.UnassignedTotalRequestCount,
		})
	}
	if trend == nil {
		trend = []dtoenterprise.DepartmentRiskTrendPoint{}
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentRiskSummaryResponse{
		Items:          items,
		TopDepartments: topDepartments,
		Trend:          trend,
		Unassigned:     mapDepartmentRiskSummaryItem(result.Unassigned),
		Formula: dtoenterprise.DepartmentRiskFormula{
			Expression:       result.Formula.Expression,
			NumeratorLabel:   result.Formula.NumeratorLabel,
			DenominatorLabel: result.Formula.DenominatorLabel,
		},
		DisclaimerKey: result.DisclaimerKey,
	})
}

func ListAlertDeliveries(c *gin.Context) {
	var query dtoenterprise.AlertDeliveriesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}

	result, err := entservice.NewAlertService(model.DB).ListAlertDeliveries(entservice.AlertDeliveryQuery{
		TenantId:       tenantId,
		RuleId:         query.RuleId,
		EventId:        query.EventId,
		ManualParentId: query.ManualParentId,
		ChannelType:    readOptionalString(query.ChannelType),
		Status:         readOptionalString(query.Status),
		TriggerSource:  readOptionalString(query.TriggerSource),
		Page:           valueOrZero(query.Page),
		PageSize:       valueOrZero(query.PageSize),
	})
	if err != nil {
		writeAlertEventError(c, err)
		return
	}

	items := make([]dtoenterprise.AlertDeliveryItem, 0, len(result.Items))
	for _, item := range result.Items {
		var trace *dtoenterprise.AlertDeliveryTraceItem
		if item.Trace != nil {
			trace = &dtoenterprise.AlertDeliveryTraceItem{
				EventId:            item.Trace.EventId,
				RequestId:          item.Trace.RequestId,
				TenantId:           item.Trace.TenantId,
				UserId:             item.Trace.UserId,
				Username:           item.Trace.Username,
				DisplayName:        item.Trace.DisplayName,
				ModelName:          item.Trace.ModelName,
				RiskType:           item.Trace.RiskType,
				ActionResult:       item.Trace.ActionResult,
				EventCreatedAt:     item.Trace.EventCreatedAt,
				DepartmentSnapshot: mapAlertEventDepartmentSnapshotItems(item.Trace.DepartmentSnapshot),
				DepartmentSummary:  item.Trace.DepartmentSummary,
				EventSummary:       item.Trace.EventSummary,
				RuleId:             item.Trace.RuleId,
				RuleName:           item.Trace.RuleName,
				DetailRoute:        item.Trace.DetailRoute,
				DetailAPIPath:      item.Trace.DetailAPIPath,
			}
		}
		items = append(items, dtoenterprise.AlertDeliveryItem{
			Id:             item.Id,
			TenantId:       item.TenantId,
			EventId:        item.EventId,
			RuleId:         item.RuleId,
			ChannelType:    item.ChannelType,
			Status:         item.Status,
			AttemptCount:   item.AttemptCount,
			MaxAttempts:    item.MaxAttempts,
			NextRetryAt:    item.NextRetryAt,
			LastAttemptAt:  item.LastAttemptAt,
			SentAt:         item.SentAt,
			FinalFailedAt:  item.FinalFailedAt,
			ErrorReason:    item.ErrorReason,
			DedupeKey:      item.DedupeKey,
			TriggerSource:  item.TriggerSource,
			ManualParentId: item.ManualParentId,
			TraceSummary:   item.TraceSummary,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
			Trace:          trace,
		})
	}
	if items == nil {
		items = []dtoenterprise.AlertDeliveryItem{}
	}

	common.ApiSuccess(c, dtoenterprise.AlertDeliveriesResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func ResendAlertDelivery(c *gin.Context) {
	deliveryId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	var query dtoenterprise.AlertDeliveryResendQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}

	var result entservice.AlertDeliveryResendResult
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		service := entservice.NewAlertService(tx)
		var err error
		result, err = service.ResendAlertDelivery(tenantId, deliveryId, c.GetInt("id"))
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    tenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionAlertDeliveryResend,
			ObjectType:  entservice.AdminObjectAlertDelivery,
			ObjectId:    strconv.Itoa(result.Item.Id),
			DiffSummary: result.AuditSummary,
			Payload:     result.AuditPayload,
		})
	})
	if err != nil {
		writeAlertEventError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.AlertDeliveryResendResponse{
		Item:    mapAlertDeliveryDTOItem(result.Item),
		Created: result.Created,
	})
}

func ListAlertRules(c *gin.Context) {
	var query dtoenterprise.AlertRulesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}

	result, err := entservice.NewAlertService(model.DB).ListAlertRules(tenantId)
	if err != nil {
		writeAlertEventError(c, err)
		return
	}

	items := make([]dtoenterprise.AlertRuleItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapAlertRuleItem(item))
	}
	if items == nil {
		items = []dtoenterprise.AlertRuleItem{}
	}

	common.ApiSuccess(c, dtoenterprise.AlertRulesResponse{
		Items: items,
		Total: result.Total,
	})
}

func GetAlertRule(c *gin.Context) {
	ruleId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	var query dtoenterprise.AlertRulesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}

	item, err := entservice.NewAlertService(model.DB).GetAlertRule(tenantId, ruleId)
	if err != nil {
		writeAlertEventError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.AlertRuleResponse{
		Item: mapAlertRuleItem(item),
	})
}

func SaveAlertRule(c *gin.Context) {
	var req dtoenterprise.AlertRuleUpsertRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	input := entservice.AlertRuleInput{
		Id:                  valueOrZero(req.Id),
		TenantId:            tenantId,
		Name:                readOptionalString(req.Name),
		Enabled:             req.Enabled,
		RiskTypes:           append([]string{}, req.RiskTypes...),
		DepartmentIds:       append([]int{}, req.DepartmentIds...),
		DedupeWindowSeconds: req.DedupeWindowSeconds,
		ActorId:             c.GetInt("id"),
	}
	input.ChannelConfigs = mapAlertRuleChannelInputs(req.ChannelConfigs)

	var result entservice.AlertRuleMutationResult
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		service := entservice.NewAlertService(tx)
		var err error
		result, err = service.SaveAlertRule(input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    tenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionAlertRuleSave,
			ObjectType:  entservice.AdminObjectAlertRule,
			ObjectId:    strconv.Itoa(result.Item.Id),
			DiffSummary: result.AuditSummary,
			Payload:     result.AuditPayload,
		})
	})
	if err != nil {
		writeAlertEventError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.AlertRuleResponse{
		Item: mapAlertRuleItem(result.Item),
	})
}

func DeleteAlertRule(c *gin.Context) {
	ruleId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	var query dtoenterprise.AlertRulesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}

	var result entservice.AlertRuleMutationResult
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		service := entservice.NewAlertService(tx)
		var err error
		result, err = service.DeleteAlertRule(tenantId, ruleId, c.GetInt("id"))
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    tenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionAlertRuleDelete,
			ObjectType:  entservice.AdminObjectAlertRule,
			ObjectId:    strconv.Itoa(result.Item.Id),
			DiffSummary: result.AuditSummary,
			Payload:     result.AuditPayload,
		})
	})
	if err != nil {
		writeAlertEventError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.AlertRuleResponse{
		Item: mapAlertRuleItem(result.Item),
	})
}

func writeAlertEventError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrInvalidAlertEventQuery):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrInvalidDepartmentRiskSummaryQuery):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrInvalidAlertDeliveryQuery):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrAlertRuleInvalidInput), errors.Is(err, entservice.ErrAlertRuleChannelRequired):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrAlertRuleInvalidEmail):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseAlertRuleInvalidEmail)
	case errors.Is(err, entservice.ErrAlertRuleInvalidWebhookURL):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseAlertRuleInvalidWebhook)
	case errors.Is(err, entservice.ErrAlertRuleNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseAlertRuleNotFound)
	case errors.Is(err, entservice.ErrAlertDeliveryNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseAlertDeliveryNotFound)
	case errors.Is(err, entservice.ErrAlertDeliveryResendNotAllowed):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseAlertDeliveryResendNotAllowed)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func mapAlertEventDepartmentSnapshotItems(items []entmodel.AlertEventDepartmentSnapshot) []dtoenterprise.AlertEventDepartmentSnapshot {
	out := make([]dtoenterprise.AlertEventDepartmentSnapshot, 0, len(items))
	for _, department := range items {
		out = append(out, dtoenterprise.AlertEventDepartmentSnapshot{
			DepartmentId:   department.DepartmentId,
			DepartmentName: department.DepartmentName,
			ExternalSource: department.ExternalSource,
			Status:         department.Status,
		})
	}
	if out == nil {
		out = []dtoenterprise.AlertEventDepartmentSnapshot{}
	}
	return out
}

func mapAlertRuleChannelInputs(items []dtoenterprise.AlertRuleChannelConfigInput) []entservice.AlertRuleChannelInput {
	out := make([]entservice.AlertRuleChannelInput, 0, len(items))
	for _, item := range items {
		out = append(out, entservice.AlertRuleChannelInput{
			Type:          item.Type,
			Enabled:       item.Enabled,
			Receivers:     append([]string{}, item.Receivers...),
			WebhookURL:    readOptionalString(item.WebhookURL),
			WebhookSecret: item.WebhookSecret,
			RobotWebhook:  readOptionalString(item.DingTalkRobotURL),
			RobotSecret:   item.DingTalkRobotSecret,
		})
	}
	return out
}

func mapAlertRuleItem(item entservice.AlertRuleItem) dtoenterprise.AlertRuleItem {
	dto := dtoenterprise.AlertRuleItem{
		Id:                  item.Id,
		TenantId:            item.TenantId,
		Name:                item.Name,
		Enabled:             item.Enabled,
		RiskTypes:           append([]string{}, item.RiskTypes...),
		DepartmentIds:       append([]int{}, item.DepartmentIds...),
		ChannelConfigs:      []dtoenterprise.AlertRuleChannelConfigItem{},
		DedupeWindowSeconds: item.DedupeWindowSeconds,
		CreatedBy:           item.CreatedBy,
		UpdatedBy:           item.UpdatedBy,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
	for _, channel := range item.ChannelConfigs {
		dto.ChannelConfigs = append(dto.ChannelConfigs, dtoenterprise.AlertRuleChannelConfigItem{
			Type:                     channel.Type,
			Enabled:                  channel.Enabled,
			Receivers:                append([]string{}, channel.Receivers...),
			WebhookURL:               optionalString(channel.WebhookURL),
			WebhookSecretConfigured:  channel.WebhookSecretConfigured,
			WebhookSecretMasked:      channel.WebhookSecretMasked,
			DingTalkRobotURL:         optionalString(channel.DingTalkRobotURL),
			DingTalkSecretConfigured: channel.DingTalkSecretConfigured,
			DingTalkSecretMasked:     channel.DingTalkSecretMasked,
		})
	}
	return dto
}

func mapAlertDeliveryDTOItem(item entservice.AlertDeliveryItem) dtoenterprise.AlertDeliveryItem {
	var trace *dtoenterprise.AlertDeliveryTraceItem
	if item.Trace != nil {
		trace = &dtoenterprise.AlertDeliveryTraceItem{
			EventId:            item.Trace.EventId,
			RequestId:          item.Trace.RequestId,
			TenantId:           item.Trace.TenantId,
			UserId:             item.Trace.UserId,
			Username:           item.Trace.Username,
			DisplayName:        item.Trace.DisplayName,
			ModelName:          item.Trace.ModelName,
			RiskType:           item.Trace.RiskType,
			ActionResult:       item.Trace.ActionResult,
			EventCreatedAt:     item.Trace.EventCreatedAt,
			DepartmentSnapshot: mapAlertEventDepartmentSnapshotItems(item.Trace.DepartmentSnapshot),
			DepartmentSummary:  item.Trace.DepartmentSummary,
			EventSummary:       item.Trace.EventSummary,
			RuleId:             item.Trace.RuleId,
			RuleName:           item.Trace.RuleName,
			DetailRoute:        item.Trace.DetailRoute,
			DetailAPIPath:      item.Trace.DetailAPIPath,
		}
	}
	return dtoenterprise.AlertDeliveryItem{
		Id:             item.Id,
		TenantId:       item.TenantId,
		EventId:        item.EventId,
		RuleId:         item.RuleId,
		ChannelType:    item.ChannelType,
		Status:         item.Status,
		AttemptCount:   item.AttemptCount,
		MaxAttempts:    item.MaxAttempts,
		NextRetryAt:    item.NextRetryAt,
		LastAttemptAt:  item.LastAttemptAt,
		SentAt:         item.SentAt,
		FinalFailedAt:  item.FinalFailedAt,
		ErrorReason:    item.ErrorReason,
		DedupeKey:      item.DedupeKey,
		TriggerSource:  item.TriggerSource,
		ManualParentId: item.ManualParentId,
		TraceSummary:   item.TraceSummary,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
		Trace:          trace,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func mapDepartmentRiskSummaryItem(item entservice.DepartmentRiskSummaryItem) dtoenterprise.DepartmentRiskSummaryItem {
	return dtoenterprise.DepartmentRiskSummaryItem{
		DeptId:            item.DeptId,
		DeptName:          item.DeptName,
		IsUnassigned:      item.IsUnassigned,
		WindowStart:       item.WindowStart,
		WindowEnd:         item.WindowEnd,
		RiskEventCount:    item.RiskEventCount,
		TotalRequestCount: item.TotalRequestCount,
		RiskRate:          item.RiskRate,
		EventEntry: dtoenterprise.DepartmentRiskEventEntry{
			DetailRoute:    item.EventEntry.DetailRoute,
			DetailAPIPath:  item.EventEntry.DetailAPIPath,
			DepartmentId:   item.EventEntry.DepartmentId,
			DepartmentName: item.EventEntry.DepartmentName,
			From:           item.EventEntry.From,
			To:             item.EventEntry.To,
			UnassignedOnly: item.EventEntry.UnassignedOnly,
		},
	}
}
