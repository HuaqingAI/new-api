package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	serviceenterprise "github.com/QuantumNous/new-api/service/enterprise"
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
	var emptyResponse dtoenterprise.DepartmentUsageSummaryResponse
	require.NoError(t, common.Unmarshal(emptyPayload.Data, &emptyResponse))
	require.Empty(t, emptyResponse.Items)
	require.Empty(t, emptyResponse.Scope.DepartmentName)
	require.False(t, emptyResponse.Scope.IncludeDescendants)
	require.Empty(t, emptyResponse.Scope.DepartmentIds)
	require.Zero(t, emptyResponse.Scope.RequestCount)
	require.Zero(t, emptyResponse.Scope.Quota)

	emptyForRollingWindow := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-summary?from=1699913600&to=1700000000", adminCookies)
	emptyRollingPayload := decodeAdminActionsAPIResponse(t, emptyForRollingWindow)
	require.True(t, emptyRollingPayload.Success, emptyRollingPayload.Message)
	var emptyRollingResponse dtoenterprise.DepartmentUsageSummaryResponse
	require.NoError(t, common.Unmarshal(emptyRollingPayload.Data, &emptyRollingResponse))
	require.Empty(t, emptyRollingResponse.Items)
	require.Empty(t, emptyRollingResponse.Scope.DepartmentIds)
	require.Zero(t, emptyRollingResponse.Scope.RequestCount)
}

func TestEnterpriseUsageSummaryAPIHonorsSummarySortParams(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	engineerID := 11
	opsID := 22
	engineer := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           &engineerID,
		DeptName:         "Engineering",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     9,
		PromptTokens:     90,
		CompletionTokens: 30,
		Quota:            200,
	}
	require.NoError(t, engineer.SetModelDistribution(nil))
	require.NoError(t, engineer.SetUserIds([]int{101, 102, 103}))

	ops := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           &opsID,
		DeptName:         "Operations",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     3,
		PromptTokens:     30,
		CompletionTokens: 10,
		Quota:            100,
	}
	require.NoError(t, ops.SetModelDistribution(nil))
	require.NoError(t, ops.SetUserIds([]int{104}))
	require.NoError(t, fixture.db.Create(&engineer).Error)
	require.NoError(t, fixture.db.Create(&ops).Error)

	recorder := fixture.performEnterpriseRequest(
		t,
		http.MethodGet,
		"/api/enterprise/usage/department-summary?from=1700000000&to=1700003600&summary_sort=users&summary_order=asc",
		adminCookies,
	)
	payload := decodeAdminActionsAPIResponse(t, recorder)
	require.True(t, payload.Success, payload.Message)

	var response dtoenterprise.DepartmentUsageSummaryResponse
	require.NoError(t, common.Unmarshal(payload.Data, &response))
	require.Len(t, response.Items, 2)
	require.Equal(t, "Operations", response.Items[0].DeptName)
	require.Equal(t, "Engineering", response.Items[1].DeptName)
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

