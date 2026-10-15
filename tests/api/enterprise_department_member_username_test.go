package api_test

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestEnterpriseDepartmentMemberUsernameRenameAPIUpdatesCurrentViewsAndPreservesLogSnapshot(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.Log{}))
	require.NoError(t, fixture.db.Create(&model.User{
		Id:          2001,
		Username:    "alice",
		DisplayName: "Alice",
		Password:    "password123",
		Group:       "vip",
		AffCode:     "alice-api",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.UserDepartment{
		TenantId:       0,
		UserId:         2001,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
		JoinedAt:       1700000000,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&model.Log{
		UserId:    2001,
		Username:  "alice",
		CreatedAt: 1700000100,
		Type:      model.LogTypeConsume,
		ModelName: "gpt-4o-mini",
		Content:   "historical request",
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	rename := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/departments/1/members/2001/username", cookies, dtoenterprise.RenameDepartmentMemberRequest{
		NewUsername: "alice_ops",
	})
	renamePayload := decodeDepartmentMembersAPIResponse(t, rename)
	require.True(t, renamePayload.Success, renamePayload.Message)
	require.Contains(t, string(renamePayload.Data), `"username":"alice_ops"`)

	members := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/1/members", cookies)
	membersPayload := decodeDepartmentMembersAPIResponse(t, members)
	require.True(t, membersPayload.Success, membersPayload.Message)
	require.Contains(t, string(membersPayload.Data), `"username":"alice_ops"`)
	require.NotContains(t, string(membersPayload.Data), `"username":"alice"`)

	var user model.User
	require.NoError(t, fixture.db.Where("id = ?", 2001).First(&user).Error)
	require.Equal(t, "alice_ops", user.Username)

	var historicalLog model.Log
	require.NoError(t, fixture.db.Where("user_id = ?", 2001).First(&historicalLog).Error)
	require.Equal(t, "alice", historicalLog.Username)

	var action modelenterprise.AdminAction
	require.NoError(t, fixture.db.Where("action_type = ?", "enterprise.organization.membership.rename").First(&action).Error)
	require.Equal(t, "enterprise_department_member", action.ObjectType)
	require.Equal(t, "1:2001", action.ObjectId)
	require.Contains(t, action.Payload, `"previous_username":"alice"`)
	require.Contains(t, action.Payload, `"new_username":"alice_ops"`)
}

func TestEnterpriseDepartmentMemberUsernameRenameAPIRejectsInvalidDuplicateAndOutOfDepartmentTargets(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}))
	require.NoError(t, fixture.db.Create(&[]model.User{
		{
			Id:       2001,
			Username: "alice",
			Password: "password123",
			Group:    "vip",
			AffCode:  "alice-api",
		},
		{
			Id:       2002,
			Username: "bob",
			Password: "password123",
			Group:    "vip",
			AffCode:  "bob-api",
		},
		{
			Id:       2003,
			Username: "charlie",
			Password: "password123",
			Group:    "vip",
			AffCode:  "charlie-api",
		},
	}).Error)
	require.NoError(t, fixture.db.Create(&[]modelenterprise.Department{
		{
			Id:          1,
			TenantId:    0,
			Name:        "Engineering",
			Status:      constant.DepartmentStatusEnabled,
			SourceType:  constant.DepartmentSourceTypeManual,
			SyncStatus:  constant.DepartmentSyncStatusOK,
			NameHistory: "[]",
		},
		{
			Id:          2,
			TenantId:    0,
			Name:        "Finance",
			Status:      constant.DepartmentStatusEnabled,
			SourceType:  constant.DepartmentSourceTypeManual,
			SyncStatus:  constant.DepartmentSyncStatusOK,
			NameHistory: "[]",
		},
	}).Error)
	require.NoError(t, fixture.db.Create(&[]modelenterprise.UserDepartment{
		{
			TenantId:     0,
			UserId:       2001,
			DepartmentId: 1,
			Status:       constant.EnterpriseMembershipStatusActive,
		},
		{
			TenantId:     0,
			UserId:       2003,
			DepartmentId: 2,
			Status:       constant.EnterpriseMembershipStatusActive,
		},
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	invalid := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/departments/1/members/2001/username", cookies, dtoenterprise.RenameDepartmentMemberRequest{
		NewUsername: "Alice Ops",
	})
	invalidPayload := decodeDepartmentMembersAPIResponse(t, invalid)
	require.False(t, invalidPayload.Success)
	require.Equal(t, "enterprise.organization.username_invalid", invalidPayload.Message)

	duplicate := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/departments/1/members/2001/username", cookies, dtoenterprise.RenameDepartmentMemberRequest{
		NewUsername: "bob",
	})
	duplicatePayload := decodeDepartmentMembersAPIResponse(t, duplicate)
	require.False(t, duplicatePayload.Success)
	require.Equal(t, "enterprise.organization.username_exists", duplicatePayload.Message)

	outOfDepartment := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/departments/1/members/2003/username", cookies, dtoenterprise.RenameDepartmentMemberRequest{
		NewUsername: "charlie_ops",
	})
	outOfDepartmentPayload := decodeDepartmentMembersAPIResponse(t, outOfDepartment)
	require.False(t, outOfDepartmentPayload.Success)
	require.Equal(t, "enterprise.organization.membership_not_found", outOfDepartmentPayload.Message)

	var alice model.User
	require.NoError(t, fixture.db.Where("id = ?", 2001).First(&alice).Error)
	require.Equal(t, "alice", alice.Username)
	var charlie model.User
	require.NoError(t, fixture.db.Where("id = ?", 2003).First(&charlie).Error)
	require.Equal(t, "charlie", charlie.Username)
}
