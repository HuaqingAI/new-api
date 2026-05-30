package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type AlertEventDepartmentSnapshot struct {
	DepartmentId   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
	ExternalSource string `json:"external_source,omitempty"`
	Status         int    `json:"status,omitempty"`
}

type AlertEvent struct {
	Id                 int    `json:"id" gorm:"primaryKey"`
	TenantId           int    `json:"tenant_id" gorm:"type:int;not null;default:0;index:idx_alert_events_tenant_created,priority:1"`
	UserId             int    `json:"user_id" gorm:"type:int;not null;default:0;index:idx_alert_events_user_id"`
	Username           string `json:"username" gorm:"type:varchar(64);not null;default:''"`
	RequestId          string `json:"request_id" gorm:"type:varchar(96);not null;default:'';index:idx_alert_events_request_id"`
	ModelName          string `json:"model_name" gorm:"type:varchar(255);not null;default:''"`
	RiskType           string `json:"risk_type" gorm:"type:varchar(64);not null;default:'';index:idx_alert_events_risk_type"`
	ActionResult       string `json:"action_result" gorm:"type:varchar(64);not null;default:''"`
	DepartmentSnapshot string `json:"department_snapshot" gorm:"type:text;not null"`
	Summary            string `json:"summary" gorm:"type:text;not null"`
	CreatedAt          int64  `json:"created_at" gorm:"type:bigint;not null;default:0;index:idx_alert_events_tenant_created,priority:2"`
	UpdatedAt          int64  `json:"updated_at" gorm:"type:bigint;not null;default:0"`
}

func (AlertEvent) TableName() string {
	return "enterprise_alert_events"
}

func (e *AlertEvent) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if e.CreatedAt == 0 {
		e.CreatedAt = now
	}
	if e.UpdatedAt == 0 {
		e.UpdatedAt = now
	}
	return e.normalize()
}

func (e *AlertEvent) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now().Unix()
	return e.normalize()
}

func (e *AlertEvent) SetDepartmentSnapshot(snapshot []AlertEventDepartmentSnapshot) error {
	if snapshot == nil {
		snapshot = []AlertEventDepartmentSnapshot{}
	}
	data, err := common.Marshal(snapshot)
	if err != nil {
		return err
	}
	e.DepartmentSnapshot = string(data)
	return nil
}

func (e AlertEvent) ParsedDepartmentSnapshot() ([]AlertEventDepartmentSnapshot, error) {
	if e.DepartmentSnapshot == "" {
		return []AlertEventDepartmentSnapshot{}, nil
	}
	var snapshot []AlertEventDepartmentSnapshot
	if err := common.UnmarshalJsonStr(e.DepartmentSnapshot, &snapshot); err != nil {
		return nil, err
	}
	if snapshot == nil {
		return []AlertEventDepartmentSnapshot{}, nil
	}
	return snapshot, nil
}

func (e *AlertEvent) normalize() error {
	if e.DepartmentSnapshot == "" {
		if err := e.SetDepartmentSnapshot(nil); err != nil {
			return err
		}
	} else {
		snapshot, err := e.ParsedDepartmentSnapshot()
		if err != nil {
			return err
		}
		if err := e.SetDepartmentSnapshot(snapshot); err != nil {
			return err
		}
	}
	if e.Summary == "" {
		e.Summary = ""
	}
	return nil
}
