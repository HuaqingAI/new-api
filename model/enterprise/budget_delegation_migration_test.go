package enterprise_test

import (
	"testing"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnterpriseBudgetDelegationMigratesTableAndColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	require.True(t, db.Migrator().HasTable(&entmodel.BudgetDelegation{}))
	for _, column := range []string{
		"tenant_id",
		"source_department_id",
		"source_budget_id",
		"target_department_id",
		"target_budget_id",
		"actor_id",
		"committed_quota",
		"budget_type_snapshot",
		"cycle_type_snapshot",
		"status",
		"before_source_budget_snapshot",
		"after_source_budget_snapshot",
		"before_target_budget_snapshot",
		"after_target_budget_snapshot",
		"superseded_by_id",
		"processed_at",
		"reason",
	} {
		require.Truef(t, db.Migrator().HasColumn(&entmodel.BudgetDelegation{}, column), "missing column %s", column)
	}
}
