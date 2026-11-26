package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/requestplugin"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/request_plugin_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRelaySensitiveWordsRecordsAlertEventWithoutLeakingWordsOrChangingResponse(t *testing.T) {
	db, logBuffer := setupSensitiveRelayTest(t)

	require.NoError(t, db.Create(&model.User{Id: 1001, Username: "alice", Password: "password123", Group: "default", AffCode: "alice-aff"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 11, TenantId: 0, Name: "Engineering", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         1001,
		DepartmentId:   11,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
	}).Error)

	successRecorder := performSensitiveRelayRequest(t, "req-alert-1", "/v1/chat/completions", 1001, "alice", `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"please keep this secret"}]}`)
	assert.Equal(t, http.StatusBadRequest, successRecorder.Code)
	assert.Contains(t, successRecorder.Body.String(), `"code":"request_rejected"`)
	assert.Contains(t, successRecorder.Body.String(), "req-alert-1")
	assert.NotContains(t, logBuffer.String(), "please keep this secret")
	assert.NotContains(t, logBuffer.String(), "secret")

	var events []entmodel.AlertEvent
	require.NoError(t, db.Find(&events).Error)
	require.Len(t, events, 1)
	assert.Equal(t, "req-alert-1", events[0].RequestId)
	assert.Equal(t, "alice", events[0].Username)
	assert.Equal(t, "gpt-4o-mini", events[0].ModelName)
	assert.Equal(t, "sensitive words detected", events[0].Summary)
	snapshot, err := events[0].ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot, 1)
	assert.Equal(t, 11, snapshot[0].DepartmentId)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	logBuffer.Reset()
	failureRecorder := performSensitiveRelayRequest(t, "req-alert-2", "/v1/chat/completions", 1001, "alice", `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"this secret should still block"}]}`)
	assert.Equal(t, successRecorder.Code, failureRecorder.Code)
	assert.Contains(t, failureRecorder.Body.String(), `"code":"request_rejected"`)
	assert.Contains(t, failureRecorder.Body.String(), "req-alert-2")
	assert.NotContains(t, logBuffer.String(), "this secret should still block")
	assert.NotContains(t, logBuffer.String(), "secret")
	assert.Contains(t, logBuffer.String(), "enterprise alert event persistence skipped")
}

