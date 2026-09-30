package aionui

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
)

const clientHeartbeatMaxBodyBytes = 16 << 10

func ReportClientHeartbeat(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, clientHeartbeatMaxBodyBytes)
	var req dtoaionui.ClientHeartbeatRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		writeClientHeartbeatError(c, clientHeartbeatValidationError())
		return
	}
	result, err := clientHeartbeatService().Report(serviceaionui.ClientHeartbeatInput{
		UserId:        c.GetInt("aionui_user_id"),
		DeviceId:      c.GetString("aionui_device_id"),
		ClientVersion: req.ClientVersion,
		Platform:      req.Platform,
		LanIP:         req.LanIP,
		User:          req.User,
	})
	if err != nil {
		writeClientHeartbeatError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminListClientInstallations(c *gin.Context) {
	var query dtoaionui.ClientInstallationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeClientHeartbeatError(c, clientHeartbeatValidationError())
		return
	}
	result, err := clientHeartbeatService().List(serviceaionui.ClientInstallationQuery{
		Page:       intValue(query.Page),
		PageSize:   intValue(query.PageSize),
		Keyword:    stringValue(query.Keyword),
		Platform:   stringValue(query.Platform),
		Version:    stringValue(query.Version),
		ActiveOnly: boolValue(query.ActiveOnly),
	})
	if err != nil {
		writeClientHeartbeatError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func clientHeartbeatService() *serviceaionui.ClientHeartbeatService {
	return serviceaionui.NewClientHeartbeatService(model.DB)
}

func writeClientHeartbeatError(c *gin.Context, err error) {
	var validationErr serviceaionui.ClientHeartbeatValidationError
	if errors.As(err, &validationErr) || errors.Is(err, serviceaionui.ErrClientHeartbeatInvalidInput) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request params"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "internal server error"})
}

func clientHeartbeatValidationError() error {
	return serviceaionui.ClientHeartbeatValidationError{Message: "invalid request params"}
}

func boolValue(value *bool) bool {
	return value != nil && *value
}
