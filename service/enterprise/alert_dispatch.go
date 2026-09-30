package enterprise

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/service"
	"gorm.io/gorm"
)

var alertRetryBackoffSeconds = []int64{30, 120, 600}

type alertEmailSender func(subject string, receiver string, content string) error
type alertWebhookSender func(webhookURL string, secret string, data dto.Notify) error
type alertNowFunc func() time.Time

type AlertDispatchService struct {
	db          *gorm.DB
	now         alertNowFunc
	sendEmail   alertEmailSender
	sendWebhook alertWebhookSender
}

type AlertDispatchResult struct {
	Processed   int
	Sent        int
	Retried     int
	FinalFailed int
}

func NewAlertDispatchService(db *gorm.DB) *AlertDispatchService {
	if db == nil {
		db = model.DB
	}
	return &AlertDispatchService{
		db:          db,
		now:         time.Now,
		sendEmail:   common.SendEmail,
		sendWebhook: service.SendWebhookNotify,
	}
}

func NewAlertDispatchServiceForTest(
	db *gorm.DB,
	now alertNowFunc,
	sendEmail alertEmailSender,
	sendWebhook alertWebhookSender,
) *AlertDispatchService {
	svc := NewAlertDispatchService(db)
	if now != nil {
		svc.now = now
	}
	if sendEmail != nil {
		svc.sendEmail = sendEmail
	}
	if sendWebhook != nil {
		svc.sendWebhook = sendWebhook
	}
	return svc
}

func (s *AlertDispatchService) DispatchDueDeliveries(ctx context.Context, limit int) (AlertDispatchResult, error) {
	if s == nil || s.db == nil {
		return AlertDispatchResult{}, nil
	}
	if limit <= 0 {
		limit = 100
	}
	nowUnix := s.now().Unix()

	var deliveries []entmodel.AlertDelivery
	if err := s.db.Where("status IN ? AND next_retry_at <= ?", []string{
		entmodel.AlertDeliveryStatusPending,
		entmodel.AlertDeliveryStatusFailed,
	}, nowUnix).
		Order("next_retry_at ASC, id ASC").
		Limit(limit).
		Find(&deliveries).Error; err != nil {
		return AlertDispatchResult{}, err
	}

	result := AlertDispatchResult{}
	for _, delivery := range deliveries {
		result.Processed++
		dispatched, err := s.dispatchDelivery(ctx, &delivery)
		if err != nil {
			return result, err
		}
		switch dispatched {
		case entmodel.AlertDeliveryStatusSent, entmodel.AlertDeliveryStatusResent:
			result.Sent++
		case entmodel.AlertDeliveryStatusFailed:
			result.Retried++
		case entmodel.AlertDeliveryStatusFinalFailed:
			result.FinalFailed++
		}
	}
	return result, nil
}

