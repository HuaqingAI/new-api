package enterprise

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAlertEventsAPIValidatesQueryAndReturnsPaginationEnvelope(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	event := entmodel.AlertEvent{
		TenantId:     0,
		UserId:       100,
		Username:     "alice",
		RequestId:    "req-alert-1",
		ModelName:    "gpt-4o-mini",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "2 sensitive word hits",
		CreatedAt:    1717117201,
		UpdatedAt:    1717117201,
	}
	require.NoError(t, event.SetDepartmentSnapshot([]entmodel.AlertEventDepartmentSnapshot{
		{DepartmentId: 1, DepartmentName: "Engineering"},
	}))
	require.NoError(t, db.Create(&event).Error)

	invalid := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/events?department_id=bad", nil)
	invalidResponse := decodeEnterpriseAPIResponse(t, invalid)
	require.False(t, invalidResponse.Success)
	require.Equal(t, "common.invalid_params", invalidResponse.Message)

	ok := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/events?department_id=1&page=1&page_size=20", nil)
	okResponse := decodeEnterpriseAPIResponse(t, ok)
	require.True(t, okResponse.Success, okResponse.Message)
	require.NotContains(t, ok.Body.String(), "prompt")
	require.NotContains(t, ok.Body.String(), "messages")

	payload := decodeEnterpriseData[dtoenterprise.AlertEventsResponse](t, okResponse)
	require.Equal(t, 1, payload.Total)
	require.Equal(t, 1, payload.Page)
	require.Equal(t, 20, payload.PageSize)
	require.Len(t, payload.Items, 1)
	require.Equal(t, "alice", payload.Items[0].Username)
	require.Equal(t, "Alice", payload.Items[0].DisplayName)
	require.Equal(t, "2 sensitive word hits", payload.Items[0].Summary)
	require.NotNil(t, payload.Items[0].DepartmentSnapshot)
	require.Len(t, payload.Items[0].DepartmentSnapshot, 1)
}

func TestAlertEventsAPIReturnsEmptyItemsArray(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/events?page=1&page_size=20", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	require.JSONEq(t, `{"items":[],"total":0,"page":1,"page_size":20}`, string(response.Data))
}

func TestAlertEventsAPISupportsEventIDFilter(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	first := entmodel.AlertEvent{
		TenantId:     0,
		UserId:       100,
		Username:     "alice",
		RequestId:    "req-alert-1",
		ModelName:    "gpt-4o-mini",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "first",
		CreatedAt:    1717117201,
		UpdatedAt:    1717117201,
	}
	second := entmodel.AlertEvent{
		TenantId:     0,
		UserId:       101,
		Username:     "bob",
		RequestId:    "req-alert-2",
		ModelName:    "gpt-4o-mini",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "second",
		CreatedAt:    1717117202,
		UpdatedAt:    1717117202,
	}
	require.NoError(t, first.SetDepartmentSnapshot(nil))
	require.NoError(t, second.SetDepartmentSnapshot(nil))
	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&second).Error)

	ok := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/events?event_id=2&page=1&page_size=20", nil)
	okResponse := decodeEnterpriseAPIResponse(t, ok)
	require.True(t, okResponse.Success, okResponse.Message)

	payload := decodeEnterpriseData[dtoenterprise.AlertEventsResponse](t, okResponse)
	require.Len(t, payload.Items, 1)
	require.Equal(t, 2, payload.Items[0].Id)
	require.Equal(t, "bob", payload.Items[0].Username)
}

