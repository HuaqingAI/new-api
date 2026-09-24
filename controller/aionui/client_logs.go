package aionui

import (
	"errors"
	"fmt"
	"math"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
)

const clientLogUploadMultipartOverheadBytes = int64(1024 * 1024)

var clientLogUploadAuditVersionPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._+-]{0,63}$`)

func UploadClientLogs(c *gin.Context) {
	startedAt := time.Now()
	audit := clientLogUploadAudit{
		date:          sanitizedClientLogAuditDate(c.Param("date")),
		deviceHash:    common.AionUiDeviceAuditHash(c.GetString("aionui_device_id")),
		clientVersion: sanitizedClientLogAuditVersion(c.GetHeader("X-AionUI-Client-Version")),
		resultCode:    "internal_error",
		userID:        c.GetInt("aionui_user_id"),
	}
	defer func() {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf(
			"event=aionui_client_log_upload user_id=%d device_hash=%s client_version=%s log_date=%s file_count=%d total_bytes=%d result=%s duration_ms=%d",
			audit.userID,
			audit.deviceHash,
			audit.clientVersion,
			audit.date,
			audit.fileCount,
			audit.totalBytes,
			audit.resultCode,
			time.Since(startedAt).Milliseconds(),
		))
	}()
	if !common.AionUiClientLogUploadEnabled {
		audit.resultCode = "storage_unavailable"
		writeClientLogUploadError(c, serviceaionui.ClientLogUploadError{Code: "storage_unavailable", HTTPStatus: http.StatusServiceUnavailable})
		return
	}
	mediaType, parameters, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") || strings.TrimSpace(parameters["boundary"]) == "" {
		audit.resultCode = "unsupported_media_type"
		writeClientLogUploadError(c, serviceaionui.ClientLogUploadError{Code: "unsupported_media_type", HTTPStatus: http.StatusUnsupportedMediaType})
		return
	}
	limits := serviceaionui.DefaultClientLogUploadLimits()
	if limits.MaxTotalBytes > math.MaxInt64-clientLogUploadMultipartOverheadBytes {
		audit.resultCode = "storage_unavailable"
		writeClientLogUploadError(c, serviceaionui.ClientLogUploadError{Code: "storage_unavailable", HTTPStatus: http.StatusServiceUnavailable})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limits.MaxTotalBytes+clientLogUploadMultipartOverheadBytes)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		audit.resultCode = "invalid_multipart"
		writeClientLogUploadError(c, serviceaionui.ClientLogUploadError{Code: "invalid_multipart", HTTPStatus: http.StatusBadRequest})
		return
	}
	result, err := serviceaionui.NewClientLogUploadService(limits).Upload(c.Request.Context(), serviceaionui.ClientLogUploadInput{
		ClientVersion: c.GetHeader("X-AionUI-Client-Version"),
		Date:          c.Param("date"),
		Email:         c.GetString("aionui_email"),
		Reader:        reader,
		UserID:        c.GetInt("aionui_user_id"),
	})
	if err != nil {
		var uploadError serviceaionui.ClientLogUploadError
		if errors.As(err, &uploadError) {
			audit.resultCode = uploadError.Code
			writeClientLogUploadError(c, uploadError)
			return
		}
		audit.resultCode = "storage_unavailable"
		writeClientLogUploadError(c, serviceaionui.ClientLogUploadError{Code: "storage_unavailable", HTTPStatus: http.StatusBadGateway})
		return
	}
	audit.date = result.Date
	audit.clientVersion = result.ClientVersion
	audit.fileCount = result.FileCount
	audit.totalBytes = result.TotalBytes
	audit.resultCode = "success"
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": result})
}

func writeClientLogUploadError(c *gin.Context, err serviceaionui.ClientLogUploadError) {
	c.JSON(err.HTTPStatus, gin.H{"success": false, "message": err.Code, "code": err.Code})
}

type clientLogUploadAudit struct {
	clientVersion string
	date          string
	deviceHash    string
	fileCount     int
	resultCode    string
	totalBytes    int64
	userID        int
}

func sanitizedClientLogAuditDate(value string) string {
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Format("2006-01-02") != value {
		return ""
	}
	return value
}

func sanitizedClientLogAuditVersion(value string) string {
	if !clientLogUploadAuditVersionPattern.MatchString(value) {
		return ""
	}
	return value
}
