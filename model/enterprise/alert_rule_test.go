package enterprise

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAlertRuleJSONWrappersNormalizeEmptyValues(t *testing.T) {
	rule := AlertRule{}

	require.NoError(t, rule.SetRiskTypes([]string{"abuse", "sensitive_words"}))
	require.JSONEq(t, `["abuse","sensitive_words"]`, rule.RiskTypes)

	require.NoError(t, rule.SetDepartmentIds([]int{11, 22}))
	require.JSONEq(t, `[11,22]`, rule.DepartmentIds)

	require.NoError(t, rule.SetChannelConfigs([]AlertRuleChannelConfig{
		{
			Type:      AlertRuleChannelEmail,
			Enabled:   true,
			Receivers: []string{"alice@example.com"},
		},
		{
			Type:          AlertRuleChannelWebhook,
			Enabled:       true,
			WebhookURL:    "https://hooks.example.com/alerts?token=secret",
			WebhookSecret: "top-secret",
		},
	}))
	require.Contains(t, rule.ChannelConfigs, `"type":"email"`)
	require.Contains(t, rule.ChannelConfigs, `"type":"webhook"`)

	riskTypes, err := rule.ParsedRiskTypes()
	require.NoError(t, err)
	require.Equal(t, []string{"abuse", "sensitive_words"}, riskTypes)

	departmentIds, err := rule.ParsedDepartmentIds()
	require.NoError(t, err)
	require.Equal(t, []int{11, 22}, departmentIds)

	configs, err := rule.ParsedChannelConfigs()
	require.NoError(t, err)
	require.Len(t, configs, 2)
	require.Equal(t, "alice@example.com", configs[0].Receivers[0])
	require.Equal(t, "top-secret", configs[1].WebhookSecret)

	require.NoError(t, rule.SetRiskTypes(nil))
	require.NoError(t, rule.SetDepartmentIds(nil))
	require.NoError(t, rule.SetChannelConfigs(nil))
	require.JSONEq(t, `[]`, rule.RiskTypes)
	require.JSONEq(t, `[]`, rule.DepartmentIds)
	require.JSONEq(t, `[]`, rule.ChannelConfigs)
}

func TestAlertRuleTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	models := []any{
		AlertRule{},
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

func TestMigrateCreatesAlertRuleTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&AlertRule{}))
	for _, column := range []string{
		"id",
		"tenant_id",
		"name",
		"enabled",
		"risk_types",
		"department_ids",
		"channel_configs",
		"dedupe_window_seconds",
		"created_by",
		"updated_by",
		"created_at",
		"updated_at",
	} {
		require.True(t, db.Migrator().HasColumn(&AlertRule{}, column), column)
	}
}

func TestAlertRuleBeforeCreateAppliesApplicationDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	rule := AlertRule{
		Name: "Critical abuse",
	}
	require.NoError(t, db.Create(&rule).Error)
	require.JSONEq(t, `[]`, rule.RiskTypes)
	require.JSONEq(t, `[]`, rule.DepartmentIds)
	require.JSONEq(t, `[]`, rule.ChannelConfigs)
	require.True(t, rule.Enabled)
	require.NotZero(t, rule.CreatedAt)
	require.NotZero(t, rule.UpdatedAt)
}
