package agentplatform

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func CreateExposure(c *gin.Context) {
	var req dtoagentplatform.CreateExposureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := exposureService().Create(c.Param("id"), apservice.CreateExposureInput{
		ResourceVersion:     req.ResourceVersion,
		ClientKey:           req.ClientKey,
		ClientScope:         req.ClientScope,
		VisibilityState:     req.VisibilityState,
		CallableState:       req.CallableState,
		FreshnessTTLSeconds: valueOrZero(req.FreshnessTTLSeconds),
		ETag:                req.ETag,
		Extensions:          req.Extensions,
	})
	if err != nil {
		writeExposureError(c, err)
		return
	}
	common.ApiSuccess(c, mapExposureItem(item))
}

func UpdateExposure(c *gin.Context) {
	var req dtoagentplatform.UpdateExposureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := exposureService().Update(c.Param("id"), c.Param("target"), apservice.UpdateExposureInput{
		VisibilityState:     req.VisibilityState,
		CallableState:       req.CallableState,
		FreshnessTTLSeconds: req.FreshnessTTLSeconds,
		ETag:                req.ETag,
		Extensions:          req.Extensions,
	})
	if err != nil {
		writeExposureError(c, err)
		return
	}
	common.ApiSuccess(c, mapExposureItem(item))
}

func RevokeExposure(c *gin.Context) {
	item, err := exposureService().Revoke(c.Param("id"), c.Param("target"))
	if err != nil {
		writeExposureError(c, err)
		return
	}
	common.ApiSuccess(c, mapExposureItem(item))
}

func GetExposure(c *gin.Context) {
	item, err := exposureService().Get(c.Param("id"), c.Param("target"))
	if err != nil {
		writeExposureError(c, err)
		return
	}
	common.ApiSuccess(c, mapExposureItem(item))
}

func ListExposures(c *gin.Context) {
	var query dtoagentplatform.ExposureQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := exposureService().List(apservice.ExposureQuery{
		ResourceId:      query.ResourceId,
		ResourceVersion: query.ResourceVersion,
		ClientKey:       query.ClientKey,
		ClientScope:     query.ClientScope,
	})
	if err != nil {
		writeExposureError(c, err)
		return
	}
	items := make([]dtoagentplatform.ExposureItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapExposureItem(item))
	}
	common.ApiSuccess(c, dtoagentplatform.ExposureListResponse{Items: items, Total: result.Total})
}

func exposureService() *apservice.ExposureService {
	return apservice.NewExposureService(model.DB)
}

func writeExposureError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apservice.ErrInvalidExposureInput):
		common.ApiErrorMsg(c, "invalid request params")
	case errors.Is(err, apservice.ErrExposureNotFound), errors.Is(err, apservice.ErrResourceVersionNotFound):
		common.ApiErrorMsg(c, "resource not found")
	default:
		common.ApiError(c, err)
	}
}

func mapExposureItem(item apservice.ExposureItem) dtoagentplatform.ExposureItem {
	response := dtoagentplatform.ExposureItem{
		Id:                  item.Id,
		ResourceId:          item.ResourceId,
		ResourceVersion:     item.ResourceVersion,
		ClientKey:           item.ClientKey,
		ClientScope:         item.ClientScope,
		VisibilityState:     item.VisibilityState,
		CallableState:       item.CallableState,
		FreshnessTTLSeconds: item.FreshnessTTLSeconds,
		ETag:                item.ETag,
		Extensions:          jsonTextToRawMessage(item.ExtensionsJSON),
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
	if item.PublishedAt != nil {
		publishedAt := item.PublishedAt.Unix()
		response.PublishedAt = &publishedAt
	}
	if item.RevokedAt != nil {
		revokedAt := item.RevokedAt.Unix()
		response.RevokedAt = &revokedAt
	}
	return response
}
