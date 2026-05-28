package enterprise

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDepartmentNameHistoryUsesJSONWrapperShape(t *testing.T) {
	department := Department{}
	err := department.SetNameHistory([]DepartmentNameHistoryEntry{
		{Name: "Old name", ChangedAt: 1700000000},
	})
	require.NoError(t, err)
	require.JSONEq(t, `[{"name":"Old name","changed_at":1700000000}]`, department.NameHistory)

	history, err := department.ParsedNameHistory()
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "Old name", history[0].Name)
	require.Equal(t, int64(1700000000), history[0].ChangedAt)
}

func TestMigrateCreatesSQLiteDepartmentTableAndIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&Department{}))
	for _, column := range []string{
		"tenant_id",
		"name",
		"parent_id",
		"status",
		"source_type",
		"external_id",
		"sync_status",
		"sync_error",
		"name_history",
		"deleted_at",
	} {
		require.True(t, db.Migrator().HasColumn(&Department{}, column), column)
	}
	for _, index := range []string{
		"idx_departments_tenant",
		"idx_departments_parent",
		"idx_departments_status",
		"idx_departments_source_external",
	} {
		require.True(t, db.Migrator().HasIndex(&Department{}, index), index)
		require.LessOrEqual(t, len(index), 64)
	}
	require.True(t, db.Migrator().HasConstraint(&Department{}, "chk_departments_parent_not_self"))
}

func TestDepartmentBeforeCreateDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	department := Department{Name: "Root"}
	require.NoError(t, db.Create(&department).Error)
	require.Equal(t, constant.DepartmentStatusEnabled, department.Status)
	require.Equal(t, constant.DepartmentSourceTypeManual, department.SourceType)
	require.JSONEq(t, `[]`, department.NameHistory)
	require.NotZero(t, department.CreatedAt)
	require.NotZero(t, department.UpdatedAt)
}
