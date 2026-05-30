package api_test

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestEnterpriseAlertsAPIRequiresEnterpriseAdmin(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)

	event := modelenterprise.AlertEvent{
		TenantId:     0,
		UserId:       2001,
		Username:     "member",
		RequestId:    "req-alert-api-1",
		ModelName:    "gpt-4o-mini",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "2 sensitive word hits",
		CreatedAt:    1717117201,
		UpdatedAt:    1717117201,
	}
	require.NoError(t, event.SetDepartmentSnapshot([]modelenterprise.AlertEventDepartmentSnapshot{
		{DepartmentId: 1, DepartmentName: "Engineering"},
	}))
	require.NoError(t, fixture.db.Create(&event).Error)

	commonUser := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/events?page=1&page_size=20", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	deptAdmin := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/events?page=1&page_size=20", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))
	deptAdminPayload := decodeAdminActionsAPIResponse(t, deptAdmin)
	require.False(t, deptAdminPayload.Success)
	require.Contains(t, deptAdminPayload.Message, "error.enterprise.permission.admin_required")

	admin := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/events?department_id=1&page=1&page_size=20", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	adminPayload := decodeAdminActionsAPIResponse(t, admin)
	require.True(t, adminPayload.Success, adminPayload.Message)
	require.NotContains(t, string(adminPayload.Data), "prompt")
	require.NotContains(t, string(adminPayload.Data), "messages")
	require.NotContains(t, string(adminPayload.Data), "sensitive_hits")

	var response dtoenterprise.AlertEventsResponse
	require.NoError(t, common.Unmarshal(adminPayload.Data, &response))
	require.Equal(t, 1, response.Total)
	require.Equal(t, 1, response.Page)
	require.Equal(t, 20, response.PageSize)
	require.Len(t, response.Items, 1)
	require.Equal(t, "member", response.Items[0].Username)
}

func TestEnterpriseAlertsAPIHonorsTenantScopeAndEmptyResult(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	event := modelenterprise.AlertEvent{
		TenantId:     7,
		UserId:       7001,
		Username:     "tenant-user",
		RequestId:    "req-alert-api-tenant",
		ModelName:    "claude-sonnet-4",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "review requested",
		CreatedAt:    1717117201,
		UpdatedAt:    1717117201,
	}
	require.NoError(t, event.SetDepartmentSnapshot(nil))
	require.NoError(t, fixture.db.Create(&event).Error)

	empty := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/events?page=1&page_size=20", adminCookies)
	emptyPayload := decodeAdminActionsAPIResponse(t, empty)
	require.True(t, emptyPayload.Success, emptyPayload.Message)
	require.JSONEq(t, `{"items":[],"total":0,"page":1,"page_size":20}`, string(emptyPayload.Data))

	scoped := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/events?tenant_id=7&user_id=7001&page=1&page_size=20", adminCookies)
	scopedPayload := decodeAdminActionsAPIResponse(t, scoped)
	require.True(t, scopedPayload.Success, scopedPayload.Message)

	var response dtoenterprise.AlertEventsResponse
	require.NoError(t, common.Unmarshal(scopedPayload.Data, &response))
	require.Len(t, response.Items, 1)
	require.NotNil(t, response.Items[0].DepartmentSnapshot)
	require.Empty(t, response.Items[0].DepartmentSnapshot)
}

