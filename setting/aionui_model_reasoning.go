package setting

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

const (
	ModelReasoningPolicyKey      = "AionUiModelReasoningPolicy"
	DefaultModelReasoningPolicy  = `{"schema_version":1,"rules":[]}`
	MaxModelReasoningPolicyRules = 1000
	MaxModelReasoningPolicyBytes = 60 * 1024
	MaxModelReasoningModelLength = 255
)

var (
	ErrModelReasoningPolicyInvalid  = common.NewMessage("Invalid model reasoning settings")
	ErrModelReasoningPolicyTooLarge = common.NewMessage("Model reasoning settings are too large")
	modelReasoningFamilies          = []string{"deepseek", "qwen", "glm", "gpt"}
	modelReasoningEfforts           = []string{"none", "low", "medium", "high", "xhigh", "max"}
)

type ModelReasoningRule struct {
	MatchType  string   `json:"match_type"`
	MatchValue string   `json:"match_value"`
	Efforts    []string `json:"efforts"`
}

type ModelReasoningPolicy struct {
	SchemaVersion int                  `json:"schema_version"`
	Rules         []ModelReasoningRule `json:"rules"`
}

type EffectiveModelReasoningPolicy struct {
	SchemaVersion  int                  `json:"schema_version"`
	Scope          string               `json:"scope"`
	Revision       string               `json:"revision"`
	DefaultEfforts []string             `json:"default_efforts"`
	Rules          []ModelReasoningRule `json:"rules"`
}

// NormalizeModelReasoningPolicy is shared by dedicated and generic option writes.
func NormalizeModelReasoningPolicy(raw string) (ModelReasoningPolicy, string, error) {
	invalid := ModelReasoningPolicy{}
	if !utf8.ValidString(raw) {
		return invalid, "", ErrModelReasoningPolicyInvalid
	}
	var fields map[string]common.RawMessage
	if common.UnmarshalJsonStr(raw, &fields) != nil || len(fields) != 2 {
		return invalid, "", ErrModelReasoningPolicyInvalid
	}
	// The JSON codec replaces unpaired UTF-16 escapes with U+FFFD. Reject them
	// before decoding strings, while retaining a literal valid U+FFFD model ID.
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		i++
		if raw[i] != 'u' {
			continue
		}
		code, _ := strconv.ParseUint(raw[i+1:i+5], 16, 16)
		i += 4
		if code >= 0xdc00 && code <= 0xdfff {
			return invalid, "", ErrModelReasoningPolicyInvalid
		}
		if code < 0xd800 || code > 0xdbff {
			continue
		}
		if len(raw) < i+7 || raw[i+1:i+3] != `\u` {
			return invalid, "", ErrModelReasoningPolicyInvalid
		}
		low, err := strconv.ParseUint(raw[i+3:i+7], 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return invalid, "", ErrModelReasoningPolicyInvalid
		}
		i += 6
	}
	var version int
	if common.Unmarshal(fields["schema_version"], &version) != nil || version != 1 {
		return invalid, "", ErrModelReasoningPolicyInvalid
	}
	var rules []map[string]common.RawMessage
	if common.Unmarshal(fields["rules"], &rules) != nil || rules == nil || len(rules) > MaxModelReasoningPolicyRules {
		return invalid, "", ErrModelReasoningPolicyInvalid
	}
	policy := ModelReasoningPolicy{SchemaVersion: 1, Rules: make([]ModelReasoningRule, 0, len(rules))}
	seen := make(map[string]struct{}, len(rules))
	for _, fields := range rules {
		var rule ModelReasoningRule
		if len(fields) != 3 || common.Unmarshal(fields["match_type"], &rule.MatchType) != nil ||
			common.Unmarshal(fields["match_value"], &rule.MatchValue) != nil ||
			common.Unmarshal(fields["efforts"], &rule.Efforts) != nil {
			return invalid, "", ErrModelReasoningPolicyInvalid
		}
		for _, r := range rule.MatchValue {
			if r < 0x20 || r == 0x7f {
				return invalid, "", ErrModelReasoningPolicyInvalid
			}
		}
		rule.MatchValue = strings.TrimSpace(rule.MatchValue)
		switch rule.MatchType {
		case "family":
			rule.MatchValue = strings.ToLower(rule.MatchValue)
			if !slices.Contains(modelReasoningFamilies, rule.MatchValue) {
				return invalid, "", ErrModelReasoningPolicyInvalid
			}
		case "model":
			if rule.MatchValue == "" || utf8.RuneCountInString(rule.MatchValue) > MaxModelReasoningModelLength {
				return invalid, "", ErrModelReasoningPolicyInvalid
			}
		default:
			return invalid, "", ErrModelReasoningPolicyInvalid
		}
		key := rule.MatchType + ":" + rule.MatchValue
		if _, duplicate := seen[key]; duplicate {
			return invalid, "", ErrModelReasoningPolicyInvalid
		}
		seen[key] = struct{}{}
		if len(rule.Efforts) == 0 || len(rule.Efforts) > len(modelReasoningEfforts) {
			return invalid, "", ErrModelReasoningPolicyInvalid
		}
		for _, effort := range rule.Efforts {
			if !slices.Contains(modelReasoningEfforts, effort) {
				return invalid, "", ErrModelReasoningPolicyInvalid
			}
		}
		efforts := make([]string, 0, len(rule.Efforts))
		for _, effort := range modelReasoningEfforts {
			if slices.Contains(rule.Efforts, effort) {
				efforts = append(efforts, effort)
			}
		}
		rule.Efforts = efforts
		policy.Rules = append(policy.Rules, rule)
	}
	slices.SortFunc(policy.Rules, func(a, b ModelReasoningRule) int {
		if a.MatchType != b.MatchType {
			return strings.Compare(a.MatchType, b.MatchType)
		}
		if a.MatchType == "family" {
			return slices.Index(modelReasoningFamilies, a.MatchValue) - slices.Index(modelReasoningFamilies, b.MatchValue)
		}
		return strings.Compare(a.MatchValue, b.MatchValue)
	})
	encoded, err := common.Marshal(policy)
	if err != nil {
		return invalid, "", err
	}
	if len(encoded) > MaxModelReasoningPolicyBytes {
		return invalid, "", ErrModelReasoningPolicyTooLarge
	}
	return policy, string(encoded), nil
}

func (policy ModelReasoningPolicy) Effective() (EffectiveModelReasoningPolicy, error) {
	defaults := []string{"low", "medium", "high"}
	content := struct {
		SchemaVersion  int                  `json:"schema_version"`
		DefaultEfforts []string             `json:"default_efforts"`
		Rules          []ModelReasoningRule `json:"rules"`
	}{policy.SchemaVersion, defaults, policy.Rules}
	encoded, err := common.Marshal(content)
	if err != nil {
		return EffectiveModelReasoningPolicy{}, err
	}
	return EffectiveModelReasoningPolicy{
		SchemaVersion:  policy.SchemaVersion,
		Scope:          "global",
		Revision:       fmt.Sprintf("sha256:%x", sha256.Sum256(encoded)),
		DefaultEfforts: defaults,
		Rules:          policy.Rules,
	}, nil
}
