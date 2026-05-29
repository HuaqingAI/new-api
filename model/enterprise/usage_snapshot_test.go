package enterprise

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUsageSnapshotModelDistributionUsesJSONWrapperShape(t *testing.T) {
	snapshot := UsageSnapshot{}

	require.NoError(t, snapshot.SetModelDistribution([]UsageSnapshotModelStat{
		{
			ModelName:        "gpt-4o",
			RequestCount:     2,
			PromptTokens:     100,
			CompletionTokens: 40,
			Quota:            1200,
		},
	}))
	require.JSONEq(t, `[{"model_name":"gpt-4o","request_count":2,"prompt_tokens":100,"completion_tokens":40,"quota":1200}]`, snapshot.ModelDistribution)

	stats, err := snapshot.ParsedModelDistribution()
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Equal(t, "gpt-4o", stats[0].ModelName)
	require.Equal(t, int64(2), stats[0].RequestCount)
}

func TestUsageSnapshotModelDistributionNormalizesEmptyValues(t *testing.T) {
	snapshot := UsageSnapshot{}

	stats, err := snapshot.ParsedModelDistribution()
	require.NoError(t, err)
	require.NotNil(t, stats)
	require.Empty(t, stats)

	require.NoError(t, snapshot.SetModelDistribution(nil))
	require.JSONEq(t, `[]`, snapshot.ModelDistribution)
}

func TestEnterpriseTextFieldsIncludeUsageSnapshotWithoutDatabaseDefaults(t *testing.T) {
	models := []any{
		UsageSnapshot{},
	}

	for _, model := range models {
		modelType := reflect.TypeOf(model)
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			gormTag := field.Tag.Get("gorm")
			if strings.Contains(gormTag, "type:text") {
				require.NotContainsf(t, gormTag, "default:", "%s.%s text field must not declare a DB default", modelType.Name(), field.Name)
			}
		}
	}
}

func TestMigrateCreatesUsageSnapshotTableAndIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&UsageSnapshot{}))
	for _, column := range []string{
		"tenant_id",
		"dept_id",
		"dept_name",
		"window_start",
		"window_end",
		"request_count",
		"prompt_tokens",
		"completion_tokens",
		"quota",
		"user_count",
		"model_distribution",
		"created_at",
		"updated_at",
	} {
		require.True(t, db.Migrator().HasColumn(&UsageSnapshot{}, column), column)
	}
	for _, index := range []string{
		"idx_usage_snapshots_tenant_window",
		"idx_usage_snapshots_tenant_dept_window",
		"uq_usage_snapshots_bucket",
	} {
		require.True(t, db.Migrator().HasIndex(&UsageSnapshot{}, index), index)
	}
}

