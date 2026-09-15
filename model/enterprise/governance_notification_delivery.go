package enterprise

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	GovernanceNotificationSourceQuotaRequest    = "quota_request"
	GovernanceNotificationSourceQuotaAllocation = "quota_allocation"

	GovernanceNotificationRecipientRequester = "requester"
	GovernanceNotificationRecipientOwner     = "owner"
	GovernanceNotificationRecipientMember    = "member"
	GovernanceNotificationRecipientAdmin     = "admin"

	GovernanceNotificationChannelDingTalkRobot = AlertRuleChannelDingTalkRobot

	GovernanceNotificationStatusPending      = AlertDeliveryStatusPending
	GovernanceNotificationStatusSent         = AlertDeliveryStatusSent
	GovernanceNotificationStatusFailed       = AlertDeliveryStatusFailed
	GovernanceNotificationStatusFinalFailed  = AlertDeliveryStatusFinalFailed
	GovernanceNotificationStatusResent       = AlertDeliveryStatusResent
	GovernanceNotificationStatusUnconfigured = "unconfigured"

	GovernanceNotificationTriggerGovernanceAction = "governance_action"
	GovernanceNotificationTriggerManual           = AlertDeliveryTriggerManual

	GovernanceNotificationDefaultMaxAttempts = AlertDeliveryDefaultMaxAttempts
)

type GovernanceNotificationTracePayload struct {
	TraceId            string `json:"trace_id"`
	SourceType         string `json:"source_type"`
	SourceId           int    `json:"source_id"`
	ActionType         string `json:"action_type"`
	TenantId           int    `json:"tenant_id"`
	DepartmentId       int    `json:"department_id"`
	DepartmentName     string `json:"department_name"`
	BudgetId           int    `json:"budget_id"`
	AllocationId       int    `json:"allocation_id"`
	RequestId          int    `json:"request_id"`
	ActorId            int    `json:"actor_id"`
	ActorName          string `json:"actor_name"`
	TargetUserId       int    `json:"target_user_id"`
	TargetUsername     string `json:"target_username"`
	TargetDisplayName  string `json:"target_display_name"`
	QuotaDelta         int64  `json:"quota_delta"`
	CommittedQuota     int64  `json:"committed_quota"`
	RequestedQuota     int64  `json:"requested_quota"`
	ApprovedQuota      int64  `json:"approved_quota"`
	Status             string `json:"status"`
	Fallback           string `json:"fallback,omitempty"`
	OccurredAt         int64  `json:"occurred_at"`
	DetailRoute        string `json:"detail_route"`
	DetailAPIPath      string `json:"detail_api_path"`
	Summary            string `json:"summary"`
	RecipientUserId    int    `json:"recipient_user_id"`
	RecipientKind      string `json:"recipient_kind"`
	NotificationTarget string `json:"notification_target,omitempty"`
}