func TestRelaySensitiveWordsRecordsTenantScopedAlertEvent(t *testing.T) {
	db, _ := setupSensitiveRelayTest(t)

	require.NoError(t, db.Create(&model.User{Id: 2002, Username: "bob", Password: "password123", Group: "default", AffCode: "bob-aff"}).Error)
	require.NoError(t, db.Create(&[]entmodel.Department{
		{Id: 11, TenantId: 0, Name: "Default Tenant", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"},
		{Id: 19, TenantId: 9, Name: "Tenant Nine", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"},
	}).Error)
	require.NoError(t, db.Create(&[]entmodel.UserDepartment{
		{TenantId: 0, UserId: 2002, DepartmentId: 11, ExternalSource: constant.EnterpriseExternalSourceManual, Status: constant.EnterpriseMembershipStatusActive},
		{TenantId: 9, UserId: 2002, DepartmentId: 19, ExternalSource: constant.EnterpriseExternalSourceManual, Status: constant.EnterpriseMembershipStatusActive},
	}).Error)

	recorder := performSensitiveRelayRequest(t, "req-alert-tenant", "/v1/chat/completions?tenant_id=9", 2002, "bob", `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"tenant secret"}]}`)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var event entmodel.AlertEvent
	require.NoError(t, db.First(&event).Error)
	assert.Equal(t, 9, event.TenantId)
	snapshot, err := event.ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot, 1)
	assert.Equal(t, "Tenant Nine", snapshot[0].DepartmentName)
	assert.Equal(t, "sensitive words detected", event.Summary)
	assert.NotContains(t, event.Summary, "tenant secret")
}

func TestRelaySensitiveWordsPluginClassificationAndDefaults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tenant string
		plugin string
		config string
		stage  string
		code   types.ErrorCode
		record bool
	}{
		{name: "invalid tenant and absent model retain defaults", tenant: "invalid", plugin: "request-filter", stage: "ingress", code: types.ErrorCodeRequestRejected, record: true},
		{name: "negative tenant retains default tenant", tenant: "-1", plugin: "request-filter", stage: "ingress", code: types.ErrorCodeRequestRejected, record: true},
		{name: "egress preserves original model and records once", plugin: "request-filter", stage: "egress", code: types.ErrorCodeRequestRejected, record: true},
		{name: "other plugin using same code is not a sensitive event", plugin: "rp-other-filter", stage: "ingress", code: types.ErrorCodeRequestRejected},
		{name: "filter failure is not a sensitive event", plugin: "request-filter", config: `{"rules":[{"pattern":"[","type":"regex"}]}`, stage: "ingress", code: types.ErrorCodeRequestPluginFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := setupSensitiveRelayTest(t)
			require.NoError(t, db.Create(&entmodel.Department{Id: 11, TenantId: 0, Name: "Engineering", NameHistory: "[]"}).Error)
			require.NoError(t, db.Create(&entmodel.UserDepartment{TenantId: 0, UserId: 1001, DepartmentId: 11, Status: constant.EnterpriseMembershipStatusActive}).Error)
			if tc.plugin == "rp-other-filter" {
				plugin, err := jsplugin.DefaultRegistry.Register(`export const meta = {apiVersion: 1, key: "rp-other-filter", name: "Other filter", version: "1.0.0", author: {name: "Test"}};
export function ingress() { return {action: "reject", code: "request_filtered"}; }`, jsplugin.Options{})
				require.NoError(t, err)
				t.Cleanup(func() { _ = jsplugin.DefaultRegistry.Unregister(plugin.Meta.Key) })
			}
			configured := request_plugin_setting.GetInstances()
			require.Len(t, configured, 1)
			instance := configured[0]
			instance.Plugin = tc.plugin
			if tc.config != "" {
				instance.Config = []byte(tc.config)
			}
			instances, err := common.Marshal([]request_plugin_setting.Instance{instance})
			require.NoError(t, err)
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{request_plugin_setting.InstancesOption: string(instances)}))
			body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"private-request-body secret"}]}`
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions?tenant_id="+tc.tenant, strings.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(ctx, constant.ContextKeyUserId, 1001)
			common.SetContextKey(ctx, constant.ContextKeyUserName, "alice")
			ctx.Set(common.RequestIdKey, "req-alert-classification")
			wantModel := ""
			if tc.stage == "egress" {
				wantModel = "client-model@effort:high"
				common.SetContextKey(ctx, constant.ContextKeyOriginalModel, wantModel)
				ctx.Request.Body = http.NoBody
			}
			_, _, apiErr := requestplugin.Ingress(ctx, types.RelayFormatOpenAI)
			if tc.stage == "egress" {
				require.Nil(t, apiErr)
				apiErr = requestplugin.Egress(ctx, &relaycommon.RelayInfo{OriginModelName: wantModel}, []byte(body))
			}
			require.NotNil(t, apiErr)
			assert.Equal(t, tc.code, apiErr.GetErrorCode())
			if tc.record {
				// The same rejection reaching another stage does not duplicate its alert.
				assert.NotNil(t, requestplugin.Egress(ctx, &relaycommon.RelayInfo{OriginModelName: wantModel}, []byte(body)))
			}
			var events []entmodel.AlertEvent
			require.NoError(t, db.Find(&events).Error)
			if !tc.record {
				assert.Empty(t, events)
				return
			}
			require.Len(t, events, 1)
			assert.Zero(t, events[0].TenantId)
			assert.Equal(t, wantModel, events[0].ModelName)
			assert.Equal(t, "sensitive_words", events[0].RiskType)
			assert.Equal(t, "blocked", events[0].ActionResult)
			assert.Equal(t, "sensitive words detected", events[0].Summary)
			snapshot, err := events[0].ParsedDepartmentSnapshot()
			require.NoError(t, err)
			require.Len(t, snapshot, 1)
			assert.Equal(t, 11, snapshot[0].DepartmentId)
			encoded, err := common.Marshal(events[0])
			require.NoError(t, err)
			for _, secret := range []string{"private-request-body", "secret", "private-rule-id", "private-rejection-message"} {
				assert.NotContains(t, string(encoded), secret)
			}
		})
	}
}

func setupSensitiveRelayTest(t *testing.T) (*gorm.DB, *bytes.Buffer) {
	t.Helper()
	require.NoError(t, i18n.Init())
	oldDB := model.DB
	oldWriter, oldErrorWriter := gin.DefaultWriter, gin.DefaultErrorWriter
	oldInstances, err := common.Marshal(request_plugin_setting.GetInstances())
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		model.DB = oldDB
		gin.DefaultWriter, gin.DefaultErrorWriter = oldWriter, oldErrorWriter
		require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{request_plugin_setting.InstancesOption: string(oldInstances)}))
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, entmodel.AutoMigrate(db))
	model.DB = db
	source, err := os.ReadFile(filepath.Join("..", "plugins", "tasks", "request-filter", "plugin.js"))
	require.NoError(t, err)
	plugin, err := jsplugin.DefaultRegistry.Register(string(source), jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = jsplugin.DefaultRegistry.Unregister(plugin.Meta.Key) })
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{request_plugin_setting.InstancesOption: `[{"id":"sensitive-filter","name":"Filter","plugin":"request-filter","enabled":true,"global":true,"config":{"rules":[{"pattern":"secret","id":"private-rule-id"}],"message":"private-rejection-message"}}]`}))
	logBuffer := &bytes.Buffer{}
	gin.DefaultWriter, gin.DefaultErrorWriter = logBuffer, logBuffer
	return db, logBuffer
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
