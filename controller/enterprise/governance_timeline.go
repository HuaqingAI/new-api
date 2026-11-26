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

func ListGovernanceTimeline(c *gin.Context) {
	var query dtoenterprise.GovernanceTimelineQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}
	result, err := entservice.NewGovernanceTimelineService(model.DB).List(entservice.GovernanceTimelineQuery{
		TenantId:     tenantId,
		DepartmentId: query.DepartmentId,
		SourceId:     query.SourceId,
		ActorId:      query.ActorId,
		ActionType:   stringPtrValue(query.ActionType),
		SourceType:   stringPtrValue(query.SourceType),
		Status:       stringPtrValue(query.Status),
		StartAt:      query.StartAt,
		EndAt:        query.EndAt,
		Page:         valueOrZero(query.Page),
		PageSize:     valueOrZero(query.PageSize),
		ViewerId:     c.GetInt("id"),
	})
	if err != nil {
		writeGovernanceError(c, err)
		return
	}
	items := make([]dtoenterprise.GovernanceTimelineItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapGovernanceTimelineItem(item))
	}
	common.ApiSuccess(c, dtoenterprise.GovernanceTimelineResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func mapGovernanceTimelineItem(item entservice.GovernanceTimelineItem) dtoenterprise.GovernanceTimelineItem {
	return dtoenterprise.GovernanceTimelineItem{
		TraceId:    item.TraceId,
		SourceType: item.SourceType,
		SourceId:   item.SourceId,
		ActionType: item.ActionType,
		TenantId:   item.TenantId,
		ActorId:    item.ActorId,
		ActorName:  item.ActorName,
		Target: dtoenterprise.GovernanceTimelineTarget{
			DepartmentId:   item.Target.DepartmentId,
			DepartmentName: item.Target.DepartmentName,
			UserId:         item.Target.UserId,
			Username:       item.Target.Username,
			DisplayName:    item.Target.DisplayName,
			ObjectType:     item.Target.ObjectType,
			ObjectId:       item.Target.ObjectId,
		},
		QuotaDelta:    item.QuotaDelta,
		BeforeQuota:   item.BeforeQuota,
		AfterQuota:    item.AfterQuota,
		Status:        item.Status,
		OccurredAt:    item.OccurredAt,
		DetailRoute:   item.DetailRoute,
		DetailAPIPath: item.DetailAPIPath,
		Summary:       item.Summary,
	}
}

func writeGovernanceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrInvalidGovernanceTimelineQuery),
		errors.Is(err, entservice.ErrInvalidGovernanceNotificationQuery):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrGovernanceNotificationDeliveryNotFound):
		common.ApiErrorI18n(c, i18n.MsgNotFound)
	case errors.Is(err, entservice.ErrGovernanceNotificationResendNotAllowed):
		common.ApiErrorI18n(c, i18n.MsgForbidden)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
