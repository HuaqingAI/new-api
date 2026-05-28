package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newPermissionTestService(t *testing.T) (*entservice.PermissionService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	return entservice.NewPermissionService(db), db
}

func TestPermissionServiceExpandsDepartmentAdminScopeToDescendants(t *testing.T) {
	svc, db := newPermissionTestService(t)
	parentId := 1
	childId := 2
	require.NoError(t, db.Create(&[]entmodel.Department{
		{Id: 1, TenantId: 0, Name: "Company"},
		{Id: 2, TenantId: 0, Name: "Engineering", ParentId: &parentId},
		{Id: 3, TenantId: 0, Name: "Platform", ParentId: &childId},
		{Id: 4, TenantId: 0, Name: "Finance"},
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       100,
		DepartmentId: 2,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	ids, err := svc.ListManageableDepartmentIds(100, 0)

	require.NoError(t, err)
	require.Equal(t, []int{2, 3}, ids)

	canManageChild, err := svc.CanManageDepartment(100, 0, 3)
	require.NoError(t, err)
	require.True(t, canManageChild)

	canManageSibling, err := svc.CanManageDepartment(100, 0, 4)
	require.NoError(t, err)
	require.False(t, canManageSibling)
}

func TestPermissionServiceIgnoresInactiveRolesAndMissingDepartments(t *testing.T) {
	svc, db := newPermissionTestService(t)
	require.NoError(t, db.Create(&entmodel.Department{Id: 1, TenantId: 0, Name: "Company"}).Error)
	require.NoError(t, db.Create(&[]entmodel.DepartmentRole{
		{
			TenantId:     0,
			UserId:       100,
			DepartmentId: 1,
			Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
			Status:       constant.EnterpriseDepartmentRoleStatusInactive,
		},
		{
			TenantId:     0,
			UserId:       100,
			DepartmentId: 999,
			Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
			Status:       constant.EnterpriseDepartmentRoleStatusActive,
		},
	}).Error)

	ids, err := svc.ListManageableDepartmentIds(100, 0)

	require.NoError(t, err)
	require.NotNil(t, ids)
	require.Empty(t, ids)
}
