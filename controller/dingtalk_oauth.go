package controller

import (
	"context"
	"errors"
	"html"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/service"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
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
	if payload.DesktopRedirectURI != "" || payload.DesktopState != "" {
		handleAionUiDesktopLogin(c, result.User, identity, payload.DesktopRedirectURI, payload.DesktopState)
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

func handleAionUiDesktopLogin(
	c *gin.Context,
	user *model.User,
	identity entservice.DingTalkOAuthIdentity,
	redirectURI string,
	state string,
) {
	if redirectURI == "" || state == "" {
		writeAionUiDesktopLoginError(c, "desktop login flow is incomplete")
		return
	}
	if user != nil {
		email := model.NormalizeEmail(user.Email)
		if email == "" {
			email = model.NormalizeEmail(identity.Email)
		}
		if email != "" {
			currentUser, err := model.GetUserById(user.Id, false)
			if err != nil {
				writeAionUiDesktopLoginError(c, "failed to load DingTalk user: "+err.Error())
				return
			}
			if model.NormalizeEmail(currentUser.Email) == "" {
				if err := model.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("email", email).Error; err != nil {
					writeAionUiDesktopLoginError(c, "failed to persist DingTalk email: "+err.Error())
					return
				}
			}
			user.Email = email
		}
	}

	code, err := serviceaionui.DefaultDesktopAuthService().IssueCode(user, redirectURI, state)
	if err != nil {
		writeAionUiDesktopLoginError(c, err.Error())
		return
	}
	callbackURL, err := serviceaionui.BuildDesktopCallbackURL(redirectURI, code, state)
	if err != nil {
		writeAionUiDesktopLoginError(c, err.Error())
		return
	}
	bundle, err := service.CreateLoginSession(user.Id, loginMethodFromContext(c), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		writeAionUiDesktopLoginError(c, err.Error())
		return
	}
	service.WriteRefreshCookie(c, bundle.RefreshToken)
	setAuthNoStore(c)
	model.UpdateUserLastLoginAt(user.Id)
	recordLoginAudit(user, c)
	c.Redirect(http.StatusFound, callbackURL)
}

func writeAionUiDesktopLoginError(c *gin.Context, message string) {
	common.SysError("[AionUi Desktop Login] failed: " + message)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(
		http.StatusBadRequest,
		`<!doctype html><html><head><meta charset="utf-8"><title>AionUi login failed</title></head><body style="font-family:system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;margin:48px;line-height:1.6"><h2>AionUi login failed</h2><p>new-api completed DingTalk login but could not issue a desktop credential.</p><pre style="white-space:pre-wrap;background:#f5f5f5;padding:16px;border-radius:6px">`+
			html.EscapeString(message)+
			`</pre></body></html>`,
	)
}
