package enterprise

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestDepartmentOwnerAPIListsFactsEffectiveOwnersAndFallback(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:       0,
		UserId:         100,
		DepartmentId:   1,
		Role:           constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:         constant.EnterpriseDepartmentRoleSourceDingTalkOwner,
		Effect:         constant.EnterpriseDepartmentRoleEffectAllow,
		ExternalSource: constant.EnterpriseExternalSourceDingTalk,
		Status:         constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       100,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualDenyOverride,
		Effect:       constant.EnterpriseDepartmentRoleEffectDeny,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/1/owners", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)

	require.True(t, response.Success, response.Message)
	data := decodeEnterpriseData[dtoenterprise.DepartmentOwnersResponse](t, response)
	require.Len(t, data.Facts, 2)
	require.Empty(t, data.EffectiveOwners)
	require.Equal(t, 0, data.OwnerCount)
	require.Equal(t, "admin", data.Fallback)
	require.Contains(t, recorder.Body.String(), "manual_deny_override")
	require.Contains(t, recorder.Body.String(), "dingtalk_synced_owner")
}

func TestDepartmentOwnerAPIGrantDenyAndRevokeWritesAudit(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	grantRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/owners/grants", dtoenterprise.DepartmentOwnerMutationRequest{
		UserId: 100,
	})
	grantResponse := decodeEnterpriseAPIResponse(t, grantRecorder)
	require.True(t, grantResponse.Success, grantResponse.Message)
	grant := decodeEnterpriseData[dtoenterprise.DepartmentRoleItem](t, grantResponse)
	require.Equal(t, constant.EnterpriseDepartmentRoleSourceManualGrant, grant.Source)

	denyRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/owners/denies", dtoenterprise.DepartmentOwnerMutationRequest{
		UserId: 100,
	})
	denyResponse := decodeEnterpriseAPIResponse(t, denyRecorder)
	require.True(t, denyResponse.Success, denyResponse.Message)
	deny := decodeEnterpriseData[dtoenterprise.DepartmentRoleItem](t, denyResponse)
	require.Equal(t, constant.EnterpriseDepartmentRoleSourceManualDenyOverride, deny.Source)
	require.Equal(t, constant.EnterpriseDepartmentRoleEffectDeny, deny.Effect)

	revokeRecorder := performEnterpriseRequest(t, router, http.MethodDelete, "/api/enterprise/departments/1/owners/denies/100", dtoenterprise.DepartmentOwnerMutationRequest{
		UserId: 100,
	})
	revokeResponse := decodeEnterpriseAPIResponse(t, revokeRecorder)
	require.True(t, revokeResponse.Success, revokeResponse.Message)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 3)
	require.Equal(t, entservice.AdminActionDeptOwnerManualGrant, actions[0].ActionType)
	require.Equal(t, entservice.AdminActionDeptOwnerManualDeny, actions[1].ActionType)
	require.Equal(t, entservice.AdminActionDeptOwnerManualDenyRevoke, actions[2].ActionType)
	require.Contains(t, actions[1].Payload, `"source":"manual_deny_override"`)
	require.Contains(t, actions[1].Payload, `"actor_id":999`)
}

func TestDepartmentAdminCompatibilityRoutesUseManualGrantSemantics(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	grantRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/admins", dtoenterprise.DepartmentAdminRoleRequest{
		UserId: 101,
	})
	grantResponse := decodeEnterpriseAPIResponse(t, grantRecorder)
	require.True(t, grantResponse.Success, grantResponse.Message)
	grant := decodeEnterpriseData[dtoenterprise.DepartmentRoleItem](t, grantResponse)
	require.Equal(t, constant.EnterpriseDepartmentRoleSourceManualGrant, grant.Source)
	require.Equal(t, constant.EnterpriseDepartmentRoleEffectAllow, grant.Effect)

	revokeRecorder := performEnterpriseRequest(t, router, http.MethodDelete, "/api/enterprise/departments/1/admins/101", nil)
	revokeResponse := decodeEnterpriseAPIResponse(t, revokeRecorder)
	require.True(t, revokeResponse.Success, revokeResponse.Message)

	var role entmodel.DepartmentRole
	require.NoError(t, db.Where("user_id = ? AND department_id = ? AND source = ?", 101, 1, constant.EnterpriseDepartmentRoleSourceManualGrant).First(&role).Error)
	require.Equal(t, constant.EnterpriseDepartmentRoleStatusInactive, role.Status)
}
