package agentplatform

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func CreateResource(c *gin.Context) {
	var req dtoagentplatform.CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}

	item, err := resourceService().Create(apservice.CreateResourceInput{
		ResourceType: req.ResourceType,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Avatar:       req.Avatar,
		OwnerUserId:  req.OwnerUserId,
		TenantId:     valueOrZero(req.TenantId),
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item))
}

func GetResource(c *gin.Context) {
	item, err := resourceService().GetByResourceID(c.Param("id"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item))
}

func ListResources(c *gin.Context) {
	var query dtoagentplatform.ResourceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}

	result, err := resourceService().List(apservice.ListResourcesQuery{
		ResourceType: query.ResourceType,
		OwnerUserId:  query.OwnerUserId,
		TenantId:     query.TenantId,
		Page:         valueOrZero(query.Page),
		PageSize:     valueOrZero(query.PageSize),
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}

	items := make([]dtoagentplatform.ResourceItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapResourceItem(item))
	}
	common.ApiSuccess(c, dtoagentplatform.ResourceListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func resourceService() *apservice.ResourceService {
	return apservice.NewResourceService(model.DB)
}

func writeResourceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apservice.ErrInvalidResourceInput):
		common.ApiErrorMsg(c, "invalid request params")
	case errors.Is(err, apservice.ErrResourceNotFound):
		common.ApiErrorMsg(c, "resource not found")
	default:
		common.ApiError(c, err)
	}
}

func mapResourceItem(item apservice.ResourceItem) dtoagentplatform.ResourceItem {
	avatarURL, _ := apservice.ResolveAgentAvatarURL(context.Background(), item.Avatar, apservice.ArtifactPresignExpiresForAionUI())
	return dtoagentplatform.ResourceItem{
		Id:            item.Id,
		ResourceId:    item.ResourceId,
		ResourceType:  item.ResourceType,
		DisplayName:   item.DisplayName,
		Description:   item.Description,
		Avatar:        item.Avatar,
		AvatarURL:     avatarURL,
		OwnerUserId:   item.OwnerUserId,
		OwnerName:     item.OwnerName,
		Status:        item.Status,
		LatestVersion: item.LatestVersion,
		TenantId:      item.TenantId,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
