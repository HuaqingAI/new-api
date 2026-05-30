package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRelaySensitiveWordsRecordsAlertEventWithoutLeakingWordsOrChangingResponse(t *testing.T) {
	oldDB := model.DB
	oldSensitiveWords := append([]string(nil), setting.SensitiveWords...)
	oldCheckEnabled := setting.CheckSensitiveEnabled
	oldPromptCheckEnabled := setting.CheckSensitiveOnPromptEnabled
	oldWriter := gin.DefaultWriter
	oldErrorWriter := gin.DefaultErrorWriter
	defer func() {
		model.DB = oldDB
		setting.SensitiveWords = oldSensitiveWords
		setting.CheckSensitiveEnabled = oldCheckEnabled
		setting.CheckSensitiveOnPromptEnabled = oldPromptCheckEnabled
		gin.DefaultWriter = oldWriter
		gin.DefaultErrorWriter = oldErrorWriter
	}()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, entmodel.AutoMigrate(db))
	model.DB = db

	require.NoError(t, db.Create(&model.User{Id: 1001, Username: "alice", Password: "password123", Group: "default", AffCode: "alice-aff"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 11, TenantId: 0, Name: "Engineering", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         1001,
		DepartmentId:   11,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
	}).Error)

	setting.SensitiveWords = []string{"secret"}
	setting.CheckSensitiveEnabled = true
	setting.CheckSensitiveOnPromptEnabled = true

	logBuffer := &bytes.Buffer{}
	gin.DefaultWriter = logBuffer
	gin.DefaultErrorWriter = logBuffer

	successRecorder := performSensitiveRelayRequest(t, "req-alert-1", "/v1/chat/completions", 1001, "alice", `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"please keep this secret"}]}`)
	require.Equal(t, http.StatusInternalServerError, successRecorder.Code)
	require.Contains(t, successRecorder.Body.String(), `"code":"sensitive_words_detected"`)
	require.Contains(t, successRecorder.Body.String(), "req-alert-1")
	require.NotContains(t, logBuffer.String(), "please keep this secret")
	require.NotContains(t, logBuffer.String(), "secret")
	require.Contains(t, logBuffer.String(), "user sensitive words detected (1 hits)")

	var events []entmodel.AlertEvent
	require.NoError(t, db.Find(&events).Error)
	require.Len(t, events, 1)
	require.Equal(t, "req-alert-1", events[0].RequestId)
	require.Equal(t, "alice", events[0].Username)
	require.Equal(t, "gpt-4o-mini", events[0].ModelName)
	require.Equal(t, "1 sensitive word hits", events[0].Summary)
	snapshot, err := events[0].ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot, 1)
	require.Equal(t, 11, snapshot[0].DepartmentId)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	logBuffer.Reset()
	failureRecorder := performSensitiveRelayRequest(t, "req-alert-2", "/v1/chat/completions", 1001, "alice", `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"this secret should still block"}]}`)
	require.Equal(t, successRecorder.Code, failureRecorder.Code)
	require.Contains(t, failureRecorder.Body.String(), `"code":"sensitive_words_detected"`)
	require.Contains(t, failureRecorder.Body.String(), "req-alert-2")
	require.NotContains(t, logBuffer.String(), "this secret should still block")
	require.NotContains(t, logBuffer.String(), "secret")
	require.Contains(t, logBuffer.String(), "enterprise alert event persistence skipped")
}

func TestRelaySensitiveWordsRecordsTenantScopedAlertEvent(t *testing.T) {
	oldDB := model.DB
	oldSensitiveWords := append([]string(nil), setting.SensitiveWords...)
	oldCheckEnabled := setting.CheckSensitiveEnabled
	oldPromptCheckEnabled := setting.CheckSensitiveOnPromptEnabled
	defer func() {
		model.DB = oldDB
		setting.SensitiveWords = oldSensitiveWords
		setting.CheckSensitiveEnabled = oldCheckEnabled
		setting.CheckSensitiveOnPromptEnabled = oldPromptCheckEnabled
	}()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, entmodel.AutoMigrate(db))
	model.DB = db

	require.NoError(t, db.Create(&model.User{Id: 2002, Username: "bob", Password: "password123", Group: "default", AffCode: "bob-aff"}).Error)
	require.NoError(t, db.Create(&[]entmodel.Department{
		{Id: 11, TenantId: 0, Name: "Default Tenant", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"},
		{Id: 19, TenantId: 9, Name: "Tenant Nine", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"},
	}).Error)
	require.NoError(t, db.Create(&[]entmodel.UserDepartment{
		{TenantId: 0, UserId: 2002, DepartmentId: 11, ExternalSource: constant.EnterpriseExternalSourceManual, Status: constant.EnterpriseMembershipStatusActive},
		{TenantId: 9, UserId: 2002, DepartmentId: 19, ExternalSource: constant.EnterpriseExternalSourceManual, Status: constant.EnterpriseMembershipStatusActive},
	}).Error)

	setting.SensitiveWords = []string{"secret"}
	setting.CheckSensitiveEnabled = true
	setting.CheckSensitiveOnPromptEnabled = true

	recorder := performSensitiveRelayRequest(t, "req-alert-tenant", "/v1/chat/completions?tenant_id=9", 2002, "bob", `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"tenant secret"}]}`)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	var event entmodel.AlertEvent
	require.NoError(t, db.First(&event).Error)
	require.Equal(t, 9, event.TenantId)
	snapshot, err := event.ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot, 1)
	require.Equal(t, "Tenant Nine", snapshot[0].DepartmentName)
	require.Equal(t, "1 sensitive word hits", event.Summary)
	require.NotContains(t, event.Summary, "tenant secret")
}

func performSensitiveRelayRequest(t *testing.T, requestID string, target string, userID int, username string, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(common.RequestIdKey, requestID)
	common.SetContextKey(ctx, constant.ContextKeyRequestStartTime, time.Unix(1717117200, 0))
	common.SetContextKey(ctx, constant.ContextKeyUserId, userID)
	common.SetContextKey(ctx, constant.ContextKeyUserName, username)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyOriginalModel, "gpt-4o-mini")

	Relay(ctx, types.RelayFormatOpenAI)
	return recorder
}