func TestEnterpriseUsageExportAPIRequiresEnterpriseAdminAndStreamsSnapshotCSV(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	parentID := 10
	deptID := 11

	require.NoError(t, fixture.db.Create(&[]modelenterprise.Department{
		enterpriseDepartment(parentID, nil, "Platform", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
		enterpriseDepartment(deptID, intPtr(parentID), "Engineering", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
	}).Error)

	snapshotA := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           intPtr(deptID),
		DeptName:         "Engineering",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     8,
		PromptTokens:     80,
		CompletionTokens: 16,
		Quota:            240,
	}
	require.NoError(t, snapshotA.SetModelDistribution([]modelenterprise.UsageSnapshotModelStat{
		{ModelName: "gpt-4o", RequestCount: 8, PromptTokens: 80, CompletionTokens: 16, Quota: 240},
	}))
	require.NoError(t, snapshotA.SetUserIds([]int{101, 102}))

	snapshotB := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           nil,
		DeptName:         "",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     3,
		PromptTokens:     30,
		CompletionTokens: 9,
		Quota:            90,
	}
	require.NoError(t, snapshotB.SetModelDistribution(nil))
	require.NoError(t, snapshotB.SetUserIds([]int{999}))

	require.NoError(t, fixture.db.Create(&snapshotA).Error)
	require.NoError(t, fixture.db.Create(&snapshotB).Error)
	commonUser := fixture.performEnterpriseRequest(
		t,
		http.MethodGet,
		"/api/enterprise/usage/export?from=1714521600&to=1714608000",
		fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled),
	)
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	admin := fixture.performEnterpriseRequest(
		t,
		http.MethodGet,
		"/api/enterprise/usage/export?from=1714521600&to=1714608000&summary_sort=dept_name&summary_order=asc",
		fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled),
	)
	require.Equal(t, http.StatusOK, admin.Code)
	require.Equal(t, "text/csv; charset=utf-8", admin.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="usage-department-20240501-20240501.csv"`, admin.Header().Get("Content-Disposition"))
	require.Contains(t, admin.Body.String(), "# 注意：用量按用户当前所属部门重复计入，部门间数值不可加和")
	require.Contains(t, admin.Body.String(), "部门 ID,部门名称,父部门,周期开始,周期结束,请求数,prompt_tokens,completion_tokens,quota,用户数")
	require.Contains(t, admin.Body.String(), "11,Engineering,Platform,1714521600,1714608000,8,80,16,240,2")
	require.Contains(t, admin.Body.String(), ",未归属,,1714521600,1714608000,3,30,9,90,1")
	lines := strings.Split(strings.TrimSpace(admin.Body.String()), "\n")
	require.GreaterOrEqual(t, len(lines), 4)
	require.Equal(t, "# 注意：用量按用户当前所属部门重复计入，部门间数值不可加和", lines[0])
	require.Equal(t, "部门 ID,部门名称,父部门,周期开始,周期结束,请求数,prompt_tokens,completion_tokens,quota,用户数", lines[1])
	require.NotContains(t, admin.Body.String(), "should-not-be-read")
	require.NotContains(t, admin.Body.String(), "null")
	require.NotContains(t, admin.Body.String(), "<nil>")
}

func TestEnterpriseUsageDetailAPIRequiresEnterpriseAdminAndReturnsFilterContext(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.Log{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 101, Username: "alice", Password: "password123", Group: "default", AffCode: "alice-usage"}).Error)
	require.NoError(t, fixture.db.Create(&model.User{Id: 102, Username: "bob", Password: "password123", Group: "default", AffCode: "bob-usage"}).Error)
	require.NoError(t, fixture.db.Create(&[]modelenterprise.Department{
		enterpriseDepartment(11, nil, "Engineering", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
		enterpriseDepartment(22, nil, "Operations", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
	}).Error)
	require.NoError(t, fixture.db.Create(&[]modelenterprise.UserDepartment{
		{
			UserId:         101,
			DepartmentId:   11,
			ExternalSource: constant.EnterpriseExternalSourceManual,
			Status:         constant.EnterpriseMembershipStatusActive,
		},
		{
			UserId:         101,
			DepartmentId:   22,
			ExternalSource: constant.EnterpriseExternalSourceManual,
			Status:         constant.EnterpriseMembershipStatusActive,
		},
		{
			UserId:         102,
			DepartmentId:   11,
			ExternalSource: constant.EnterpriseExternalSourceManual,
			Status:         constant.EnterpriseMembershipStatusActive,
		},
	}).Error)

	snapshotA := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           intPtr(11),
		DeptName:         "Engineering",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     2,
		PromptTokens:     60,
		CompletionTokens: 30,
		Quota:            150,
	}
	require.NoError(t, snapshotA.SetModelDistribution([]modelenterprise.UsageSnapshotModelStat{
		{ModelName: "gpt-4o", RequestCount: 2, PromptTokens: 60, CompletionTokens: 30, Quota: 150},
	}))
	require.NoError(t, snapshotA.SetUserIds([]int{101, 102}))
	snapshotB := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           intPtr(11),
		DeptName:         "Engineering",
		WindowStart:      1700003600,
		WindowEnd:        1700007200,
		RequestCount:     1,
		PromptTokens:     80,
		CompletionTokens: 40,
		Quota:            200,
	}
	require.NoError(t, snapshotB.SetModelDistribution([]modelenterprise.UsageSnapshotModelStat{
		{ModelName: "claude-sonnet-4", RequestCount: 1, PromptTokens: 80, CompletionTokens: 40, Quota: 200},
	}))
	require.NoError(t, snapshotB.SetUserIds([]int{101}))
	require.NoError(t, fixture.db.Create(&snapshotA).Error)
	require.NoError(t, fixture.db.Create(&snapshotB).Error)
	require.NoError(t, fixture.db.Create(&[]model.Log{
		{
			Id:               1,
			UserId:           101,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			ModelName:        "gpt-4o",
			Quota:            100,
			PromptTokens:     40,
			CompletionTokens: 20,
			CreatedAt:        1700000100,
		},
		{
			Id:               2,
			UserId:           102,
			Username:         "bob",
			Type:             model.LogTypeConsume,
			ModelName:        "gpt-4o",
			Quota:            50,
			PromptTokens:     20,
			CompletionTokens: 10,
			CreatedAt:        1700000200,
		},
		{
			Id:               3,
			UserId:           101,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			ModelName:        "claude-sonnet-4",
			Quota:            200,
			PromptTokens:     80,
			CompletionTokens: 40,
			CreatedAt:        1700003900,
		},
	}).Error)

	commonUser := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=11&from=1700000000&to=1700007200", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	admin := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=11&from=1700000000&to=1700007200", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	adminPayload := decodeAdminActionsAPIResponse(t, admin)
	require.True(t, adminPayload.Success, adminPayload.Message)

	var response dtoenterprise.DepartmentUsageDetailResponse
	require.NoError(t, common.Unmarshal(adminPayload.Data, &response))
	require.Equal(t, "Engineering", response.DeptName)
	require.Equal(t, int64(3), response.RequestCount)
	require.Equal(t, int64(210), response.TokenCount)
	require.Len(t, response.UserRanking, 2)
	require.Equal(t, "alice", response.UserRanking[0].Username)
	require.Len(t, response.ModelDistribution, 2)
	require.Len(t, response.Trend, 2)
	require.Equal(t, "/usage-logs/common", response.RecentLogsEntry.Path)
	require.Equal(t, int64(1700000000), response.RecentLogsEntry.Filters.StartTimestamp)
	require.Equal(t, int64(1700007199), response.RecentLogsEntry.Filters.EndTimestamp)
	require.Equal(t, []string{"alice", "bob"}, response.RecentLogsEntry.Filters.UsernameOptions)
}

func TestEnterpriseUsageReportConfigAPIRequiresEnterpriseAdminAndPersists(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)

	commonUser := fixture.performEnterpriseRequestWithBody(
		t,
		http.MethodPut,
		"/api/enterprise/usage/reports",
		fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled),
		map[string]any{
			"receivers":  []string{"ops@example.com"},
			"frequency":  "daily",
			"range_type": "last7d",
			"enabled":    true,
		},
	)
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")

	admin := fixture.performEnterpriseRequestWithBody(
		t,
		http.MethodPut,
		"/api/enterprise/usage/reports",
		fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled),
		map[string]any{
			"receivers":  []string{"ops@example.com", "cto@example.com"},
			"frequency":  "monthly",
			"range_type": "last30d",
			"enabled":    true,
		},
	)
	adminPayload := decodeAdminActionsAPIResponse(t, admin)
	require.True(t, adminPayload.Success, adminPayload.Message)
	require.Contains(t, string(adminPayload.Data), `"frequency":"monthly"`)
	require.Contains(t, string(adminPayload.Data), `"range_type":"last30d"`)

	getRecorder := fixture.performEnterpriseRequest(
		t,
		http.MethodGet,
		"/api/enterprise/usage/reports",
		fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled),
	)
	getPayload := decodeAdminActionsAPIResponse(t, getRecorder)
	require.True(t, getPayload.Success, getPayload.Message)
	require.Contains(t, string(getPayload.Data), `"receivers":["ops@example.com","cto@example.com"]`)
}

func TestEnterpriseUsageReportConfigAPIReflectsJobStatusAfterRun(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.Local)

	job := modelenterprise.UsageReportJob{
		TenantId:  0,
		Frequency: modelenterprise.UsageReportFrequencyDaily,
		RangeType: modelenterprise.UsageReportRangeLast7Days,
		Enabled:   true,
		Status:    modelenterprise.UsageReportStatusPending,
		NextRunAt: now.Unix() - 1,
	}
	require.NoError(t, job.SetReceivers([]string{"ops@example.com"}))
	require.NoError(t, job.SetLastSnapshot(nil))
	require.NoError(t, fixture.db.Create(&job).Error)

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	windowStart := startOfDay.AddDate(0, 0, -6).Unix()
	windowEnd := startOfDay.Add(24 * time.Hour).Unix()
	deptID := 11
	snapshot := modelenterprise.UsageSnapshot{
		TenantId:         0,
		DeptId:           &deptID,
		DeptName:         "Engineering",
		WindowStart:      windowStart,
		WindowEnd:        windowEnd,
		RequestCount:     8,
		PromptTokens:     80,
		CompletionTokens: 20,
		Quota:            160,
	}
	require.NoError(t, snapshot.SetModelDistribution(nil))
	require.NoError(t, snapshot.SetUserIds([]int{1001, 1002}))
	require.NoError(t, fixture.db.Create(&snapshot).Error)

	service := serviceenterprise.NewUsageReportServiceForTest(
		fixture.db,
		func() time.Time { return now },
		func(subject string, receiver string, content string) error { return nil },
	)
	result, err := service.RunDueReports(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	require.Zero(t, result.Failed)

	getRecorder := fixture.performEnterpriseRequest(
		t,
		http.MethodGet,
		"/api/enterprise/usage/reports",
		fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled),
	)
	getPayload := decodeAdminActionsAPIResponse(t, getRecorder)
	require.True(t, getPayload.Success, getPayload.Message)
	require.Contains(t, string(getPayload.Data), `"status":"success"`)
	require.Contains(t, string(getPayload.Data), `"last_success_at":`)
	require.Contains(t, string(getPayload.Data), `"run_count":1`)
	require.Contains(t, string(getPayload.Data), `"top_departments":[`)
	require.Contains(t, string(getPayload.Data), `"dept_name":"Engineering"`)
}

func TestEnterpriseUsageDetailAPIRejectsInvalidParamsAndReturnsEmptyArrays(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	invalid := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=11&from=1700003600&to=1700000000", adminCookies)
	invalidPayload := decodeAdminActionsAPIResponse(t, invalid)
	require.False(t, invalidPayload.Success)
	require.Equal(t, "common.invalid_params", invalidPayload.Message)

	empty := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=11&from=1700000000&to=1700003600", adminCookies)
	emptyPayload := decodeAdminActionsAPIResponse(t, empty)
	require.True(t, emptyPayload.Success, emptyPayload.Message)
	require.Contains(t, string(emptyPayload.Data), `"user_ranking":[]`)
	require.Contains(t, string(emptyPayload.Data), `"model_distribution":[]`)
	require.Contains(t, string(emptyPayload.Data), `"username_options":[]`)
}

func intPtr(value int) *int {
	return &value
}
