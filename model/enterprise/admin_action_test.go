package enterprise_test

import (
	"testing"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAdminActionMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, entmodel.AutoMigrate(db))
	require.True(t, db.Migrator().HasTable("enterprise_admin_actions"))
	for _, column := range []string{
		"action_id",
		"tenant_id",
		"actor_id",
		"action_type",
		"object_type",
		"object_id",
		"created_at",
		"diff_summary",
		"payload",
	} {
		require.True(t, db.Migrator().HasColumn(&entmodel.AdminAction{}, column), column)
	}
	require.False(t, db.Migrator().HasColumn(&entmodel.AdminAction{}, "secret"))
	require.False(t, db.Migrator().HasColumn(&entmodel.AdminAction{}, "token"))
}
