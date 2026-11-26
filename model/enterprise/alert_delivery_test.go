package enterprise

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAlertDeliveryJSONWrappersNormalizeEmptyValues(t *testing.T) {
	delivery := AlertDelivery{}

	require.NoError(t, delivery.SetTracePayload(&AlertDeliveryTracePayload{
		EventId:           10,
		RequestId:         "req-1",
		TenantId:          7,
		Username:          "alice",
		ModelName:         "gpt-4o-mini",
		RiskType:          "abuse",
		ActionResult:      "blocked",
		EventCreatedAt:    1717117201,
		DepartmentSummary: "Engineering (#11), Security (#22)",
		DepartmentSnapshot: []AlertEventDepartmentSnapshot{
			{DepartmentId: 11, DepartmentName: "Engineering"},
			{DepartmentId: 22, DepartmentName: "Security"},
		},
	}))
	require.Contains(t, delivery.TracePayload, `"request_id":"req-1"`)

	payload, err := delivery.ParsedTracePayload()
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Len(t, payload.DepartmentSnapshot, 2)
	require.Equal(t, "alice", payload.Username)

	require.NoError(t, delivery.SetTracePayload(nil))
	require.JSONEq(t, `{}`, delivery.TracePayload)
	parsed, err := delivery.ParsedTracePayload()
	require.NoError(t, err)
	require.Nil(t, parsed)
}

func TestAlertDeliveryTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	models := []any{
		AlertDelivery{},
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

func TestMigrateCreatesAlertDeliveryTableAndIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&AlertDelivery{}))
	for _, column := range []string{
		"id",
		"tenant_id",
		"event_id",
		"rule_id",
		"channel_type",
		"status",
		"attempt_count",
		"max_attempts",
		"next_retry_at",
		"last_attempt_at",
		"sent_at",
		"final_failed_at",
		"error_reason",
		"dedupe_key",
		"trace_payload",
		"trigger_source",
		"manual_parent_id",
		"created_at",
		"updated_at",
	} {
		require.True(t, db.Migrator().HasColumn(&AlertDelivery{}, column), column)
	}
	require.True(t, db.Migrator().HasIndex(&AlertDelivery{}, "uq_alert_deliveries_dedupe"))
}

func TestAlertDeliveryBeforeCreateAppliesDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	delivery := AlertDelivery{
		TenantId:    0,
		EventId:     1,
		RuleId:      2,
		ChannelType: AlertRuleChannelEmail,
		DedupeKey:   "0:1:2:email:1717117200",
	}
	require.NoError(t, db.Create(&delivery).Error)
	require.Equal(t, AlertDeliveryStatusPending, delivery.Status)
	require.Equal(t, AlertDeliveryDefaultMaxAttempts, delivery.MaxAttempts)
	require.Equal(t, AlertDeliveryTriggerRuleMatch, delivery.TriggerSource)
	require.NotZero(t, delivery.CreatedAt)
	require.NotZero(t, delivery.UpdatedAt)
}

func TestAlertDeliveryUniqueDedupeKeyConstraint(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	delivery := AlertDelivery{
		TenantId:    0,
		EventId:     1,
		RuleId:      2,
		ChannelType: AlertRuleChannelEmail,
		DedupeKey:   "0:1:2:email:1717117200",
	}
	require.NoError(t, db.Create(&delivery).Error)

	duplicate := AlertDelivery{
		TenantId:    0,
		EventId:     1,
		RuleId:      2,
		ChannelType: AlertRuleChannelEmail,
		DedupeKey:   "0:1:2:email:1717117200",
	}
	require.Error(t, db.Create(&duplicate).Error)
}