func (s *AlertDispatchService) dispatchDelivery(ctx context.Context, delivery *entmodel.AlertDelivery) (string, error) {
	nowUnix := s.now().Unix()
	trace, err := delivery.ParsedTracePayload()
	if err != nil {
		return "", err
	}
	if trace == nil {
		trace = &entmodel.AlertDeliveryTracePayload{}
	}
	normalizeAlertTracePayloadDetailPaths(trace)

	rule, err := loadAlertRuleByID(s.db, delivery.TenantId, delivery.RuleId)
	if err != nil {
		if errors.Is(err, ErrAlertRuleNotFound) {
			return s.markDeliveryConfigurationFailure(delivery, nowUnix, err)
		}
		return "", err
	}
	channelConfig, err := findAlertRuleChannelConfig(rule, delivery.ChannelType)
	if err != nil {
		if errors.Is(err, ErrAlertRuleNotFound) {
			return s.markDeliveryConfigurationFailure(delivery, nowUnix, err)
		}
		return "", err
	}

	notify := buildAlertNotifyPayload(delivery, trace)
	delivery.AttemptCount++
	delivery.LastAttemptAt = nowUnix

	sendErr := s.sendDeliveryByChannel(*channelConfig, notify)
	if sendErr == nil {
		nextStatus := entmodel.AlertDeliveryStatusSent
		if delivery.AttemptCount > 1 || delivery.TriggerSource == entmodel.AlertDeliveryTriggerManual {
			nextStatus = entmodel.AlertDeliveryStatusResent
		}
		updates := map[string]any{
			"status":          nextStatus,
			"attempt_count":   delivery.AttemptCount,
			"last_attempt_at": delivery.LastAttemptAt,
			"sent_at":         nowUnix,
			"error_reason":    "",
			"updated_at":      nowUnix,
		}
		if err := s.db.Model(delivery).Updates(updates).Error; err != nil {
			return "", err
		}
		return nextStatus, nil
	}

	errorReason := summarizeAlertDispatchError(sendErr)
	nextStatus := entmodel.AlertDeliveryStatusFailed
	nextRetryAt := nowUnix
	finalFailedAt := int64(0)
	if delivery.AttemptCount >= delivery.MaxAttempts {
		nextStatus = entmodel.AlertDeliveryStatusFinalFailed
		nextRetryAt = 0
		finalFailedAt = nowUnix
	} else {
		nextRetryAt = nowUnix + retryBackoffSecondsForAttempt(delivery.AttemptCount)
	}
	updates := map[string]any{
		"status":          nextStatus,
		"attempt_count":   delivery.AttemptCount,
		"last_attempt_at": delivery.LastAttemptAt,
		"next_retry_at":   nextRetryAt,
		"final_failed_at": finalFailedAt,
		"error_reason":    errorReason,
		"updated_at":      nowUnix,
	}
	if err := s.db.Model(delivery).Updates(updates).Error; err != nil {
		return "", err
	}
	common.SysLog(fmt.Sprintf(
		"enterprise alert delivery failed: delivery_id=%d event_id=%d rule_id=%d channel_type=%s attempt_count=%d error=%s",
		delivery.Id,
		delivery.EventId,
		delivery.RuleId,
		delivery.ChannelType,
		delivery.AttemptCount,
		errorReason,
	))
	return nextStatus, nil
}

func (s *AlertDispatchService) markDeliveryConfigurationFailure(delivery *entmodel.AlertDelivery, nowUnix int64, reason error) (string, error) {
	attemptCount := delivery.AttemptCount + 1
	if attemptCount < 1 {
		attemptCount = 1
	}
	if delivery.MaxAttempts > 0 && attemptCount > delivery.MaxAttempts {
		attemptCount = delivery.MaxAttempts
	}
	errorReason := summarizeAlertDispatchError(reason)
	updates := map[string]any{
		"status":          entmodel.AlertDeliveryStatusFinalFailed,
		"attempt_count":   attemptCount,
		"last_attempt_at": nowUnix,
		"next_retry_at":   0,
		"final_failed_at": nowUnix,
		"error_reason":    errorReason,
		"updated_at":      nowUnix,
	}
	if err := s.db.Model(delivery).Updates(updates).Error; err != nil {
		return "", err
	}
	common.SysLog(fmt.Sprintf(
		"enterprise alert delivery permanently failed: delivery_id=%d event_id=%d rule_id=%d channel_type=%s error=%s",
		delivery.Id,
		delivery.EventId,
		delivery.RuleId,
		delivery.ChannelType,
		errorReason,
	))
	return entmodel.AlertDeliveryStatusFinalFailed, nil
}

func (s *AlertDispatchService) sendDeliveryByChannel(channel entmodel.AlertRuleChannelConfig, notify dto.Notify) error {
	switch channel.Type {
	case entmodel.AlertRuleChannelEmail:
		return s.sendEmail(notify.Title, strings.Join(channel.Receivers, ";"), notify.Content)
	case entmodel.AlertRuleChannelWebhook:
		return s.sendWebhook(channel.WebhookURL, channel.WebhookSecret, notify)
	case entmodel.AlertRuleChannelDingTalkRobot:
		return s.sendWebhook(channel.RobotWebhook, channel.RobotSecret, notify)
	default:
		return ErrAlertRuleInvalidInput
	}
}

