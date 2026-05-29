package enterprise

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDepartmentBudgetBeforeCreateDefaultsBalance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	budget := DepartmentBudget{
		TenantId:     1,
		DepartmentId: 10,
		Type:         DepartmentBudgetTypeBalance,
		TotalQuota:   200,
	}
	require.NoError(t, db.Create(&budget).Error)
	require.Equal(t, DepartmentBudgetStatusActive, budget.Status)
	require.Equal(t, int64(200), budget.Remaining)
	require.Equal(t, "never", budget.CycleType)
	require.NotZero(t, budget.CreatedAt)
	require.NotZero(t, budget.UpdatedAt)
}

func TestDepartmentBudgetBeforeCreateDefaultsSubscription(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	budget := DepartmentBudget{
		TenantId:       1,
		DepartmentId:   10,
		Type:           DepartmentBudgetTypeSubscription,
		CycleQuota:     300,
		CycleType:      "weekly",
		CycleStartedAt: 1700000000,
	}
	require.NoError(t, db.Create(&budget).Error)
	require.Equal(t, int64(300), budget.Remaining)
	require.Equal(t, "weekly", budget.CycleType)
}

func TestDepartmentBudgetMigrationCreatesExpectedColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&DepartmentBudget{}))
	for _, column := range []string{
		"tenant_id",
		"department_id",
		"type",
		"status",
		"total_quota",
		"remaining",
		"cycle_quota",
		"cycle_type",
		"cycle_started_at",
		"custom_seconds",
		"expires_at",
		"parent_status",
	} {
		require.True(t, db.Migrator().HasColumn(&DepartmentBudget{}, column), column)
	}
}

func TestDepartmentBudgetTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	modelType := reflect.TypeOf(DepartmentBudget{})
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		gormTag := field.Tag.Get("gorm")
		if strings.Contains(gormTag, "type:text") {
			require.NotContainsf(t, gormTag, "default:", "%s.%s text field must not declare a DB default", modelType.Name(), field.Name)
		}
	}
}
