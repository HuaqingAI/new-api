package agentplatform

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func ListNotifications(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	page, pageErr := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, sizeErr := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageErr != nil || sizeErr != nil {
		writeNotificationError(c, apservice.ErrInvalidNotification)
		return
	}
	result, err := apservice.NewNotificationService(model.DB.WithContext(c.Request.Context())).List(page, pageSize, c.DefaultQuery("status", "all"))
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetNotification(c *gin.Context) {
	record, err := apservice.NewNotificationService(model.DB.WithContext(c.Request.Context())).Get(c.Param("id"), false)
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	writeNotificationDetail(c, record)
}

func CreateNotification(c *gin.Context) {
	var request dtoagentplatform.NotificationWriteRequest
	if !bindNotificationBody(c, &request, []string{"title", "summary", "content", "enabled"}) {
		return
	}
	if request.Title == nil || request.Content == nil {
		writeNotificationError(c, apservice.ErrInvalidNotification)
		return
	}
	input := apservice.NotificationInput{Title: *request.Title, Content: *request.Content}
	if request.Summary != nil {
		input.Summary = *request.Summary
	}
	if request.Enabled != nil {
		input.Enabled = *request.Enabled
	}
	record, err := apservice.NewNotificationService(model.DB.WithContext(c.Request.Context())).Create(input, c.GetInt("id"))
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	writeNotificationDetail(c, record)
}

func UpdateNotification(c *gin.Context) {
	version, ok := notificationVersion(c)
	if !ok {
		return
	}
	var request dtoagentplatform.NotificationWriteRequest
	if !bindNotificationBody(c, &request, []string{"title", "summary", "content", "enabled"}) {
		return
	}
	if request.Title == nil || request.Summary == nil || request.Content == nil || request.Enabled == nil {
		writeNotificationError(c, apservice.ErrInvalidNotification)
		return
	}
	record, err := apservice.NewNotificationService(model.DB.WithContext(c.Request.Context())).Update(c.Param("id"), version,
		apservice.NotificationInput{Title: *request.Title, Summary: *request.Summary, Content: *request.Content, Enabled: *request.Enabled}, c.GetInt("id"))
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	writeNotificationDetail(c, record)
}

func UpdateNotificationStatus(c *gin.Context) {
	version, ok := notificationVersion(c)
	if !ok {
		return
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if !bindNotificationBody(c, &request, []string{"enabled"}) {
		return
	}
	if request.Enabled == nil {
		writeNotificationError(c, apservice.ErrInvalidNotification)
		return
	}
	record, err := apservice.NewNotificationService(model.DB.WithContext(c.Request.Context())).SetEnabled(c.Param("id"), version, *request.Enabled, c.GetInt("id"))
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	writeNotificationDetail(c, record)
}

func DeleteNotification(c *gin.Context) {
	version, ok := notificationVersion(c)
	if !ok {
		return
	}
	if err := apservice.NewNotificationService(model.DB.WithContext(c.Request.Context())).Delete(c.Param("id"), version); err != nil {
		writeNotificationError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	common.ApiSuccess(c, gin.H{"id": c.Param("id"), "deleted": true})
}

func notificationVersion(c *gin.Context) (int64, bool) {
	c.Header("Cache-Control", "private, no-store")
	values := c.Request.Header.Values("If-Match")
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		common.ApiErrorStatus(c, http.StatusPreconditionRequired, common.NewMessage("Notification version is required"), gin.H{"code": "NOTIFICATION_VERSION_REQUIRED"})
		return 0, false
	}
	value := strings.TrimSpace(values[0])
	if len(values) != 1 || len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		writeNotificationError(c, apservice.ErrInvalidNotification)
		return 0, false
	}
	value = value[1 : len(value)-1]
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil || version <= 0 || strconv.FormatInt(version, 10) != value {
		writeNotificationError(c, apservice.ErrInvalidNotification)
		return 0, false
	}
	return version, true
}

func bindNotificationBody(c *gin.Context, target any, fields []string) bool {
	c.Header("Cache-Control", "private, no-store")
	const maxBytes = 64 * 1024
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBytes+1))
	if len(body) > maxBytes {
		common.ApiErrorStatus(c, http.StatusRequestEntityTooLarge, common.NewMessage("Notification request is too large"), gin.H{"code": "NOTIFICATION_TOO_LARGE"})
		return false
	}
	var object map[string]common.RawMessage
	if err != nil || !utf8.Valid(body) || common.Unmarshal(body, &object) != nil || object == nil {
		writeNotificationError(c, apservice.ErrInvalidNotification)
		return false
	}
	for key, value := range object {
		if !slices.Contains(fields, key) || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			writeNotificationError(c, apservice.ErrInvalidNotification)
			return false
		}
	}
	if common.Unmarshal(body, target) != nil {
		writeNotificationError(c, apservice.ErrInvalidNotification)
		return false
	}
	return true
}

func writeNotificationDetail(c *gin.Context, record apmodel.Notification) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("ETag", `"`+strconv.FormatInt(record.Version, 10)+`"`)
	common.ApiSuccess(c, dtoagentplatform.NotificationDetail{NotificationItem: apservice.NotificationManagementItem(record), Content: record.Content})
}

func writeNotificationError(c *gin.Context, err error) {
	c.Header("Cache-Control", "private, no-store")
	status, code, message := http.StatusInternalServerError, "NOTIFICATION_UNAVAILABLE", "Notifications are temporarily unavailable"
	switch {
	case errors.Is(err, apservice.ErrInvalidNotification):
		status, code, message = http.StatusBadRequest, "NOTIFICATION_INVALID", "Invalid notification request"
	case errors.Is(err, apservice.ErrNotificationNotFound):
		status, code, message = http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Notification not found"
	case errors.Is(err, apservice.ErrNotificationConflict):
		status, code, message = http.StatusPreconditionFailed, "NOTIFICATION_VERSION_CONFLICT", "Notification has changed. Reload and try again."
	default:
		common.SysError(common.LogText("notification operation failed: %v", err))
	}
	common.ApiErrorStatus(c, status, common.NewMessage(message), gin.H{"code": code})
}
