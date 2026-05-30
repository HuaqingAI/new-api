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
		TenantId:     tenantId,
		DepartmentId: query.DepartmentId,
		UserId:       query.UserId,
		Username:     readOptionalString(query.Username),
		ModelName:    readOptionalString(query.ModelName),
		RiskType:     readOptionalString(query.RiskType),
		From:         query.From,
		To:           query.To,
		Page:         valueOrZero(query.Page),
		PageSize:     valueOrZero(query.PageSize),
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
		TenantId:    tenantId,
		RuleId:      query.RuleId,
		EventId:     query.EventId,
		ChannelType: readOptionalString(query.ChannelType),
		Status:      readOptionalString(query.Status),
		Page:        valueOrZero(query.Page),
		PageSize:    valueOrZero(query.PageSize),
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
				Username:           item.Trace.Username,
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

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
