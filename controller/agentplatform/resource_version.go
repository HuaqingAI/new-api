package agentplatform

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func CreateResourceVersion(c *gin.Context) {
	var req dtoagentplatform.CreateResourceVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}

	item, err := resourceVersionService().Create(c.Param("id"), apservice.CreateResourceVersionInput{
		Version:         req.Version,
		ContractVersion: req.ContractVersion,
		Summary:         req.Summary,
		CreatedBy:       c.GetInt("id"),
	})
	if err != nil {
		writeResourceVersionError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceVersionItem(item))
}

func GetResourceVersion(c *gin.Context) {
	item, err := resourceVersionService().Get(c.Param("id"), c.Param("version"))
	if err != nil {
		writeResourceVersionError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceVersionItem(item))
}

func resourceVersionService() *apservice.ResourceVersionService {
	return apservice.NewResourceVersionService(model.DB)
}

func writeResourceVersionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apservice.ErrInvalidResourceInput), errors.Is(err, apservice.ErrInvalidResourceVersionInput):
		common.ApiErrorMsg(c, "invalid request params")
	case errors.Is(err, apservice.ErrResourceNotFound), errors.Is(err, apservice.ErrResourceVersionNotFound):
		common.ApiErrorMsg(c, "resource not found")
	default:
		common.ApiError(c, err)
	}
}

func mapResourceVersionItem(item apservice.ResourceVersionItem) dtoagentplatform.ResourceVersionItem {
	response := dtoagentplatform.ResourceVersionItem{
		ResourceId:      item.ResourceId,
		ResourceType:    item.ResourceType,
		Version:         item.Version,
		ContractVersion: item.ContractVersion,
		Summary:         item.Summary,
		Status:          item.Status,
		CreatedBy:       item.CreatedBy,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
	if item.PublishedAt != nil {
		publishedAt := item.PublishedAt.Unix()
		response.PublishedAt = &publishedAt
	}
	return response
}
