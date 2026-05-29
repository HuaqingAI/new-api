package enterprise

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUsageReportJobJSONWrappersNormalizeEmptyValues(t *testing.T) {
	job := UsageReportJob{}

	require.NoError(t, job.SetReceivers([]string{"ops@example.com", "cto@example.com"}))
	require.JSONEq(t, `["ops@example.com","cto@example.com"]`, job.Receivers)

	receivers, err := job.ParsedReceivers()
	require.NoError(t, err)
	require.Equal(t, []string{"ops@example.com", "cto@example.com"}, receivers)

	require.NoError(t, job.SetLastSnapshot(&UsageReportSnapshot{
		WindowStart:     100,
		WindowEnd:       200,
		DepartmentCount: 2,
		TopDepartments: []UsageReportTopDepartment{
			{DeptName: "Engineering", RequestCount: 9},
		},
	}))
	snapshot, err := job.ParsedLastSnapshot()
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, int64(100), snapshot.WindowStart)
	require.Len(t, snapshot.TopDepartments, 1)

	require.NoError(t, job.SetReceivers(nil))
	require.JSONEq(t, `[]`, job.Receivers)
	require.NoError(t, job.SetLastSnapshot(nil))
	require.JSONEq(t, `{}`, job.LastSnapshot)
	snapshot, err = job.ParsedLastSnapshot()
	require.NoError(t, err)
	require.Nil(t, snapshot)
}

func TestUsageReportJobTextFieldsDoNotDeclareDBDefaults(t *testing.T) {
	models := []any{
		UsageSnapshot{},
		UsageReportJob{},
	}

	for _, item := range models {
		modelType := reflect.TypeOf(item)
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			gormTag := field.Tag.Get("gorm")
			if strings.Contains(gormTag, "type:text") {
				require.NotContainsf(t, gormTag, "default:", "%s.%s text field must not declare a DB default", modelType.Name(), field.Name)
			}
		}
	}
}

func TestMigrateCreatesUsageReportJobTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&UsageReportJob{}))
	for _, column := range []string{
		"tenant_id",
		"receivers",
		"frequency",
		"range_type",
		"enabled",
		"status",
		"last_run_at",
		"next_run_at",
		"last_success_at",
		"error_reason",
		"last_snapshot",
	} {
		require.True(t, db.Migrator().HasColumn(&UsageReportJob{}, column), column)
	}
}
