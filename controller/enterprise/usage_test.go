package enterprise

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageSummaryAPIValidatesTimeRangeAndNormalizesArrays(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	require.NoError(t, db.Create(&entmodel.UsageSnapshot{
		TenantId:          0,
		DeptId:            nil,
		DeptName:          "",
		WindowStart:       1700000000,
		WindowEnd:         1700003600,
		RequestCount:      1,
		PromptTokens:      10,
		CompletionTokens:  5,
		Quota:             20,
		UserCount:         1,
		ModelDistribution: "",
	}).Error)

	invalidRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-summary?from=bad&to=1700003600", nil)
	invalidResponse := decodeEnterpriseAPIResponse(t, invalidRecorder)
	require.False(t, invalidResponse.Success)
	require.Equal(t, "common.invalid_params", invalidResponse.Message)

	rangeRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-summary?from=1700003600&to=1700000000", nil)
	rangeResponse := decodeEnterpriseAPIResponse(t, rangeRecorder)
	require.False(t, rangeResponse.Success)
	require.Equal(t, "common.invalid_params", rangeResponse.Message)

	okRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-summary?from=1700000000&to=1700003600", nil)
	okResponse := decodeEnterpriseAPIResponse(t, okRecorder)
	require.True(t, okResponse.Success, okResponse.Message)
	require.Contains(t, okRecorder.Body.String(), `"dept_name":"未归属"`)
	require.Contains(t, okRecorder.Body.String(), `"model_distribution":[]`)

	var payload dtoenterprise.DepartmentUsageSummaryResponse
	require.NoError(t, common.Unmarshal(okResponse.Data, &payload))
	require.Len(t, payload.Items, 1)
	require.NotNil(t, payload.Items[0].ModelDistribution)
	require.Equal(t, int64(1700000000), payload.Items[0].WindowStart)
	require.Equal(t, int64(1700003600), payload.Items[0].WindowEnd)
}

func TestUsageSummaryAPIAppliesRequestedSummarySort(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	engineerID := 1
	securityID := 2
	engineer := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &engineerID,
		DeptName:         "Engineering",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     9,
		PromptTokens:     90,
		CompletionTokens: 20,
		Quota:            300,
	}
	require.NoError(t, engineer.SetModelDistribution(nil))
	require.NoError(t, engineer.SetUserIds([]int{100, 101, 102}))

	security := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &securityID,
		DeptName:         "Security",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     3,
		PromptTokens:     30,
		CompletionTokens: 10,
		Quota:            100,
	}
	require.NoError(t, security.SetModelDistribution(nil))
	require.NoError(t, security.SetUserIds([]int{103}))
	require.NoError(t, db.Create(&engineer).Error)
	require.NoError(t, db.Create(&security).Error)

	recorder := performEnterpriseRequest(
		t,
		router,
		http.MethodGet,
		"/api/enterprise/usage/department-summary?from=1714521600&to=1714608000&summary_sort=users&summary_order=asc",
		nil,
	)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var payload dtoenterprise.DepartmentUsageSummaryResponse
	require.NoError(t, common.Unmarshal(response.Data, &payload))
	require.Len(t, payload.Items, 2)
	require.Equal(t, "Security", payload.Items[0].DeptName)
	require.Equal(t, "Engineering", payload.Items[1].DeptName)
}

