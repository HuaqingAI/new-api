package aionui

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/service"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
)

const desktopOAuthFlowTTL = 10 * time.Minute

type desktopOAuthFlowPayload struct {
	DesktopRedirectURI string `json:"desktop_redirect_uri"`
	DesktopState       string `json:"desktop_state"`
}

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

	if user, err := currentDesktopSessionUser(c); err != nil {
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

	payload, err := common.Marshal(desktopOAuthFlowPayload{
		DesktopRedirectURI: redirectURI,
		DesktopState:       state,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	flowState, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeOAuth,
		Provider:  "dingtalk",
		Intent:    model.AuthFlowIntentLogin,
		Payload:   string(payload),
		ExpiresAt: time.Now().Add(desktopOAuthFlowTTL),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	authURL, err := buildDingTalkAuthURL(status.AppKey, status.CallbackUrl, flowState)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

func currentDesktopSessionUser(c *gin.Context) (*model.User, error) {
	rawRefreshToken, err := c.Cookie(service.RefreshCookieName)
	if err != nil || rawRefreshToken == "" {
		return nil, nil
	}
	bundle, user, err := service.RefreshLoginSession(rawRefreshToken, "", c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, service.ErrRefreshTokenInvalid) || errors.Is(err, service.ErrLoginSessionRevoked) {
			return nil, nil
		}
		return nil, err
	}
	service.WriteRefreshCookie(c, bundle.RefreshToken)
	return user, nil
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
