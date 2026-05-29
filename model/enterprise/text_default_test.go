package enterprise_test

import (
	"reflect"
	"strings"
	"testing"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnterpriseTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	models := []any{
		entmodel.Department{},
		entmodel.DepartmentBudget{},
		entmodel.UserDepartment{},
		entmodel.DepartmentRole{},
		entmodel.AdminAction{},
		entmodel.DingTalkConfig{},
		entmodel.DingTalkIdentity{},
		entmodel.DingTalkSyncTask{},
		entmodel.DingTalkSyncLog{},
		entmodel.DingTalkSyncConflict{},
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

func TestEnterpriseTextFieldApplicationDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	department := entmodel.Department{Name: "Root"}
	require.NoError(t, db.Create(&department).Error)
	require.JSONEq(t, "[]", department.NameHistory)

	action := entmodel.AdminAction{
		ActorId:     100,
		ActionType:  "enterprise.organization.membership.add",
		ObjectType:  "enterprise_department_member",
		ObjectId:    "1:200",
		DiffSummary: "Added department member",
	}
	require.NoError(t, db.Create(&action).Error)
	require.JSONEq(t, "{}", action.Payload)

	config := entmodel.DingTalkConfig{TenantId: 1}
	require.NoError(t, db.Create(&config).Error)
	require.Equal(t, "", config.SyncScope)
}
