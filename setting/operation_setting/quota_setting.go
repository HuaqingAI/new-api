package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type QuotaSetting struct {
	EnableFreeModelPreConsume         bool `json:"enable_free_model_pre_consume"`        // 是否对免费模型启用预消耗
	EnterpriseBudgetWarningThreshold  int  `json:"enterprise_budget_warning_threshold"`  // 企业预算预警阈值
	EnterpriseBudgetCriticalThreshold int  `json:"enterprise_budget_critical_threshold"` // 企业预算紧急阈值
}

// 默认配置
var quotaSetting = QuotaSetting{
	EnableFreeModelPreConsume:         true,
	EnterpriseBudgetWarningThreshold:  80,
	EnterpriseBudgetCriticalThreshold: 95,
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("quota_setting", &quotaSetting)
}

func GetQuotaSetting() *QuotaSetting {
	return &quotaSetting
}

func IsEnterpriseBudgetThresholdsValid(warning, critical int) bool {
	return warning > 0 && warning < critical && critical <= 100
}
