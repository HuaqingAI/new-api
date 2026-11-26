package agentplatform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGenerateResourceIDUsesStableOpaquePrefix(t *testing.T) {
	first, err := GenerateResourceID()
	require.NoError(t, err)
	second, err := GenerateResourceID()
	require.NoError(t, err)

	require.NotEqual(t, first, second)
	require.True(t, strings.HasPrefix(first, "res_"))
	require.True(t, strings.HasPrefix(second, "res_"))
	require.Len(t, first, 36)
	require.Len(t, second, 36)
}

func TestResourceBeforeCreateAppliesDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	resource := Resource{
		ResourceType: ResourceTypeSkill,
		DisplayName:  "First Skill",
		OwnerUserId:  100,
	}
	require.NoError(t, db.Create(&resource).Error)
	require.Equal(t, ResourceStatusDraft, resource.Status)
	require.Equal(t, "", resource.LatestVersion)
	require.True(t, strings.HasPrefix(resource.ResourceId, "res_"))
	require.NotZero(t, resource.CreatedAt)
	require.NotZero(t, resource.UpdatedAt)
}

func TestResourceRejectsInvalidResourceType(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	resource := Resource{
		ResourceType: "workflow",
		DisplayName:  "Invalid",
		OwnerUserId:  100,
	}
	err = db.Create(&resource).Error
	require.ErrorIs(t, err, ErrInvalidResourceType)
}

func TestResourceRejectsMissingGovernanceFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	resource := Resource{
		ResourceType: ResourceTypeSkill,
		DisplayName:  "   ",
		OwnerUserId:  0,
	}
	err = db.Create(&resource).Error
	require.ErrorIs(t, err, ErrInvalidResourceBody)
}

func TestResourceIdentityIsImmutableAfterCreation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	resource := Resource{
		ResourceType: ResourceTypeSkill,
		DisplayName:  "Immutable Skill",
		OwnerUserId:  100,
	}
	require.NoError(t, db.Create(&resource).Error)

	resource.ResourceId = "res_overwrite_attempt"
	err = db.Save(&resource).Error
	require.ErrorIs(t, err, ErrImmutableResourceID)

	var persisted Resource
	require.NoError(t, db.First(&persisted, resource.Id).Error)
	require.NotEqual(t, "res_overwrite_attempt", persisted.ResourceId)

	resource = persisted
	resource.ResourceType = ResourceTypeAgent
	err = db.Save(&resource).Error
	require.ErrorIs(t, err, ErrImmutableResourceType)
}

func TestResourceTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	modelType := reflect.TypeOf(Resource{})
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		gormTag := field.Tag.Get("gorm")
		if strings.Contains(gormTag, "type:text") {
			require.NotContainsf(t, gormTag, "default:", "%s.%s text field must not declare a DB default", modelType.Name(), field.Name)
		}
	}
}

func TestMigrateCreatesResourceTableAndIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&Resource{}))
	for _, column := range []string{
		"resource_id",
		"resource_type",
		"display_name",
		"owner_user_id",
		"status",
		"latest_version",
		"tenant_id",
		"created_at",
		"updated_at",
	} {
		require.True(t, db.Migrator().HasColumn(&Resource{}, column), column)
	}
	for _, index := range []string{
		"idx_ap_resource_id",
		"idx_ap_resource_type",
		"idx_ap_resource_owner",
		"idx_ap_resource_status",
		"idx_ap_resource_tenant",
	} {
		require.True(t, db.Migrator().HasIndex(&Resource{}, index), index)
		require.LessOrEqual(t, len(index), 64)
	}
}
