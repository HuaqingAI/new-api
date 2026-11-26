package enterprise

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	AlertDeliveryStatusPending     = "pending"
	AlertDeliveryStatusSent        = "sent"
	AlertDeliveryStatusFailed      = "failed"
	AlertDeliveryStatusFinalFailed = "final_failed"
	AlertDeliveryStatusResent      = "resent"

	AlertDeliveryTriggerRuleMatch = "rule_match"
	AlertDeliveryTriggerManual    = "manual_resend"

	AlertDeliveryDefaultMaxAttempts = 4
)

type AlertDeliveryTracePayload struct {
	EventId            int                            `json:"event_id"`
	RequestId          string                         `json:"request_id"`
	TenantId           int                            `json:"tenant_id"`
	UserId             int                            `json:"user_id"`
	Username           string                         `json:"username"`
	DisplayName        string                         `json:"display_name"`
	ModelName          string                         `json:"model_name"`
	RiskType           string                         `json:"risk_type"`
	ActionResult       string                         `json:"action_result"`
	EventCreatedAt     int64                          `json:"event_created_at"`
	DepartmentSnapshot []AlertEventDepartmentSnapshot `json:"department_snapshot"`
	DepartmentSummary  string                         `json:"department_summary"`
	EventSummary       string                         `json:"event_summary"`
	RuleId             int                            `json:"rule_id"`
	RuleName           string                         `json:"rule_name"`
	DetailRoute        string                         `json:"detail_route"`
	DetailAPIPath      string                         `json:"detail_api_path"`
}

type AlertDelivery struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	TenantId       int    `json:"tenant_id" gorm:"type:int;not null;default:0;index:idx_alert_deliveries_tenant_created,priority:1;index:idx_alert_deliveries_due,priority:1;index:idx_alert_deliveries_event_rule_channel,priority:1"`
	EventId        int    `json:"event_id" gorm:"type:int;not null;default:0;index:idx_alert_deliveries_event_rule_channel,priority:2"`
	RuleId         int    `json:"rule_id" gorm:"type:int;not null;default:0;index:idx_alert_deliveries_event_rule_channel,priority:3"`
	ChannelType    string `json:"channel_type" gorm:"type:varchar(32);not null;default:'';index:idx_alert_deliveries_event_rule_channel,priority:4"`
	Status         string `json:"status" gorm:"type:varchar(32);not null;default:'pending';index:idx_alert_deliveries_due,priority:2"`
	AttemptCount   int    `json:"attempt_count" gorm:"type:int;not null;default:0"`
	MaxAttempts    int    `json:"max_attempts" gorm:"type:int;not null;default:4"`
	NextRetryAt    int64  `json:"next_retry_at" gorm:"type:bigint;not null;default:0;index:idx_alert_deliveries_due,priority:3"`
	LastAttemptAt  int64  `json:"last_attempt_at" gorm:"type:bigint;not null;default:0"`
	SentAt         int64  `json:"sent_at" gorm:"type:bigint;not null;default:0"`
	FinalFailedAt  int64  `json:"final_failed_at" gorm:"type:bigint;not null;default:0"`
	ErrorReason    string `json:"error_reason" gorm:"type:text;not null"`
	DedupeKey      string `json:"dedupe_key" gorm:"type:varchar(255);not null;uniqueIndex:uq_alert_deliveries_dedupe"`
	TracePayload   string `json:"trace_payload" gorm:"type:text;not null"`
	TriggerSource  string `json:"trigger_source" gorm:"type:varchar(64);not null;default:'rule_match'"`
	ManualParentId *int   `json:"manual_parent_id" gorm:"type:int;index:idx_alert_deliveries_manual_parent"`
	CreatedAt      int64  `json:"created_at" gorm:"type:bigint;not null;default:0;index:idx_alert_deliveries_tenant_created,priority:2"`
	UpdatedAt      int64  `json:"updated_at" gorm:"type:bigint;not null;default:0"`
}

func (AlertDelivery) TableName() string {
	return "enterprise_alert_deliveries"
}

func (d *AlertDelivery) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if d.CreatedAt == 0 {
		d.CreatedAt = now
	}
	if d.UpdatedAt == 0 {
		d.UpdatedAt = now
	}
	return d.normalize()
}

func (d *AlertDelivery) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = time.Now().Unix()
	return d.normalize()
}

func (d *AlertDelivery) SetTracePayload(payload *AlertDeliveryTracePayload) error {
	if payload == nil {
		data, err := common.Marshal(map[string]any{})
		if err != nil {
			return err
		}
		d.TracePayload = string(data)
		return nil
	}
	if payload.DepartmentSnapshot == nil {
		payload.DepartmentSnapshot = []AlertEventDepartmentSnapshot{}
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	d.TracePayload = string(data)
	return nil
}

func (d AlertDelivery) ParsedTracePayload() (*AlertDeliveryTracePayload, error) {
	if strings.TrimSpace(d.TracePayload) == "" || strings.TrimSpace(d.TracePayload) == "{}" {
		return nil, nil
	}
	var payload AlertDeliveryTracePayload
	if err := common.UnmarshalJsonStr(d.TracePayload, &payload); err != nil {
		return nil, err
	}
	if payload.DepartmentSnapshot == nil {
		payload.DepartmentSnapshot = []AlertEventDepartmentSnapshot{}
	}
	return &payload, nil
}

func (d *AlertDelivery) normalize() error {
	if d.Status == "" {
		d.Status = AlertDeliveryStatusPending
	}
	if d.MaxAttempts <= 0 {
		d.MaxAttempts = AlertDeliveryDefaultMaxAttempts
	}
	if d.AttemptCount < 0 {
		d.AttemptCount = 0
	}
	if d.TriggerSource == "" {
		d.TriggerSource = AlertDeliveryTriggerRuleMatch
	}
	if d.ErrorReason == "" {
		d.ErrorReason = ""
	}
	if d.TracePayload == "" {
		if err := d.SetTracePayload(nil); err != nil {
			return err
		}
	} else {
		payload, err := d.ParsedTracePayload()
		if err != nil {
			return err
		}
		if err := d.SetTracePayload(payload); err != nil {
			return err
		}
	}
	return nil
}
