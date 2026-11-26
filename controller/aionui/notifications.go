package aionui

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func ListClientNotifications(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	items, err := apservice.NewNotificationService(model.DB.WithContext(c.Request.Context())).ListEnabled()
	if err != nil {
		writeClientNotificationError(c, err)
		return
	}
	common.ApiSuccess(c, dtoagentplatform.ClientNotificationList{
		ViewerScope: "user:" + strconv.Itoa(c.GetInt("aionui_user_id")), Items: items, Total: len(items),
	})
}

func GetClientNotification(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	record, err := apservice.NewNotificationService(model.DB.WithContext(c.Request.Context())).Get(c.Param("id"), true)
	if err != nil {
		writeClientNotificationError(c, err)
		return
	}
	common.ApiSuccess(c, dtoagentplatform.ClientNotificationDetail{ClientNotificationItem: apservice.NotificationClientItem(record), Content: record.Content})
}

func writeClientNotificationError(c *gin.Context, err error) {
	if errors.Is(err, apservice.ErrNotificationNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": i18n.T(c, i18n.MsgClientNotificationNotFound), "code": "NOTIFICATION_NOT_FOUND"})
		return
	}
	common.SysError(common.LogText("client notification query failed: %v", err))
	c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": i18n.T(c, i18n.MsgClientNotificationsUnavailable), "code": "NOTIFICATION_UNAVAILABLE"})
}
