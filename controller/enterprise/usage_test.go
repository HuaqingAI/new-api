package enterprise

import (
	"net/http"
	"testing"

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
}