func TestDepartmentRiskSummaryAPIValidatesQueryAndReturnsOverviewEnvelope(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	deptID := 1
	snapshot := entmodel.UsageSnapshot{
		TenantId:     0,
		DeptId:       &deptID,
		DeptName:     "Engineering",
		WindowStart:  1717117200,
		WindowEnd:    1717120800,
		RequestCount: 8,
	}
	require.NoError(t, snapshot.SetModelDistribution([]entmodel.UsageSnapshotModelStat{{ModelName: "gpt-4o-mini", RequestCount: 8}}))
	require.NoError(t, snapshot.SetUserIds([]int{100}))
	require.NoError(t, db.Create(&snapshot).Error)

	unassignedSnapshot := entmodel.UsageSnapshot{
		TenantId:     0,
		WindowStart:  1717117200,
		WindowEnd:    1717120800,
		RequestCount: 2,
	}
	require.NoError(t, unassignedSnapshot.SetModelDistribution([]entmodel.UsageSnapshotModelStat{{ModelName: "gpt-4o-mini", RequestCount: 2}}))
	require.NoError(t, unassignedSnapshot.SetUserIds([]int{101}))
	require.NoError(t, db.Create(&unassignedSnapshot).Error)

	event := entmodel.AlertEvent{
		TenantId:     0,
		UserId:       100,
		Username:     "alice",
		RequestId:    "req-risk-summary-1",
		ModelName:    "gpt-4o-mini",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "policy only",
		CreatedAt:    1717117300,
		UpdatedAt:    1717117300,
	}
	require.NoError(t, event.SetDepartmentSnapshot([]entmodel.AlertEventDepartmentSnapshot{
		{DepartmentId: 1, DepartmentName: "Engineering"},
	}))
	require.NoError(t, db.Create(&event).Error)

	invalid := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/department-summary?from=bad&to=1717120800", nil)
	invalidResponse := decodeEnterpriseAPIResponse(t, invalid)
	require.False(t, invalidResponse.Success)
	require.Equal(t, "common.invalid_params", invalidResponse.Message)

	ok := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/department-summary?from=1717117200&to=1717120800&summary_sort=quota&summary_order=desc", nil)
	okResponse := decodeEnterpriseAPIResponse(t, ok)
	require.True(t, okResponse.Success, okResponse.Message)

	payload := decodeEnterpriseData[dtoenterprise.DepartmentRiskSummaryResponse](t, okResponse)
	require.NotNil(t, payload.Items)
	require.NotNil(t, payload.TopDepartments)
	require.NotNil(t, payload.Trend)
	require.Equal(t, "enterprise.usage.multi_dept_disclaimer", payload.DisclaimerKey)
	require.Equal(t, "Risk rate = risky requests / total requests", payload.Formula.Expression)
	require.Len(t, payload.Items, 2)
	require.Equal(t, "Engineering", payload.TopDepartments[0].DeptName)
	require.Equal(t, int64(1), payload.TopDepartments[0].RiskEventCount)
	require.Equal(t, int64(8), payload.TopDepartments[0].TotalRequestCount)
	require.False(t, payload.TopDepartments[0].EventEntry.UnassignedOnly)
	require.Contains(t, payload.TopDepartments[0].EventEntry.DetailRoute, "department_id=1")
	require.True(t, payload.Unassigned.IsUnassigned)
	require.Contains(t, payload.Unassigned.EventEntry.DetailRoute, "unassigned_only=true")
}

