package enterprise

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGovernanceNotificationDispatchFailureBackoffAndFinalFailed(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	rule := entmodel.AlertRule{
		TenantId: 0,
		Name:     "governance dingtalk",
		Enabled:  true,
	}
	require.NoError(t, rule.SetRiskTypes([]string{"governance"}))
	require.NoError(t, rule.SetDepartmentIds([]int{7}))
	require.NoError(t, rule.SetChannelConfigs([]entmodel.AlertRuleChannelConfig{
		{
			Type:         entmodel.AlertRuleChannelDingTalkRobot,
			Enabled:      true,
			RobotWebhook: "https://oapi.dingtalk.com/robot/send?access_token=token",
			RobotSecret:  "secret",
		},
	}))
	require.NoError(t, db.Create(&rule).Error)

	delivery := entmodel.GovernanceNotificationDelivery{
		TenantId:        0,
		SourceType:      GovernanceSourceQuotaRequest,
		SourceId:        42,
		TraceId:         "quota_request:42",
		ActionType:      GovernanceActionQuotaRequestApproved,
		RecipientUserId: 1001,
		RecipientKind:   entmodel.GovernanceNotificationRecipientRequester,
		DedupeKey:       "quota_request:42:requester",
		Status:          entmodel.GovernanceNotificationStatusPending,
		MaxAttempts:     2,
		NextRetryAt:     1000,
	}
	require.NoError(t, delivery.SetTracePayload(&entmodel.GovernanceNotificationTracePayload{
		TraceId:      "quota_request:42",
		ActionType:   GovernanceActionQuotaRequestApproved,
		DepartmentId: 7,
		Status:       "fulfilled",
	}))
	require.NoError(t, db.Create(&delivery).Error)

	now := time.Unix(1000, 0)
	svc := NewGovernanceNotificationDispatchServiceForTest(
		db,
		func() time.Time { return now },
		func(string, string, dto.Notify) error { return errors.New("webhook token should be redacted") },
	)
	result, err := svc.DispatchDueDeliveries(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, result.Retried)

	var updated entmodel.GovernanceNotificationDelivery
	require.NoError(t, db.First(&updated, delivery.Id).Error)
	require.Equal(t, entmodel.GovernanceNotificationStatusFailed, updated.Status)
	require.Equal(t, 1, updated.AttemptCount)
	require.Equal(t, int64(1030), updated.NextRetryAt)
	require.NotContains(t, updated.ErrorReason, "token")

	now = time.Unix(1030, 0)
	result, err = svc.DispatchDueDeliveries(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, result.FinalFailed)
	require.NoError(t, db.First(&updated, delivery.Id).Error)
	require.Equal(t, entmodel.GovernanceNotificationStatusFinalFailed, updated.Status)
	require.Equal(t, 2, updated.AttemptCount)
	require.Equal(t, int64(0), updated.NextRetryAt)
	require.Equal(t, int64(1030), updated.FinalFailedAt)
}

func TestGovernanceNotificationDispatchMissingConfigurationIsNonBlocking(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	delivery := entmodel.GovernanceNotificationDelivery{
		TenantId:        0,
		SourceType:      GovernanceSourceQuotaRequest,
		SourceId:        42,
		TraceId:         "quota_request:42",
		ActionType:      GovernanceActionQuotaRequestApproved,
		RecipientUserId: 1001,
		RecipientKind:   entmodel.GovernanceNotificationRecipientRequester,
		DedupeKey:       "quota_request:42:requester",
		Status:          entmodel.GovernanceNotificationStatusPending,
		MaxAttempts:     2,
		NextRetryAt:     1000,
	}
	require.NoError(t, delivery.SetTracePayload(&entmodel.GovernanceNotificationTracePayload{
		TraceId:      "quota_request:42",
		ActionType:   GovernanceActionQuotaRequestApproved,
		DepartmentId: 7,
		Status:       "fulfilled",
	}))
	require.NoError(t, db.Create(&delivery).Error)

	sendCalled := false
	svc := NewGovernanceNotificationDispatchServiceForTest(
		db,
		func() time.Time { return time.Unix(1000, 0) },
		func(string, string, dto.Notify) error {
			sendCalled = true
			return nil
		},
	)
	result, err := svc.DispatchDueDeliveries(context.Background(), 10)
	require.NoError(t, err)
	require.False(t, sendCalled)
	require.Equal(t, 1, result.Processed)
	require.Equal(t, 0, result.FinalFailed)
	require.Equal(t, 0, result.Retried)

	var updated entmodel.GovernanceNotificationDelivery
	require.NoError(t, db.First(&updated, delivery.Id).Error)
	require.Equal(t, entmodel.GovernanceNotificationStatusUnconfigured, updated.Status)
	require.Equal(t, 0, updated.AttemptCount)
	require.Equal(t, int64(0), updated.NextRetryAt)
	require.Equal(t, int64(0), updated.LastAttemptAt)
	require.Equal(t, int64(0), updated.FinalFailedAt)
	require.Contains(t, updated.ErrorReason, "not configured")
	require.NotContains(t, updated.ErrorReason, "token")

	result, err = svc.DispatchDueDeliveries(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 0, result.Processed)
}

