package enterprise

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestDingTalkConfigAPIStoresSecretWithoutReturningIt(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/dingtalk/config", GetDingTalkConfig)
	router.PUT("/api/enterprise/dingtalk/config", SaveDingTalkConfig)

	secret := "plain-secret"
	recorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/dingtalk/config", dtoenterprise.DingTalkConfigRequest{
		CorpId:       "corp-id",
		AppKey:       "app-key",
		AppSecret:    &secret,
		CallbackUrl:  "https://example.com/api/oauth/dingtalk",
		SyncScope:    "1,2",
		LoginEnabled: boolPtr(true),
		SyncEnabled:  boolPtr(true),
	})

	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	require.NotContains(t, recorder.Body.String(), secret)
	require.NotContains(t, recorder.Body.String(), `"app_secret":`)
	saved := decodeEnterpriseData[dtoenterprise.DingTalkConfigResponse](t, response)
	require.True(t, saved.HasAppSecret)

	getRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/dingtalk/config", nil)
	getResponse := decodeEnterpriseAPIResponse(t, getRecorder)
	require.True(t, getResponse.Success, getResponse.Message)
	require.NotContains(t, getRecorder.Body.String(), secret)
	require.NotContains(t, getRecorder.Body.String(), `"app_secret":`)
	fetched := decodeEnterpriseData[dtoenterprise.DingTalkConfigResponse](t, getResponse)
	require.True(t, fetched.HasAppSecret)
	require.Equal(t, "corp-id", fetched.CorpId)

	var config entmodel.DingTalkConfig
	require.NoError(t, db.Where("tenant_id = ?", 0).First(&config).Error)
	require.Equal(t, secret, config.AppSecret)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Where("action_type = ?", entservice.AdminActionDingTalkConfigSet).Find(&actions).Error)
	require.Len(t, actions, 1)
	require.NotContains(t, actions[0].Payload, secret)
	require.NotContains(t, actions[0].Payload, "app-key")
	require.Contains(t, actions[0].Payload, "[REDACTED]")
}

func TestDingTalkConfigAPIRejectsEnableWithDistinctErrors(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)
	router.PUT("/api/enterprise/dingtalk/config", SaveDingTalkConfig)

	missingRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/dingtalk/config", dtoenterprise.DingTalkConfigRequest{
		CorpId:       "corp-id",
		CallbackUrl:  "https://example.com/api/oauth/dingtalk",
		LoginEnabled: boolPtr(true),
	})
	missingResponse := decodeEnterpriseAPIResponse(t, missingRecorder)
	require.False(t, missingResponse.Success)
	require.Equal(t, "enterprise.dingtalk.missing_credentials", missingResponse.Message)

	secret := "plain-secret"
	invalidCallbackRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/dingtalk/config", dtoenterprise.DingTalkConfigRequest{
		CorpId:      "corp-id",
		AppKey:      "app-key",
		AppSecret:   &secret,
		CallbackUrl: "http://example.com/api/oauth/dingtalk",
		SyncEnabled: boolPtr(true),
	})
	invalidCallbackResponse := decodeEnterpriseAPIResponse(t, invalidCallbackRecorder)
	require.False(t, invalidCallbackResponse.Success)
	require.Equal(t, "enterprise.dingtalk.invalid_callback_url", invalidCallbackResponse.Message)
	require.NotContains(t, invalidCallbackRecorder.Body.String(), secret)
}

func TestDingTalkConnectivityAPIWritesSanitizedAudit(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/dingtalk/connectivity-test", TestDingTalkConnectivity)

	secret := "plain-secret"
	require.NoError(t, db.Create(&entmodel.DingTalkConfig{
		TenantId:    0,
		CorpId:      "corp-id",
		AppKey:      "app-key",
		AppSecret:   secret,
		CallbackUrl: "http://example.com/api/oauth/dingtalk",
	}).Error)

	recorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/dingtalk/connectivity-test", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	require.Contains(t, recorder.Body.String(), `"code":"callback_misconfigured"`)
	require.NotContains(t, recorder.Body.String(), secret)
	require.NotContains(t, recorder.Body.String(), `"app_secret"`)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Where("action_type = ?", entservice.AdminActionDingTalkTest).Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Contains(t, actions[0].Payload, "callback_misconfigured")
	require.Contains(t, actions[0].Payload, "callback_url_invalid")
	require.NotContains(t, actions[0].Payload, secret)
	require.NotContains(t, actions[0].Payload, "app-key")
	require.NotContains(t, actions[0].Payload, "access_token")
}

func TestDingTalkSyncConflictsAPIListsPendingConflicts(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/dingtalk/sync/conflicts", ListDingTalkSyncConflicts)
	require.NoError(t, db.Create(&entmodel.DingTalkSyncConflict{
		TenantId:        0,
		TaskId:          9,
		LastTaskId:      9,
		ExternalUserId:  "staff-conflict",
		UnionId:         "union-conflict",
		Email:           "taken@example.com",
		Name:            "Conflict",
		ConflictType:    "email",
		CandidateUserId: 100,
		Details:         "email_matches_existing_local_user",
		Status:          constant.DingTalkSyncConflictStatusPending,
	}).Error)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/dingtalk/sync/conflicts?status=pending", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)

	require.True(t, response.Success, response.Message)
	data := decodeEnterpriseData[entservice.DingTalkSyncConflictsResult](t, response)
	require.Equal(t, 1, data.Total)
	require.Len(t, data.Items, 1)
	require.Equal(t, "staff-conflict", data.Items[0].ExternalUserId)
	require.Equal(t, "email", data.Items[0].ConflictType)
}