type GovernanceNotificationDelivery struct {
	Id              int    `json:"id" gorm:"primaryKey"`
	TenantId        int    `json:"tenant_id" gorm:"type:int;not null;default:0;index:idx_ent_gov_notif_tenant_created,priority:1;index:idx_ent_gov_notif_due,priority:1;index:idx_ent_gov_notif_source,priority:1"`
	SourceType      string `json:"source_type" gorm:"type:varchar(48);not null;default:'';index:idx_ent_gov_notif_source,priority:2"`
	SourceId        int    `json:"source_id" gorm:"type:int;not null;default:0;index:idx_ent_gov_notif_source,priority:3"`
	TraceId         string `json:"trace_id" gorm:"type:varchar(128);not null;default:'';index:idx_ent_gov_notif_trace"`
	ActionType      string `json:"action_type" gorm:"type:varchar(96);not null;default:'';index:idx_ent_gov_notif_action"`
	RecipientUserId int    `json:"recipient_user_id" gorm:"type:int;not null;default:0;index:idx_ent_gov_notif_recipient"`
	RecipientKind   string `json:"recipient_kind" gorm:"type:varchar(32);not null;default:''"`
	ChannelType     string `json:"channel_type" gorm:"type:varchar(32);not null;default:'dingtalk_robot';index:idx_ent_gov_notif_due,priority:2"`
	Status          string `json:"status" gorm:"type:varchar(32);not null;default:'pending';index:idx_ent_gov_notif_due,priority:3"`
	AttemptCount    int    `json:"attempt_count" gorm:"type:int;not null;default:0"`
	MaxAttempts     int    `json:"max_attempts" gorm:"type:int;not null;default:4"`
	NextRetryAt     int64  `json:"next_retry_at" gorm:"type:bigint;not null;default:0;index:idx_ent_gov_notif_due,priority:4"`
	LastAttemptAt   int64  `json:"last_attempt_at" gorm:"type:bigint;not null;default:0"`
	SentAt          int64  `json:"sent_at" gorm:"type:bigint;not null;default:0"`
	FinalFailedAt   int64  `json:"final_failed_at" gorm:"type:bigint;not null;default:0"`
	ErrorReason     string `json:"error_reason" gorm:"type:text;not null"`
	DedupeKey       string `json:"dedupe_key" gorm:"type:varchar(255);not null;uniqueIndex:uq_ent_gov_notif_dedupe"`
	TracePayload    string `json:"trace_payload" gorm:"type:text;not null"`
	TriggerSource   string `json:"trigger_source" gorm:"type:varchar(64);not null;default:'governance_action'"`
	ManualParentId  *int   `json:"manual_parent_id" gorm:"type:int;index:idx_ent_gov_notif_manual_parent"`
	CreatedAt       int64  `json:"created_at" gorm:"type:bigint;not null;default:0;index:idx_ent_gov_notif_tenant_created,priority:2"`
	UpdatedAt       int64  `json:"updated_at" gorm:"type:bigint;not null;default:0"`
}

func (GovernanceNotificationDelivery) TableName() string {
	return "enterprise_governance_notification_deliveries"
}

func (d *GovernanceNotificationDelivery) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if d.CreatedAt == 0 {
		d.CreatedAt = now
	}
	if d.UpdatedAt == 0 {
		d.UpdatedAt = now
	}
	return d.normalize()
}

func (d *GovernanceNotificationDelivery) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = time.Now().Unix()
	return d.normalize()
}

func (d *GovernanceNotificationDelivery) SetTracePayload(payload *GovernanceNotificationTracePayload) error {
	if payload == nil {
		data, err := common.Marshal(map[string]any{})
		if err != nil {
			return err
		}
		d.TracePayload = string(data)
		return nil
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	d.TracePayload = string(data)
	return nil
}

func (d GovernanceNotificationDelivery) ParsedTracePayload() (*GovernanceNotificationTracePayload, error) {
	if strings.TrimSpace(d.TracePayload) == "" || strings.TrimSpace(d.TracePayload) == "{}" {
		return nil, nil
	}
	var payload GovernanceNotificationTracePayload
	if err := common.UnmarshalJsonStr(d.TracePayload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (d *GovernanceNotificationDelivery) normalize() error {
	if d.ChannelType == "" {
		d.ChannelType = GovernanceNotificationChannelDingTalkRobot
	}
	if d.Status == "" {
		d.Status = GovernanceNotificationStatusPending
	}
	if d.MaxAttempts <= 0 {
		d.MaxAttempts = GovernanceNotificationDefaultMaxAttempts
	}
	if d.AttemptCount < 0 {
		d.AttemptCount = 0
	}
	if d.TriggerSource == "" {
		d.TriggerSource = GovernanceNotificationTriggerGovernanceAction
	}
	if d.TracePayload == "" {
		return d.SetTracePayload(nil)
	}
	payload, err := d.ParsedTracePayload()
	if err != nil {
		return err
	}
	return d.SetTracePayload(payload)
}