func TestUsageDashboardAndPeersAPIReadScopeSnapshots(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	parentID := 1
	require.NoError(t, db.Model(&entmodel.Department{}).Where("id = ?", 2).Update("parent_id", parentID).Error)
	require.NoError(t, db.Create(&entmodel.Department{
		Id:       3,
		TenantId: 0,
		Name:     "Reliability",
		ParentId: &parentID,
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)
	securityID := 2
	require.NoError(t, db.Create(&entmodel.Department{
		Id:       4,
		TenantId: 0,
		Name:     "Authentication",
		ParentId: &securityID,
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)

	departmentID := 1
	tenantSnapshot := entmodel.UsageScopeSnapshot{
		TenantId:         0,
		ScopeType:        entmodel.UsageScopeTypeTenant,
		ScopeKey:         entmodel.UsageScopeTypeTenant,
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     2,
		PromptTokens:     20,
		CompletionTokens: 10,
		Quota:            60,
	}
	require.NoError(t, tenantSnapshot.SetModelDistribution(nil))
	require.NoError(t, tenantSnapshot.SetUserIds([]int{100, 101}))
	departmentSnapshot := entmodel.UsageScopeSnapshot{
		TenantId:         0,
		ScopeType:        entmodel.UsageScopeTypeDepartment,
		ScopeKey:         "department:1",
		DepartmentId:     &departmentID,
		DepartmentName:   "Engineering",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     2,
		PromptTokens:     20,
		CompletionTokens: 10,
		Quota:            60,
	}
	require.NoError(t, departmentSnapshot.SetModelDistribution(nil))
	require.NoError(t, departmentSnapshot.SetUserIds([]int{100, 101}))
	require.NoError(t, db.Create(&tenantSnapshot).Error)
	require.NoError(t, db.Create(&departmentSnapshot).Error)

	for _, child := range []struct {
		id           int
		name         string
		requestCount int64
	}{
		{id: 2, name: "Security", requestCount: 2},
		{id: 3, name: "Reliability", requestCount: 1},
	} {
		childID := child.id
		snapshot := entmodel.UsageScopeSnapshot{
			TenantId:       0,
			ScopeType:      entmodel.UsageScopeTypeDepartment,
			ScopeKey:       "department:" + strconv.Itoa(child.id),
			DepartmentId:   &childID,
			DepartmentName: child.name,
			WindowStart:    1700000000,
			WindowEnd:      1700003600,
			RequestCount:   child.requestCount,
		}
		require.NoError(t, snapshot.SetModelDistribution(nil))
		require.NoError(t, snapshot.SetUserIds([]int{child.id + 98}))
		require.NoError(t, db.Create(&snapshot).Error)
	}
	grandchildID := 4
	grandchildSnapshot := entmodel.UsageScopeSnapshot{
		TenantId:       0,
		ScopeType:      entmodel.UsageScopeTypeDepartment,
		ScopeKey:       "department:4",
		DepartmentId:   &grandchildID,
		DepartmentName: "Authentication",
		WindowStart:    1700000000,
		WindowEnd:      1700003600,
		RequestCount:   1,
	}
	require.NoError(t, grandchildSnapshot.SetModelDistribution(nil))
	require.NoError(t, grandchildSnapshot.SetUserIds([]int{100}))
	require.NoError(t, db.Create(&grandchildSnapshot).Error)

	overviewRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-overview?from=1700000000&to=1700003600", nil)
	overviewResponse := decodeEnterpriseAPIResponse(t, overviewRecorder)
	require.True(t, overviewResponse.Success, overviewResponse.Message)
	var overview dtoenterprise.DepartmentUsageOverviewResponse
	require.NoError(t, common.Unmarshal(overviewResponse.Data, &overview))
	assert.Equal(t, int64(2), overview.Metrics.RequestCount)
	require.Len(t, overview.Items, 1)
	assert.Equal(t, "Engineering", overview.Items[0].DeptName)
	require.Len(t, overview.SecondLevelItems, 2)
	assert.Equal(t, []string{"Security", "Reliability"}, []string{overview.SecondLevelItems[0].DeptName, overview.SecondLevelItems[1].DeptName})
	assert.Equal(t, []int64{2, 1}, []int64{overview.SecondLevelItems[0].RequestCount, overview.SecondLevelItems[1].RequestCount})
	require.Len(t, overview.Trend, 1)

	peersRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-peers?department_id=2&from=1700000000&to=1700003600&include_descendants=true", nil)
	peersResponse := decodeEnterpriseAPIResponse(t, peersRecorder)
	require.True(t, peersResponse.Success, peersResponse.Message)
	var peers dtoenterprise.DepartmentUsagePeersResponse
	require.NoError(t, common.Unmarshal(peersResponse.Data, &peers))
	assert.Equal(t, "Engineering", peers.ParentDepartmentName)
	require.Len(t, peers.Items, 2)
	assert.Equal(t, []string{"Security", "Reliability"}, []string{peers.Items[0].DeptName, peers.Items[1].DeptName})
}

func TestUsageExportAPIStreamsCSVWithHeadersAndSorting(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	parentID := 9
	require.NoError(t, db.Create(&entmodel.Department{
		Id:       parentID,
		TenantId: 0,
		Name:     "Platform",
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)

	deptID := 1
	require.NoError(t, db.Model(&entmodel.Department{}).Where("id = ?", deptID).Updates(map[string]any{
		"parent_id": &parentID,
	}).Error)

	snapshotA := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &deptID,
		DeptName:         "Engineering",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     5,
		PromptTokens:     50,
		CompletionTokens: 10,
		Quota:            120,
	}
	require.NoError(t, snapshotA.SetModelDistribution(nil))
	require.NoError(t, snapshotA.SetUserIds([]int{100, 101}))
	snapshotB := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           nil,
		DeptName:         "",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     2,
		PromptTokens:     20,
		CompletionTokens: 4,
		Quota:            40,
	}
	require.NoError(t, snapshotB.SetModelDistribution(nil))
	require.NoError(t, snapshotB.SetUserIds([]int{999}))
	require.NoError(t, db.Create(&snapshotA).Error)
	require.NoError(t, db.Create(&snapshotB).Error)
	rootScope := entmodel.UsageScopeSnapshot{
		TenantId:         0,
		ScopeKey:         "department:9",
		ScopeType:        entmodel.UsageScopeTypeDepartment,
		DepartmentId:     &parentID,
		DepartmentName:   "Platform",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     5,
		PromptTokens:     50,
		CompletionTokens: 10,
		Quota:            120,
	}
	require.NoError(t, rootScope.SetModelDistribution(nil))
	require.NoError(t, rootScope.SetUserIds([]int{100, 101}))
	require.NoError(t, db.Create(&rootScope).Error)

	recorder := performEnterpriseRequest(
		t,
		router,
		http.MethodGet,
		"/api/enterprise/usage/export?from=1714521600&to=1714608000&summary_sort=dept_name&summary_order=asc",
		nil,
	)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/csv; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="usage-department-20240501-20240501.csv"`, recorder.Header().Get("Content-Disposition"))

	body := recorder.Body.String()
	require.Contains(t, body, "# 注意：企业总览按一级部门完整子树展示，未归属用量仅计入企业总量，部门间数值不可加和")
	require.Contains(t, body, "统计口径,部门 ID,部门名称,父部门,周期开始,周期结束,请求数,prompt_tokens,completion_tokens,quota,用户数")
	require.Contains(t, body, "root_subtree,9,Platform,,1714521600,1714608000,5,50,10,120,2")

	lines := strings.Split(strings.TrimSpace(body), "\n")
	require.GreaterOrEqual(t, len(lines), 4)
	require.Equal(t, "# 注意：企业总览按一级部门完整子树展示，未归属用量仅计入企业总量，部门间数值不可加和", lines[0])
	require.Equal(t, "统计口径,部门 ID,部门名称,父部门,周期开始,周期结束,请求数,prompt_tokens,completion_tokens,quota,用户数", lines[1])
	require.NotContains(t, body, "null")
	require.NotContains(t, body, "<nil>")
}

func TestUsageExportAPIRejectsInvalidRange(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)

	invalidFromRecorder := performEnterpriseRequest(
		t,
		router,
		http.MethodGet,
		"/api/enterprise/usage/export?from=bad&to=1714608000",
		nil,
	)
	invalidFromResponse := decodeEnterpriseAPIResponse(t, invalidFromRecorder)
	require.False(t, invalidFromResponse.Success)
	require.Equal(t, "common.invalid_params", invalidFromResponse.Message)

	invalidRangeRecorder := performEnterpriseRequest(
		t,
		router,
		http.MethodGet,
		"/api/enterprise/usage/export?from=1714608000&to=1714521600",
		nil,
	)
	invalidRangeResponse := decodeEnterpriseAPIResponse(t, invalidRangeRecorder)
	require.False(t, invalidRangeResponse.Success)
	require.Equal(t, "common.invalid_params", invalidRangeResponse.Message)
}

func TestUsageDetailAPIValidatesTimeRangeAndNormalizesArrays(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	deptID := 1
	require.NoError(t, db.Create(&entmodel.UsageSnapshot{
		TenantId:          0,
		DeptId:            &deptID,
		DeptName:          "Engineering",
		WindowStart:       1700000000,
		WindowEnd:         1700003600,
		RequestCount:      1,
		PromptTokens:      10,
		CompletionTokens:  5,
		Quota:             20,
		UserCount:         1,
		ModelDistribution: "",
		UserIds:           "[100]",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		Id:               1,
		UserId:           100,
		Username:         "alice",
		Type:             model.LogTypeConsume,
		ModelName:        "gpt-4o",
		Quota:            20,
		PromptTokens:     10,
		CompletionTokens: 5,
		CreatedAt:        1700000100,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         100,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
		JoinedAt:       0,
		LeftAt:         0,
	}).Error)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", 100).Updates(map[string]any{
		"username":     "alice_ops",
		"display_name": "Alice Zhang",
	}).Error)

	invalidRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=bad&from=1700000000&to=1700003600", nil)
	invalidResponse := decodeEnterpriseAPIResponse(t, invalidRecorder)
	require.False(t, invalidResponse.Success)
	require.Equal(t, "common.invalid_params", invalidResponse.Message)

	missingDeptRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-detail?from=1700000000&to=1700003600", nil)
	missingDeptResponse := decodeEnterpriseAPIResponse(t, missingDeptRecorder)
	require.False(t, missingDeptResponse.Success)
	require.Equal(t, "common.invalid_params", missingDeptResponse.Message)

	rangeRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=1&from=1700003600&to=1700000000", nil)
	rangeResponse := decodeEnterpriseAPIResponse(t, rangeRecorder)
	require.False(t, rangeResponse.Success)
	require.Equal(t, "common.invalid_params", rangeResponse.Message)

	okRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=1&from=1700000000&to=1700003600", nil)
	okResponse := decodeEnterpriseAPIResponse(t, okRecorder)
	require.True(t, okResponse.Success, okResponse.Message)
	require.Contains(t, okRecorder.Body.String(), `"dept_name":"Engineering"`)
	require.Contains(t, okRecorder.Body.String(), `"model_distribution":[]`)
	require.Contains(t, okRecorder.Body.String(), `"user_ranking":[`)
	require.Contains(t, okRecorder.Body.String(), `"recent_logs_entry":`)
	require.Contains(t, okRecorder.Body.String(), `"username_options":["alice_ops"]`)

	var payload dtoenterprise.DepartmentUsageDetailResponse
	require.NoError(t, common.Unmarshal(okResponse.Data, &payload))
	require.Equal(t, int64(1700000000), payload.WindowStart)
	require.Equal(t, int64(1700003600), payload.WindowEnd)
	require.NotNil(t, payload.ModelDistribution)
	require.Empty(t, payload.ModelDistribution)
	require.Len(t, payload.UserRanking, 1)
	require.Equal(t, "alice_ops", payload.UserRanking[0].Username)
	require.Equal(t, "Alice Zhang", payload.UserRanking[0].DisplayName)
	require.Equal(t, int64(15), payload.UserRanking[0].TokenCount)
	require.NotNil(t, payload.Trend)
	require.Len(t, payload.Trend, 1)
	require.Equal(t, "/usage-logs/common", payload.RecentLogsEntry.Path)
	require.Equal(t, "common", payload.RecentLogsEntry.Section)
	require.Equal(t, int64(1700003599), payload.RecentLogsEntry.Filters.EndTimestamp)
	require.Equal(t, "", payload.RecentLogsEntry.Filters.Username)
	require.Equal(t, []string{"alice_ops"}, payload.RecentLogsEntry.Filters.UsernameOptions)
	require.Len(t, payload.RecentLogsEntry.Filters.UserOptions, 1)
	require.Equal(t, 100, payload.RecentLogsEntry.Filters.UserOptions[0].UserId)
	require.Equal(t, "alice_ops", payload.RecentLogsEntry.Filters.UserOptions[0].Username)
	require.Equal(t, "Alice Zhang", payload.RecentLogsEntry.Filters.UserOptions[0].DisplayName)

	var historicalLog model.Log
	require.NoError(t, db.Where("id = ?", 1).First(&historicalLog).Error)
	require.Equal(t, "alice", historicalLog.Username)
}

func TestUsageDetailAPIKeepsUsernameAndUserIDWhenDisplayNameMissing(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	require.NoError(t, db.Model(&model.User{}).Where("id = ?", 101).Update("display_name", "").Error)
	deptID := 1
	require.NoError(t, db.Create(&entmodel.UsageSnapshot{
		TenantId:          0,
		DeptId:            &deptID,
		DeptName:          "Engineering",
		WindowStart:       1700000000,
		WindowEnd:         1700003600,
		RequestCount:      1,
		PromptTokens:      10,
		CompletionTokens:  5,
		Quota:             20,
		UserCount:         1,
		ModelDistribution: "",
		UserIds:           "[101]",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		Id:               2,
		UserId:           101,
		Username:         "bob",
		Type:             model.LogTypeConsume,
		ModelName:        "gpt-4o",
		Quota:            20,
		PromptTokens:     10,
		CompletionTokens: 5,
		CreatedAt:        1700000100,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         101,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
		JoinedAt:       0,
		LeftAt:         0,
	}).Error)

	okRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=1&from=1700000000&to=1700003600", nil)
	okResponse := decodeEnterpriseAPIResponse(t, okRecorder)
	require.True(t, okResponse.Success, okResponse.Message)

	var payload dtoenterprise.DepartmentUsageDetailResponse
	require.NoError(t, common.Unmarshal(okResponse.Data, &payload))
	require.Len(t, payload.UserRanking, 1)
	require.Equal(t, 101, payload.UserRanking[0].UserId)
	require.Equal(t, "bob", payload.UserRanking[0].Username)
	require.Equal(t, "", payload.UserRanking[0].DisplayName)
	require.Len(t, payload.RecentLogsEntry.Filters.UserOptions, 1)
	require.Equal(t, 101, payload.RecentLogsEntry.Filters.UserOptions[0].UserId)
	require.Equal(t, "bob", payload.RecentLogsEntry.Filters.UserOptions[0].Username)
	require.Equal(t, "", payload.RecentLogsEntry.Filters.UserOptions[0].DisplayName)
	require.Equal(t, []string{"bob"}, payload.RecentLogsEntry.Filters.UsernameOptions)
}

func TestUsageDetailAPIValidatesParamsAndNormalizesArrays(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	deptId := 1
	snapshot := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &deptId,
		DeptName:         "Engineering",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     1,
		PromptTokens:     10,
		CompletionTokens: 5,
		Quota:            20,
	}
	require.NoError(t, snapshot.SetModelDistribution(nil))
	require.NoError(t, snapshot.SetUserIds(nil))
	require.NoError(t, db.Create(&snapshot).Error)

	invalidRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=1&from=bad&to=1700003600", nil)
	invalidResponse := decodeEnterpriseAPIResponse(t, invalidRecorder)
	require.False(t, invalidResponse.Success)
	require.Equal(t, "common.invalid_params", invalidResponse.Message)

	missingDeptRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-detail?from=1700000000&to=1700003600", nil)
	missingDeptResponse := decodeEnterpriseAPIResponse(t, missingDeptRecorder)
	require.False(t, missingDeptResponse.Success)
	require.Equal(t, "common.invalid_params", missingDeptResponse.Message)

	okRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-detail?dept_id=1&from=1700000000&to=1700003600", nil)
	okResponse := decodeEnterpriseAPIResponse(t, okRecorder)
	require.True(t, okResponse.Success, okResponse.Message)
	require.Contains(t, okRecorder.Body.String(), `"model_distribution":[]`)
	require.Contains(t, okRecorder.Body.String(), `"user_ranking":[]`)
	require.Contains(t, okRecorder.Body.String(), `"trend":[`)

	var payload dtoenterprise.DepartmentUsageDetailResponse
	require.NoError(t, common.Unmarshal(okResponse.Data, &payload))
	require.Equal(t, "Engineering", payload.DeptName)
	require.NotNil(t, payload.ModelDistribution)
	require.Empty(t, payload.ModelDistribution)
	require.NotNil(t, payload.UserRanking)
	require.Empty(t, payload.UserRanking)
	require.NotNil(t, payload.Trend)
	require.Len(t, payload.Trend, 1)
	require.Equal(t, "/usage-logs/common", payload.RecentLogsEntry.Path)
	require.NotNil(t, payload.RecentLogsEntry.Filters.UsernameOptions)
}

func TestUsageReportConfigAPIValidatesAndPersists(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	invalidRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/usage/reports", dtoenterprise.DepartmentUsageReportConfigRequest{
		Receivers: []string{"bad-email"},
		Frequency: strPtr("daily"),
		RangeType: strPtr("last7d"),
		Enabled:   boolPtr(true),
	})
	invalidResponse := decodeEnterpriseAPIResponse(t, invalidRecorder)
	require.False(t, invalidResponse.Success)
	require.Equal(t, "enterprise.usage.report_invalid_email", invalidResponse.Message)

	saveRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/usage/reports", dtoenterprise.DepartmentUsageReportConfigRequest{
		Receivers: []string{"ops@example.com", "cto@example.com"},
		Frequency: strPtr("weekly"),
		RangeType: strPtr("last30d"),
		Enabled:   boolPtr(true),
	})
	saveResponse := decodeEnterpriseAPIResponse(t, saveRecorder)
	require.True(t, saveResponse.Success, saveResponse.Message)

	var payload dtoenterprise.DepartmentUsageReportConfigResponse
	require.NoError(t, common.Unmarshal(saveResponse.Data, &payload))
	require.Equal(t, []string{"ops@example.com", "cto@example.com"}, payload.Item.Receivers)
	require.Equal(t, "weekly", payload.Item.Frequency)
	require.Equal(t, "last30d", payload.Item.RangeType)
	require.True(t, payload.Item.Enabled)

	getRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/reports", nil)
	getResponse := decodeEnterpriseAPIResponse(t, getRecorder)
	require.True(t, getResponse.Success, getResponse.Message)
	require.Contains(t, getRecorder.Body.String(), `"receivers":["ops@example.com","cto@example.com"]`)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.NotEmpty(t, actions)
	require.Equal(t, entservice.AdminActionUsageReportSet, actions[len(actions)-1].ActionType)
	require.Equal(t, entservice.AdminObjectUsageReportJob, actions[len(actions)-1].ObjectType)
	require.Contains(t, actions[len(actions)-1].Payload, "ops@example.com")
}

func TestUsageSummaryAPIAllowsScopedDepartmentOwnerAndReturnsScopeEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	db := setupEnterpriseBudgetPermissionDB(t)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.Create(&model.User{Id: 201, Username: "owner", Password: "password123", AffCode: "owner-aff"}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       201,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualGrant,
		Effect:       constant.EnterpriseDepartmentRoleEffectAllow,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	deptID := 1
	snapshot := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &deptID,
		DeptName:         "Engineering",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     2,
		PromptTokens:     20,
		CompletionTokens: 10,
		Quota:            40,
	}
	require.NoError(t, snapshot.SetModelDistribution(nil))
	require.NoError(t, snapshot.SetUserIds([]int{201}))
	require.NoError(t, db.Create(&snapshot).Error)

	router.Use(func(c *gin.Context) {
		c.Set("id", 201)
		c.Set("role", common.RoleCommonUser)
		c.Next()
	})
	router.GET("/api/enterprise/usage/department-summary", GetDepartmentUsageSummary)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/usage/department-summary?department_id=1&from=1700000000&to=1700003600&include_descendants=true", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var payload dtoenterprise.DepartmentUsageSummaryResponse
	require.NoError(t, common.Unmarshal(response.Data, &payload))
	require.Equal(t, "Engineering", payload.Scope.DepartmentName)
	require.True(t, payload.Scope.IncludeDescendants)
	require.Equal(t, []int{1}, payload.Scope.DepartmentIds)
}

func strPtr(value string) *string {
	return &value
}
