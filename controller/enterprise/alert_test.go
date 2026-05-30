package enterprise

import (
	"net/http"
	"testing"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
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