func TestAlertDeliveriesAPIValidatesQueryAndReturnsEnvelope(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	delivery := entmodel.AlertDelivery{
		TenantId:      0,
		EventId:       1,
		RuleId:        2,
		ChannelType:   entmodel.AlertRuleChannelEmail,
		Status:        entmodel.AlertDeliveryStatusFinalFailed,
		AttemptCount:  4,
		MaxAttempts:   4,
		FinalFailedAt: 1717117201,
		ErrorReason:   "smtp timeout",
		DedupeKey:     "0:1:2:email:1717117200",
		NextRetryAt:   1717117200,
		CreatedAt:     1717117200,
		UpdatedAt:     1717117201,
	}
	require.NoError(t, delivery.SetTracePayload(&entmodel.AlertDeliveryTracePayload{
		EventId:            1,
		RequestId:          "req-1",
		TenantId:           0,
		UserId:             101,
		Username:           "alice",
		DisplayName:        "Alice",
		ModelName:          "gpt-4o-mini",
		RiskType:           "abuse",
		ActionResult:       "blocked",
		EventCreatedAt:     1717117200,
		DepartmentSnapshot: []entmodel.AlertEventDepartmentSnapshot{{DepartmentId: 1, DepartmentName: "Engineering"}},
		DepartmentSummary:  "Engineering (#1)",
		EventSummary:       "policy only",
		RuleId:             2,
		RuleName:           "Critical",
		DetailRoute:        "/enterprise-alerts?event_id=1",
		DetailAPIPath:      "/api/enterprise/alerts/events?tenant_id=0",
	}))
	require.NoError(t, db.Create(&delivery).Error)

	invalid := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/deliveries?rule_id=bad", nil)
	invalidResponse := decodeEnterpriseAPIResponse(t, invalid)
	require.False(t, invalidResponse.Success)
	require.Equal(t, "common.invalid_params", invalidResponse.Message)

	ok := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/deliveries?status=final_failed&page=1&page_size=20", nil)
	okResponse := decodeEnterpriseAPIResponse(t, ok)
	require.True(t, okResponse.Success, okResponse.Message)
	require.NotContains(t, ok.Body.String(), "secret")

	payload := decodeEnterpriseData[dtoenterprise.AlertDeliveriesResponse](t, okResponse)
	require.Equal(t, 1, payload.Total)
	require.Len(t, payload.Items, 1)
	require.Equal(t, entmodel.AlertDeliveryStatusFinalFailed, payload.Items[0].Status)
	require.NotNil(t, payload.Items[0].Trace)
	require.Equal(t, "req-1", payload.Items[0].Trace.RequestId)
	require.Equal(t, 101, payload.Items[0].Trace.UserId)
	require.Equal(t, "Alice", payload.Items[0].Trace.DisplayName)
	require.Equal(t, "Engineering (#1) · req-1 · /enterprise-alerts?tab=events&event_id=1", payload.Items[0].TraceSummary)
}

func TestAlertDeliveryResendAPICreatesManualDeliveryAndWritesAudit(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	parent := entmodel.AlertDelivery{
		TenantId:      0,
		EventId:       1,
		RuleId:        2,
		ChannelType:   entmodel.AlertRuleChannelEmail,
		Status:        entmodel.AlertDeliveryStatusFinalFailed,
		AttemptCount:  4,
		MaxAttempts:   4,
		FinalFailedAt: 1717117201,
		ErrorReason:   "smtp timeout",
		DedupeKey:     "0:1:2:email:1717117200",
		NextRetryAt:   1717117200,
		CreatedAt:     1717117200,
		UpdatedAt:     1717117201,
	}
	require.NoError(t, parent.SetTracePayload(&entmodel.AlertDeliveryTracePayload{
		EventId:           1,
		RequestId:         "req-1",
		TenantId:          0,
		UserId:            101,
		Username:          "alice",
		DisplayName:       "Alice",
		ModelName:         "gpt-4o-mini",
		RiskType:          "abuse",
		ActionResult:      "blocked",
		EventCreatedAt:    1717117200,
		DepartmentSummary: "Engineering (#1)",
		EventSummary:      "policy only",
		RuleId:            2,
		RuleName:          "Critical",
		DetailRoute:       "/enterprise-alerts?event_id=1",
		DetailAPIPath:     "/api/enterprise/alerts/events?tenant_id=0",
	}))
	require.NoError(t, db.Create(&parent).Error)

	recorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/alerts/deliveries/1/resend", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	payload := decodeEnterpriseData[dtoenterprise.AlertDeliveryResendResponse](t, response)
	require.True(t, payload.Created)
	require.Equal(t, entmodel.AlertDeliveryStatusPending, payload.Item.Status)
	require.Equal(t, entmodel.AlertDeliveryTriggerManual, payload.Item.TriggerSource)
	require.NotNil(t, payload.Item.ManualParentId)
	require.Equal(t, 1, *payload.Item.ManualParentId)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, entservice.AdminActionAlertDeliveryResend, actions[0].ActionType)
	require.Equal(t, entservice.AdminObjectAlertDelivery, actions[0].ObjectType)
}

