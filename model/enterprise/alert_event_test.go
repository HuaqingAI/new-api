package enterprise

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type legacyAlertEvent struct {
	Id                 int    `gorm:"primaryKey"`
	TenantId           int    `gorm:"type:int;not null;default:0"`
	UserId             int    `gorm:"type:int;not null;default:0"`
	Username           string `gorm:"type:text;not null;default:''"`
	RequestId          string `gorm:"type:text;not null;default:''"`
	ModelName          string `gorm:"type:text;not null;default:''"`
	RiskType           string `gorm:"type:text;not null;default:''"`
	ActionResult       string `gorm:"type:text;not null;default:''"`
	DepartmentSnapshot string `gorm:"type:text;not null"`
	Summary            string `gorm:"type:text;not null"`
	CreatedAt          int64  `gorm:"type:bigint;not null;default:0"`
	UpdatedAt          int64  `gorm:"type:bigint;not null;default:0"`
}

func (legacyAlertEvent) TableName() string {
	return "enterprise_alert_events"
}

func TestAlertEventJSONWrappersNormalizeEmptyValues(t *testing.T) {
	event := AlertEvent{}

	require.NoError(t, event.SetDepartmentSnapshot([]AlertEventDepartmentSnapshot{
		{
			DepartmentId:   11,
			DepartmentName: "Engineering",
			ExternalSource: "manual",
			Status:         1,
		},
		{
			DepartmentId:   22,
			DepartmentName: "Security",
			ExternalSource: "manual",
			Status:         1,
		},
	}))
	require.JSONEq(t, `[{"department_id":11,"department_name":"Engineering","external_source":"manual","status":1},{"department_id":22,"department_name":"Security","external_source":"manual","status":1}]`, event.DepartmentSnapshot)

	snapshot, err := event.ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot, 2)
	require.Equal(t, "Engineering", snapshot[0].DepartmentName)

	require.NoError(t, event.SetDepartmentSnapshot(nil))
	require.JSONEq(t, `[]`, event.DepartmentSnapshot)

	snapshot, err = event.ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Empty(t, snapshot)
}

func TestAlertEventTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	models := []any{
		AlertEvent{},
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

func TestMigrateCreatesAlertEventTableAndIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&AlertEvent{}))
	for _, column := range []string{
		"id",
		"tenant_id",
		"user_id",
		"username",
		"request_id",
		"model_name",
		"risk_type",
		"action_result",
		"department_snapshot",
		"department_tokens",
		"summary",
		"created_at",
		"updated_at",
	} {
		require.True(t, db.Migrator().HasColumn(&AlertEvent{}, column), column)
	}
	for _, index := range []string{
		"idx_alert_events_tenant_created",
		"idx_alert_events_request_id",
		"idx_alert_events_user_id",
		"idx_alert_events_risk_type",
	} {
		require.True(t, db.Migrator().HasIndex(&AlertEvent{}, index), index)
	}
}

func TestAlertEventBeforeCreateAppliesApplicationDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	event := AlertEvent{
		UserId:       10,
		Username:     "alice",
		RequestId:    "req-1",
		ModelName:    "gpt-4o-mini",
		RiskType:     "sensitive_words",
		ActionResult: "blocked",
	}
	require.NoError(t, db.Create(&event).Error)
	require.JSONEq(t, `[]`, event.DepartmentSnapshot)
	require.Equal(t, "|", event.DepartmentTokens)
	require.Equal(t, "", event.Summary)
	require.NotZero(t, event.CreatedAt)
	require.NotZero(t, event.UpdatedAt)
}

func TestAlertEventSetDepartmentSnapshotBuildsCrossDBFilterTokens(t *testing.T) {
	event := AlertEvent{}

	require.NoError(t, event.SetDepartmentSnapshot([]AlertEventDepartmentSnapshot{
		{DepartmentId: 11, DepartmentName: "Engineering"},
		{DepartmentId: 22, DepartmentName: "Security"},
		{DepartmentId: 11, DepartmentName: "Engineering"},
	}))

	require.Equal(t, "|11|22|", event.DepartmentTokens)
}

func TestMigrateBackfillsDepartmentTokensForLegacyAlertEvents(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&legacyAlertEvent{}))

	legacy := AlertEvent{
		TenantId:     0,
		UserId:       9,
		Username:     "legacy",
		RequestId:    "req-legacy",
		ModelName:    "gpt-4o-mini",
		RiskType:     "abuse",
		ActionResult: "blocked",
		Summary:      "legacy row",
		CreatedAt:    1717117201,
		UpdatedAt:    1717117201,
	}
	require.NoError(t, legacy.SetDepartmentSnapshot([]AlertEventDepartmentSnapshot{
		{DepartmentId: 11, DepartmentName: "Engineering"},
		{DepartmentId: 22, DepartmentName: "Security"},
	}))

	require.NoError(t, db.Create(&legacyAlertEvent{
		TenantId:           legacy.TenantId,
		UserId:             legacy.UserId,
		Username:           legacy.Username,
		RequestId:          legacy.RequestId,
		ModelName:          legacy.ModelName,
		RiskType:           legacy.RiskType,
		ActionResult:       legacy.ActionResult,
		DepartmentSnapshot: legacy.DepartmentSnapshot,
		Summary:            legacy.Summary,
		CreatedAt:          legacy.CreatedAt,
		UpdatedAt:          legacy.UpdatedAt,
	}).Error)

	require.NoError(t, Migrate(db))

	var stored AlertEvent
	require.NoError(t, db.First(&stored).Error)
	require.Equal(t, "|11|22|", stored.DepartmentTokens)
}
