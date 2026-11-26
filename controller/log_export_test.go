package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// callLogHandler runs a log handler as the given user and returns the JSON
// lines of the response.
func callLogHandler(t *testing.T, handler gin.HandlerFunc, userId int, role int, query string, username ...string) []map[string]any {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/log?"+query, nil)
	ctx.Set("id", userId)
	ctx.Set("role", role)
	if len(username) > 0 {
		ctx.Set("username", username[0])
	}
	handler(ctx)
	var lines []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(recorder.Body.String()), "\n") {
		var value map[string]any
		require.NoError(t, common.UnmarshalJsonStr(line, &value))
		lines = append(lines, value)
	}
	return lines
}

func exportedRequestIds(lines []map[string]any) []any {
	ids := make([]any, 0, len(lines))
	for _, line := range lines[:len(lines)-1] {
		ids = append(ids, line["request_id"])
	}
	return ids
}

// seedExportLogs stores logs of two users: three in the last hour and one
// from 400 days ago.
func seedExportLogs(t *testing.T, mainDB *gorm.DB, logDB *gorm.DB, now int64) {
	t.Helper()
	require.NoError(t, mainDB.Create(&model.Channel{Id: 9, Name: "primary"}).Error)
	logs := []model.Log{
		{UserId: 1, Username: "alice", CreatedAt: now - 3600, Type: model.LogTypeConsume, ModelName: "gpt-a", ChannelId: 9, Quota: 10, RequestId: "a-1", ClientRequestId: "client-1", InboundRequestId: "inbound-1", Other: `{"admin_info":{"use_channel":["9"]}}`},
		{UserId: 2, Username: "bob", CreatedAt: now - 1800, Type: model.LogTypeConsume, ModelName: "gpt-b", ChannelId: 9, Quota: 20, RequestId: "b-1"},
		{UserId: 1, Username: "alice", CreatedAt: now - 60, Type: model.LogTypeError, ModelName: "gpt-a", RequestId: "a-2", ClientRequestId: "client-2", InboundRequestId: "inbound-2"},
		{UserId: 1, Username: "alice", CreatedAt: now - 400*24*3600, Type: model.LogTypeConsume, ModelName: "gpt-a", RequestId: "a-old"},
	}
	for i := range logs {
		require.NoError(t, logDB.Create(&logs[i]).Error)
	}
}

func useLogExportDatabases(t *testing.T, mainDB *gorm.DB, logDB *gorm.DB, mainType common.DatabaseType, logType common.DatabaseType) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	previousRedis, previousMemory := common.RedisEnabled, common.MemoryCacheEnabled
	model.DB, model.LOG_DB = mainDB, logDB
	common.SetDatabaseTypes(mainType, logType)
	common.RedisEnabled, common.MemoryCacheEnabled = false, false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMain, previousLog)
		common.RedisEnabled, common.MemoryCacheEnabled = previousRedis, previousMemory
	})
}

func assertLogClientAppIndex(t *testing.T, db *gorm.DB, clickHouse bool) {
	t.Helper()
	assert.True(t, db.Migrator().HasColumn(&model.Log{}, "ClientApp"))
	if clickHouse {
		var index struct {
			Expression string `gorm:"column:expr"`
			Type       string `gorm:"column:type"`
		}
		require.NoError(t, db.Raw("SELECT expr, type FROM system.data_skipping_indices WHERE database = currentDatabase() AND table = 'logs' AND name = 'idx_logs_client_app_created_at'").Scan(&index).Error)
		assert.Equal(t, "client_app", index.Expression)
		assert.Equal(t, "bloom_filter", index.Type)
		return
	}
	indexes, err := db.Migrator().GetIndexes(&model.Log{})
	require.NoError(t, err)
	var sourceColumns []string
	for _, index := range indexes {
		if index.Name() == "idx_logs_client_app_created_at" {
			sourceColumns = index.Columns()
		}
	}
	assert.Equal(t, []string{"client_app", "created_at"}, sourceColumns)
	assert.True(t, db.Migrator().HasIndex(&model.Log{}, "idx_created_at_id"))
	assert.True(t, db.Migrator().HasIndex(&model.Log{}, "idx_logs_request_id"))
}

func TestLogExportDatabaseMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousMaster := common.IsMasterNode
	common.IsMasterNode = true
	t.Cleanup(func() { common.IsMasterNode = previousMaster })
	for _, tc := range []struct {
		kind, env string
		typ       common.DatabaseType
	}{
		{"sqlite", "", common.DatabaseTypeSQLite},
		{"mysql", "AUDIT_MYSQL_DSN", common.DatabaseTypeMySQL},
		{"postgres", "AUDIT_POSTGRES_DSN", common.DatabaseTypePostgreSQL},
		{"clickhouse", "AUDIT_CLICKHOUSE_DSN", common.DatabaseTypeClickHouse},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			dsn := os.Getenv(tc.env)
			if tc.env != "" && dsn == "" {
				t.Skip(tc.env + " is not configured")
			}
			db, isolatedDSN := newAuditTestDatabase(t, tc.kind, dsn)
			mainDB, mainType := db, tc.typ
			if tc.kind == "clickhouse" {
				mainDB, _ = newAuditTestDatabase(t, "sqlite", "")
				mainType = common.DatabaseTypeSQLite
			}
			useLogExportDatabases(t, mainDB, db, mainType, tc.typ)
			require.NoError(t, mainDB.AutoMigrate(&model.Channel{}, &model.User{}))
			if tc.kind == "clickhouse" {
				t.Setenv("LOG_SQL_DSN", isolatedDSN)
				require.NoError(t, model.InitLogDB())
			} else {
				require.NoError(t, db.AutoMigrate(&model.Log{}))
			}
			versionSQL := "SELECT version()"
			if tc.kind == "sqlite" {
				versionSQL = "SELECT sqlite_version()"
			}
			var version string
			require.NoError(t, model.LOG_DB.Raw(versionSQL).Scan(&version).Error)
			t.Logf("log database: %s %s", tc.kind, version)
			now := time.Now().Unix()
			seedExportLogs(t, mainDB, model.LOG_DB, now)
			query := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-2*24*3600, now+3600)

			listed := callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, query)[0]
			require.Equal(t, true, listed["success"])
			assert.EqualValues(t, 2, listed["data"].(map[string]any)["total"])
			listed = callLogHandler(t, GetAllLogs, 3, common.RoleAdminUser, query+"&username=alice")[0]
			require.Equal(t, true, listed["success"])
			assert.EqualValues(t, 2, listed["data"].(map[string]any)["total"])

			own := callLogHandler(t, ExportUserLogs, 1, common.RoleCommonUser, query)
			require.Len(t, own, 3)
			assert.Equal(t, []any{"a-2", "a-1"}, exportedRequestIds(own))
			assert.Equal(t, map[string]any{"success": true, "message": "", "data": map[string]any{"count": float64(2)}}, own[2])
			assert.Equal(t, float64(1), own[0]["id"], "user exports carry list positions, not row ids")
			assert.Empty(t, own[1]["channel_name"])
			assert.NotContains(t, own[1]["other"], "admin_info")

			all := callLogHandler(t, ExportAllLogs, 3, common.RoleAdminUser, query+"&model_name=gpt-%25")
			require.Len(t, all, 4)
			assert.Equal(t, []any{"a-2", "b-1", "a-1"}, exportedRequestIds(all))
			assert.Equal(t, "primary", all[2]["channel_name"])
			assert.Contains(t, all[2]["other"], "admin_info")
			assert.Equal(t, true, all[3]["success"])

			for _, filter := range []struct {
				query     string
				requestID string
			}{
				{"&client_request_id=client-1", "a-1"},
				{"&inbound_request_id=inbound-2", "a-2"},
			} {
				for _, handler := range []gin.HandlerFunc{GetAllLogs, GetUserLogs} {
					listed := callLogHandler(t, handler, 1, common.RoleAdminUser, query+filter.query)[0]
					require.Equal(t, true, listed["success"])
					data := listed["data"].(map[string]any)
					assert.EqualValues(t, 1, data["total"])
					items := data["items"].([]any)
					require.Len(t, items, 1)
					assert.Equal(t, filter.requestID, items[0].(map[string]any)["request_id"])
				}
				for _, handler := range []gin.HandlerFunc{ExportAllLogs, ExportUserLogs} {
					exported := callLogHandler(t, handler, 1, common.RoleAdminUser, query+filter.query)
					require.Len(t, exported, 2)
					assert.Equal(t, []any{filter.requestID}, exportedRequestIds(exported))
				}
			}
		})
	}
}

func TestLogClientAppFilterDatabaseMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousMaster := common.IsMasterNode
	common.IsMasterNode = true
	t.Cleanup(func() { common.IsMasterNode = previousMaster })
	for _, tc := range []struct {
		kind, env string
		typ       common.DatabaseType
	}{
		{"sqlite", "", common.DatabaseTypeSQLite},
		{"mysql", "AUDIT_MYSQL_DSN", common.DatabaseTypeMySQL},
		{"postgres", "AUDIT_POSTGRES_DSN", common.DatabaseTypePostgreSQL},
		{"clickhouse", "AUDIT_CLICKHOUSE_DSN", common.DatabaseTypeClickHouse},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			dsn := os.Getenv(tc.env)
			if tc.env != "" && dsn == "" {
				t.Skip(tc.env + " is not configured")
			}
			db, isolatedDSN := newAuditTestDatabase(t, tc.kind, dsn)
			mainDB, mainType := db, tc.typ
			if tc.kind == "clickhouse" {
				mainDB, _ = newAuditTestDatabase(t, "sqlite", "")
				mainType = common.DatabaseTypeSQLite
			}
			useLogExportDatabases(t, mainDB, db, mainType, tc.typ)
			require.NoError(t, mainDB.AutoMigrate(&model.User{}))
			for range 2 {
				if tc.kind == "clickhouse" {
					t.Setenv("LOG_SQL_DSN", isolatedDSN)
					require.NoError(t, model.InitLogDB())
				} else {
					require.NoError(t, model.MigrateLogClientApp())
					require.NoError(t, model.LOG_DB.AutoMigrate(&model.Log{}))
				}
			}
			assertLogClientAppIndex(t, model.LOG_DB, tc.kind == "clickhouse")
			require.NoError(t, mainDB.Create(&[]model.User{
				{Id: 1, Username: "alice", AffCode: "alice", Setting: "{}"},
				{Id: 2, Username: "bob", AffCode: "bob", Setting: "{}"},
			}).Error)
			var version string
			versionSQL := "SELECT version()"
			if tc.kind == "sqlite" {
				versionSQL = "SELECT sqlite_version()"
			}
			require.NoError(t, model.LOG_DB.Raw(versionSQL).Scan(&version).Error)
			t.Logf("log database: %s %s", tc.kind, version)
			now := time.Now().Unix()
			logs := []model.Log{
				{UserId: 1, Username: "alice", CreatedAt: now - 10, Type: model.LogTypeConsume, Quota: 10, PromptTokens: 7, CompletionTokens: 3, RequestId: "a-codex", ClientApp: "hqbuddy-codex", Other: `{"client_app":"hqbuddy-codex"}`},
				{UserId: 1, Username: "alice", CreatedAt: now - 120, Type: model.LogTypeConsume, Quota: 20, RequestId: "a-codex-old", ClientApp: "hqbuddy-codex", Other: `{"client_app" : "hqbuddy-codex"}`},
				{UserId: 1, Username: "alice", CreatedAt: now - 8, Type: model.LogTypeConsume, Quota: 30, PromptTokens: 9, CompletionTokens: 3, RequestId: "a-opencode", ClientApp: "hqbuddy-opencode", Other: `{"client_app":"hqbuddy-opencode"}`},
				{UserId: 1, Username: "alice", CreatedAt: now - 1, Type: model.LogTypeError, RequestId: "a-error", ClientApp: "hqbuddy-codex", Other: `{"client_app":"hqbuddy-codex"}`},
				{UserId: 2, Username: "bob", CreatedAt: now - 5, Type: model.LogTypeConsume, Quota: 40, PromptTokens: 10, CompletionTokens: 5, RequestId: "b-codex", ClientApp: "hqbuddy-codex", Other: `{"client_app":"hqbuddy-codex"}`},
				{UserId: 1, Username: "alice", CreatedAt: now - 2, Type: model.LogTypeError, RequestId: "indexed-priority", ClientApp: "indexed-priority", Other: `{"client_app":"legacy-source"}`},
			}
			for i, other := range []string{"", `{"admin_info":{"keep":true}}`} {
				logs = append(logs, model.Log{UserId: 1, Username: "alice", CreatedAt: now - 3, Type: model.LogTypeConsume, RequestId: fmt.Sprintf("unattributed-%d", i), Quota: 100, Other: other})
			}
			for i := range logs {
				require.NoError(t, model.LOG_DB.Create(&logs[i]).Error)
			}
			query := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-300, now+1)
			for _, filter := range []struct {
				source           string
				own, all         int
				ownStat, allStat model.Stat
			}{
				{"hqbuddy-codex", 2, 3, model.Stat{Quota: 30, Rpm: 1, Tpm: 10}, model.Stat{Quota: 70, Rpm: 2, Tpm: 25}},
				{"hqbuddy-opencode", 1, 1, model.Stat{Quota: 30, Rpm: 1, Tpm: 12}, model.Stat{Quota: 30, Rpm: 1, Tpm: 12}},
				{"hqbuddy-%", 3, 4, model.Stat{Quota: 60, Rpm: 2, Tpm: 22}, model.Stat{Quota: 100, Rpm: 3, Tpm: 37}},
				{"HQBUDDY-%", 3, 4, model.Stat{Quota: 60, Rpm: 2, Tpm: 22}, model.Stat{Quota: 100, Rpm: 3, Tpm: 37}},
				{"hqbuddy_codex", 0, 0, model.Stat{}, model.Stat{}},
				{"HQBUDDY-CODEX", 0, 0, model.Stat{}, model.Stat{}},
			} {
				t.Run(filter.source, func(t *testing.T) {
					filtered := query + "&type=2&client_app=" + url.QueryEscape(filter.source)
					for _, scope := range []struct {
						list, export, stat gin.HandlerFunc
						count              int
						want               model.Stat
					}{
						{GetUserLogs, ExportUserLogs, GetLogsSelfStat, filter.own, filter.ownStat},
						{GetAllLogs, ExportAllLogs, GetLogsStat, filter.all, filter.allStat},
					} {
						listed := callLogHandler(t, scope.list, 1, common.RoleAdminUser, filtered+"&p=1&page_size=1", "alice")[0]
						require.Equal(t, true, listed["success"])
						data := listed["data"].(map[string]any)
						assert.EqualValues(t, scope.count, data["total"])
						if scope.count > 0 {
							assert.Len(t, data["items"], 1)
						}
						exported := callLogHandler(t, scope.export, 1, common.RoleAdminUser, filtered, "alice")
						require.Len(t, exported, scope.count+1)
						assert.EqualValues(t, scope.count, exported[len(exported)-1]["data"].(map[string]any)["count"])
						stat := callLogHandler(t, scope.stat, 1, common.RoleAdminUser, filtered, "alice")[0]
						require.Equal(t, true, stat["success"])
						assert.Equal(t, map[string]any{"quota": float64(scope.want.Quota), "rpm": float64(scope.want.Rpm), "tpm": float64(scope.want.Tpm)}, stat["data"])
					}
				})
			}
			for _, handler := range []gin.HandlerFunc{GetAllLogs, GetUserLogs} {
				listed := callLogHandler(t, handler, 1, common.RoleAdminUser, query+"&type=5&client_app=hqbuddy-codex")[0]
				require.Equal(t, true, listed["success"])
				assert.EqualValues(t, 1, listed["data"].(map[string]any)["total"])
			}
			for _, source := range []struct {
				name  string
				count int
			}{{"indexed-priority", 1}, {"legacy-source", 0}} {
				listed := callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, query+"&type=5&client_app="+source.name)[0]
				require.Equal(t, true, listed["success"])
				assert.EqualValues(t, source.count, listed["data"].(map[string]any)["total"])
			}

			t.Run("upgrades released logs without assigning historical sources", func(t *testing.T) {
				// v0.3.1-20261109100448 matches the pre-change Log schema. Removing
				// only this new column/index reproduces its released logs table.
				var previousIndexes map[string][]string
				if tc.kind == "clickhouse" {
					require.NoError(t, model.LOG_DB.Exec("ALTER TABLE logs DROP INDEX IF EXISTS idx_logs_client_app_created_at").Error)
					require.NoError(t, model.LOG_DB.Exec("ALTER TABLE logs DROP COLUMN client_app").Error)
				} else {
					indexes, err := model.LOG_DB.Migrator().GetIndexes(&model.Log{})
					require.NoError(t, err)
					previousIndexes = make(map[string][]string)
					for _, index := range indexes {
						if index.Name() != "idx_logs_client_app_created_at" {
							previousIndexes[index.Name()] = index.Columns()
						}
					}
					require.NoError(t, model.LOG_DB.Migrator().DropIndex(&model.Log{}, "idx_logs_client_app_created_at"))
					require.NoError(t, model.LOG_DB.Migrator().DropColumn(&model.Log{}, "ClientApp"))
				}
				require.False(t, model.LOG_DB.Migrator().HasColumn(&model.Log{}, "ClientApp"))
				var before []model.Log
				// Use a distinct query before the simulated restart so PostgreSQL's
				// connection cache does not reuse the old SELECT * result schema.
				require.NoError(t, model.LOG_DB.Select("logs.*").Order("request_id").Find(&before).Error)
				for run := range 2 {
					require.NoError(t, model.MigrateLogClientApp())
					if tc.kind == "clickhouse" {
						require.NoError(t, model.InitLogDB())
					} else {
						require.NoError(t, model.LOG_DB.AutoMigrate(&model.Log{}))
					}
					var after []model.Log
					require.NoError(t, model.LOG_DB.Order("request_id").Find(&after).Error)
					require.Len(t, after, len(before))
					for i := range after {
						assert.Empty(t, after[i].ClientApp, "run %d leaves historical request %s unattributed", run+1, after[i].RequestId)
					}
					assert.Equal(t, before, after, "run %d preserves every existing log field", run+1)
					assertLogClientAppIndex(t, model.LOG_DB, tc.kind == "clickhouse")
					if tc.kind != "clickhouse" {
						indexes, err := model.LOG_DB.Migrator().GetIndexes(&model.Log{})
						require.NoError(t, err)
						currentIndexes := make(map[string][]string)
						for _, index := range indexes {
							if index.Name() != "idx_logs_client_app_created_at" {
								currentIndexes[index.Name()] = index.Columns()
							}
						}
						assert.Equal(t, previousIndexes, currentIndexes, "run %d preserves existing indexes", run+1)
					}
				}

				previousConsume, previousExport := common.LogConsumeEnabled, common.DataExportEnabled
				common.LogConsumeEnabled, common.DataExportEnabled = true, false
				t.Cleanup(func() {
					common.LogConsumeEnabled, common.DataExportEnabled = previousConsume, previousExport
				})
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
				ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
				ctx.Request.Header.Set("X-Client-App", "hqbuddy-codex")
				ctx.Set("username", "alice")
				ctx.Set(common.RequestIdKey, "post-upgrade")
				model.RecordConsumeLog(ctx, 1, model.RecordConsumeLogParams{ModelName: "upgrade-check", Quota: 23})
				listed := callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, "type=2&client_app=hqbuddy-codex")[0]
				require.Equal(t, true, listed["success"])
				data := listed["data"].(map[string]any)
				assert.EqualValues(t, 1, data["total"])
				items := data["items"].([]any)
				require.Len(t, items, 1)
				assert.Equal(t, "post-upgrade", items[0].(map[string]any)["request_id"])
				assert.Equal(t, "hqbuddy-codex", items[0].(map[string]any)["client_app"])
			})
		})
	}
}

func TestLogExportRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := newAuditTestDatabase(t, "sqlite", "")
	useLogExportDatabases(t, db, db, common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Log{}, &model.QuotaData{}))
	now := time.Now().Unix()
	seedExportLogs(t, db, db, now)
	logSetting := operation_setting.GetLogSetting()
	previousSetting := *logSetting
	t.Cleanup(func() { *logSetting = previousSetting })
	recent := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-2*24*3600, now)
	tooLong := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-366*24*3600, now)

	t.Run("time range", func(t *testing.T) {
		rejected := callLogHandler(t, ExportAllLogs, 3, common.RoleAdminUser, tooLong)
		require.Len(t, rejected, 1)
		assert.Equal(t, false, rejected[0]["success"])
		assert.Equal(t, "The time range cannot exceed {{days}} days", rejected[0]["message_key"])
		assert.Equal(t, map[string]any{"days": float64(365)}, rejected[0]["message_params"])

		assert.Equal(t, false, callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, tooLong)[0]["success"])
		assert.Equal(t, false, callLogHandler(t, GetLogsSelfStat, 1, common.RoleCommonUser, tooLong)[0]["success"])
		assert.Equal(t, true, callLogHandler(t, GetUserLogs, 3, common.RoleAdminUser, tooLong)[0]["success"], "administrators search their own logs without a limit")

		// The dashboard follows the same range instead of its old one-month limit.
		fortyDays := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-40*24*3600, now)
		for _, handler := range []gin.HandlerFunc{GetUserQuotaDates, GetUserFlowQuotaDates} {
			assert.Equal(t, "The time range cannot exceed {{days}} days", callLogHandler(t, handler, 1, common.RoleCommonUser, tooLong)[0]["message_key"])
			assert.Equal(t, true, callLogHandler(t, handler, 1, common.RoleCommonUser, fortyDays)[0]["success"])
		}

		// Without a start, a user's search reaches back the whole range.
		listed := callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, "")[0]
		assert.Equal(t, true, listed["success"])
		assert.EqualValues(t, 2, listed["data"].(map[string]any)["total"])

		logSetting.MaxRangeDays = 400
		assert.Equal(t, true, callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, tooLong)[0]["success"])
		logSetting.MaxRangeDays = previousSetting.MaxRangeDays
	})

	t.Run("user export switch", func(t *testing.T) {
		logSetting.UserExportEnabled = false
		t.Cleanup(func() { logSetting.UserExportEnabled = previousSetting.UserExportEnabled })
		disabled := callLogHandler(t, ExportUserLogs, 1, common.RoleCommonUser, recent)
		require.Len(t, disabled, 1)
		assert.Equal(t, "Log export is disabled", disabled[0]["message_key"])
		assert.Equal(t, true, callLogHandler(t, ExportUserLogs, 3, common.RoleAdminUser, recent)[0]["success"], "administrators export their own logs even when users cannot")
	})

	t.Run("one export per user", func(t *testing.T) {
		release, ok, err := service.AcquireLogExportLock(1)
		require.NoError(t, err)
		require.True(t, ok)
		busy := callLogHandler(t, ExportUserLogs, 1, common.RoleCommonUser, recent)
		require.Len(t, busy, 1)
		assert.Equal(t, "Another log export is running. Try again after it finishes.", busy[0]["message_key"])
		other := callLogHandler(t, ExportUserLogs, 2, common.RoleCommonUser, recent)
		assert.Equal(t, true, other[len(other)-1]["success"], "other users are not blocked")
		release()
		assert.Len(t, callLogHandler(t, ExportUserLogs, 1, common.RoleCommonUser, recent), 3)
	})
}

func TestAcquireLogExportLock(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%v", useRedis), func(t *testing.T) {
			previousRedis, previousClient := common.RedisEnabled, common.RDB
			common.RedisEnabled = useRedis
			if useRedis {
				server := miniredis.RunT(t)
				common.RDB = redis.NewClient(&redis.Options{Addr: server.Addr()})
				require.NoError(t, common.RDB.Ping(context.Background()).Err())
			}
			t.Cleanup(func() { common.RedisEnabled, common.RDB = previousRedis, previousClient })

			release, ok, err := service.AcquireLogExportLock(1)
			require.NoError(t, err)
			require.True(t, ok)
			_, ok, err = service.AcquireLogExportLock(1)
			require.NoError(t, err)
			assert.False(t, ok, "a second export of the same user must wait")
			releaseOther, ok, err := service.AcquireLogExportLock(2)
			require.NoError(t, err)
			assert.True(t, ok)
			releaseOther()
			release()
			releaseAgain, ok, err := service.AcquireLogExportLock(1)
			require.NoError(t, err)
			assert.True(t, ok, "a finished export frees the user")
			releaseAgain()
		})
	}
}
