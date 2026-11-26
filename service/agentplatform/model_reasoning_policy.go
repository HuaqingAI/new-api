package agentplatform

import (
	"errors"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
)

// GetModelReasoningPolicy reads the committed primary-database value so another
// instance does not serve an older OptionMap immediately after an admin save.
func GetModelReasoningPolicy() (setting.EffectiveModelReasoningPolicy, error) {
	var option model.Option
	err := model.DB.First(&option, &model.Option{Key: setting.ModelReasoningPolicyKey}).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return setting.EffectiveModelReasoningPolicy{}, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		option.Value = setting.DefaultModelReasoningPolicy
	}
	policy, _, err := setting.NormalizeModelReasoningPolicy(option.Value)
	if err != nil {
		return setting.EffectiveModelReasoningPolicy{}, err
	}
	return policy.Effective()
}
