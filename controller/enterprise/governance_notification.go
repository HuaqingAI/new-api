package enterprise

import (
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

func ListGovernanceNotifications(c *gin.Context) {
	var query dtoenterprise.GovernanceNotificationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}
	result, err := entservice.NewGovernanceNotificationService(model.DB).ListDeliveries(entservice.GovernanceNotificationQuery{
		TenantId:        tenantId,
		DepartmentId:    query.DepartmentId,
		RecipientUserId: query.RecipientUserId,
		SourceType:      stringPtrValue(query.SourceType),
		SourceId:        query.SourceId,
		ActionType:      stringPtrValue(query.ActionType),
		Status:          stringPtrValue(query.Status),
		Page:            valueOrZero(query.Page),
		PageSize:        valueOrZero(query.PageSize),
		ViewerId:        c.GetInt("id"),
	})
	if err != nil {
		writeGovernanceError(c, err)
		return
	}
	items := make([]dtoenterprise.GovernanceNotificationItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapGovernanceNotificationItem(item))
	}
	common.ApiSuccess(c, dtoenterprise.GovernanceNotificationResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func ResendGovernanceNotification(c *gin.Context) {
	deliveryId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	var query dtoenterprise.GovernanceNotificationResendQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}

	var result entservice.GovernanceNotificationResendResult
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		service := entservice.NewGovernanceNotificationService(tx)
		var err error
		result, err = service.ResendDelivery(tenantId, deliveryId, c.GetInt("id"))
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    tenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  "enterprise.governance.notification.resend",
			ObjectType:  "enterprise_governance_notification_delivery",
			ObjectId:    strconv.Itoa(result.Item.Id),
			DiffSummary: result.AuditSummary,
			Payload:     result.AuditPayload,
		})
	})
	if err != nil {
		writeGovernanceError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.GovernanceNotificationResendResponse{
		Item:    mapGovernanceNotificationItem(result.Item),
		Created: result.Created,
	})
}

func mapGovernanceNotificationItem(item entservice.GovernanceNotificationItem) dtoenterprise.GovernanceNotificationItem {
	return dtoenterprise.GovernanceNotificationItem{
		Id:              item.Id,
		TenantId:        item.TenantId,
		SourceType:      item.SourceType,
		SourceId:        item.SourceId,
		TraceId:         item.TraceId,
		ActionType:      item.ActionType,
		RecipientUserId: item.RecipientUserId,
		RecipientKind:   item.RecipientKind,
		ChannelType:     item.ChannelType,
		Status:          item.Status,
		AttemptCount:    item.AttemptCount,
		MaxAttempts:     item.MaxAttempts,
		NextRetryAt:     item.NextRetryAt,
		LastAttemptAt:   item.LastAttemptAt,
		SentAt:          item.SentAt,
		FinalFailedAt:   item.FinalFailedAt,
		ErrorReason:     item.ErrorReason,
		DedupeKey:       item.DedupeKey,
		TriggerSource:   item.TriggerSource,
		ManualParentId:  item.ManualParentId,
		TraceSummary:    item.TraceSummary,
		Trace:           mapGovernanceNotificationTrace(item.Trace),
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func mapGovernanceNotificationTrace(trace *entmodel.GovernanceNotificationTracePayload) *dtoenterprise.GovernanceNotificationTraceItem {
	if trace == nil {
		return nil
	}
	return &dtoenterprise.GovernanceNotificationTraceItem{
		TraceId:            trace.TraceId,
		SourceType:         trace.SourceType,
		SourceId:           trace.SourceId,
		ActionType:         trace.ActionType,
		TenantId:           trace.TenantId,
		DepartmentId:       trace.DepartmentId,
		DepartmentName:     trace.DepartmentName,
		BudgetId:           trace.BudgetId,
		AllocationId:       trace.AllocationId,
		RequestId:          trace.RequestId,
		ActorId:            trace.ActorId,
		ActorName:          trace.ActorName,
		TargetUserId:       trace.TargetUserId,
		TargetUsername:     trace.TargetUsername,
		TargetDisplayName:  trace.TargetDisplayName,
		QuotaDelta:         trace.QuotaDelta,
		CommittedQuota:     trace.CommittedQuota,
		RequestedQuota:     trace.RequestedQuota,
		ApprovedQuota:      trace.ApprovedQuota,
		Status:             trace.Status,
		Fallback:           trace.Fallback,
		OccurredAt:         trace.OccurredAt,
		DetailRoute:        trace.DetailRoute,
		DetailAPIPath:      trace.DetailAPIPath,
		Summary:            trace.Summary,
		RecipientUserId:    trace.RecipientUserId,
		RecipientKind:      trace.RecipientKind,
		NotificationTarget: trace.NotificationTarget,
	}
}
