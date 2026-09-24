package enterprise

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	AlertRuleChannelEmail         = "email"
	AlertRuleChannelWebhook       = "webhook"
	AlertRuleChannelDingTalkRobot = "dingtalk_robot"
)

type AlertRuleChannelConfig struct {
	Type                 string   `json:"type"`
	Enabled              bool     `json:"enabled"`
	Receivers            []string `json:"receivers,omitempty"`
	WebhookURL           string   `json:"webhook_url,omitempty"`
	WebhookSecret        string   `json:"webhook_secret,omitempty"`
	WebhookSecretMasked  string   `json:"webhook_secret_masked,omitempty"`
	WebhookSecretPresent bool     `json:"webhook_secret_configured,omitempty"`
	RobotWebhook         string   `json:"robot_webhook,omitempty"`
	RobotSecret          string   `json:"robot_secret,omitempty"`
	RobotSecretMasked    string   `json:"robot_secret_masked,omitempty"`
	RobotSecretPresent   bool     `json:"robot_secret_configured,omitempty"`
}

type AlertRule struct {
	Id                  int    `json:"id" gorm:"primaryKey"`
	TenantId            int    `json:"tenant_id" gorm:"type:int;not null;default:0;index:idx_alert_rules_tenant_enabled,priority:1;uniqueIndex:uq_alert_rules_tenant_name,priority:1"`
	Name                string `json:"name" gorm:"type:varchar(128);not null;default:'';uniqueIndex:uq_alert_rules_tenant_name,priority:2"`
	Enabled             bool   `json:"enabled" gorm:"not null;default:true;index:idx_alert_rules_tenant_enabled,priority:2"`
	RiskTypes           string `json:"risk_types" gorm:"type:text;not null"`
	DepartmentIds       string `json:"department_ids" gorm:"type:text;not null"`
	ChannelConfigs      string `json:"channel_configs" gorm:"type:text;not null"`
	DedupeWindowSeconds int    `json:"dedupe_window_seconds" gorm:"type:int;not null;default:0"`
	CreatedBy           int    `json:"created_by" gorm:"type:int;not null;default:0"`
	UpdatedBy           int    `json:"updated_by" gorm:"type:int;not null;default:0"`
	CreatedAt           int64  `json:"created_at" gorm:"type:bigint;not null;default:0;index:idx_alert_rules_created_at"`
	UpdatedAt           int64  `json:"updated_at" gorm:"type:bigint;not null;default:0"`
}

func (AlertRule) TableName() string {
	return "enterprise_alert_rules"
}

func (r *AlertRule) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	if r.UpdatedAt == 0 {
		r.UpdatedAt = now
	}
	return r.normalize()
}

func (r *AlertRule) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now().Unix()
	return r.normalize()
}

func (r *AlertRule) SetRiskTypes(riskTypes []string) error {
	if riskTypes == nil {
		riskTypes = []string{}
	}
	data, err := common.Marshal(riskTypes)
	if err != nil {
		return err
	}
	r.RiskTypes = string(data)
	return nil
}

func (r AlertRule) ParsedRiskTypes() ([]string, error) {
	if strings.TrimSpace(r.RiskTypes) == "" {
		return []string{}, nil
	}
	var riskTypes []string
	if err := common.UnmarshalJsonStr(r.RiskTypes, &riskTypes); err != nil {
		return nil, err
	}
	if riskTypes == nil {
		return []string{}, nil
	}
	return riskTypes, nil
}

func (r *AlertRule) SetDepartmentIds(departmentIds []int) error {
	if departmentIds == nil {
		departmentIds = []int{}
	}
	data, err := common.Marshal(departmentIds)
	if err != nil {
		return err
	}
	r.DepartmentIds = string(data)
	return nil
}

func (r AlertRule) ParsedDepartmentIds() ([]int, error) {
	if strings.TrimSpace(r.DepartmentIds) == "" {
		return []int{}, nil
	}
	var departmentIds []int
	if err := common.UnmarshalJsonStr(r.DepartmentIds, &departmentIds); err != nil {
		return nil, err
	}
	if departmentIds == nil {
		return []int{}, nil
	}
	return departmentIds, nil
}

func (r *AlertRule) SetChannelConfigs(configs []AlertRuleChannelConfig) error {
	if configs == nil {
		configs = []AlertRuleChannelConfig{}
	}
	data, err := common.Marshal(configs)
	if err != nil {
		return err
	}
	r.ChannelConfigs = string(data)
	return nil
}

func (r AlertRule) ParsedChannelConfigs() ([]AlertRuleChannelConfig, error) {
	if strings.TrimSpace(r.ChannelConfigs) == "" {
		return []AlertRuleChannelConfig{}, nil
	}
	var configs []AlertRuleChannelConfig
	if err := common.UnmarshalJsonStr(r.ChannelConfigs, &configs); err != nil {
		return nil, err
	}
	if configs == nil {
		return []AlertRuleChannelConfig{}, nil
	}
	return configs, nil
}

func (r *AlertRule) normalize() error {
	if strings.TrimSpace(r.Name) == "" {
		r.Name = ""
	}
	if r.DedupeWindowSeconds < 0 {
		r.DedupeWindowSeconds = 0
	}
	if r.RiskTypes == "" {
		if err := r.SetRiskTypes(nil); err != nil {
			return err
		}
	} else {
		riskTypes, err := r.ParsedRiskTypes()
		if err != nil {
			return err
		}
		if err := r.SetRiskTypes(riskTypes); err != nil {
			return err
		}
	}
	if r.DepartmentIds == "" {
		if err := r.SetDepartmentIds(nil); err != nil {
			return err
		}
	} else {
		departmentIds, err := r.ParsedDepartmentIds()
		if err != nil {
			return err
		}
		if err := r.SetDepartmentIds(departmentIds); err != nil {
			return err
		}
	}
	if r.ChannelConfigs == "" {
		if err := r.SetChannelConfigs(nil); err != nil {
			return err
		}
	} else {
		configs, err := r.ParsedChannelConfigs()
		if err != nil {
			return err
		}
		if err := r.SetChannelConfigs(configs); err != nil {
			return err
		}
	}
	return nil
}
