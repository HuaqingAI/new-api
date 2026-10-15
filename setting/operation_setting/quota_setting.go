package operation_setting

import (
	"fmt"
	"math"
	"strconv"

	"github.com/QuantumNous/new-api/setting/config"
)

type QuotaSetting struct {
	EnableFreeModelPreConsume         bool    `json:"enable_free_model_pre_consume"`
	EnterpriseBudgetWarningThreshold  int     `json:"enterprise_budget_warning_threshold"`
	EnterpriseBudgetCriticalThreshold int     `json:"enterprise_budget_critical_threshold"`
	TrustQuotaUSD                     float64 `json:"trust_quota_usd"`
	PreConsumeMultiplier              float64 `json:"pre_consume_multiplier"`
}

var quotaSetting = QuotaSetting{
	EnableFreeModelPreConsume:         true,
	EnterpriseBudgetWarningThreshold:  80,
	EnterpriseBudgetCriticalThreshold: 95,
	TrustQuotaUSD:                     10,
	PreConsumeMultiplier:              1,
}

func init() {
	config.GlobalConfig.Register("quota_setting", &quotaSetting)
}

func GetQuotaSetting() *QuotaSetting {
	return &quotaSetting
}

func IsEnterpriseBudgetThresholdsValid(warning, critical int) bool {
	return warning > 0 && warning < critical && critical <= 100
}

// ValidateQuotaOption validates reservation settings before they are persisted.
func ValidateQuotaOption(key, value string) error {
	if key != "quota_setting.trust_quota_usd" && key != "quota_setting.pre_consume_multiplier" {
		return nil
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
		return fmt.Errorf("%s must be a finite non-negative number", key)
	}
	if key == "quota_setting.pre_consume_multiplier" && number == 0 {
		return fmt.Errorf("%s must be greater than zero", key)
	}
	return nil
}

// InputPreConsumeMultiplier rejects invalid runtime settings as well as invalid saves.
func InputPreConsumeMultiplier() (float64, error) {
	multiplier := quotaSetting.PreConsumeMultiplier
	if multiplier <= 0 || math.IsNaN(multiplier) || math.IsInf(multiplier, 0) {
		return 0, fmt.Errorf("pre-consume multiplier must be a finite number greater than zero")
	}
	return multiplier, nil
}
