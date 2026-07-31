package aionui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDesktopLoginRedirectsToCallbackWhenNewApiSessionExists(t *testing.T) {
	serviceaionui.ResetDesktopAuthServiceForTest()
	router := setupDesktopLoginSessionTestRouter(t)

	target := "/api/aionui/desktop/login?redirect_uri=" + url.QueryEscape("http://127.0.0.1:49152/hth/callback") + "&state=state-1234567890"
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:3001"+target, nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusFound, recorder.Code)
	location := recorder.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "http", parsed.Scheme)
	require.Equal(t, "127.0.0.1:49152", parsed.Host)
	require.Equal(t, "/hth/callback", parsed.Path)
	require.Equal(t, "state-1234567890", parsed.Query().Get("state"))
	code := parsed.Query().Get("code")
	require.NotEmpty(t, code)

	token, err := serviceaionui.DefaultDesktopAuthService().ExchangeCode(dtoaionui.DesktopTokenRequest{
		Code:     code,
		DeviceId: "device-1",
	})
	require.NoError(t, err)
	require.Equal(t, "session@example.com", token.User.Email)
}

func setupDesktopLoginSessionTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	model.DB = newDesktopLoginControllerDB(t)
	model.LOG_DB = model.DB
	require.NoError(t, model.DB.Create(&model.User{
		Id:          410,
		Username:    "session-user",
		DisplayName: "Session User",
		Email:       "session@example.com",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     "su01",
	}).Error)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("desktop-login-session-test"))))
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 410)
		session.Set("username", "session-user")
		require.NoError(t, session.Save())
		c.Next()
	})
	router.GET("/api/aionui/desktop/login", DesktopLogin)
	return router
}

func newDesktopLoginControllerDB(t *testing.T) *gorm.DB {
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
