package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newMembershipTestService(t *testing.T) (*entservice.DepartmentMembershipService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, entmodel.AutoMigrate(db))

	return entservice.NewDepartmentMembershipService(db), db
}

func seedUserAndDepartments(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Create(&model.User{Id: 100, Username: "alice", Password: "password123", Group: "vip", AffCode: "alice-aff"}).Error)
	require.NoError(t, db.Create(&model.User{Id: 101, Username: "bob", Password: "password123", Group: "default", AffCode: "bob-aff"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 1, TenantId: 0, Name: "Engineering", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 2, TenantId: 0, Name: "Security", Status: constant.EnterpriseDepartmentStatusActive}).Error)
}

func TestListUserDepartmentsSupportsMultipleAndUnassigned(t *testing.T) {
	svc, db := newMembershipTestService(t)
	seedUserAndDepartments(t, db)

	_, err := svc.ReplaceUserDepartments(100, entservice.ReplaceUserDepartmentsInput{
		DepartmentIds:  []int{1, 2},
		ExternalSource: "manual",
	})
	require.NoError(t, err)

	result, err := svc.ListUserDepartments(100, entservice.MembershipQuery{})
	require.NoError(t, err)
	require.False(t, result.IsUnassigned)
	require.Len(t, result.Items, 2)
	require.Equal(t, []int{1, 2}, []int{result.Items[0].DepartmentId, result.Items[1].DepartmentId})

	unassigned, err := svc.ListUserDepartments(101, entservice.MembershipQuery{})
	require.NoError(t, err)
	require.True(t, unassigned.IsUnassigned)
	require.NotNil(t, unassigned.Items)
	require.Empty(t, unassigned.Items)
}

func TestMembershipLifecycleAndDuplicateGuard(t *testing.T) {
	svc, db := newMembershipTestService(t)
	seedUserAndDepartments(t, db)

	_, err := svc.AddDepartmentMember(1, entservice.AddDepartmentMemberInput{UserId: 100, ExternalSource: "manual"})
	require.NoError(t, err)
	_, err = svc.AddDepartmentMember(1, entservice.AddDepartmentMemberInput{UserId: 100, ExternalSource: "manual"})
	require.ErrorIs(t, err, entservice.ErrMembershipAlreadyExists)

	require.NoError(t, svc.DeactivateDepartmentMember(1, 100, entservice.MembershipMutationInput{ExternalSource: "manual"}))
	unassignedAfterDeactivate, err := svc.ListUserDepartments(100, entservice.MembershipQuery{})
	require.NoError(t, err)
	require.True(t, unassignedAfterDeactivate.IsUnassigned)
	require.Empty(t, unassignedAfterDeactivate.Items)

	inactive, err := svc.ListDepartmentMembers(1, entservice.MembershipQuery{Status: intPtr(constant.EnterpriseMembershipStatusInactive)})
	require.NoError(t, err)
	require.Len(t, inactive.Items, 1)
	require.Equal(t, constant.EnterpriseMembershipStatusInactive, inactive.Items[0].Status)

	_, err = svc.RestoreDepartmentMember(1, 100, entservice.MembershipMutationInput{ExternalSource: "manual"})
	require.NoError(t, err)
	active, err := svc.ListDepartmentMembers(1, entservice.MembershipQuery{})
	require.NoError(t, err)
	require.Len(t, active.Items, 1)
	require.Equal(t, constant.EnterpriseMembershipStatusActive, active.Items[0].Status)

	var user model.User
	require.NoError(t, db.First(&user, 100).Error)
	require.Equal(t, "vip", user.Group)
}

func TestReplaceUserDepartmentsRejectsDuplicateDepartmentIds(t *testing.T) {
	svc, db := newMembershipTestService(t)
	seedUserAndDepartments(t, db)

	_, err := svc.ReplaceUserDepartments(100, entservice.ReplaceUserDepartmentsInput{
		DepartmentIds:  []int{1, 1},
		ExternalSource: "manual",
	})
	require.ErrorIs(t, err, entservice.ErrDuplicateDepartment)
}

func intPtr(v int) *int {
	return &v
}