func TestEnterpriseAlertDeliveriesAPIRequiresEnterpriseAdminAndSanitizesTracePayload(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)

	delivery := modelenterprise.AlertDelivery{
		TenantId:     7,
		EventId:      77,
		RuleId:       5,
		ChannelType:  modelenterprise.AlertRuleChannelWebhook,
		Status:       modelenterprise.AlertDeliveryStatusFinalFailed,
		AttemptCount: 4,
		MaxAttempts:  4,
		ErrorReason:  "webhook request failed",
		DedupeKey:    "7:77:5:webhook:1717117200",
		CreatedAt:    1717117200,
		UpdatedAt:    1717117201,
	}
	require.NoError(t, delivery.SetTracePayload(&modelenterprise.AlertDeliveryTracePayload{
		EventId:           77,
		RequestId:         "req-delivery-1",
		TenantId:          7,
		Username:          "tenant-user",
		ModelName:         "claude-sonnet-4",
		RiskType:          "abuse",
		ActionResult:      "blocked",
		EventCreatedAt:    1717117200,
		DepartmentSummary: "Unassigned",
		EventSummary:      "review requested",
		RuleId:            5,
		RuleName:          "Tenant 7 abuse",
		DetailRoute:       "/enterprise-alerts?event_id=77",
		DetailAPIPath:     "/api/enterprise/alerts/events?tenant_id=7",
	}))
	require.NoError(t, fixture.db.Create(&delivery).Error)

	deliveryTwo := modelenterprise.AlertDelivery{
		TenantId:     7,
		EventId:      88,
		RuleId:       6,
		ChannelType:  modelenterprise.AlertRuleChannelEmail,
		Status:       modelenterprise.AlertDeliveryStatusSent,
		AttemptCount: 1,
		MaxAttempts:  4,
		SentAt:       1717117800,
		DedupeKey:    "7:88:6:email:1717117500",
		CreatedAt:    1717117500,
		UpdatedAt:    1717117800,
	}
	require.NoError(t, deliveryTwo.SetTracePayload(&modelenterprise.AlertDeliveryTracePayload{
		EventId:           88,
		RequestId:         "req-delivery-2",
		TenantId:          7,
		Username:          "tenant-user",
		ModelName:         "gpt-4o-mini",
		RiskType:          "abuse",
		ActionResult:      "blocked",
		EventCreatedAt:    1717117500,
		DepartmentSummary: "Engineering (#1)",
		EventSummary:      "emailed",
		RuleId:            6,
		RuleName:          "Tenant 7 email",
		DetailRoute:       "/enterprise-alerts?event_id=88",
		DetailAPIPath:     "/api/enterprise/alerts/events?tenant_id=7",
	}))
	require.NoError(t, fixture.db.Create(&deliveryTwo).Error)

	commonUser := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/deliveries?page=1&page_size=20", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	admin := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/deliveries?tenant_id=7&status=final_failed&page=1&page_size=20", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	adminPayload := decodeAdminActionsAPIResponse(t, admin)
	require.True(t, adminPayload.Success, adminPayload.Message)
	require.NotContains(t, string(adminPayload.Data), "secret")

	var response dtoenterprise.AlertDeliveriesResponse
	require.NoError(t, common.Unmarshal(adminPayload.Data, &response))
	require.Len(t, response.Items, 1)
	require.Equal(t, 1, response.Total)
	require.Equal(t, 1, response.Page)
	require.Equal(t, 20, response.PageSize)
	require.Equal(t, "req-delivery-1", response.Items[0].Trace.RequestId)
	require.Equal(t, "webhook", response.Items[0].ChannelType)
	require.Equal(t, "webhook request failed", response.Items[0].ErrorReason)
}

func TestEnterpriseAlertRulesAPIRequiresEnterpriseAdminAndSanitizesSecrets(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)

	commonUser := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/alerts/rules", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled), map[string]any{
		"name":       "Rule",
		"risk_types": []string{"abuse"},
		"channel_configs": []map[string]any{
			{
				"type":      "email",
				"receivers": []string{"ops@example.com"},
			},
		},
	})
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	deptAdmin := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/alerts/rules", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled), map[string]any{
		"name":       "Rule",
		"risk_types": []string{"abuse"},
		"channel_configs": []map[string]any{
			{
				"type":      "email",
				"receivers": []string{"ops@example.com"},
			},
		},
	})
	deptAdminPayload := decodeAdminActionsAPIResponse(t, deptAdmin)
	require.False(t, deptAdminPayload.Success)
	require.Contains(t, deptAdminPayload.Message, "error.enterprise.permission.admin_required")

	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)
	save := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/alerts/rules?tenant_id=7", adminCookies, map[string]any{
		"name":            "Tenant 7 abuse",
		"risk_types":      []string{"abuse"},
		"department_ids":  []int{101},
		"channel_configs": []map[string]any{{"type": "email", "receivers": []string{"ops@example.com"}}, {"type": "webhook", "enabled": true, "webhook_url": "https://hooks.example.com/path?credential=plain", "webhook_secret": "plain-secret"}},
	})
	savePayload := decodeAdminActionsAPIResponse(t, save)
	require.True(t, savePayload.Success, savePayload.Message)
	require.NotContains(t, string(savePayload.Data), "plain-secret")
	require.NotContains(t, string(savePayload.Data), "credential=plain")

	list := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/rules?tenant_id=7", adminCookies)
	listPayload := decodeAdminActionsAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	require.NotContains(t, string(listPayload.Data), "plain-secret")

	var actions []modelenterprise.AdminAction
	require.NoError(t, fixture.db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, "enterprise.alert.rule.save", actions[0].ActionType)
	require.Equal(t, "enterprise_alert_rule", actions[0].ObjectType)
	require.NotContains(t, actions[0].Payload, "plain-secret")
	require.NotContains(t, actions[0].Payload, "credential=plain")
}

func TestEnterpriseAlertRulesAPIRejectsOptionalOnlyChannelsWithoutEmailLoop(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	recorder := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/alerts/rules", adminCookies, map[string]any{
		"name":       "Webhook only",
		"risk_types": []string{"abuse"},
		"channel_configs": []map[string]any{
			{
				"type":        "webhook",
				"enabled":     true,
				"webhook_url": "https://hooks.example.com/path",
			},
		},
	})
	response := decodeAdminActionsAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "common.invalid_params", response.Message)
}
