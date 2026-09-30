package enterprise

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/service"
	"gorm.io/gorm"
)

type governanceWebhookSender func(webhookURL string, secret string, data dto.Notify) error

type GovernanceNotificationDispatchService struct {
	db          *gorm.DB
	now         func() time.Time
	sendWebhook governanceWebhookSender
}

type GovernanceNotificationDispatchResult struct {
	Processed   int
	Sent        int
	Retried     int
	FinalFailed int
}

func NewGovernanceNotificationDispatchService(db *gorm.DB) *GovernanceNotificationDispatchService {
	if db == nil {
		db = model.DB
	}
	return &GovernanceNotificationDispatchService{
		db:          db,
		now:         time.Now,
		sendWebhook: service.SendWebhookNotify,
	}
}

func NewGovernanceNotificationDispatchServiceForTest(db *gorm.DB, now func() time.Time, sendWebhook governanceWebhookSender) *GovernanceNotificationDispatchService {
	svc := NewGovernanceNotificationDispatchService(db)
	if now != nil {
		svc.now = now
	}
	if sendWebhook != nil {
		svc.sendWebhook = sendWebhook
	}
	return svc
}

func (s *GovernanceNotificationDispatchService) DispatchDueDeliveries(ctx context.Context, limit int) (GovernanceNotificationDispatchResult, error) {
	if s == nil || s.db == nil {
		return GovernanceNotificationDispatchResult{}, nil
	}
	if limit <= 0 {
		limit = 100
	}
	nowUnix := s.now().Unix()
	var deliveries []entmodel.GovernanceNotificationDelivery
	if err := s.db.Where("status IN ? AND next_retry_at <= ?", []string{
		entmodel.GovernanceNotificationStatusPending,
		entmodel.GovernanceNotificationStatusFailed,
	}, nowUnix).Order("next_retry_at ASC, id ASC").Limit(limit).Find(&deliveries).Error; err != nil {
		return GovernanceNotificationDispatchResult{}, err
	}
	result := GovernanceNotificationDispatchResult{}
	for _, delivery := range deliveries {
		result.Processed++
		status, err := s.dispatchDelivery(ctx, &delivery)
		if err != nil {
			return result, err
		}
		switch status {
		case entmodel.GovernanceNotificationStatusSent, entmodel.GovernanceNotificationStatusResent:
			result.Sent++
		case entmodel.GovernanceNotificationStatusFailed:
			result.Retried++
		case entmodel.GovernanceNotificationStatusFinalFailed:
			result.FinalFailed++
		}
	}
	return result, nil
}

func (s *GovernanceNotificationDispatchService) dispatchDelivery(ctx context.Context, delivery *entmodel.GovernanceNotificationDelivery) (string, error) {
	nowUnix := s.now().Unix()
	trace, err := delivery.ParsedTracePayload()
	if err != nil {
		return "", err
	}
	channel, err := s.findDingTalkChannel(delivery.TenantId)
	if err != nil {
		return s.markDeliveryConfigurationFailure(delivery, nowUnix, err)
	}
	notify := buildGovernanceNotificationPayload(delivery, trace)
	delivery.AttemptCount++
	delivery.LastAttemptAt = nowUnix
	sendErr := s.sendWebhook(channel.RobotWebhook, channel.RobotSecret, notify)
	if sendErr == nil {
		nextStatus := entmodel.GovernanceNotificationStatusSent
		if delivery.AttemptCount > 1 || delivery.TriggerSource == entmodel.GovernanceNotificationTriggerManual {
			nextStatus = entmodel.GovernanceNotificationStatusResent
		}
		return nextStatus, s.db.Model(delivery).Updates(map[string]any{
			"status":          nextStatus,
			"attempt_count":   delivery.AttemptCount,
			"last_attempt_at": delivery.LastAttemptAt,
			"sent_at":         nowUnix,
			"error_reason":    "",
			"updated_at":      nowUnix,
		}).Error
	}
	return s.markDeliverySendFailure(delivery, nowUnix, sendErr)
}

func (s *GovernanceNotificationDispatchService) findDingTalkChannel(tenantId int) (*entmodel.AlertRuleChannelConfig, error) {
	var rules []entmodel.AlertRule
	if err := s.db.Where("tenant_id = ? AND enabled = ?", tenantId, true).Order("id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	for _, rule := range rules {
		configs, err := rule.ParsedChannelConfigs()
		if err != nil {
			return nil, err
		}
		for _, config := range configs {
			if config.Type == entmodel.AlertRuleChannelDingTalkRobot && config.Enabled && strings.TrimSpace(config.RobotWebhook) != "" {
				return &config, nil
			}
		}
	}
	return nil, ErrAlertRuleNotFound
}