func TestAlertDeliveryResendAPIRejectsNonFinalFailedStatus(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	delivery := entmodel.AlertDelivery{
		TenantId:     0,
		EventId:      1,
		RuleId:       2,
		ChannelType:  entmodel.AlertRuleChannelEmail,
		Status:       entmodel.AlertDeliveryStatusSent,
		AttemptCount: 1,
		MaxAttempts:  4,
		DedupeKey:    "0:1:2:email:1717117200",
		CreatedAt:    1717117200,
		UpdatedAt:    1717117201,
	}
	require.NoError(t, delivery.SetTracePayload(&entmodel.AlertDeliveryTracePayload{EventId: 1, RuleId: 2}))
	require.NoError(t, db.Create(&delivery).Error)

	recorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/alerts/deliveries/1/resend", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "enterprise.alert.delivery_resend_not_allowed", response.Message)
}

func TestAlertRulesAPIValidatesPersistsAndSanitizesSecrets(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	invalidRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/alerts/rules", map[string]any{
		"name":       "Broken",
		"risk_types": []string{"abuse"},
		"channel_configs": []map[string]any{
			{
				"type":      "email",
				"receivers": []string{"bad-email"},
			},
		},
	})
	invalidResponse := decodeEnterpriseAPIResponse(t, invalidRecorder)
	require.False(t, invalidResponse.Success)
	require.Equal(t, "enterprise.alert.rule_invalid_email", invalidResponse.Message)

	saveRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/alerts/rules", map[string]any{
		"name":                  "Critical abuse",
		"enabled":               true,
		"risk_types":            []string{"abuse", "sensitive_words"},
		"department_ids":        []int{1, 2},
		"dedupe_window_seconds": 300,
		"channel_configs": []map[string]any{
			{
				"type":      "email",
				"enabled":   true,
				"receivers": []string{"ops@example.com"},
			},
			{
				"type":           "webhook",
				"enabled":        true,
				"webhook_url":    "https://hooks.example.com/alerts?token=secret",
				"webhook_secret": "plain-secret",
			},
		},
	})
	saveResponse := decodeEnterpriseAPIResponse(t, saveRecorder)
	require.True(t, saveResponse.Success, saveResponse.Message)
	require.NotContains(t, saveRecorder.Body.String(), "plain-secret")
	require.NotContains(t, saveRecorder.Body.String(), "token=secret")

	payload := decodeEnterpriseData[dtoenterprise.AlertRuleResponse](t, saveResponse)
	require.Equal(t, "Critical abuse", payload.Item.Name)
	require.Len(t, payload.Item.ChannelConfigs, 2)
	require.True(t, payload.Item.ChannelConfigs[1].WebhookSecretConfigured)
	require.NotEmpty(t, payload.Item.ChannelConfigs[1].WebhookSecretMasked)
	require.NotContains(t, payload.Item.ChannelConfigs[1].WebhookSecretMasked, "plain-secret")

	listRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/rules", nil)
	listResponse := decodeEnterpriseAPIResponse(t, listRecorder)
	require.True(t, listResponse.Success, listResponse.Message)
	require.NotContains(t, listRecorder.Body.String(), "plain-secret")

	detailRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/rules/1", nil)
	detailResponse := decodeEnterpriseAPIResponse(t, detailRecorder)
	require.True(t, detailResponse.Success, detailResponse.Message)
	require.NotContains(t, detailRecorder.Body.String(), "plain-secret")

	deleteRecorder := performEnterpriseRequest(t, router, http.MethodDelete, "/api/enterprise/alerts/rules/1", nil)
	deleteResponse := decodeEnterpriseAPIResponse(t, deleteRecorder)
	require.True(t, deleteResponse.Success, deleteResponse.Message)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 2)
	require.Equal(t, entservice.AdminActionAlertRuleSave, actions[0].ActionType)
	require.Equal(t, entservice.AdminActionAlertRuleDelete, actions[1].ActionType)
	require.NotContains(t, actions[0].Payload, "plain-secret")
	require.NotContains(t, actions[0].Payload, "token=secret")
}

