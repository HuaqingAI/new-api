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

func ListAdminActions(c *gin.Context) {
	var query dtoenterprise.AdminActionQuery
	_ = c.ShouldBindQuery(&query)

	result, err := adminActionService().List(entservice.AdminActionQuery{
		TenantId:   query.TenantId,
		ActorId:    query.ActorId,
		ActionType: query.ActionType,
		ObjectType: query.ObjectType,
		ObjectId:   query.ObjectId,
		StartAt:    query.StartAt,
		EndAt:      query.EndAt,
		Page:       valueOrZero(query.Page),
		PageSize:   valueOrZero(query.PageSize),
	})
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	common.ApiSuccess(c, dtoenterprise.AdminActionsResponse{
		Items:    mapAdminActionItems(result.Items, false),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func GetAdminAction(c *gin.Context) {
	actionId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	item, err := adminActionService().Get(actionId)
	if err != nil {
		writeAdminActionError(c, err)
		return
	}
	common.ApiSuccess(c, mapAdminActionItem(item, true))
}

func adminActionService() *entservice.AdminActionService {
	return entservice.NewAdminActionService(model.DB)
}

func writeAdminActionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrAdminActionNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseAdminActionNotFound)
	case errors.Is(err, entservice.ErrInvalidAdminActionInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func mapAdminActionItems(items []entservice.AdminActionItem, includePayload bool) []dtoenterprise.AdminActionItem {
	out := make([]dtoenterprise.AdminActionItem, 0, len(items))
	for _, item := range items {
		out = append(out, mapAdminActionItem(item, includePayload))
	}
	return out
}

func mapAdminActionItem(item entservice.AdminActionItem, includePayload bool) dtoenterprise.AdminActionItem {
	dto := dtoenterprise.AdminActionItem{
		ActionId:    item.ActionId,
		TenantId:    item.TenantId,
		ActorId:     item.ActorId,
		ActionType:  item.ActionType,
		ObjectType:  item.ObjectType,
		ObjectId:    item.ObjectId,
		CreatedAt:   item.CreatedAt,
		DiffSummary: item.DiffSummary,
	}
	if includePayload {
		dto.Payload = item.Payload
	}
	return dto
}
