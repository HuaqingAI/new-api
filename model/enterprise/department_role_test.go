package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDepartmentRoleMigrationAndDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, entmodel.AutoMigrate(db))
	require.True(t, db.Migrator().HasTable("enterprise_department_roles"))
	for _, column := range []string{
		"tenant_id",
		"user_id",
		"department_id",
		"role",
		"status",
		"created_at",
		"updated_at",
	} {
		require.True(t, db.Migrator().HasColumn(&entmodel.DepartmentRole{}, column), column)
	}
	require.False(t, db.Migrator().HasColumn(&entmodel.DepartmentRole{}, "group"))
	require.True(t, db.Migrator().HasIndex(&entmodel.DepartmentRole{}, "uq_ent_dept_roles_role"))

	role := entmodel.DepartmentRole{UserId: 100, DepartmentId: 20}
	require.NoError(t, db.Create(&role).Error)
	require.Equal(t, constant.EnterpriseDepartmentRoleDeptAdmin, role.Role)
	require.Equal(t, constant.EnterpriseDepartmentRoleStatusActive, role.Status)
	require.NotZero(t, role.CreatedAt)
	require.NotZero(t, role.UpdatedAt)
}