func TestAlertRulesAPISupportsTenantScopeOnDetailAndDelete(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)

	saveRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/alerts/rules?tenant_id=7", map[string]any{
		"name":       "Tenant scoped",
		"risk_types": []string{"abuse"},
		"channel_configs": []map[string]any{
			{
				"type":      "email",
				"receivers": []string{"ops@example.com"},
			},
		},
	})
	saveResponse := decodeEnterpriseAPIResponse(t, saveRecorder)
	require.True(t, saveResponse.Success, saveResponse.Message)

	detailRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/rules/1?tenant_id=7", nil)
	detailResponse := decodeEnterpriseAPIResponse(t, detailRecorder)
	require.True(t, detailResponse.Success, detailResponse.Message)

	notFoundRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/rules/1", nil)
	notFoundResponse := decodeEnterpriseAPIResponse(t, notFoundRecorder)
	require.False(t, notFoundResponse.Success)
	require.Equal(t, "enterprise.alert.rule_not_found", notFoundResponse.Message)

	deleteRecorder := performEnterpriseRequest(t, router, http.MethodDelete, "/api/enterprise/alerts/rules/1?tenant_id=7", nil)
	deleteResponse := decodeEnterpriseAPIResponse(t, deleteRecorder)
	require.True(t, deleteResponse.Success, deleteResponse.Message)
}

func TestDepartmentRiskSummaryAPIAllowsScopedDepartmentOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	db := setupEnterpriseBudgetPermissionDB(t)
	model.DB = db
	require.NoError(t, db.Create(&model.User{Id: 202, Username: "risk-owner", Password: "password123", AffCode: "risk-owner-aff"}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       202,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualGrant,
		Effect:       constant.EnterpriseDepartmentRoleEffectAllow,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	deptID := 1
	snapshot := entmodel.UsageSnapshot{
		TenantId:     0,
		DeptId:       &deptID,
		DeptName:     "Engineering",
		WindowStart:  1717117200,
		WindowEnd:    1717120800,
		RequestCount: 3,
	}
	require.NoError(t, snapshot.SetModelDistribution(nil))
	require.NoError(t, snapshot.SetUserIds([]int{202}))
	require.NoError(t, db.Create(&snapshot).Error)
	event := entmodel.AlertEvent{
		TenantId:     0,
		UserId:       202,
		Username:     "risk-owner",
		RequestId:    "req-risk-owner",
		ModelName:    "gpt-4o-mini",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "policy only",
		CreatedAt:    1717117300,
		UpdatedAt:    1717117300,
	}
	require.NoError(t, event.SetDepartmentSnapshot([]entmodel.AlertEventDepartmentSnapshot{
		{DepartmentId: 1, DepartmentName: "Engineering"},
	}))
	require.NoError(t, db.Create(&event).Error)

	router.Use(func(c *gin.Context) {
		c.Set("id", 202)
		c.Set("role", common.RoleCommonUser)
		c.Next()
	})
	router.GET("/api/enterprise/alerts/department-summary", GetDepartmentRiskSummary)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/alerts/department-summary?department_id=1&from=1717117200&to=1717120800&include_descendants=true", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	payload := decodeEnterpriseData[dtoenterprise.DepartmentRiskSummaryResponse](t, response)
	require.True(t, payload.IncludeDescendants)
	require.Equal(t, "Engineering", payload.ScopeDepartmentName)
	require.Equal(t, []int{1}, payload.ScopeDepartmentIds)
}
