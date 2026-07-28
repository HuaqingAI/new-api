package controller

import (
	"context"
	"errors"
	"html"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	controlleraionui "github.com/QuantumNous/new-api/controller/aionui"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var newDingTalkOAuthService = func() dingTalkOAuthService {
	return entservice.NewDingTalkOAuthService(model.DB, nil)
}

type dingTalkOAuthService interface {
	ResolveIdentity(c context.Context, tenantId int, code string) (entservice.DingTalkOAuthIdentity, error)
	LoginWithIdentity(c context.Context, tenantId int, identity entservice.DingTalkOAuthIdentity, session sessions.Session) (entservice.DingTalkOAuthResult, error)
	BindIdentityToUser(c context.Context, tenantId int, userId int, identity entservice.DingTalkOAuthIdentity) (entmodel.DingTalkIdentity, error)
}

func HandleDingTalkOAuth(c *gin.Context) {
	session := sessions.Default(c)
	if !validateDingTalkOAuthState(c, session) {
		return
	}

	tenantId := parseDingTalkOAuthTenantId(c)
	oauthService := newDingTalkOAuthService()
	identity, err := oauthService.ResolveIdentity(c.Request.Context(), tenantId, c.Query("code"))
	if err != nil {
		writeDingTalkOAuthError(c, err)
		return
	}

	if session.Get("username") != nil {
		userId, ok := sessionInt(session.Get("id"))
		if !ok {
			common.ApiErrorI18n(c, i18n.MsgAuthUserInfoInvalid)
			return
		}
		if _, err := oauthService.BindIdentityToUser(c.Request.Context(), tenantId, userId, identity); err != nil {
			writeDingTalkOAuthError(c, err)
			return
		}
		common.ApiSuccessI18n(c, i18n.MsgOAuthBindSuccess, gin.H{"action": "bind"})
		return
	}

	result, err := oauthService.LoginWithIdentity(c.Request.Context(), tenantId, identity, session)
	if err != nil {
		writeDingTalkOAuthError(c, err)
		return
	}
	if handleAionUiDesktopLogin(c, result.User, identity) {
		return
	}
	setupLogin(result.User, c)
}

func validateDingTalkOAuthState(c *gin.Context, session sessions.Session) bool {
	state := c.Query("state")
	expected := session.Get("oauth_state")
	expectedState, ok := expected.(string)
	if state == "" || !ok || state != expectedState {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgOAuthStateInvalid),
		})
		return false
	}
	if c.Query("error") != "" {
		common.ApiErrorMsg(c, c.Query("error_description"))
		return false
	}
	return true
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

func sessionInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func handleAionUiDesktopLogin(c *gin.Context, user *model.User, identity entservice.DingTalkOAuthIdentity) bool {
	session := sessions.Default(c)
	redirectURI, _ := session.Get(controlleraionui.DesktopSessionRedirectURI).(string)
	state, _ := session.Get(controlleraionui.DesktopSessionState).(string)
	if redirectURI == "" && state == "" {
		return false
	}
	if redirectURI == "" || state == "" {
		writeAionUiDesktopLoginError(c, "desktop login session is incomplete")
		return true
	}

	if user != nil && model.NormalizeEmail(user.Email) == "" && model.NormalizeEmail(identity.Email) != "" {
		email := model.NormalizeEmail(identity.Email)
		if err := model.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("email", email).Error; err != nil {
			writeAionUiDesktopLoginError(c, "failed to persist DingTalk email: "+err.Error())
			return true
		}
		user.Email = email
	}

	code, err := serviceaionui.DefaultDesktopAuthService().IssueCode(user, redirectURI, state)
	if err != nil {
		writeAionUiDesktopLoginError(c, err.Error())
		return true
	}
	callbackURL, err := serviceaionui.BuildDesktopCallbackURL(redirectURI, code, state)
	if err != nil {
		writeAionUiDesktopLoginError(c, err.Error())
		return true
	}
	if err := setupLoginSession(user, c); err != nil {
		writeAionUiDesktopLoginError(c, err.Error())
		return true
	}
	session.Delete(controlleraionui.DesktopSessionRedirectURI)
	session.Delete(controlleraionui.DesktopSessionState)
	session.Delete("oauth_state")
	if err := session.Save(); err != nil {
		writeAionUiDesktopLoginError(c, err.Error())
		return true
	}
	c.Redirect(http.StatusFound, callbackURL)
	return true
}

func writeAionUiDesktopLoginError(c *gin.Context, message string) {
	common.SysError("[AionUi Desktop Login] failed: " + message)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(
		http.StatusBadRequest,
		`<!doctype html><html><head><meta charset="utf-8"><title>AionUi 登录失败</title></head><body style="font-family:system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;margin:48px;line-height:1.6"><h2>AionUi 登录失败</h2><p>new-api 已完成钉钉登录，但无法向 AionUi 签发桌面登录凭证。</p><pre style="white-space:pre-wrap;background:#f5f5f5;padding:16px;border-radius:6px">`+
			html.EscapeString(message)+
			`</pre><p>请保留此页面并查看 new-api 控制台日志中的 [AionUi Desktop Login] failed 记录。</p></body></html>`,
	)
}
