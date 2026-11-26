package enterprise

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	minEnterpriseUsernameLength = 3
	maxUsernameGenerationTries  = 20
)

type enterpriseUsernameCandidate struct {
	Value      string
	PreferBase bool
}

func normalizeEnterpriseUsername(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastSeparator := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastSeparator = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastSeparator = false
		default:
			if builder.Len() == 0 || lastSeparator {
				continue
			}
			builder.WriteByte('_')
			lastSeparator = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func validateEnterpriseUsername(value string) error {
	value = strings.TrimSpace(value)
	if len(value) < minEnterpriseUsernameLength || len(value) > model.UserNameMaxLength {
		return ErrEnterpriseUsernameInvalid
	}
	if value != strings.ToLower(value) {
		return ErrEnterpriseUsernameInvalid
	}
	lastSeparator := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			lastSeparator = false
		case r >= '0' && r <= '9':
			lastSeparator = false
		case r == '_':
			if lastSeparator {
				return ErrEnterpriseUsernameInvalid
			}
			lastSeparator = true
		default:
			return ErrEnterpriseUsernameInvalid
		}
	}
	if strings.HasPrefix(value, "_") || strings.HasSuffix(value, "_") {
		return ErrEnterpriseUsernameInvalid
	}
	return nil
}

func firstReadableEnterpriseUsername(values ...string) string {
	for _, value := range values {
		normalized := normalizeEnterpriseUsername(value)
		if len(normalized) >= minEnterpriseUsernameLength {
			return normalized
		}
	}
	return "dingtalk_user"
}

func buildEnterpriseUsernameCandidates(base string, stableSuffixes ...string) []enterpriseUsernameCandidate {
	base = normalizeEnterpriseUsername(base)
	if len(base) > model.UserNameMaxLength {
		base = base[:model.UserNameMaxLength]
		base = strings.Trim(base, "_")
	}
	if len(base) < minEnterpriseUsernameLength {
		base = "dingtalk_user"
	}

	out := []enterpriseUsernameCandidate{{Value: base, PreferBase: true}}
	seen := map[string]struct{}{base: {}}
	appendCandidate := func(candidate string) {
		candidate = strings.Trim(candidate, "_")
		if len(candidate) < minEnterpriseUsernameLength || len(candidate) > model.UserNameMaxLength {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, enterpriseUsernameCandidate{Value: candidate})
	}

	for _, suffix := range stableSuffixes {
		suffix = normalizeEnterpriseUsername(suffix)
		if suffix == "" {
			continue
		}
		baseLen := model.UserNameMaxLength - len(suffix) - 1
		if baseLen < minEnterpriseUsernameLength {
			baseLen = minEnterpriseUsernameLength
		}
		candidateBase := base
		if len(candidateBase) > baseLen {
			candidateBase = strings.Trim(candidateBase[:baseLen], "_")
		}
		if len(candidateBase) < minEnterpriseUsernameLength {
			candidateBase = base
		}
		appendCandidate(candidateBase + "_" + suffix)
	}
	return out
}

func enterpriseUsernameExists(db *gorm.DB, username string) (bool, error) {
	var user model.User
	err := db.Unscoped().Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func maxUserIdForUsernameGeneration(db *gorm.DB) int {
	var user model.User
	if err := db.Unscoped().Order("id DESC").First(&user).Error; err != nil {
		return 0
	}
	return user.Id
}

func resolveAvailableEnterpriseUsername(db *gorm.DB, base string, stableSuffixes ...string) string {
	if db == nil {
		db = model.DB
	}
	candidates := buildEnterpriseUsernameCandidates(base, stableSuffixes...)
	for _, candidate := range candidates {
		if exists, err := enterpriseUsernameExists(db, candidate.Value); err == nil && !exists {
			return candidate.Value
		}
	}

	maxUserId := maxUserIdForUsernameGeneration(db)
	for i := 0; i < maxUsernameGenerationTries; i++ {
		suffix := strconv.Itoa(maxUserId + 1 + i)
		fallbackCandidates := buildEnterpriseUsernameCandidates(base, suffix)
		for _, candidate := range fallbackCandidates {
			if candidate.PreferBase {
				continue
			}
			if exists, err := enterpriseUsernameExists(db, candidate.Value); err == nil && !exists {
				return candidate.Value
			}
		}
	}
	return fmt.Sprintf("dingtalk_%d", time.Now().UnixNano()%100000000)
}
