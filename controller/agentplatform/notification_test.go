package agentplatform

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	controlleraionui "github.com/QuantumNous/new-api/controller/aionui"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/QuantumNous/new-api/service"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type notificationTestAPI struct {
	db         *gorm.DB
	logDB      *gorm.DB
	router     *gin.Engine
	rootToken  string
	adminToken string
	userToken  string
	desktop    dtoaionui.DesktopTokenResponse
	user       model.User
}

type notificationTestResponse struct {
	Success bool              `json:"success"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Data    common.RawMessage `json:"data"`
}

func newNotificationTestAPI(t *testing.T, dialect string) *notificationTestAPI {
	t.Helper()
	var driver gorm.Dialector
	switch dialect {
	case "sqlite":
		driver = sqlite.Open(":memory:")
	case "mysql":
		dsn := os.Getenv("TEST_MYSQL_DSN")
		if dsn == "" {
			t.Skip("TEST_MYSQL_DSN is not configured")
		}
		driver = mysql.Open(dsn)
	case "postgres":
		dsn := os.Getenv("TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("TEST_POSTGRES_DSN is not configured")
		}
		driver = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
	}
	db, err := gorm.Open(driver, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	if dialect == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	}
	var version string
	versionQuery := "SELECT version()"
	if dialect == "sqlite" {
		versionQuery = "SELECT sqlite_version()"
	}
	require.NoError(t, db.Raw(versionQuery).Scan(&version).Error)
	t.Logf("database version: %s", version)
	tables := []any{&apmodel.Notification{}, &model.User{}, &model.Token{}, &model.UserSession{}, &model.UserAccessToken{}, &model.AuthFlow{}, &model.Option{}}
	require.NoError(t, db.Migrator().DropTable(tables...))
	require.NoError(t, db.AutoMigrate(tables[1:]...))
	// Exercise the production migration, including the MySQL notification table charset.
	require.NoError(t, apmodel.Migrate(db))
	if dialect == "mysql" {
		var schemaCharset, notificationCollation string
		require.NoError(t, db.Raw("SELECT default_character_set_name FROM information_schema.schemata WHERE schema_name = DATABASE()").Scan(&schemaCharset).Error)
		require.NoError(t, db.Raw("SELECT table_collation FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", (apmodel.Notification{}).TableName()).Scan(&notificationCollation).Error)
		t.Logf("schema charset: %s; notification table collation: %s", schemaCharset, notificationCollation)
	}
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, logDB.AutoMigrate(&model.AuditLog{}))
	logSQL, err := logDB.DB()
	require.NoError(t, err)
	logSQL.SetMaxOpenConns(1)
	previousDB, previousLog := model.DB, model.LOG_DB
	previousMain, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousSecret, previousRedis := common.SessionSecret, common.RedisEnabled
	model.DB, model.LOG_DB = db, logDB
	common.SetDatabaseTypes(common.DatabaseType(dialect), common.DatabaseTypeSQLite)
	common.SessionSecret, common.RedisEnabled = "notification-test-session-secret", false
	require.NoError(t, i18n.Init())
	serviceaionui.ResetDesktopAuthServiceForTest()
	t.Cleanup(func() {
		assert.NoError(t, db.Migrator().DropTable(tables...))
		assert.NoError(t, sqlDB.Close())
		assert.NoError(t, logSQL.Close())
		model.DB, model.LOG_DB = previousDB, previousLog
		common.SetDatabaseTypes(previousMain, previousLogType)
		common.SessionSecret, common.RedisEnabled = previousSecret, previousRedis
		serviceaionui.ResetDesktopAuthServiceForTest()
	})
	require.NoError(t, model.EnsureLegacyAccessTokenRetireAt(time.Now().Unix()))
	api := &notificationTestAPI{db: db, logDB: logDB}
	users := []model.User{
		{Username: "notification-root", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "notify-root", Email: "root@example.com"},
		{Username: "notification-admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "notify-admin", Email: "admin@example.com"},
		{Username: "notification-user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "notify-user", Email: "user@example.com"},
	}
	tokens := make([]string, len(users))
	for i := range users {
		require.NoError(t, db.Create(&users[i]).Error)
		bundle, err := service.CreateLoginSession(users[i].Id, "password", "127.0.0.1", "notification-test")
		require.NoError(t, err)
		tokens[i] = bundle.AccessToken
	}
	api.rootToken, api.adminToken, api.userToken = tokens[0], tokens[1], tokens[2]
	api.user = users[2]
	code, err := serviceaionui.DefaultDesktopAuthService().IssueCode(&api.user, "http://127.0.0.1:49152/hth/callback", "notification-test-state")
	require.NoError(t, err)
	api.desktop, err = serviceaionui.DefaultDesktopAuthService().ExchangeCode(dtoaionui.DesktopTokenRequest{Code: code, DeviceId: "notification-test-device"})
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	api.router = gin.New()
	admin := api.router.Group("/api/agent-platform/notifications", middleware.RootAuth())
	admin.GET("", ListNotifications)
	admin.POST("", CreateNotification)
	admin.GET("/:id", GetNotification)
	admin.PUT("/:id", UpdateNotification)
	admin.PATCH("/:id/status", UpdateNotificationStatus)
	admin.DELETE("/:id", DeleteNotification)
	consumer := api.router.Group("/api/aionui/notifications", middleware.AionUiDesktopAuth())
	consumer.GET("", controlleraionui.ListClientNotifications)
	consumer.GET("/:id", controlleraionui.GetClientNotification)
	return api
}

func (api *notificationTestAPI) request(method, path, token, match, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if match != "" {
		req.Header.Set("If-Match", match)
	}
	response := httptest.NewRecorder()
	api.router.ServeHTTP(response, req)
	return response
}

func notificationData[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var envelope notificationTestResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &envelope))
	require.True(t, envelope.Success, response.Body.String())
	var data T
	require.NoError(t, common.Unmarshal(envelope.Data, &data))
	return data
}

func TestNotificationAPI(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			api := newNotificationTestAPI(t, dialect)
			const adminPath = "/api/agent-platform/notifications"
			const clientPath = "/api/aionui/notifications"
			created := notificationData[dtoagentplatform.NotificationDetail](t, api.request("POST", adminPath, api.rootToken, "", `{"title":"  Update  ","summary":"summary","content":"line 1\r\nline 2"}`))
			assert.Regexp(t, `^ntf_[0-9a-f]{32}$`, created.Id)
			assert.Equal(t, "Update", created.Title)
			assert.Equal(t, "line 1\nline 2", created.Content)
			assert.False(t, created.Enabled)
			assert.Zero(t, created.PublishedAt)
			assert.EqualValues(t, 1, created.Version)
			index := notificationData[dtoagentplatform.ClientNotificationList](t, api.request("GET", clientPath+"?enabled=false&user_id=1", api.desktop.AccessToken, "", ""))
			assert.Equal(t, fmt.Sprintf("user:%d", api.user.Id), index.ViewerScope)
			assert.Empty(t, index.Items)
			assert.NotNil(t, index.Items)
			assert.Equal(t, 404, api.request("GET", clientPath+"/"+created.Id, api.desktop.AccessToken, "", "").Code)
			path := adminPath + "/" + created.Id
			assert.Equal(t, 428, api.request("PATCH", path+"/status", api.rootToken, "", `{"enabled":true}`).Code)
			for _, match := range []string{`*`, `W/"1"`, `"1","2"`, `"-1"`, `"0"`, `"9223372036854775808"`} {
				assert.Equal(t, 400, api.request("PATCH", path+"/status", api.rootToken, match, `{"enabled":true}`).Code, match)
			}
			enabled := notificationData[dtoagentplatform.NotificationDetail](t, api.request("PATCH", path+"/status", api.rootToken, `"1"`, `{"enabled":true}`))
			assert.True(t, enabled.Enabled)
			assert.Positive(t, enabled.PublishedAt)
			assert.EqualValues(t, 2, enabled.Version)
			unchanged := notificationData[dtoagentplatform.NotificationDetail](t, api.request("PATCH", path+"/status", api.rootToken, `"2"`, `{"enabled":true}`))
			assert.Equal(t, enabled, unchanged, "a repeated status does not republish or create another version")
			for _, body := range []string{`{}`, `{"enabled":null}`, `{"enabled":"false"}`} {
				assert.Equal(t, 400, api.request("PATCH", path+"/status", api.rootToken, `"2"`, body).Code)
			}
			assert.Equal(t, 412, api.request("PATCH", path+"/status", api.rootToken, `"1"`, `{"enabled":false}`).Code)
			assert.Equal(t, 412, api.request("DELETE", path, api.rootToken, `"1"`, "").Code)
			clientDetail := api.request("GET", clientPath+"/"+created.Id, api.desktop.AccessToken, "", "")
			assert.Equal(t, "private, no-store", clientDetail.Header().Get("Cache-Control"))
			detail := notificationData[dtoagentplatform.ClientNotificationDetail](t, clientDetail)
			assert.Equal(t, created.Content, detail.Content)
			assert.NotContains(t, clientDetail.Body.String(), "created_by")
			assert.NotContains(t, clientDetail.Body.String(), `"enabled"`)
			closed := notificationData[dtoagentplatform.NotificationDetail](t, api.request("PUT", path, api.rootToken, `"2"`, `{"title":"Edited","summary":"","content":"new body","enabled":false}`))
			assert.False(t, closed.Enabled)
			assert.Empty(t, closed.Summary)
			assert.Equal(t, enabled.PublishedAt, closed.PublishedAt)
			assert.Equal(t, 404, api.request("GET", clientPath+"/"+created.Id, api.desktop.AccessToken, "", "").Code)
			reopened := notificationData[dtoagentplatform.NotificationDetail](t, api.request("PATCH", path+"/status", api.rootToken, `"3"`, `{"enabled":true}`))
			assert.Equal(t, enabled.PublishedAt, reopened.PublishedAt)
			assert.Equal(t, created.Id, reopened.Id)
			seed := make([]apmodel.Notification, 101)
			for i := range seed {
				seed[i] = apmodel.Notification{NotificationId: fmt.Sprintf("ntf_%032x", i+1), Title: "seed", Summary: "", Content: "private body", Enabled: true, Version: 1, PublishedAt: 1, CreatedAt: 1, UpdatedAt: 1, CreatedBy: 1, UpdatedBy: 1}
			}
			require.NoError(t, api.db.CreateInBatches(seed, 50).Error)
			indexResponse := api.request("GET", clientPath, api.desktop.AccessToken, "", "")
			index = notificationData[dtoagentplatform.ClientNotificationList](t, indexResponse)
			require.Len(t, index.Items, 102)
			assert.Equal(t, len(index.Items), index.Total)
			assert.Equal(t, created.Id, index.Items[0].Id)
			assert.Equal(t, seed[100].NotificationId, index.Items[1].Id)
			assert.NotContains(t, indexResponse.Body.String(), "private body")
			assert.NotContains(t, indexResponse.Body.String(), `"content"`)
			list := notificationData[dtoagentplatform.NotificationList](t, api.request("GET", adminPath+"?page=2&page_size=100", api.rootToken, "", ""))
			assert.EqualValues(t, 102, list.Total)
			assert.Len(t, list.Items, 2)
			for _, body := range []string{
				`{"title":"x","content":"body","enabled":null}`, `{"title":"x","content":"body","enabled":"false"}`,
				`{"title":"x","content":"body","created_by":99}`, `{"title":"x","content":"   "}`, `{"title":"line\nbreak","content":"body"}`,
				`{"title":"x","content":"nul\u0000body"}`, `{"title":"x","content":"` + strings.Repeat("x", 10001) + `"}`,
			} {
				assert.Equal(t, 400, api.request("POST", adminPath, api.rootToken, "", body).Code)
			}
			assert.Equal(t, 400, api.request("POST", adminPath, api.rootToken, "", string([]byte{'{', '"', 't', 'i', 't', 'l', 'e', '"', ':', '"', 0xff, '"', '}'})).Code)
			assert.Equal(t, 413, api.request("POST", adminPath, api.rootToken, "", strings.Repeat(" ", 64*1024+1)).Code)
			assert.Equal(t, 400, api.request("PUT", path, api.rootToken, `"4"`, `{"title":"missing"}`).Code)
			assert.Equal(t, 400, api.request("GET", adminPath+"?page=0", api.rootToken, "", "").Code)
			assert.Equal(t, 400, api.request("GET", adminPath+"?page_size=101", api.rootToken, "", "").Code)
			var wg sync.WaitGroup
			responses := make([]*httptest.ResponseRecorder, 2)
			for i := range responses {
				wg.Go(func() {
					responses[i] = api.request("PUT", path, api.rootToken, `"4"`, fmt.Sprintf(`{"title":"concurrent-%d","summary":"","content":"body","enabled":true}`, i))
				})
			}
			wg.Wait()
			assert.ElementsMatch(t, []int{200, 412}, []int{responses[0].Code, responses[1].Code})
			deleted := api.request("DELETE", path, api.rootToken, `"5"`, "")
			assert.Equal(t, 200, deleted.Code, deleted.Body.String())
			assert.Equal(t, 404, api.request("GET", clientPath+"/"+created.Id, api.desktop.AccessToken, "", "").Code)
			assert.Equal(t, 404, api.request("DELETE", path, api.rootToken, `"5"`, "").Code)
			var audits []model.AuditLog
			require.NoError(t, api.logDB.Find(&audits).Error)
			require.NotEmpty(t, audits)
			var actions []string
			for _, audit := range audits {
				actions = append(actions, audit.Action)
				encoded, err := common.Marshal(audit)
				require.NoError(t, err)
				assert.NotContains(t, string(encoded), "new body")
				assert.NotContains(t, string(encoded), "line 1")
				assert.NotContains(t, string(encoded), api.rootToken)
				assert.NotContains(t, string(encoded), api.desktop.AccessToken)
			}
			assert.Contains(t, actions, "agentplatform.notification.create")
			assert.Contains(t, actions, "agentplatform.notification.status")
			assert.Contains(t, actions, "agentplatform.notification.update")
			assert.Contains(t, actions, "agentplatform.notification.delete")

			t.Run("authorization", func(t *testing.T) {
				for _, token := range []string{"", api.userToken, api.adminToken, api.desktop.AccessToken} {
					response := api.request("GET", adminPath, token, "", "")
					assert.Contains(t, []int{401, 403}, response.Code)
				}
				for _, token := range []string{"", "invalid", api.rootToken, api.desktop.PersonalAPIKey.Key} {
					assert.Equal(t, 401, api.request("GET", clientPath, token, "", "").Code)
				}
				for _, tc := range []struct {
					name   string
					userID int
					scopes []string
					read   int
					write  int
				}{
					{"root-read", 1, []string{"option:read"}, 200, 403},
					{"root-write", 1, []string{"option:write"}, 403, 200},
					{"root-wrong", 1, []string{"agent_platform:write"}, 403, 403},
					{"admin-scoped", 2, []string{"option:read", "option:write"}, 403, 403},
				} {
					raw := model.AccessTokenPrefix + common.GetUUID()
					pat := &model.UserAccessToken{UserId: tc.userID, Name: tc.name, TokenHash: model.AccessTokenFingerprint(raw)}
					require.NoError(t, pat.SetScopes(tc.scopes))
					require.NoError(t, api.db.Create(pat).Error)
					assert.Equal(t, tc.read, api.request("GET", adminPath, raw, "", "").Code, tc.name)
					assert.Equal(t, tc.write, api.request("POST", adminPath, raw, "", `{"title":"scope","content":"body"}`).Code, tc.name)
				}
				expiredRaw := model.AccessTokenPrefix + common.GetUUID()
				expired := &model.UserAccessToken{UserId: 1, Name: "expired", TokenHash: model.AccessTokenFingerprint(expiredRaw), ExpiresAt: time.Now().Unix() - 1}
				require.NoError(t, expired.SetScopes([]string{"option:read"}))
				require.NoError(t, api.db.Create(expired).Error)
				assert.Equal(t, 401, api.request("GET", adminPath, expiredRaw, "", "").Code)
				require.NoError(t, api.db.Model(&model.User{}).Where("id = ?", api.user.Id).Update("status", common.UserStatusDisabled).Error)
				assert.Equal(t, 403, api.request("GET", clientPath, api.desktop.AccessToken, "", "").Code)
			})
		})
	}
}

func TestNotificationUnicodeBoundary(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			api := newNotificationTestAPI(t, dialect)
			body, err := common.Marshal(map[string]any{"title": "Unicode", "content": strings.Repeat("😀", 10000), "enabled": true})
			require.NoError(t, err)
			response := api.request("POST", "/api/agent-platform/notifications", api.rootToken, "", string(body))
			created := notificationData[dtoagentplatform.NotificationDetail](t, response)
			assert.Len(t, []rune(created.Content), 10000)
			assert.Len(t, []byte(created.Content), 40000)
			assert.Equal(t, `"`+strconv.FormatInt(created.Version, 10)+`"`, response.Header().Get("ETag"))
			var stored apmodel.Notification
			require.NoError(t, api.db.Where("notification_id = ?", created.Id).First(&stored).Error)
			assert.Equal(t, created.Content, stored.Content)
		})
	}
}

func TestClientNotificationErrorTranslations(t *testing.T) {
	api := newNotificationTestAPI(t, "sqlite")
	forceUnavailable := false
	const callback = "test:notification-query-error"
	require.NoError(t, api.db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if forceUnavailable && tx.Statement.Table == (apmodel.Notification{}).TableName() {
			tx.AddError(errors.New("forced notification database failure"))
		}
	}))
	t.Cleanup(func() { api.db.Callback().Query().Remove(callback) })
	for _, tc := range []struct {
		language    string
		notFound    string
		unavailable string
	}{
		{"en", "Notification not found", "Notifications are temporarily unavailable"},
		{"zh-CN", "消息不存在", "消息通知暂时不可用"},
		{"zh-TW", "訊息不存在", "訊息通知暫時無法使用"},
	} {
		for _, failure := range []struct {
			unavailable bool
			status      int
			code        string
			message     string
		}{
			{false, 404, "NOTIFICATION_NOT_FOUND", tc.notFound},
			{true, 500, "NOTIFICATION_UNAVAILABLE", tc.unavailable},
		} {
			forceUnavailable = failure.unavailable
			request := httptest.NewRequest("GET", "/api/aionui/notifications/missing", nil)
			request.Header.Set("Authorization", "Bearer "+api.desktop.AccessToken)
			request.Header.Set("Accept-Language", tc.language)
			response := httptest.NewRecorder()
			api.router.ServeHTTP(response, request)
			require.Equal(t, failure.status, response.Code, response.Body.String())
			var envelope notificationTestResponse
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &envelope))
			assert.Equal(t, failure.code, envelope.Code)
			assert.Equal(t, failure.message, envelope.Message)
			assert.False(t, envelope.Success)
		}
	}
}
