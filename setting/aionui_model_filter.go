package setting

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

const (
	ClientModelFilterExcludedModelsKey = "AionUiModelFilterExcludedModels"
	MaxClientModelFilterModels         = 1000
	MaxClientModelFilterModelLength    = 255
	MaxClientModelFilterBytes          = 60 * 1024
)

// NormalizeClientModelFilter validates exact model IDs and returns their canonical JSON.
func NormalizeClientModelFilter(raw string) ([]string, string, error) {
	if !utf8.ValidString(raw) {
		return nil, "", fmt.Errorf("client model filter must contain valid UTF-8")
	}
	var values []any
	if err := common.UnmarshalJsonStr(raw, &values); err != nil || values == nil {
		return nil, "", fmt.Errorf("client model filter must be an array of model IDs")
	}
	if len(values) > MaxClientModelFilterModels {
		return nil, "", fmt.Errorf("client model filter cannot contain more than %d models", MaxClientModelFilterModels)
	}
	models := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		id, ok := value.(string)
		if !ok {
			return nil, "", fmt.Errorf("client model filter must contain only string model IDs")
		}
		for _, r := range id {
			if r < 0x20 || r == 0x7f {
				return nil, "", fmt.Errorf("client model IDs cannot contain control characters")
			}
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if utf8.RuneCountInString(id) > MaxClientModelFilterModelLength {
			return nil, "", fmt.Errorf("client model IDs cannot exceed %d characters", MaxClientModelFilterModelLength)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		models = append(models, id)
	}
	slices.Sort(models)
	encoded, err := common.Marshal(models)
	if err != nil {
		return nil, "", err
	}
	if len(encoded) > MaxClientModelFilterBytes {
		return nil, "", fmt.Errorf("client model filter cannot exceed %d bytes", MaxClientModelFilterBytes)
	}
	return models, string(encoded), nil
}

func GetClientModelFilterExcludedModels() ([]string, error) {
	common.OptionMapRWMutex.RLock()
	raw, exists := common.OptionMap[ClientModelFilterExcludedModelsKey]
	common.OptionMapRWMutex.RUnlock()
	if !exists {
		return []string{}, nil
	}
	models, _, err := NormalizeClientModelFilter(raw)
	return models, err
}
