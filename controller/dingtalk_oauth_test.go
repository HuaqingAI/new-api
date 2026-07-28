package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	controlleraionui "github.com/QuantumNous/new-api/controller/aionui"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeDingTalkOAuthService struct {
	identity entservice.DingTalkOAuthIdentity
	login    entservice.DingTalkOAuthResult
	err      error
	boundId  int
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

func (f *fakeDingTalkOAuthService) LoginWithIdentity(_ context.Context, _ int, _ entservice.DingTalkOAuthIdentity, _ sessions.Session) (entservice.DingTalkOAuthResult, error) {
	if f.err != nil {
		return entservice.DingTalkOAuthResult{}, f.err
	}
	return f.login, nil
}

func (f *fakeDingTalkOAuthService) BindIdentityToUser(_ context.Context, _ int, userId int, _ entservice.DingTalkOAuthIdentity) (entmodel.DingTalkIdentity, error) {
	if f.err != nil {
		return entmodel.DingTalkIdentity{}, f.err
	}
	f.boundId = userId
	return entmodel.DingTalkIdentity{UserId: userId}, nil
}

func TestDingTalkOAuthCallbackLogsInWithExistingSessionFlow(t *testing.T) {
	fake := &fakeDingTalkOAuthService{
		identity: entservice.DingTalkOAuthIdentity{UnionId: "union-login"},
		login: entservice.DingTalkOAuthResult{
			User: &model.User{Id: 300, Username: "ding-login", DisplayName: "Ding Login", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"},
		},
	}
	router := setupDingTalkOAuthTestRouter(t, fake, false)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state=state-1", nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Contains(t, recorder.Body.String(), `"username":"ding-login"`)
}

func TestDingTalkOAuthCallbackRedirectsForAionUiDesktopLogin(t *testing.T) {
	serviceaionui.ResetDesktopAuthServiceForTest()
	fake := &fakeDingTalkOAuthService{
		identity: entservice.DingTalkOAuthIdentity{UnionId: "union-desktop"},
		login: entservice.DingTalkOAuthResult{
			User: &model.User{
				Id:          300,
				Username:    "ding-login",
				DisplayName: "Ding Login",
				Email:       "ding@example.com",
				Role:        common.RoleCommonUser,
				Status:      common.UserStatusEnabled,
				Group:       "default",
			},
		},
	}
	router := setupDingTalkOAuthDesktopTestRouter(t, fake)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state=state-1234567890", nil)

	require.Equal(t, http.StatusFound, recorder.Code)
	location := recorder.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "aionui", parsed.Scheme)
	require.Equal(t, "auth", parsed.Host)
	require.Equal(t, "/new-api-callback", parsed.Path)
	require.Equal(t, "state-1234567890", parsed.Query().Get("state"))
	code := parsed.Query().Get("code")
	require.NotEmpty(t, code)

	token, err := serviceaionui.DefaultDesktopAuthService().ExchangeCode(dtoaionui.DesktopTokenRequest{
		Code:     code,
		DeviceId: "device-1",
	})
	require.NoError(t, err)
	require.Equal(t, "ding@example.com", token.User.Email)
}

func TestDingTalkOAuthCallbackBindsLoggedInUser(t *testing.T) {
	fake := &fakeDingTalkOAuthService{identity: entservice.DingTalkOAuthIdentity{UnionId: "union-bind"}}
	router := setupDingTalkOAuthTestRouter(t, fake, true)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state=state-1", nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Equal(t, 301, fake.boundId)
}

func TestDingTalkOAuthCallbackRejectsInvalidState(t *testing.T) {
	router := setupDingTalkOAuthTestRouter(t, &fakeDingTalkOAuthService{}, false)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state=wrong", nil)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "oauth.state_invalid")
}

func TestDingTalkOAuthCallbackMapsOutOfScope(t *testing.T) {
	router := setupDingTalkOAuthTestRouter(t, &fakeDingTalkOAuthService{err: entservice.ErrDingTalkOAuthOutOfScope}, false)

	recorder := performDingTalkOAuthCallback(router, "/api/oauth/dingtalk?code=ok&state=state-1", nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "enterprise.dingtalk.oauth_out_of_scope")
}

func setupDingTalkOAuthTestRouter(t *testing.T, fake *fakeDingTalkOAuthService, loggedIn bool) *gin.Engine {
	t.Helper()
	return setupDingTalkOAuthTestRouterWithSession(t, fake, func(session sessions.Session) {
		session.Set("oauth_state", "state-1")
		if loggedIn {
			session.Set("id", 301)
			session.Set("username", "current")
		}
	})
}

func setupDingTalkOAuthDesktopTestRouter(t *testing.T, fake *fakeDingTalkOAuthService) *gin.Engine {
	t.Helper()
	return setupDingTalkOAuthTestRouterWithSession(t, fake, func(session sessions.Session) {
		session.Set("oauth_state", "state-1234567890")
		session.Set(controlleraionui.DesktopSessionRedirectURI, serviceaionui.DesktopRedirectURI)
		session.Set(controlleraionui.DesktopSessionState, "state-1234567890")
	})
}

func setupDingTalkOAuthTestRouterWithSession(t *testing.T, fake *fakeDingTalkOAuthService, configureSession func(sessions.Session)) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := newDingTalkOAuthControllerDB(t)

	previousFactory := newDingTalkOAuthService
	newDingTalkOAuthService = func() dingTalkOAuthService {
		return fake
	}
	t.Cleanup(func() {
		newDingTalkOAuthService = previousFactory
	})
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.Create(&model.User{Id: 300, Username: "ding-login", DisplayName: "Ding Login", Email: "ding@example.com", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "dl01"}).Error)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("dingtalk-oauth-test"))))
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		configureSession(session)
		require.NoError(t, session.Save())
		c.Next()
	})
	router.GET("/api/oauth/dingtalk", HandleDingTalkOAuth)
	return router
}

func newDingTalkOAuthControllerDB(t *testing.T) *gorm.DB {
	t.Helper()
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
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
