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

func TestEnterpriseDepartmentRiskSummaryAPIRequiresEnterpriseAdminAndReturnsDrilldownContext(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	deptID := 1
	snapshot := modelenterprise.UsageSnapshot{
		TenantId:     0,
		DeptId:       &deptID,
		DeptName:     "Engineering",
		WindowStart:  1717117200,
		WindowEnd:    1717120800,
		RequestCount: 8,
	}
	require.NoError(t, snapshot.SetModelDistribution([]modelenterprise.UsageSnapshotModelStat{{ModelName: "gpt-4o-mini", RequestCount: 8}}))
	require.NoError(t, snapshot.SetUserIds([]int{1001}))
	require.NoError(t, fixture.db.Create(&snapshot).Error)

	unassignedSnapshot := modelenterprise.UsageSnapshot{
		TenantId:     0,
		WindowStart:  1717117200,
		WindowEnd:    1717120800,
		RequestCount: 1,
	}
	require.NoError(t, unassignedSnapshot.SetModelDistribution([]modelenterprise.UsageSnapshotModelStat{{ModelName: "gpt-4o-mini", RequestCount: 1}}))
	require.NoError(t, unassignedSnapshot.SetUserIds([]int{1002}))
	require.NoError(t, fixture.db.Create(&unassignedSnapshot).Error)

	event := modelenterprise.AlertEvent{
		TenantId:     0,
		UserId:       1001,
		Username:     "member",
		RequestId:    "req-risk-overview-api",
		ModelName:    "gpt-4o-mini",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "1 hit",
		CreatedAt:    1717117300,
		UpdatedAt:    1717117300,
	}
	require.NoError(t, event.SetDepartmentSnapshot([]modelenterprise.AlertEventDepartmentSnapshot{
		{DepartmentId: 1, DepartmentName: "Engineering"},
	}))
	require.NoError(t, fixture.db.Create(&event).Error)

	commonUser := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/department-summary?from=1717117200&to=1717120800", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	admin := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/department-summary?from=1717117200&to=1717120800&summary_sort=quota&summary_order=desc", adminCookies)
	adminPayload := decodeAdminActionsAPIResponse(t, admin)
	require.True(t, adminPayload.Success, adminPayload.Message)
	require.NotContains(t, string(adminPayload.Data), "prompt")
	require.NotContains(t, string(adminPayload.Data), "messages")

	var response dtoenterprise.DepartmentRiskSummaryResponse
	require.NoError(t, common.Unmarshal(adminPayload.Data, &response))
	require.NotEmpty(t, response.Items)
	require.Equal(t, "enterprise.usage.multi_dept_disclaimer", response.DisclaimerKey)
	require.Equal(t, "Engineering", response.TopDepartments[0].DeptName)
	require.Contains(t, response.TopDepartments[0].EventEntry.DetailRoute, "/enterprise-alerts?tab=events")
	require.Contains(t, response.TopDepartments[0].EventEntry.DetailAPIPath, "/api/enterprise/alerts/events?")
	require.True(t, response.Unassigned.IsUnassigned)
	require.True(t, response.Unassigned.EventEntry.UnassignedOnly)
}

func TestEnterpriseDepartmentRiskSummaryAPIRejectsInvalidRangeAndReturnsEmptyResult(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	invalidFrom := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/department-summary?from=bad&to=1717120800", adminCookies)
	invalidFromPayload := decodeAdminActionsAPIResponse(t, invalidFrom)
	require.False(t, invalidFromPayload.Success)
	require.Equal(t, "common.invalid_params", invalidFromPayload.Message)

	invalidRange := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/department-summary?from=1717120800&to=1717117200", adminCookies)
	invalidRangePayload := decodeAdminActionsAPIResponse(t, invalidRange)
	require.False(t, invalidRangePayload.Success)
	require.Equal(t, "common.invalid_params", invalidRangePayload.Message)

	empty := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/department-summary?from=1717117200&to=1717120800", adminCookies)
	emptyPayload := decodeAdminActionsAPIResponse(t, empty)
	require.True(t, emptyPayload.Success, emptyPayload.Message)

	var response dtoenterprise.DepartmentRiskSummaryResponse
	require.NoError(t, common.Unmarshal(emptyPayload.Data, &response))
	require.Len(t, response.Items, 1)
	require.Empty(t, response.TopDepartments)
	require.Empty(t, response.Trend)
	require.Equal(t, "enterprise.usage.multi_dept_disclaimer", response.DisclaimerKey)
	require.Equal(t, "Risk rate = risky requests / total requests", response.Formula.Expression)
	require.True(t, response.Unassigned.IsUnassigned)
	require.Equal(t, "未归属", response.Unassigned.DeptName)
	require.Zero(t, response.Unassigned.RiskEventCount)
	require.Zero(t, response.Unassigned.TotalRequestCount)
	require.True(t, response.Unassigned.EventEntry.UnassignedOnly)
	require.Equal(t, response.Unassigned, response.Items[0])
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
	require.Equal(t, "Unassigned · req-delivery-1 · /enterprise-alerts?event_id=77", response.Items[0].TraceSummary)
}

