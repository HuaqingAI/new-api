package agentplatform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestExposureTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	modelType := reflect.TypeOf(Exposure{})
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		gormTag := field.Tag.Get("gorm")
		if strings.Contains(gormTag, "type:text") {
			require.NotContainsf(t, gormTag, "default:", "%s.%s text field must not declare a DB default", modelType.Name(), field.Name)
		}
	}
}

func TestExposureModelSeparatesVisibilityAndCallableState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	resource := Resource{ResourceType: ResourceTypeSkill, DisplayName: "Skill", OwnerUserId: 1}
	require.NoError(t, db.Create(&resource).Error)
	version := ResourceVersion{
		ResourceId:      resource.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          ResourceStatusDraft,
		CreatedBy:       1,
	}
	require.NoError(t, db.Create(&version).Error)

	exposure := Exposure{
		ResourceId:          resource.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           "demo-client",
		ClientScope:         "placeholder",
		VisibilityState:     ExposureVisibilityVisible,
		CallableState:       ExposureCallableDisabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-001",
	}
	require.NoError(t, db.Create(&exposure).Error)

	var persisted Exposure
	require.NoError(t, db.First(&persisted, exposure.Id).Error)
	require.Equal(t, ExposureVisibilityVisible, persisted.VisibilityState)
	require.Equal(t, ExposureCallableDisabled, persisted.CallableState)
}
