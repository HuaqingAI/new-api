package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserDepartmentMigrationAndConstraints(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, entmodel.AutoMigrate(db))
	require.True(t, db.Migrator().HasTable("enterprise_user_departments"))
	require.True(t, db.Migrator().HasColumn(&entmodel.UserDepartment{}, "tenant_id"))
	require.False(t, db.Migrator().HasColumn(&entmodel.UserDepartment{}, "is_primary"))
	require.False(t, db.Migrator().HasColumn(&entmodel.UserDepartment{}, "primary_department_id"))
	require.False(t, db.Migrator().HasColumn(&entmodel.UserDepartment{}, "group"))

	first := entmodel.UserDepartment{
		TenantId:       0,
		UserId:         10,
		DepartmentId:   20,
		ExternalSource: "manual",
		Status:         constant.EnterpriseMembershipStatusActive,
	}
	require.NoError(t, db.Create(&first).Error)

	duplicate := first
	duplicate.Id = 0
	require.Error(t, db.Create(&duplicate).Error)
}
