package aionui

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	DesktopSessionRedirectURI = "aionui_desktop_redirect_uri"
	DesktopSessionState       = "aionui_desktop_state"
)

func DesktopLogin(c *gin.Context) {
	redirectURI := strings.TrimSpace(c.Query("redirect_uri"))
	state := strings.TrimSpace(c.Query("state"))
	if err := serviceaionui.ValidateDesktopRedirectURI(redirectURI); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := serviceaionui.ValidateDesktopState(state); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	session := sessions.Default(c)
	if user, err := currentDesktopSessionUser(session); err != nil {
		common.ApiError(c, err)
		return
	} else if user != nil {
		redirectToDesktopCallback(c, user, redirectURI, state)
		return
	}

	status := getDingTalkOAuthStatusForDesktop()
	if status == nil {
		common.ApiErrorMsg(c, "dingtalk oauth is not enabled")
		return
	}

	session.Set("oauth_state", state)
	session.Set(DesktopSessionRedirectURI, redirectURI)
	session.Set(DesktopSessionState, state)
	if err := session.Save(); err != nil {
		common.ApiError(c, err)
		return
	}

	authURL, err := buildDingTalkAuthURL(status.AppKey, status.CallbackUrl, state)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

func currentDesktopSessionUser(session sessions.Session) (*model.User, error) {
	if session.Get("username") == nil {
		return nil, nil
	}
	userId, ok := desktopSessionInt(session.Get("id"))
	if !ok {
		return nil, errors.New("new-api session user id is invalid")
	}
	return model.GetUserById(userId, false)
}

func desktopSessionInt(value any) (int, bool) {
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

func redirectToDesktopCallback(c *gin.Context, user *model.User, redirectURI string, state string) {
	code, err := serviceaionui.DefaultDesktopAuthService().IssueCode(user, redirectURI, state)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	callbackURL, err := serviceaionui.BuildDesktopCallbackURL(redirectURI, code, state)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Redirect(http.StatusFound, callbackURL)
}

func DesktopToken(c *gin.Context) {
	var req dtoaionui.DesktopTokenRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := serviceaionui.DefaultDesktopAuthService().ExchangeCode(req)
	if err != nil {
		status := http.StatusOK
		if errors.Is(err, serviceaionui.ErrInvalidCode) {
			status = http.StatusUnauthorized
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, result)
}

func buildDingTalkAuthURL(clientID string, callbackURL string, state string) (string, error) {
	u, err := url.Parse("https://login.dingtalk.com/oauth2/auth")
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("client_id", clientID)
	query.Set("redirect_uri", callbackURL)
	query.Set("response_type", "code")
	query.Set("scope", "openid")
	query.Set("state", state)
	query.Set("prompt", "consent")
	u.RawQuery = query.Encode()
	return u.String(), nil
}

type dingTalkOAuthStatus struct {
	AppKey      string
	CallbackUrl string
}

func getDingTalkOAuthStatusForDesktop() *dingTalkOAuthStatus {
	if model.DB == nil {
		return nil
	}
	var config entmodel.DingTalkConfig
	if err := model.DB.Where("tenant_id = ? AND login_enabled = ?", 0, true).First(&config).Error; err != nil {
		return nil
	}
	if strings.TrimSpace(config.AppKey) == "" || strings.TrimSpace(config.CallbackUrl) == "" {
		return nil
	}
	return &dingTalkOAuthStatus{AppKey: config.AppKey, CallbackUrl: config.CallbackUrl}
}
