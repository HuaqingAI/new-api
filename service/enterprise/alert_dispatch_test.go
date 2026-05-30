package enterprise

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestAlertServiceEnqueueAlertDeliveriesMatchesRulesAndDedupesWithinWindow(t *testing.T) {
	service, db := setupAlertServiceTest(t)
	require.NoError(t, db.Create(&model.User{Id: 1001, Username: "alice", Password: "password123", Group: "default", AffCode: "alice-aff"}).Error)
	require.NoError(t, db.Create(&[]entmodel.Department{
		{Id: 11, TenantId: 0, Name: "Engineering", Status: 1, SourceType: 1, NameHistory: "[]"},
		{Id: 22, TenantId: 0, Name: "Security", Status: 1, SourceType: 1, NameHistory: "[]"},
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{TenantId: 0, UserId: 1001, DepartmentId: 11, ExternalSource: "manual", Status: 1}).Error)

	_, err := service.SaveAlertRule(AlertRuleInput{
		TenantId:      0,
		Name:          "Critical abuse",
		RiskTypes:     []string{"abuse"},
		DepartmentIds: []int{11},
		ChannelConfigs: []AlertRuleChannelInput{
			{Type: entmodel.AlertRuleChannelEmail, Receivers: []string{"ops@example.com"}},
			{Type: entmodel.AlertRuleChannelWebhook, Enabled: alertBoolPtr(true), WebhookURL: "https://hooks.example.com/alerts"},
			{Type: entmodel.AlertRuleChannelDingTalkRobot, Enabled: alertBoolPtr(true), RobotWebhook: "https://oapi.dingtalk.com/robot/send"},
		},
		ActorId: 999,
	})
	require.NoError(t, err)

	ctx := newAlertTestContext(t, "req-1", 1001, "alice")
	require.NoError(t, service.RecordRiskEvent(ctx, RecordRiskEventInput{
		TenantId:        0,
		ModelName:       "gpt-4o-mini",
		RiskType:        "abuse",
		ActionResult:    AlertActionBlocked,
		Summary:         "policy only",
		EventOccurredAt: time.Unix(1717117201, 0),
	}))
	require.NoError(t, service.RecordRiskEvent(ctx, RecordRiskEventInput{
		TenantId:        0,
		ModelName:       "gpt-4o-mini",
		RiskType:        "abuse",
		ActionResult:    AlertActionBlocked,
		Summary:         "policy only",
		EventOccurredAt: time.Unix(1717117202, 0),
	}))

	created, err := service.EnqueueAlertDeliveriesForPendingEvents(time.Unix(1717117205, 0), 24*time.Hour, 100)
	require.NoError(t, err)
	require.Equal(t, 6, created)

	createdAgain, err := service.EnqueueAlertDeliveriesForPendingEvents(time.Unix(1717117206, 0), 24*time.Hour, 100)
	require.NoError(t, err)
	require.Equal(t, 0, createdAgain)

	var deliveries []entmodel.AlertDelivery
	require.NoError(t, db.Order("id ASC").Find(&deliveries).Error)
	require.Len(t, deliveries, 6)
	require.Equal(t, entmodel.AlertDeliveryStatusPending, deliveries[0].Status)
	trace, err := deliveries[0].ParsedTracePayload()
	require.NoError(t, err)
	require.NotNil(t, trace)
	require.Equal(t, "alice", trace.Username)
	require.Equal(t, "Engineering (#11)", trace.DepartmentSummary)
	require.NotContains(t, deliveries[0].TracePayload, "secret")
}

func TestAlertDispatchServiceRetriesAndMarksFinalFailure(t *testing.T) {
	service, db := setupAlertServiceTest(t)

	ruleResult, err := service.SaveAlertRule(AlertRuleInput{
		TenantId:  0,
		Name:      "Email only",
		RiskTypes: []string{"abuse"},
		ChannelConfigs: []AlertRuleChannelInput{
			{Type: entmodel.AlertRuleChannelEmail, Receivers: []string{"ops@example.com"}},
		},
		ActorId: 999,
	})
	require.NoError(t, err)

	delivery := entmodel.AlertDelivery{
		TenantId:    0,
		EventId:     10,
		RuleId:      ruleResult.Item.Id,
		ChannelType: entmodel.AlertRuleChannelEmail,
		DedupeKey:   "0:10:1:email:1717117200",
		NextRetryAt: 1717117200,
	}
	require.NoError(t, delivery.SetTracePayload(&entmodel.AlertDeliveryTracePayload{
		EventId:           10,
		RequestId:         "req-10",
		TenantId:          0,
		Username:          "alice",
		ModelName:         "gpt-4o-mini",
		RiskType:          "abuse",
		ActionResult:      "blocked",
		EventCreatedAt:    1717117200,
		DepartmentSummary: "Unassigned",
		EventSummary:      "policy",
		RuleId:            ruleResult.Item.Id,
		RuleName:          ruleResult.Item.Name,
		DetailRoute:       "/enterprise-alerts?event_id=10",
		DetailAPIPath:     "/api/enterprise/alerts/events?tenant_id=0",
	}))
	require.NoError(t, db.Create(&delivery).Error)

	nowUnix := int64(1717117200)
	dispatchService := NewAlertDispatchServiceForTest(
		db,
		func() time.Time { return time.Unix(nowUnix, 0) },
		func(subject string, receiver string, content string) error {
			return errors.New("smtp timeout secret should not leak")
		},
		nil,
	)

	first, err := dispatchService.DispatchDueDeliveries(context.Background(), 20)
	require.NoError(t, err)
	require.Equal(t, 1, first.Processed)
	var afterFirst entmodel.AlertDelivery
	require.NoError(t, db.First(&afterFirst, delivery.Id).Error)
	require.Equal(t, entmodel.AlertDeliveryStatusFailed, afterFirst.Status)
	require.Equal(t, nowUnix+30, afterFirst.NextRetryAt)
	nowUnix += 31
	second, err := dispatchService.DispatchDueDeliveries(context.Background(), 20)
	require.NoError(t, err)
	require.Equal(t, 1, second.Processed)
	var afterSecond entmodel.AlertDelivery
	require.NoError(t, db.First(&afterSecond, delivery.Id).Error)
	require.Equal(t, entmodel.AlertDeliveryStatusFailed, afterSecond.Status)
	require.Equal(t, nowUnix+120, afterSecond.NextRetryAt)
	nowUnix += 121
	third, err := dispatchService.DispatchDueDeliveries(context.Background(), 20)
	require.NoError(t, err)
	require.Equal(t, 1, third.Processed)
	var afterThird entmodel.AlertDelivery
	require.NoError(t, db.First(&afterThird, delivery.Id).Error)
	require.Equal(t, entmodel.AlertDeliveryStatusFailed, afterThird.Status)
	require.Equal(t, nowUnix+600, afterThird.NextRetryAt)
	nowUnix += 601
	fourth, err := dispatchService.DispatchDueDeliveries(context.Background(), 20)
	require.NoError(t, err)
	require.Equal(t, 1, fourth.Processed)
	require.Equal(t, 1, fourth.FinalFailed)

	var stored entmodel.AlertDelivery
	require.NoError(t, db.First(&stored, delivery.Id).Error)
	require.Equal(t, entmodel.AlertDeliveryStatusFinalFailed, stored.Status)
	require.Equal(t, 4, stored.AttemptCount)
	require.NotZero(t, stored.FinalFailedAt)
	require.NotContains(t, stored.ErrorReason, "secret")
}

func TestAlertServiceListAlertDeliveriesFiltersTenantStatusAndPagination(t *testing.T) {
	service, db := setupAlertServiceTest(t)

	deliveryOne := entmodel.AlertDelivery{
		TenantId:      7,
		EventId:       101,
		RuleId:        201,
		ChannelType:   entmodel.AlertRuleChannelWebhook,
		Status:        entmodel.AlertDeliveryStatusFinalFailed,
		AttemptCount:  4,
		MaxAttempts:   4,
		FinalFailedAt: 1717117500,
		ErrorReason:   "webhook timeout",
		DedupeKey:     "7:101:201:webhook:1717117200",
		CreatedAt:     1717117200,
		UpdatedAt:     1717117500,
	}
	require.NoError(t, deliveryOne.SetTracePayload(&entmodel.AlertDeliveryTracePayload{
		EventId:           101,
		RequestId:         "req-101",
		TenantId:          7,
		Username:          "alice",
		ModelName:         "gpt-4o-mini",
		RiskType:          "abuse",
		ActionResult:      "blocked",
		EventCreatedAt:    1717117200,
		DepartmentSummary: "Engineering (#11)",
		EventSummary:      "review requested",
		RuleId:            201,
		RuleName:          "Webhook alert",
		DetailRoute:       "/enterprise-alerts?event_id=101",
		DetailAPIPath:     "/api/enterprise/alerts/events?tenant_id=7",
	}))
	require.NoError(t, db.Create(&deliveryOne).Error)

	deliveryTwo := entmodel.AlertDelivery{
		TenantId:      7,
		EventId:       102,
		RuleId:        202,
		ChannelType:   entmodel.AlertRuleChannelEmail,
		Status:        entmodel.AlertDeliveryStatusSent,
		AttemptCount:  1,
		MaxAttempts:   4,
		SentAt:        1717117600,
		DedupeKey:     "7:102:202:email:1717117500",
		CreatedAt:     1717117500,
		UpdatedAt:     1717117600,
	}
	require.NoError(t, deliveryTwo.SetTracePayload(&entmodel.AlertDeliveryTracePayload{
		EventId:           102,
		RequestId:         "req-102",
		TenantId:          7,
		Username:          "bob",
		ModelName:         "claude-sonnet-4",
		RiskType:          "sensitive_words",
		ActionResult:      "blocked",
		EventCreatedAt:    1717117500,
		DepartmentSummary: "Security (#22)",
		EventSummary:      "sent successfully",
		RuleId:            202,
		RuleName:          "Email alert",
		DetailRoute:       "/enterprise-alerts?event_id=102",
		DetailAPIPath:     "/api/enterprise/alerts/events?tenant_id=7",
	}))
	require.NoError(t, db.Create(&deliveryTwo).Error)

	otherTenant := entmodel.AlertDelivery{
		TenantId:     9,
		EventId:      103,
		RuleId:       203,
		ChannelType:  entmodel.AlertRuleChannelWebhook,
		Status:       entmodel.AlertDeliveryStatusFinalFailed,
		AttemptCount: 4,
		MaxAttempts:  4,
		ErrorReason:  "should not leak across tenants",
		DedupeKey:    "9:103:203:webhook:1717117800",
		CreatedAt:    1717117800,
		UpdatedAt:    1717117801,
	}
	require.NoError(t, db.Create(&otherTenant).Error)

	result, err := service.ListAlertDeliveries(AlertDeliveryQuery{
		TenantId:    7,
		ChannelType: entmodel.AlertRuleChannelWebhook,
		Status:      entmodel.AlertDeliveryStatusFinalFailed,
		Page:        1,
		PageSize:    1,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, 1, result.Page)
	require.Equal(t, 1, result.PageSize)
	require.Len(t, result.Items, 1)
	require.Equal(t, deliveryOne.Id, result.Items[0].Id)
	require.Equal(t, "webhook timeout", result.Items[0].ErrorReason)
	require.NotNil(t, result.Items[0].Trace)
	require.Equal(t, "req-101", result.Items[0].Trace.RequestId)
	require.Equal(t, "/enterprise-alerts?event_id=101", result.Items[0].Trace.DetailRoute)
}

func TestAlertDispatchServiceSuccessMarksResentOnRetry(t *testing.T) {
	service, db := setupAlertServiceTest(t)

	ruleResult, err := service.SaveAlertRule(AlertRuleInput{
		TenantId:  0,
		Name:      "Webhook only after email",
		RiskTypes: []string{"abuse"},
		ChannelConfigs: []AlertRuleChannelInput{
			{Type: entmodel.AlertRuleChannelEmail, Receivers: []string{"ops@example.com"}},
			{Type: entmodel.AlertRuleChannelWebhook, Enabled: alertBoolPtr(true), WebhookURL: "https://hooks.example.com/alerts"},
		},
		ActorId: 999,
	})
	require.NoError(t, err)

	manualParent := 9
	delivery := entmodel.AlertDelivery{
		TenantId:       0,
		EventId:        10,
		RuleId:         ruleResult.Item.Id,
		ChannelType:    entmodel.AlertRuleChannelWebhook,
		Status:         entmodel.AlertDeliveryStatusFailed,
		AttemptCount:   1,
		MaxAttempts:    4,
		NextRetryAt:    1717117200,
		DedupeKey:      "0:10:1:webhook:1717117200",
		TriggerSource:  entmodel.AlertDeliveryTriggerManual,
		ManualParentId: &manualParent,
	}
	require.NoError(t, delivery.SetTracePayload(&entmodel.AlertDeliveryTracePayload{
		EventId:           10,
		RequestId:         "req-10",
		TenantId:          0,
		Username:          "alice",
		ModelName:         "gpt-4o-mini",
		RiskType:          "abuse",
		ActionResult:      "blocked",
		EventCreatedAt:    1717117200,
		DepartmentSummary: "Unassigned",
		EventSummary:      "policy",
		RuleId:            ruleResult.Item.Id,
		RuleName:          ruleResult.Item.Name,
		DetailRoute:       "/enterprise-alerts?event_id=10",
		DetailAPIPath:     "/api/enterprise/alerts/events?tenant_id=0",
	}))
	require.NoError(t, db.Create(&delivery).Error)

	dispatchService := NewAlertDispatchServiceForTest(
		db,
		func() time.Time { return time.Unix(1717117201, 0) },
		nil,
		func(webhookURL string, secret string, data dto.Notify) error {
			require.Equal(t, "https://hooks.example.com/alerts", webhookURL)
			require.Contains(t, data.Content, "Request Trace ID")
			return nil
		},
	)
	result, err := dispatchService.DispatchDueDeliveries(context.Background(), 20)
	require.NoError(t, err)
	require.Equal(t, 1, result.Sent)

	var stored entmodel.AlertDelivery
	require.NoError(t, db.First(&stored, delivery.Id).Error)
	require.Equal(t, entmodel.AlertDeliveryStatusResent, stored.Status)
	require.NotZero(t, stored.SentAt)
	require.Empty(t, stored.ErrorReason)
}
