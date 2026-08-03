package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/service"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
)

var newDingTalkOAuthService = func() dingTalkOAuthService {
	return entservice.NewDingTalkOAuthService(model.DB, nil)
}

type dingTalkOAuthService interface {
	ResolveIdentity(c context.Context, tenantId int, code string) (entservice.DingTalkOAuthIdentity, error)
	LoginWithIdentity(c context.Context, tenantId int, identity entservice.DingTalkOAuthIdentity, affiliateCode string) (entservice.DingTalkOAuthResult, error)
	BindIdentityToUser(c context.Context, tenantId int, userId int, identity entservice.DingTalkOAuthIdentity) (entmodel.DingTalkIdentity, error)
}

func HandleDingTalkOAuth(c *gin.Context) {
	state := c.Query("state")
	pendingFlow, err := model.GetAuthFlow(state, model.AuthFlowMatch{
		Purpose:  model.AuthFlowPurposeOAuth,
		Provider: "dingtalk",
	})
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgOAuthStateInvalid),
		})
		return
	}
	consumeMatch := model.AuthFlowMatch{
		Purpose:  model.AuthFlowPurposeOAuth,
		Provider: "dingtalk",
		Intent:   pendingFlow.Intent,
	}
	if pendingFlow.Intent == model.AuthFlowIntentBind {
		if _, err := service.ValidateSessionReference(pendingFlow.UserId, pendingFlow.SessionId); err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgOAuthStateInvalid),
			})
			return
		}
		consumeMatch.UserId = pendingFlow.UserId
		consumeMatch.SessionId = pendingFlow.SessionId
	} else if pendingFlow.Intent != model.AuthFlowIntentLogin {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if c.Query("error") != "" {
		if _, err := model.ConsumeAuthFlow(state, consumeMatch); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": common.TranslateMessage(c, i18n.MsgOAuthStateInvalid)})
			return
		}
		common.ApiErrorMsg(c, c.Query("error_description"))
		return
	}

	tenantId := parseDingTalkOAuthTenantId(c)
	oauthService := newDingTalkOAuthService()
	identity, err := oauthService.ResolveIdentity(c.Request.Context(), tenantId, c.Query("code"))
	if err != nil {
		writeDingTalkOAuthError(c, err)
		return
	}
	flow, err := model.ConsumeAuthFlow(state, consumeMatch)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": common.TranslateMessage(c, i18n.MsgOAuthStateInvalid)})
		return
	}

	if flow.Intent == model.AuthFlowIntentBind {
		if _, err := oauthService.BindIdentityToUser(c.Request.Context(), tenantId, flow.UserId, identity); err != nil {
			writeDingTalkOAuthError(c, err)
			return
		}
		common.ApiSuccessI18n(c, i18n.MsgOAuthBindSuccess, gin.H{"action": "bind"})
		return
	}

	var payload oauthFlowPayload
	if err := common.UnmarshalJsonStr(flow.Payload, &payload); err != nil {
		common.ApiError(c, err)
		return
	}
	result, err := oauthService.LoginWithIdentity(c.Request.Context(), tenantId, identity, payload.AffiliateCode)
	if err != nil {
		writeDingTalkOAuthError(c, err)
		return
	}
	setupLogin(result.User, c)
}

func parseDingTalkOAuthTenantId(c *gin.Context) int {
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

func writeDingTalkOAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrDingTalkOAuthNotEnabled), errors.Is(err, entservice.ErrDingTalkConfigNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkOAuthNotEnabled)
	case errors.Is(err, entservice.ErrDingTalkMissingCredentials):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkMissingCredentials)
	case errors.Is(err, entservice.ErrDingTalkOAuthCodeMissing), errors.Is(err, entservice.ErrDingTalkOAuthIdentityMissing):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkOAuthInvalidIdentity)
	case errors.Is(err, entservice.ErrDingTalkOAuthProviderFailed):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkOAuthProviderFailed)
	case errors.Is(err, entservice.ErrDingTalkOAuthUserDisabled):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkOAuthUserDisabled)
	case errors.Is(err, entservice.ErrDingTalkOAuthOutOfScope):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkOAuthOutOfScope)
	case errors.Is(err, entservice.ErrDingTalkOAuthBindingConflict):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkOAuthBindingConflict)
	case errors.Is(err, entservice.ErrDingTalkOAuthRegistrationDisabled):
		common.ApiErrorI18n(c, i18n.MsgUserRegisterDisabled)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}