func TestEnterpriseAlertDeliveryResendAPIRequiresEnterpriseAdminAndCreatesManualDelivery(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)

	parent := modelenterprise.AlertDelivery{
		TenantId:      7,
		EventId:       77,
		RuleId:        5,
		ChannelType:   modelenterprise.AlertRuleChannelWebhook,
		Status:        modelenterprise.AlertDeliveryStatusFinalFailed,
		AttemptCount:  4,
		MaxAttempts:   4,
		FinalFailedAt: 1717117200,
		ErrorReason:   "webhook request failed",
		DedupeKey:     "7:77:5:webhook:1717117200",
		CreatedAt:     1717117200,
		UpdatedAt:     1717117201,
	}
	require.NoError(t, parent.SetTracePayload(&modelenterprise.AlertDeliveryTracePayload{
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
	require.NoError(t, fixture.db.Create(&parent).Error)

	commonUser := fixture.performEnterpriseRequest(t, http.MethodPost, "/api/enterprise/alerts/deliveries/1/resend?tenant_id=7", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)
	resend := fixture.performEnterpriseRequest(t, http.MethodPost, "/api/enterprise/alerts/deliveries/1/resend?tenant_id=7", adminCookies)
	resendPayload := decodeAdminActionsAPIResponse(t, resend)
	require.True(t, resendPayload.Success, resendPayload.Message)

	var resendResponse dtoenterprise.AlertDeliveryResendResponse
	require.NoError(t, common.Unmarshal(resendPayload.Data, &resendResponse))
	require.True(t, resendResponse.Created)
	require.Equal(t, modelenterprise.AlertDeliveryTriggerManual, resendResponse.Item.TriggerSource)
	require.NotNil(t, resendResponse.Item.ManualParentId)
	require.Equal(t, 1, *resendResponse.Item.ManualParentId)

	list := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/alerts/deliveries?tenant_id=7&event_id=77&page=1&page_size=20", adminCookies)
	listPayload := decodeAdminActionsAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)

	var listResponse dtoenterprise.AlertDeliveriesResponse
	require.NoError(t, common.Unmarshal(listPayload.Data, &listResponse))
	require.Len(t, listResponse.Items, 2)
	require.Equal(t, modelenterprise.AlertDeliveryStatusPending, listResponse.Items[0].Status)
	require.Equal(t, modelenterprise.AlertDeliveryStatusFinalFailed, listResponse.Items[1].Status)

	var actions []modelenterprise.AdminAction
	require.NoError(t, fixture.db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, "enterprise.alert.delivery.resend", actions[0].ActionType)
	require.NotContains(t, actions[0].Payload, "secret")
}

func TestEnterpriseAlertDeliveryResendAPIRejectsNonFinalFailedStatus(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)

	delivery := modelenterprise.AlertDelivery{
		TenantId:     7,
		EventId:      77,
		RuleId:       5,
		ChannelType:  modelenterprise.AlertRuleChannelWebhook,
		Status:       modelenterprise.AlertDeliveryStatusSent,
		AttemptCount: 1,
		MaxAttempts:  4,
		DedupeKey:    "7:77:5:webhook:1717117200",
		CreatedAt:    1717117200,
		UpdatedAt:    1717117201,
	}
	require.NoError(t, delivery.SetTracePayload(&modelenterprise.AlertDeliveryTracePayload{EventId: 77, RuleId: 5}))
	require.NoError(t, fixture.db.Create(&delivery).Error)

	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)
	resend := fixture.performEnterpriseRequest(t, http.MethodPost, "/api/enterprise/alerts/deliveries/1/resend?tenant_id=7", adminCookies)
	resendPayload := decodeAdminActionsAPIResponse(t, resend)
	require.False(t, resendPayload.Success)
	require.Equal(t, "enterprise.alert.delivery_resend_not_allowed", resendPayload.Message)
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
