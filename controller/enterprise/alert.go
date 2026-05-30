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

func writeAlertEventError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrInvalidAlertEventQuery):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}
