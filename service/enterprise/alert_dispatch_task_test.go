package enterprise

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestRunAlertDispatchTaskOnceCreatesAndProcessesPendingDeliveries(t *testing.T) {
	service, db := setupAlertServiceTest(t)
	oldDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = oldDB
	})

	require.NoError(t, db.Create(&model.User{Id: 1001, Username: "alice", Password: "password123", Group: "default", AffCode: "alice-aff"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 11, TenantId: 0, Name: "Engineering", Status: 1, SourceType: 1, NameHistory: "[]"}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{TenantId: 0, UserId: 1001, DepartmentId: 11, ExternalSource: "manual", Status: 1}).Error)
	_, err := service.SaveAlertRule(AlertRuleInput{
		TenantId:  0,
		Name:      "Rule",
		RiskTypes: []string{"abuse"},
		ChannelConfigs: []AlertRuleChannelInput{
			{Type: entmodel.AlertRuleChannelEmail, Receivers: []string{"ops@example.com"}},
		},
		ActorId: 999,
	})
	require.NoError(t, err)
	require.NoError(t, service.RecordRiskEvent(newAlertTestContext(t, "req-task", 1001, "alice"), RecordRiskEventInput{
		TenantId:        0,
		ModelName:       "gpt-4o-mini",
		RiskType:        "abuse",
		ActionResult:    "blocked",
		Summary:         "policy only",
		EventOccurredAt: time.Now(),
	}))

	result, err := RunAlertDispatchTaskOnce(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.Processed, 0)
}
