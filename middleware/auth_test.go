package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAdminAuthRefreshesStaleSessionRoleFromDatabase(t *testing.T) {
	db := setupAuthTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       91,
		Username: "member-promoted",
		Password: "password123",
		AffCode:  "member-promoted-aff",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("auth-test"))))
	router.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("username", "member-promoted")
		session.Set("role", common.RoleCommonUser)
		session.Set("id", 91)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/admin", AdminAuth(), func(c *gin.Context) {
		session := sessions.Default(c)
		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"role":         c.GetInt("role"),
			"session_role": session.Get("role"),
		})
	})

	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/login", nil))
	require.Equal(t, http.StatusNoContent, loginRecorder.Code)
	cookies := loginRecorder.Result().Cookies()

	require.NoError(t, db.Model(&model.User{}).Where("id = ?", 91).Update("role", common.RoleAdminUser).Error)

	request := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
	request.Header.Set("New-Api-User", "91")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Contains(t, recorder.Body.String(), fmt.Sprintf(`"role":%d`, common.RoleAdminUser))
	require.Contains(t, recorder.Body.String(), fmt.Sprintf(`"session_role":%d`, common.RoleAdminUser))
	require.NotEmpty(t, recorder.Result().Cookies())
}

func TestUserAuthBootstrapsSelfRequestFromSession(t *testing.T) {
	router := setupUserAuthBootstrapRouter(t)
	cookies := loginAuthTestUser(t, router)

	request := httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Contains(t, recorder.Body.String(), `"id":92`)
}

func TestUserAuthDoesNotBootstrapOtherRequestsFromSession(t *testing.T) {
	router := setupUserAuthBootstrapRouter(t)
	cookies := loginAuthTestUser(t, router)

	request := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
}

func setupUserAuthBootstrapRouter(t *testing.T) *gin.Engine {
	t.Helper()
	db := setupAuthTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       92,
		Username: "desktop-login",
		Password: "password123",
		AffCode:  "desktop-login-aff",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("auth-bootstrap-test"))))
	router.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("username", "desktop-login")
		session.Set("role", common.RoleCommonUser)
		session.Set("id", 92)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/user/self", UserAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"id":      c.GetInt("id"),
		})
	})
	router.GET("/api/protected", UserAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	return router
}

func loginAuthTestUser(t *testing.T, router *gin.Engine) []*http.Cookie {
	t.Helper()
	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/login", nil))
	require.Equal(t, http.StatusNoContent, loginRecorder.Code)
	return loginRecorder.Result().Cookies()
}

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))

	t.Cleanup(func() {
		if model.DB == db {
			model.DB = nil
		}
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}