func buildAlertNotifyPayload(delivery *entmodel.AlertDelivery, trace *entmodel.AlertDeliveryTracePayload) dto.Notify {
	title := fmt.Sprintf("Risk alert: %s / %s", safeNotifyValue(trace.RiskType, "unknown"), safeNotifyValue(trace.ModelName, "unknown"))
	var builder strings.Builder
	builder.WriteString("<div>")
	builder.WriteString("<p><strong>Department:</strong> " + html.EscapeString(safeNotifyValue(trace.DepartmentSummary, "Unassigned")) + "</p>")
	builder.WriteString("<p><strong>User:</strong> " + html.EscapeString(safeNotifyValue(trace.Username, "unknown")) + "</p>")
	builder.WriteString("<p><strong>Model:</strong> " + html.EscapeString(safeNotifyValue(trace.ModelName, "unknown")) + "</p>")
	builder.WriteString("<p><strong>Occurred At:</strong> " + html.EscapeString(formatAlertNotifyTimestamp(trace.EventCreatedAt)) + "</p>")
	builder.WriteString("<p><strong>Risk Type:</strong> " + html.EscapeString(safeNotifyValue(trace.RiskType, "unknown")) + "</p>")
	builder.WriteString("<p><strong>Request Trace ID:</strong> " + html.EscapeString(safeNotifyValue(trace.RequestId, "N/A")) + "</p>")
	builder.WriteString("<p><strong>Summary:</strong> " + html.EscapeString(safeNotifyValue(trace.EventSummary, "N/A")) + "</p>")
	builder.WriteString("<p><strong>Action Result:</strong> " + html.EscapeString(safeNotifyValue(trace.ActionResult, "unknown")) + "</p>")
	builder.WriteString("<p><strong>Rule:</strong> " + html.EscapeString(fmt.Sprintf("%s (#%d)", safeNotifyValue(trace.RuleName, "unknown"), trace.RuleId)) + "</p>")
	builder.WriteString("<p><strong>Trace Details:</strong> " + html.EscapeString(safeNotifyValue(trace.DetailRoute, "/enterprise-alerts")) + "</p>")
	builder.WriteString("<p><strong>Lookup API:</strong> " + html.EscapeString(safeNotifyValue(trace.DetailAPIPath, "/api/enterprise/alerts/events")) + "</p>")
	builder.WriteString("<p><strong>Delivery ID:</strong> " + fmt.Sprintf("%d", delivery.Id) + "</p>")
	builder.WriteString("<p><strong>Event ID:</strong> " + fmt.Sprintf("%d", trace.EventId) + "</p>")
	builder.WriteString("</div>")

	return dto.NewNotify("enterprise_alert", title, builder.String(), nil)
}

func loadAlertRuleByID(db *gorm.DB, tenantId int, ruleId int) (entmodel.AlertRule, error) {
	var rule entmodel.AlertRule
	if err := db.Where("tenant_id = ? AND id = ?", tenantId, ruleId).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return entmodel.AlertRule{}, ErrAlertRuleNotFound
		}
		return entmodel.AlertRule{}, err
	}
	return rule, nil
}

func findAlertRuleChannelConfig(rule entmodel.AlertRule, channelType string) (*entmodel.AlertRuleChannelConfig, error) {
	configs, err := rule.ParsedChannelConfigs()
	if err != nil {
		return nil, err
	}
	for _, config := range configs {
		if config.Type == channelType {
			return &config, nil
		}
	}
	return nil, ErrAlertRuleNotFound
}

func summarizeAlertDispatchError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.Join(strings.Fields(err.Error()), " ")
	replacer := strings.NewReplacer(
		"secret", "[redacted]",
		"Secret", "[redacted]",
		"token", "[redacted]",
		"Token", "[redacted]",
		"access_token", "[redacted]",
	)
	message = replacer.Replace(message)
	if len(message) > 160 {
		message = message[:160]
	}
	return message
}

func retryBackoffSecondsForAttempt(attemptCount int) int64 {
	if attemptCount <= 0 {
		return alertRetryBackoffSeconds[0]
	}
	index := attemptCount - 1
	if index >= len(alertRetryBackoffSeconds) {
		return alertRetryBackoffSeconds[len(alertRetryBackoffSeconds)-1]
	}
	return alertRetryBackoffSeconds[index]
}

func safeNotifyValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func formatAlertNotifyTimestamp(ts int64) string {
	if ts <= 0 {
		return "N/A"
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}