func TestGovernanceNotificationDispatchUnavailableDingTalkRobotIsNonBlocking(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	rule := entmodel.AlertRule{
		TenantId: 0,
		Name:     "governance without enabled robot",
		Enabled:  true,
	}
	require.NoError(t, rule.SetRiskTypes([]string{"governance"}))
	require.NoError(t, rule.SetDepartmentIds([]int{7}))
	require.NoError(t, rule.SetChannelConfigs([]entmodel.AlertRuleChannelConfig{
		{
			Type:      entmodel.AlertRuleChannelEmail,
			Enabled:   true,
			Receivers: []string{"ops@example.com"},
		},
		{
			Type:         entmodel.AlertRuleChannelDingTalkRobot,
			Enabled:      false,
			RobotWebhook: "https://oapi.dingtalk.com/robot/send?access_token=token",
			RobotSecret:  "secret",
		},
	}))
	require.NoError(t, db.Create(&rule).Error)

	delivery := entmodel.GovernanceNotificationDelivery{
		TenantId:        0,
		SourceType:      GovernanceSourceQuotaRequest,
		SourceId:        42,
		TraceId:         "quota_request:42",
		ActionType:      GovernanceActionQuotaRequestApproved,
		RecipientUserId: 1001,
		RecipientKind:   entmodel.GovernanceNotificationRecipientRequester,
		DedupeKey:       "quota_request:42:requester",
		Status:          entmodel.GovernanceNotificationStatusPending,
		MaxAttempts:     2,
		NextRetryAt:     1000,
	}
	require.NoError(t, delivery.SetTracePayload(&entmodel.GovernanceNotificationTracePayload{
		TraceId:      "quota_request:42",
		ActionType:   GovernanceActionQuotaRequestApproved,
		DepartmentId: 7,
		Status:       "fulfilled",
	}))
	require.NoError(t, db.Create(&delivery).Error)

	sendCalled := false
	svc := NewGovernanceNotificationDispatchServiceForTest(
		db,
		func() time.Time { return time.Unix(1000, 0) },
		func(string, string, dto.Notify) error {
			sendCalled = true
			return nil
		},
	)
	result, err := svc.DispatchDueDeliveries(context.Background(), 10)
	require.NoError(t, err)
	require.False(t, sendCalled)
	require.Equal(t, 1, result.Processed)
	require.Equal(t, 0, result.FinalFailed)
	require.Equal(t, 0, result.Retried)

	var updated entmodel.GovernanceNotificationDelivery
	require.NoError(t, db.First(&updated, delivery.Id).Error)
	require.Equal(t, entmodel.GovernanceNotificationStatusUnconfigured, updated.Status)
	require.Equal(t, 0, updated.AttemptCount)
	require.Equal(t, int64(0), updated.NextRetryAt)
	require.Equal(t, int64(0), updated.FinalFailedAt)
	require.Contains(t, updated.ErrorReason, "not configured")
	require.NotContains(t, updated.ErrorReason, "token")
	require.NotContains(t, updated.ErrorReason, "secret")
}
