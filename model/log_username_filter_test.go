package model

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetAllLogsUsernameFilterMatchesCurrentUsernameAndHistoricalSnapshot(t *testing.T) {
	previousDB := DB
	previousLogDB := LOG_DB
	previousUsingSQLite := common.UsingSQLite
	previousUsingMySQL := common.UsingMySQL
	previousUsingPostgreSQL := common.UsingPostgreSQL
	t.Cleanup(func() {
		DB = previousDB
		LOG_DB = previousLogDB
		common.UsingSQLite = previousUsingSQLite
		common.UsingMySQL = previousUsingMySQL
		common.UsingPostgreSQL = previousUsingPostgreSQL
	})

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	DB = db
	LOG_DB = db
	require.NoError(t, db.AutoMigrate(&User{}, &Log{}))

	require.NoError(t, db.Create(&User{Id: 1, Username: "alice_ops", Password: "password123", AffCode: "alice-ops"}).Error)
	require.NoError(t, db.Create(&Log{
		Id:        1,
		UserId:    1,
		Username:  "alice",
		Type:      LogTypeConsume,
		ModelName: "gpt-4o-mini",
		CreatedAt: 1717117200,
	}).Error)

	logs, total, err := GetAllLogs(LogTypeConsume, 0, 0, "", "alice_ops", "", 0, 20, 0, "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	require.Equal(t, "alice", logs[0].Username)
}

func TestGetLogsClientAndInboundRequestIDFilters(t *testing.T) {
	previousDB := DB
	previousLogDB := LOG_DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	t.Cleanup(func() {
		DB = previousDB
		LOG_DB = previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	DB = db
	LOG_DB = db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&Log{}))

	require.NoError(t, db.Create(&Log{
		Id:               1,
		UserId:           1,
		Type:             LogTypeConsume,
		ModelName:        "gpt-5",
		Quota:            42,
		CreatedAt:        1717117200,
		RequestId:        "new-api-request-1",
		ClientRequestId:  "upstream-client-1",
		InboundRequestId: "codex-thread-1",
	}).Error)
	require.NoError(t, db.Create(&Log{
		Id:               2,
		UserId:           1,
		Type:             LogTypeConsume,
		ModelName:        "gpt-5",
		Quota:            100,
		CreatedAt:        1717117300,
		RequestId:        "new-api-request-2",
		ClientRequestId:  "upstream-client-2",
		InboundRequestId: "codex-thread-2",
	}).Error)

	logs, total, err := GetAllLogs(LogTypeConsume, 0, 0, "", "", "", 0, 20, 0, "", "", "", "upstream-client-1", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, "upstream-client-1", logs[0].ClientRequestId)

	logs, total, err = GetUserLogs(1, LogTypeConsume, 0, 0, "", "", 0, 20, "", "", "", "upstream-client-2", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, "upstream-client-2", logs[0].ClientRequestId)

	logs, total, err = GetAllLogs(LogTypeConsume, 0, 0, "", "", "", 0, 20, 0, "", "", "", "", "codex-thread-2")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, "codex-thread-2", logs[0].InboundRequestId)

	stat, err := SumUsedQuota(LogTypeConsume, 0, 0, "", "", "", 0, "", "", "", "upstream-client-1", "")
	require.NoError(t, err)
	assert.Equal(t, 42, stat.Quota)

	stat, err = SumUsedQuota(LogTypeConsume, 0, 0, "", "", "", 0, "", "", "", "", "codex-thread-2")
	require.NoError(t, err)
	assert.Equal(t, 100, stat.Quota)
}

func TestRecordConsumeLogStoresResponseAndInboundRequestIDsSeparately(t *testing.T) {
	previousDB := DB
	previousLogDB := LOG_DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousLogConsumeEnabled := common.LogConsumeEnabled
	previousDataExportEnabled := common.DataExportEnabled
	t.Cleanup(func() {
		DB = previousDB
		LOG_DB = previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		common.LogConsumeEnabled = previousLogConsumeEnabled
		common.DataExportEnabled = previousDataExportEnabled
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	DB = db
	LOG_DB = db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.LogConsumeEnabled = true
	common.DataExportEnabled = false
	require.NoError(t, db.AutoMigrate(&Log{}))

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("username", "alice")
	ctx.Set(common.RequestIdKey, "new-api-request")
	ctx.Set(common.ClientRequestIdKey, "sub2api-request")
	ctx.Set(common.InboundRequestIdKey, "codex-thread")
	ctx.Set(common.UpstreamRequestIdKey, "upstream-oneapi-request")

	RecordConsumeLog(ctx, 1, RecordConsumeLogParams{
		ModelName:      "gpt-5",
		TokenName:      "token",
		Quota:          42,
		Content:        "ok",
		UseTimeSeconds: 1,
		Other:          NewLogOther(),
	})

	var log Log
	require.NoError(t, db.Take(&log, "content = ?", "ok").Error)
	assert.Equal(t, "new-api-request", log.RequestId)
	assert.Equal(t, "sub2api-request", log.ClientRequestId)
	assert.Equal(t, "codex-thread", log.InboundRequestId)
	assert.Equal(t, "upstream-oneapi-request", log.UpstreamRequestId)
}
