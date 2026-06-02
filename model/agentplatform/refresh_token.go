package agentplatform

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrInvalidRefreshTokenBody = errors.New("agent platform refresh token body invalid")
)

type RefreshToken struct {
	Id               int        `json:"id" gorm:"primaryKey"`
	GrantId          string     `json:"grant_id" gorm:"type:varchar(40);index:idx_ap_refresh_grant;not null"`
	RefreshTokenHash string     `json:"refresh_token_hash" gorm:"type:varchar(128);uniqueIndex:idx_ap_refresh_hash;not null"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RotatedFromId    *int       `json:"rotated_from_id" gorm:"index:idx_ap_refresh_rotated_from"`
	RevokedAt        *time.Time `json:"revoked_at"`
	LastUsedAt       *time.Time `json:"last_used_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (RefreshToken) TableName() string {
	return "agent_platform_refresh_tokens"
}

func (t *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	return t.applyDefaultsAndValidate()
}

func (t *RefreshToken) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return t.applyDefaultsAndValidate()
}

func (t *RefreshToken) applyDefaultsAndValidate() error {
	t.GrantId = strings.TrimSpace(t.GrantId)
	t.RefreshTokenHash = strings.TrimSpace(t.RefreshTokenHash)
	if t.GrantId == "" || t.RefreshTokenHash == "" || t.ExpiresAt.IsZero() {
		return ErrInvalidRefreshTokenBody
	}
	return nil
}
