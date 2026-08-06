package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeDingTalkOAuthService struct {
	identity      entservice.DingTalkOAuthIdentity
	login         entservice.DingTalkOAuthResult
	err           error
	boundId       int
	affiliateCode string
}

func (f *fakeDingTalkOAuthService) ResolveIdentity(_ context.Context, _ int, code string) (entservice.DingTalkOAuthIdentity, error) {
	if f.err != nil {
		return entservice.DingTalkOAuthIdentity{}, f.err
	}
	if code == "" {
		return entservice.DingTalkOAuthIdentity{}, entservice.ErrDingTalkOAuthCodeMissing
	}
	return f.identity, nil
}

func (f *fakeDingTalkOAuthService) LoginWithIdentity(_ context.Context, _ int, _ entservice.DingTalkOAuthIdentity, affiliateCode string) (entservice.DingTalkOAuthResult, error) {
	if f.err != nil {
		return entservice.DingTalkOAuthResult{}, f.err
	}
	f.affiliateCode = affiliateCode
	return f.login, nil
}

func (f *fakeDingTalkOAuthService) BindIdentityToUser(_ context.Context, _ int, userId int, _ entservice.DingTalkOAuthIdentity) (entmodel.DingTalkIdentity, error) {
	if f.err != nil {
		return entmodel.DingTalkIdentity{}, f.err
	}
	f.boundId = userId
	return entmodel.DingTalkIdentity{UserId: userId}, nil
}

func TestDingTalkOAuthCallbackLogsInWithAuthFlow(t *testing.T) {
	fake := &fakeDingTalkOAuthService{
		identity: entservice.DingTalkOAuthIdentity{UnionId: "union-login"},
		login: entservice.DingTalkOAuthResult{
			User: &model.User{Id: 300, Username: "ding-login", DisplayName: "Ding Login", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"},
		},
	}
	router, state := setupDingTalkOAuthTestRouter(t, fake, model.AuthFlowIntentLogin)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state="+state, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Contains(t, recorder.Body.String(), `"username":"ding-login"`)
	require.Equal(t, "invite-code", fake.affiliateCode)
	_, err := model.GetAuthFlow(state, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeOAuth})
	require.ErrorIs(t, err, model.ErrAuthFlowConsumed)
}

func TestDingTalkOAuthCallbackRedirectsForAionUiDesktopLogin(t *testing.T) {
	serviceaionui.ResetDesktopAuthServiceForTest()
	fake := &fakeDingTalkOAuthService{
		identity: entservice.DingTalkOAuthIdentity{UnionId: "union-desktop"},
		login: entservice.DingTalkOAuthResult{
			User: &model.User{
				Id: 300, Username: "ding-login", DisplayName: "Ding Login", Email: "ding@example.com",
				Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
			},
		},
	}
	router, _ := setupDingTalkOAuthTestRouter(t, fake, model.AuthFlowIntentLogin)
	payload, err := common.Marshal(oauthFlowPayload{
		DesktopRedirectURI: serviceaionui.DesktopRedirectURI,
		DesktopState:       "state-1234567890",
	})
	require.NoError(t, err)
	state, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeOAuth,
		Provider:  "dingtalk",
		Intent:    model.AuthFlowIntentLogin,
		Payload:   string(payload),
		ExpiresAt: time.Now().Add(time.Minute),
	})
	require.NoError(t, err)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state="+state, nil)
	require.Equal(t, http.StatusFound, recorder.Code)
	parsed, err := url.Parse(recorder.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "aionui", parsed.Scheme)
	require.Equal(t, "auth", parsed.Host)
	code := parsed.Query().Get("code")
	require.NotEmpty(t, code)
	require.Equal(t, "state-1234567890", parsed.Query().Get("state"))

	token, err := serviceaionui.DefaultDesktopAuthService().ExchangeCode(dtoaionui.DesktopTokenRequest{
		Code: code, DeviceId: "device-1",
	})
	require.NoError(t, err)
	require.Equal(t, "ding@example.com", token.User.Email)
}

func TestDingTalkOAuthCallbackBindsLoggedInUser(t *testing.T) {
	fake := &fakeDingTalkOAuthService{identity: entservice.DingTalkOAuthIdentity{UnionId: "union-bind"}}
	router, state := setupDingTalkOAuthTestRouter(t, fake, model.AuthFlowIntentBind)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state="+state, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Equal(t, 301, fake.boundId)
}

func TestDingTalkOAuthCallbackRejectsInvalidState(t *testing.T) {
	router, _ := setupDingTalkOAuthTestRouter(t, &fakeDingTalkOAuthService{}, model.AuthFlowIntentLogin)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state=wrong", nil)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "oauth.state_invalid")
}

func TestDingTalkOAuthCallbackMapsOutOfScope(t *testing.T) {
	router, state := setupDingTalkOAuthTestRouter(t, &fakeDingTalkOAuthService{err: entservice.ErrDingTalkOAuthOutOfScope}, model.AuthFlowIntentLogin)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state="+state, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "enterprise.dingtalk.oauth_out_of_scope")
}

func setupDingTalkOAuthTestRouter(t *testing.T, fake *fakeDingTalkOAuthService, intent string) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := newDingTalkOAuthControllerDB(t)
	previousDB := model.DB
	previousLogDB := model.LOG_DB

	previousFactory := newDingTalkOAuthService
	newDingTalkOAuthService = func() dingTalkOAuthService {
		return fake
	}
	t.Cleanup(func() {
		newDingTalkOAuthService = previousFactory
		model.DB = previousDB
		model.LOG_DB = previousLogDB
	})
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.Create(&model.User{Id: 300, Username: "ding-login", DisplayName: "Ding Login", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "dl01", AuthVersion: 1}).Error)

	flowInput := model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeOAuth,
		Provider:  "dingtalk",
		Intent:    intent,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if intent == model.AuthFlowIntentLogin {
		payload, err := common.Marshal(oauthFlowPayload{AffiliateCode: "invite-code"})
		require.NoError(t, err)
		flowInput.Payload = string(payload)
	} else {
		user := &model.User{Id: 301, Username: "current", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "cur1", AuthVersion: 1}
		require.NoError(t, db.Create(user).Error)
		now := time.Now()
		session := &model.UserSession{
			SID: "dingtalk-bind-session", UserID: user.Id, Version: 1, UserAuthVersion: user.AuthVersion,
			Status: model.UserSessionStatusActive, RefreshHash: "dingtalk-bind-refresh", LoginMethod: "password",
			CreatedAt: now.Unix(), LastActiveAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
		}
		require.NoError(t, model.CreateUserSession(session))
		flowInput.UserId = user.Id
		flowInput.SessionId = session.SID
	}
	state, _, err := model.CreateAuthFlow(flowInput)
	require.NoError(t, err)

	router := gin.New()
	router.GET("/api/oauth/dingtalk", HandleDingTalkOAuth)
	return router, state
}

func newDingTalkOAuthControllerDB(t *testing.T) *gorm.DB {
	t.Helper()
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.UserSession{}, &model.AuthFlow{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

func performDingTalkOAuthCallback(router *gin.Engine, target string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
