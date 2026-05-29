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
