package enterprise

import (
	"net/http"
	"testing"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestGovernanceTimelineAPIReturnsEmptyArray(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/governance/timeline?department_id=1&page=1&page_size=20", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	require.JSONEq(t, `{"items":[],"total":0,"page":1,"page_size":20}`, string(response.Data))
}

func TestGovernanceNotificationListAndResendAPI(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	delivery := entmodel.GovernanceNotificationDelivery{
		TenantId:        0,
		SourceType:      entservice.GovernanceSourceQuotaRequest,
		SourceId:        42,
		TraceId:         "quota_request:42",
		ActionType:      entservice.GovernanceActionQuotaRequestApproved,
		RecipientUserId: 100,
		RecipientKind:   entmodel.GovernanceNotificationRecipientRequester,
		ChannelType:     entmodel.GovernanceNotificationChannelDingTalkRobot,
		Status:          entmodel.GovernanceNotificationStatusFinalFailed,
		AttemptCount:    4,
		MaxAttempts:     4,
		FinalFailedAt:   1717117200,
		ErrorReason:     "webhook request failed",
		DedupeKey:       "quota_request:42:requester",
		CreatedAt:       1717117200,
		UpdatedAt:       1717117201,
	}
	require.NoError(t, delivery.SetTracePayload(&entmodel.GovernanceNotificationTracePayload{
		TraceId:        "quota_request:42",
		SourceType:     entservice.GovernanceSourceQuotaRequest,
		SourceId:       42,
		ActionType:     entservice.GovernanceActionQuotaRequestApproved,
		DepartmentId:   1,
		DepartmentName: "Engineering",
		Status:         "fulfilled",
	}))
	require.NoError(t, db.Create(&delivery).Error)

	list := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/governance/notifications?department_id=1&status=final_failed", nil)
	listResponse := decodeEnterpriseAPIResponse(t, list)
	require.True(t, listResponse.Success, listResponse.Message)
	listPayload := decodeEnterpriseData[dtoenterprise.GovernanceNotificationResponse](t, listResponse)
	require.Len(t, listPayload.Items, 1)
	require.Equal(t, "webhook request failed", listPayload.Items[0].ErrorReason)
	require.Equal(t, "quota_request:42", listPayload.Items[0].TraceId)

	resend := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/governance/notifications/1/resend", nil)
	resendResponse := decodeEnterpriseAPIResponse(t, resend)
	require.True(t, resendResponse.Success, resendResponse.Message)
	resendPayload := decodeEnterpriseData[dtoenterprise.GovernanceNotificationResendResponse](t, resendResponse)
	require.True(t, resendPayload.Created)
	require.Equal(t, entmodel.GovernanceNotificationStatusPending, resendPayload.Item.Status)
	require.NotNil(t, resendPayload.Item.ManualParentId)
	require.Equal(t, 1, *resendPayload.Item.ManualParentId)
}
