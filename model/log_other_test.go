package model

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestLogOtherScopesAndMerges(t *testing.T) {
	var other LogOther

	assert.True(t, other.SetPublic("request_path", "/v1/chat/completions"))
	other.MergePublic(map[string]any{
		"zero": 0,
	})
	assert.True(t, other.SetAdmin("use_channel", []string{"channel-a"}))
	other.MergeAdmin(map[string]any{
		"rejected": false,
	})
	assert.True(t, other.SetRoot("upstream_request_id", "upstream-private"))
	other.MergeRoot(map[string]any{
		"generation": 0,
	})
	assert.True(t, other.SetAudit("method", "POST"))
	other.MergeAudit(map[string]any{
		"success": false,
	})

	require.JSONEq(t, `{
		"request_path": "/v1/chat/completions",
		"zero": 0,
		"admin_info": {
			"use_channel": ["channel-a"],
			"rejected": false
		},
		"root_info": {
			"upstream_request_id": "upstream-private",
			"generation": 0
		},
		"audit_info": {
			"method": "POST",
			"success": false
		}
	}`, other.JSONString())
}

func TestLogOtherRejectsSensitivePublicFields(t *testing.T) {
	other := NewLogOther()

	for _, key := range []string{
		"admin_info",
		"root_info",
		"audit_info",
		"channel_id",
		"channel_name",
		"channel_type",
		"reject_reason",
	} {
		assert.False(t, other.SetPublic(key, "must-not-leak"), key)
	}
	other.MergePublic(map[string]any{
		"request_path": "/v1/responses",
		"channel_name": "still-must-not-leak",
		"admin_info":   map[string]any{"secret": true},
	})

	require.JSONEq(t, `{"request_path":"/v1/responses"}`, other.JSONString())
	require.JSONEq(t, `{}`, NewLogOther().JSONString())
}

func TestLogOtherJSONStringDoesNotMutateReceiver(t *testing.T) {
	other := NewLogOther()
	require.True(t, other.SetPublic("request_path", "/v1/chat/completions"))
	require.True(t, other.SetAdmin("rejected", false))

	before := other.Snapshot()
	first := other.JSONString()
	after := other.Snapshot()
	second := other.JSONString()

	require.Equal(t, before, after)
	require.Equal(t, first, second)
}

func TestLogOtherClientApp(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		want   string
	}{
		{name: "codex", header: "hqbuddy-codex", want: "hqbuddy-codex"},
		{name: "trim spaces", header: " hqbuddy-opencode ", want: "hqbuddy-opencode"},
		{name: "other client", header: "WordBuddy/1.0", want: "WordBuddy/1.0"},
		{name: "missing"},
		{name: "non ASCII", header: "华青助手 · Codex"},
		{name: "control characters", header: "hqbuddy\r\ncodex"},
		{name: "oversized", header: strings.Repeat("a", 129)},
		{name: "maximum length", header: strings.Repeat("a", 128), want: strings.Repeat("a", 128)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := NewLogOther()
			other.SetPublic("request_path", "/v1/responses")
			other = other.withClientApp(tc.header)
			assert.Equal(t, "/v1/responses", other.Snapshot()["request_path"])
			if tc.want == "" {
				assert.NotContains(t, other.Snapshot(), "client_app")
				return
			}
			assert.Equal(t, tc.want, other.Snapshot()["client_app"])
		})
	}
}

func TestRecordLogsClientApp(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
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
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{NamingStrategy: schema.NamingStrategy{TablePrefix: "client_app_test_"}})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			previousDB, previousLogDB := DB, LOG_DB
			previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
			previousConsume, previousExport, previousRedis := common.LogConsumeEnabled, common.DataExportEnabled, common.RedisEnabled
			t.Cleanup(func() {
				assert.NoError(t, db.Migrator().DropTable(&Log{}, &User{}))
				assert.NoError(t, sqlDB.Close())
				DB, LOG_DB = previousDB, previousLogDB
				common.SetDatabaseTypes(previousMainType, previousLogType)
				common.LogConsumeEnabled, common.DataExportEnabled, common.RedisEnabled = previousConsume, previousExport, previousRedis
			})
			DB, LOG_DB = db, db
			common.SetDatabaseTypes(common.DatabaseType(dialect), common.DatabaseType(dialect))
			common.LogConsumeEnabled, common.DataExportEnabled, common.RedisEnabled = true, false, false
			require.NoError(t, db.AutoMigrate(&User{}, &Log{}))
			require.NoError(t, db.Create(&User{Id: 1, Username: "client_app_test", Setting: "{}"}).Error)
			var version string
			versionSQL := "SELECT version()"
			if dialect == "sqlite" {
				versionSQL = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(versionSQL).Scan(&version).Error)
			t.Logf("database version: %s", version)

			for _, clientApp := range []string{"hqbuddy-codex", "hqbuddy-opencode", ""} {
				for _, logType := range []int{LogTypeConsume, LogTypeError} {
					ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
					ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
					ctx.Request.Header.Set("X-Client-App", clientApp)
					ctx.Set("username", "client_app_test")
					var other *LogOther
					if clientApp == "hqbuddy-opencode" {
						other = NewLogOther()
						other.SetPublic("request_path", "/v1/responses")
						other.SetAdmin("use_channel", []string{"test-channel"})
					}
					if logType == LogTypeConsume {
						RecordConsumeLog(ctx, 1, RecordConsumeLogParams{ModelName: "test-model", Other: other})
					} else {
						RecordErrorLog(ctx, 1, 0, "test-model", "", "test failure", 0, 0, false, "", other)
					}
					var log Log
					require.NoError(t, db.Order("id desc").First(&log).Error)
					assert.Equal(t, logType, log.Type)
					assert.Equal(t, clientApp, log.ClientApp)
					logs := []*Log{&log}
					formatUserLogs(logs, 0)
					if clientApp == "" {
						assert.Empty(t, log.Other)
						continue
					}
					var metadata map[string]any
					require.NoError(t, common.UnmarshalJsonStr(log.Other, &metadata))
					assert.Equal(t, clientApp, metadata["client_app"])
					if clientApp == "hqbuddy-opencode" {
						assert.Equal(t, "/v1/responses", metadata["request_path"])
						assert.NotContains(t, metadata, "admin_info")
					}
				}
			}
		})
	}
}
