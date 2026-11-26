package agentplatform

import (
	"errors"
	"io"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func PublishResource(c *gin.Context) {
	req, ok := bindLifecycleActionRequest(c)
	if !ok {
		return
	}
	result, err := lifecycleService().Publish(c.Param("id"), req.Version, c.GetInt("id"), req.RequestId)
	if err != nil {
		writeLifecycleError(c, err)
		return
	}
	common.ApiSuccess(c, mapLifecycleActionResponse(result))
}

func DisableResource(c *gin.Context) {
	req, ok := bindLifecycleActionRequest(c)
	if !ok {
		return
	}
	result, err := lifecycleService().Disable(c.Param("id"), c.GetInt("id"), req.RequestId)
	if err != nil {
		writeLifecycleError(c, err)
		return
	}
	common.ApiSuccess(c, mapLifecycleActionResponse(result))
}

func RevokeResource(c *gin.Context) {
	req, ok := bindLifecycleActionRequest(c)
	if !ok {
		return
	}
	result, err := lifecycleService().Revoke(c.Param("id"), c.GetInt("id"), req.RequestId)
	if err != nil {
		writeLifecycleError(c, err)
		return
	}
	common.ApiSuccess(c, mapLifecycleActionResponse(result))
}

func OfflineResource(c *gin.Context) {
	req, ok := bindLifecycleActionRequest(c)
	if !ok {
		return
	}
	result, err := lifecycleService().Offline(c.Param("id"), c.GetInt("id"), req.RequestId)
	if err != nil {
		writeLifecycleError(c, err)
		return
	}
	common.ApiSuccess(c, mapLifecycleActionResponse(result))
}

func RollbackResourceVersion(c *gin.Context) {
	req, ok := bindLifecycleActionRequest(c)
	if !ok {
		return
	}
	result, err := lifecycleService().Rollback(c.Param("id"), c.Param("version"), c.GetInt("id"), req.RequestId)
	if err != nil {
		writeLifecycleError(c, err)
		return
	}
	common.ApiSuccess(c, mapLifecycleActionResponse(result))
}

func bindLifecycleActionRequest(c *gin.Context) (dtoagentplatform.LifecycleActionRequest, bool) {
	var req dtoagentplatform.LifecycleActionRequest
	if c.Request.ContentLength == 0 {
		return req, true
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return req, true
		}
		common.ApiErrorMsg(c, "invalid request params")
		return req, false
	}
	return req, true
}

func lifecycleService() *apservice.LifecycleService {
	return apservice.NewLifecycleService(model.DB)
}

func writeLifecycleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apservice.ErrInvalidLifecycleInput), errors.Is(err, apservice.ErrRollbackVersionMissing):
		common.ApiErrorMsg(c, "invalid request params")
	case errors.Is(err, apservice.ErrResourceNotFound):
		common.ApiErrorMsg(c, "resource not found")
	default:
		common.ApiError(c, err)
	}
}

func mapLifecycleActionResponse(result apservice.LifecycleActionResult) dtoagentplatform.LifecycleActionResponse {
	return dtoagentplatform.LifecycleActionResponse{
		ResourceId:      result.ResourceId,
		Action:          result.Action,
		PreviousStatus:  result.PreviousStatus,
		CurrentStatus:   result.CurrentStatus,
		PreviousVersion: result.PreviousVersion,
		CurrentVersion:  result.CurrentVersion,
		TargetVersion:   result.TargetVersion,
		RequestId:       result.RequestId,
		AuditActionId:   result.AuditActionId,
	}
}
