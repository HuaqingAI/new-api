package enterprise

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDingTalkConfig(c *gin.Context) {
	tenantId := parseTenantIdQuery(c)
	result, err := entservice.NewDingTalkConfigService(model.DB).Get(tenantId)
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func SaveDingTalkConfig(c *gin.Context) {
	var req dtoenterprise.DingTalkConfigRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	input := entservice.DingTalkConfigInput{
		TenantId:     valueOrZero(req.TenantId),
		CorpId:       req.CorpId,
		AppKey:       req.AppKey,
		AppSecret:    req.AppSecret,
		CallbackUrl:  req.CallbackUrl,
		SyncScope:    req.SyncScope,
		LoginEnabled: boolValue(req.LoginEnabled),
		SyncEnabled:  boolValue(req.SyncEnabled),
	}

	var result dtoenterprise.DingTalkConfigResponse
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = entservice.NewDingTalkConfigService(tx).Save(input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    result.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionDingTalkConfigSet,
			ObjectType:  entservice.AdminObjectDingTalkConfig,
			ObjectId:    strconv.Itoa(result.TenantId),
			DiffSummary: "Saved DingTalk enterprise app configuration",
			Payload: map[string]any{
				"tenant_id":      result.TenantId,
				"corp_id":        result.CorpId,
				"app_key":        result.AppKey,
				"app_secret":     req.AppSecret,
				"callback_url":   result.CallbackUrl,
				"sync_scope":     result.SyncScope,
				"login_enabled":  result.LoginEnabled,
				"sync_enabled":   result.SyncEnabled,
				"has_app_secret": result.HasAppSecret,
			},
		})
	})
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}

	common.ApiSuccessI18n(c, i18n.MsgEnterpriseDingTalkConfigSaved, result)
}

func parseTenantIdQuery(c *gin.Context) int {
	raw := c.Query("tenant_id")
	if raw == "" {
		return 0
	}
	tenantId, err := strconv.Atoi(raw)
	if err != nil || tenantId < 0 {
		return 0
	}
	return tenantId
}

func writeDingTalkConfigError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrDingTalkMissingCredentials):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkMissingCredentials)
	case errors.Is(err, entservice.ErrDingTalkInvalidCallbackURL):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkInvalidCallbackURL)
	case errors.Is(err, entservice.ErrDingTalkConfigNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkConfigNotFound)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}
