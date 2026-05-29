package enterprise

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestUsageSummaryAPIValidatesTimeRangeAndNormalizesArrays(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/usage/department-summary", GetDepartmentUsageSummary)

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

func TestUsageDetailAPIValidatesTimeRangeAndNormalizesArrays(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/usage/department-detail", GetDepartmentUsageDetail)

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
	require.Contains(t, okRecorder.Body.String(), `"username_options":["alice"]`)

	var payload dtoenterprise.DepartmentUsageDetailResponse
	require.NoError(t, common.Unmarshal(okResponse.Data, &payload))
	require.Equal(t, int64(1700000000), payload.WindowStart)
	require.Equal(t, int64(1700003600), payload.WindowEnd)
	require.NotNil(t, payload.ModelDistribution)
	require.Empty(t, payload.ModelDistribution)
	require.Len(t, payload.UserRanking, 1)
	require.Equal(t, int64(15), payload.UserRanking[0].TokenCount)
	require.NotNil(t, payload.Trend)
	require.Len(t, payload.Trend, 1)
	require.Equal(t, "/usage-logs/common", payload.RecentLogsEntry.Path)
	require.Equal(t, "common", payload.RecentLogsEntry.Section)
	require.Equal(t, int64(1700003599), payload.RecentLogsEntry.Filters.EndTimestamp)
	require.Equal(t, []string{"alice"}, payload.RecentLogsEntry.Filters.UsernameOptions)
}

func TestUsageDetailAPIValidatesParamsAndNormalizesArrays(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/usage/department-detail", GetDepartmentUsageDetail)

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
