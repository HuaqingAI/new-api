package aionui

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PersonalAPIKey struct {
	Name      string
	Key       string
	MaskedKey string
}

// GetPersonalAPIKeyGroup returns the current pricing group of the desktop user's
// personal API key. The group can be changed from the admin token settings, so
// it must be read from the database instead of relying on the login request.
func GetPersonalAPIKeyGroup(userID int) (string, error) {
	if model.DB == nil || userID <= 0 {
		return "", ErrUserUnavailable
	}

	var token model.Token
	err := model.DB.Where("user_id = ? AND name = ? AND status = ?", userID, DefaultPersonalAPIKeyName, common.TokenStatusEnabled).
		Order("id ASC").
		First(&token).Error
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(token.Group), nil
}

func EnsurePersonalAPIKey(userID int, requestedGroup string) (PersonalAPIKey, error) {
	if model.DB == nil || userID <= 0 {
		return PersonalAPIKey{}, ErrUserUnavailable
	}
	group := strings.TrimSpace(requestedGroup)
	if group != HTHBuddyPersonalAPIKeyGroup {
		group = DefaultPersonalAPIKeyGroup
	}
	var token model.Token
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND name = ? AND status = ?", userID, DefaultPersonalAPIKeyName, common.TokenStatusEnabled).
			Order("id ASC").
			First(&token).Error; err == nil {
			return nil
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		key, err := common.GenerateKey()
		if err != nil {
			return err
		}
		token = model.Token{
			UserId:             userID,
			Name:               DefaultPersonalAPIKeyName,
			Key:                key,
			Status:             common.TokenStatusEnabled,
			CreatedTime:        common.GetTimestamp(),
			AccessedTime:       common.GetTimestamp(),
			ExpiredTime:        -1,
			RemainQuota:        0,
			UnlimitedQuota:     true,
			ModelLimitsEnabled: false,
			ModelLimits:        "",
			Group:              group,
		}
		return tx.Create(&token).Error
	})
	if err != nil {
		return PersonalAPIKey{}, err
	}
	return PersonalAPIKey{
		Name:      token.Name,
		Key:       exportedPersonalAPIKey(token.Key),
		MaskedKey: model.MaskTokenKey(exportedPersonalAPIKey(token.Key)),
	}, nil
}

func exportedPersonalAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(key, "sk-") {
		return key
	}
	return "sk-" + key
}