func (s *GovernanceNotificationDispatchService) markDeliveryConfigurationFailure(delivery *entmodel.GovernanceNotificationDelivery, nowUnix int64, reason error) (string, error) {
	errorReason := "governance notification channel is not configured"
	if reason != nil && !errors.Is(reason, ErrAlertRuleNotFound) {
		errorReason = summarizeAlertDispatchError(reason)
	}
	return entmodel.GovernanceNotificationStatusUnconfigured, s.db.Model(delivery).Updates(map[string]any{
		"status":          entmodel.GovernanceNotificationStatusUnconfigured,
		"last_attempt_at": int64(0),
		"next_retry_at":   int64(0),
		"final_failed_at": int64(0),
		"error_reason":    errorReason,
		"updated_at":      nowUnix,
	}).Error
}

func (s *GovernanceNotificationDispatchService) markDeliverySendFailure(delivery *entmodel.GovernanceNotificationDelivery, nowUnix int64, sendErr error) (string, error) {
	errorReason := summarizeAlertDispatchError(sendErr)
	nextStatus := entmodel.GovernanceNotificationStatusFailed
	nextRetryAt := nowUnix
	finalFailedAt := int64(0)
	if delivery.AttemptCount >= delivery.MaxAttempts {
		nextStatus = entmodel.GovernanceNotificationStatusFinalFailed
		nextRetryAt = 0
		finalFailedAt = nowUnix
	} else {
		nextRetryAt = nowUnix + retryBackoffSecondsForAttempt(delivery.AttemptCount)
	}
	return nextStatus, s.db.Model(delivery).Updates(map[string]any{
		"status":          nextStatus,
		"attempt_count":   delivery.AttemptCount,
		"last_attempt_at": delivery.LastAttemptAt,
		"next_retry_at":   nextRetryAt,
		"final_failed_at": finalFailedAt,
		"error_reason":    errorReason,
		"updated_at":      nowUnix,
	}).Error
}

func buildGovernanceNotificationPayload(delivery *entmodel.GovernanceNotificationDelivery, trace *entmodel.GovernanceNotificationTracePayload) dto.Notify {
	if trace == nil {
		trace = &entmodel.GovernanceNotificationTracePayload{}
	}
	title := fmt.Sprintf("Governance action: %s", safeNotifyValue(trace.ActionType, delivery.ActionType))
	var builder strings.Builder
	builder.WriteString("<div>")
	builder.WriteString("<p><strong>Action:</strong> " + html.EscapeString(safeNotifyValue(trace.ActionType, delivery.ActionType)) + "</p>")
	builder.WriteString("<p><strong>Department:</strong> " + html.EscapeString(formatGovernanceDepartment(trace)) + "</p>")
	builder.WriteString("<p><strong>Budget / Allocation / Request:</strong> " + html.EscapeString(formatGovernanceObjectIds(trace)) + "</p>")
	builder.WriteString("<p><strong>Actor:</strong> " + html.EscapeString(formatGovernanceUser(trace.ActorId, trace.ActorName)) + "</p>")
	builder.WriteString("<p><strong>Target User:</strong> " + html.EscapeString(formatGovernanceUser(trace.TargetUserId, firstNonEmptyGovernance(trace.TargetDisplayName, trace.TargetUsername))) + "</p>")
	builder.WriteString("<p><strong>Quota Change:</strong> " + fmt.Sprintf("%d", trace.QuotaDelta) + "</p>")
	builder.WriteString("<p><strong>Status:</strong> " + html.EscapeString(safeNotifyValue(trace.Status, delivery.Status)) + "</p>")
	builder.WriteString("<p><strong>Processed At:</strong> " + html.EscapeString(formatAlertNotifyTimestamp(trace.OccurredAt)) + "</p>")
	builder.WriteString("<p><strong>Trace ID:</strong> " + html.EscapeString(safeNotifyValue(trace.TraceId, delivery.TraceId)) + "</p>")
	builder.WriteString("<p><strong>Details:</strong> " + html.EscapeString(safeNotifyValue(trace.DetailRoute, "/enterprise-organization")) + "</p>")
	builder.WriteString("<p><strong>Lookup API:</strong> " + html.EscapeString(safeNotifyValue(trace.DetailAPIPath, "/api/enterprise/governance/timeline")) + "</p>")
	builder.WriteString("</div>")
	return dto.NewNotify("enterprise_governance", title, builder.String(), nil)
}

func formatGovernanceDepartment(trace *entmodel.GovernanceNotificationTracePayload) string {
	if trace.DepartmentId <= 0 {
		return safeNotifyValue(trace.DepartmentName, "unknown")
	}
	if strings.TrimSpace(trace.DepartmentName) == "" {
		return fmt.Sprintf("#%d", trace.DepartmentId)
	}
	return fmt.Sprintf("%s (#%d)", trace.DepartmentName, trace.DepartmentId)
}

func formatGovernanceObjectIds(trace *entmodel.GovernanceNotificationTracePayload) string {
	return fmt.Sprintf("budget #%d / allocation #%d / request #%d", trace.BudgetId, trace.AllocationId, trace.RequestId)
}

func formatGovernanceUser(userId int, name string) string {
	if userId <= 0 {
		return safeNotifyValue(name, "admin")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Sprintf("#%d", userId)
	}
	return fmt.Sprintf("%s (#%d)", name, userId)
}
