package api_test

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestEnterpriseUsageSummaryAPIRequiresEnterpriseAdminAndReadsSnapshots(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.Log{}))
	require.NoError(t, fixture.db.Create(&modelenterprise.UsageSnapshot{
		TenantId:          0,
		DeptId:            nil,
		DeptName:          "",
		WindowStart:       1700000000,
		WindowEnd:         1700003600,
		RequestCount:      2,
		PromptTokens:      20,
		CompletionTokens:  5,
		Quota:             90,
		UserCount:         1,
		ModelDistribution: `[{"model_name":"gpt-4o","request_count":2,"prompt_tokens":20,"completion_tokens":5,"quota":90}]`,
	}).Error)
	require.NoError(t, fixture.db.Create(&model.Log{
		Id:               1,
		UserId:           9999,
		Username:         "ignored-log",
		Type:             model.LogTypeConsume,
		ModelName:        "should-not-be-read",
		Quota:            999,
		PromptTokens:     888,
		CompletionTokens: 777,
		CreatedAt:        1700000100,
	}).Error)

	commonUser := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-summary?from=1700000000&to=1700003600", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	admin := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-summary?from=1700000000&to=1700003600", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	adminPayload := decodeAdminActionsAPIResponse(t, admin)
	require.True(t, adminPayload.Success, adminPayload.Message)
	require.Contains(t, string(adminPayload.Data), `"dept_name":"未归属"`)
	require.Contains(t, string(adminPayload.Data), `"request_count":2`)
	require.Contains(t, string(adminPayload.Data), `"quota":90`)
	require.Contains(t, string(adminPayload.Data), `"model_distribution":[`)
	require.NotContains(t, string(adminPayload.Data), "should-not-be-read")
}

func TestEnterpriseUsageSummaryAPIRejectsInvalidRangeAndReturnsEmptySnapshotList(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	invalidFrom := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-summary?from=bad&to=1700003600", adminCookies)
	invalidFromPayload := decodeAdminActionsAPIResponse(t, invalidFrom)
	require.False(t, invalidFromPayload.Success)
	require.Equal(t, "common.invalid_params", invalidFromPayload.Message)

	invalidRange := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-summary?from=1700003600&to=1700000000", adminCookies)
	invalidRangePayload := decodeAdminActionsAPIResponse(t, invalidRange)
	require.False(t, invalidRangePayload.Success)
	require.Equal(t, "common.invalid_params", invalidRangePayload.Message)

	empty := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-summary?from=1700000000&to=1700003600", adminCookies)
	emptyPayload := decodeAdminActionsAPIResponse(t, empty)
	require.True(t, emptyPayload.Success, emptyPayload.Message)
	require.JSONEq(t, `{"items":[]}`, string(emptyPayload.Data))

	emptyForRollingWindow := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-summary?from=1699913600&to=1700000000", adminCookies)
	emptyRollingPayload := decodeAdminActionsAPIResponse(t, emptyForRollingWindow)
	require.True(t, emptyRollingPayload.Success, emptyRollingPayload.Message)
	require.JSONEq(t, `{"items":[]}`, string(emptyRollingPayload.Data))
}

func TestEnterpriseUsageSummaryAPIPreservesMultiDepartmentAttributionAcrossWindows(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	engineeringId := 11
	operationsId := 22

	snapshotA := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           &engineeringId,
		DeptName:         "Engineering",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     2,
		PromptTokens:     200,
		CompletionTokens: 80,
		Quota:            500,
	}
	require.NoError(t, snapshotA.SetModelDistribution([]modelenterprise.UsageSnapshotModelStat{
		{
			ModelName:        "gpt-4o",
			RequestCount:     2,
			PromptTokens:     200,
			CompletionTokens: 80,
			Quota:            500,
		},
	}))
	require.NoError(t, snapshotA.SetUserIds([]int{101}))

	snapshotB := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           &operationsId,
		DeptName:         "Operations",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     2,
		PromptTokens:     200,
		CompletionTokens: 80,
		Quota:            500,
	}
	require.NoError(t, snapshotB.SetModelDistribution([]modelenterprise.UsageSnapshotModelStat{
		{
			ModelName:        "gpt-4o",
			RequestCount:     2,
			PromptTokens:     200,
			CompletionTokens: 80,
			Quota:            500,
		},
	}))
	require.NoError(t, snapshotB.SetUserIds([]int{101}))

	snapshotC := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           &engineeringId,
		DeptName:         "Engineering",
		WindowStart:      1700003600,
		WindowEnd:        1700007200,
		RequestCount:     3,
		PromptTokens:     300,
		CompletionTokens: 120,
		Quota:            900,
	}
	require.NoError(t, snapshotC.SetModelDistribution([]modelenterprise.UsageSnapshotModelStat{
		{
			ModelName:        "claude-sonnet-4",
			RequestCount:     3,
			PromptTokens:     300,
			CompletionTokens: 120,
			Quota:            900,
		},
	}))
	require.NoError(t, snapshotC.SetUserIds([]int{101, 202}))

	require.NoError(t, fixture.db.Create(&snapshotA).Error)
	require.NoError(t, fixture.db.Create(&snapshotB).Error)
	require.NoError(t, fixture.db.Create(&snapshotC).Error)

	recorder := fixture.performEnterpriseRequest(
		t,
		http.MethodGet,
		"/api/enterprise/usage/department-summary?from=1700000000&to=1700007200",
		adminCookies,
	)
	payload := decodeAdminActionsAPIResponse(t, recorder)
	require.True(t, payload.Success, payload.Message)

	var response dtoenterprise.DepartmentUsageSummaryResponse
	require.NoError(t, common.Unmarshal(payload.Data, &response))
	require.Len(t, response.Items, 2)

	itemsByDept := make(map[string]dtoenterprise.DepartmentUsageSummaryItem, len(response.Items))
	for _, item := range response.Items {
		itemsByDept[item.DeptName] = item
	}

	require.Contains(t, itemsByDept, "Engineering")
	require.Contains(t, itemsByDept, "Operations")

	engineering := itemsByDept["Engineering"]
	require.Equal(t, int64(1700000000), engineering.WindowStart)
	require.Equal(t, int64(1700007200), engineering.WindowEnd)
	require.Equal(t, int64(5), engineering.RequestCount)
	require.Equal(t, int64(2), engineering.UserCount)
	require.Len(t, engineering.ModelDistribution, 2)

	operations := itemsByDept["Operations"]
	require.Equal(t, int64(1700000000), operations.WindowStart)
	require.Equal(t, int64(1700003600), operations.WindowEnd)
	require.Equal(t, int64(2), operations.RequestCount)
	require.Equal(t, int64(1), operations.UserCount)
	require.Len(t, operations.ModelDistribution, 1)
}
