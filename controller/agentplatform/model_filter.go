package agentplatform

import (
	"io"
	"net/http"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func GetClientModelFilter(c *gin.Context) {
	models, err := setting.GetClientModelFilterExcludedModels()
	if err != nil {
		common.SysError("failed to read client model filter")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "code": "CLIENT_MODEL_FILTER_UNAVAILABLE", "message": common.TranslateMessage(c, i18n.MsgClientModelFilterLoadFailed)})
		return
	}
	common.ApiSuccess(c, gin.H{
		"target_token_name": serviceaionui.DefaultPersonalAPIKeyName,
		"default_model":     serviceaionui.DefaultAgentModel,
		"excluded_models":   models,
	})
}

func UpdateClientModelFilter(c *gin.Context) {
	const maxRequestBytes = 64 * 1024
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxRequestBytes+1))
	if len(body) > maxRequestBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "code": "CLIENT_MODEL_FILTER_TOO_LARGE", "message": common.TranslateMessage(c, i18n.MsgClientModelFilterTooLarge)})
		return
	}
	var request struct {
		ExcludedModels common.RawMessage `json:"excluded_models"`
	}
	if err != nil || !utf8.Valid(body) || common.Unmarshal(body, &request) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "CLIENT_MODEL_FILTER_INVALID", "message": common.TranslateMessage(c, i18n.MsgClientModelFilterInvalid)})
		return
	}
	models, canonical, err := setting.NormalizeClientModelFilter(string(request.ExcludedModels))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "CLIENT_MODEL_FILTER_INVALID", "message": common.TranslateMessage(c, i18n.MsgClientModelFilterInvalid)})
		return
	}
	if err := model.UpdateOptionsBulk(map[string]string{setting.ClientModelFilterExcludedModelsKey: canonical}); err != nil {
		common.SysError("failed to save client model filter")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "code": "CLIENT_MODEL_FILTER_SAVE_FAILED", "message": common.TranslateMessage(c, i18n.MsgClientModelFilterSaveFailed)})
		return
	}
	common.ApiSuccess(c, gin.H{
		"target_token_name": serviceaionui.DefaultPersonalAPIKeyName,
		"default_model":     serviceaionui.DefaultAgentModel,
		"excluded_models":   models,
	})
}
