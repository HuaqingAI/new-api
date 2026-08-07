package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRetiredFrontendAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	_, hasAsyncCleanup := routes[http.MethodPost+" /api/system-task/log-cleanup"]
	_, hasDirectDelete := routes[http.MethodDelete+" /api/log/"]
	_, hasConsoleMigration := routes[http.MethodPost+" /api/option/migrate_console_setting"]
	assert.True(t, hasAsyncCleanup)
	assert.False(t, hasDirectDelete)
	assert.False(t, hasConsoleMigration)
}

func TestLoginRoutesBypassGlobalAPIRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousEnabled := common.GlobalApiRateLimitEnable
	previousLimit := common.GlobalApiRateLimitNum
	previousDuration := common.GlobalApiRateLimitDuration
	previousRedisEnabled := common.RedisEnabled
	common.GlobalApiRateLimitEnable = true
	common.GlobalApiRateLimitNum = 1
	common.GlobalApiRateLimitDuration = 60
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.GlobalApiRateLimitEnable = previousEnabled
		common.GlobalApiRateLimitNum = previousLimit
		common.GlobalApiRateLimitDuration = previousDuration
		common.RedisEnabled = previousRedisEnabled
	})

	engine := gin.New()
	SetApiRouter(engine)

	assert.Equal(t, http.StatusOK, performRouterRequest(engine, http.MethodGet, "/api/status", "", "192.0.2.71:12345").Code)
	assert.Equal(t, http.StatusTooManyRequests, performRouterRequest(engine, http.MethodGet, "/api/status", "", "192.0.2.71:12345").Code)

	for range 2 {
		recorder := performRouterRequest(engine, http.MethodPost, "/api/user/auth/refresh", "", "192.0.2.72:12345")
		assert.NotEqual(t, http.StatusTooManyRequests, recorder.Code)
	}
	for range 2 {
		recorder := performRouterRequest(engine, http.MethodPost, "/api/aionui/desktop/token", `{"code":"invalid","device_id":"desktop-test"}`, "192.0.2.73:12345")
		assert.NotEqual(t, http.StatusTooManyRequests, recorder.Code)
	}
}

func performRouterRequest(router http.Handler, method string, path string, body string, remoteAddr string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.RemoteAddr = remoteAddr
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(recorder, request)
	return recorder
}
