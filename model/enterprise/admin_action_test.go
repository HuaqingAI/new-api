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

func TestAdminActionBeforeCreateDefaultsPayload(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	action := entmodel.AdminAction{
		ActorId:     100,
		ActionType:  "enterprise.organization.membership.add",
		ObjectType:  "enterprise_department_member",
		ObjectId:    "1:200",
		DiffSummary: "Added department member",
	}

	require.NoError(t, db.Create(&action).Error)
	require.Equal(t, "{}", action.Payload)

	var saved entmodel.AdminAction
	require.NoError(t, db.First(&saved, action.ActionId).Error)
	require.Equal(t, "{}", saved.Payload)
}
