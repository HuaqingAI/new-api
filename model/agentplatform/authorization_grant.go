package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrInvalidAuthorizationGrantBody = errors.New("agent platform authorization grant body invalid")
)

type AuthorizationGrant struct {
	Id               int        `json:"id" gorm:"primaryKey"`
	GrantId          string     `json:"grant_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_grant_id;not null"`
	ClientId         string     `json:"client_id" gorm:"type:varchar(40);index:idx_ap_grant_client;not null"`
	UserId           *int       `json:"user_id" gorm:"index:idx_ap_grant_user"`
	GrantType        string     `json:"grant_type" gorm:"type:varchar(32);not null"`
	ScopeText        string     `json:"scope_text" gorm:"type:text;not null"`
	Status           string     `json:"status" gorm:"type:varchar(32);index:idx_ap_grant_status;not null"`
	TokenVersion     int        `json:"token_version" gorm:"not null;default:1"`
	ContractVersion  string     `json:"contract_version" gorm:"type:varchar(64);not null"`
	CodeChallenge    string     `json:"code_challenge" gorm:"type:varchar(128);not null"`
	CodeMethod       string     `json:"code_method" gorm:"type:varchar(16);not null"`
	AuthorizationCode string    `json:"authorization_code" gorm:"type:varchar(128);not null"`
	CodeExpiresAt    time.Time  `json:"code_expires_at"`
	ConsentedAt      *time.Time `json:"consented_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (AuthorizationGrant) TableName() string {
	return "agent_platform_authorization_grants"
}

func (g *AuthorizationGrant) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(g.GrantId) == "" {
		grantID, err := GenerateGrantID()
		if err != nil {
			return err
		}
		g.GrantId = grantID
	}
	return g.applyDefaultsAndValidate()
}

func (g *AuthorizationGrant) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	if g.Id == 0 && strings.TrimSpace(g.GrantId) == "" {
		grantID, err := GenerateGrantID()
		if err != nil {
			return err
		}
		g.GrantId = grantID
	}
	return g.applyDefaultsAndValidate()
}

func (g *AuthorizationGrant) applyDefaultsAndValidate() error {
	g.GrantId = strings.TrimSpace(g.GrantId)
	g.ClientId = strings.TrimSpace(g.ClientId)
	g.GrantType = strings.TrimSpace(strings.ToLower(g.GrantType))
	g.ScopeText = strings.TrimSpace(g.ScopeText)
	g.Status = strings.TrimSpace(strings.ToLower(g.Status))
	g.ContractVersion = strings.TrimSpace(g.ContractVersion)
	g.CodeChallenge = strings.TrimSpace(g.CodeChallenge)
	g.CodeMethod = strings.TrimSpace(strings.ToUpper(g.CodeMethod))
	g.AuthorizationCode = strings.TrimSpace(g.AuthorizationCode)

	if g.Status == "" {
		g.Status = "pending"
	}
	if g.TokenVersion <= 0 {
		g.TokenVersion = 1
	}
	if g.GrantId == "" || g.ClientId == "" || g.GrantType == "" || g.ScopeText == "" || g.ContractVersion == "" || g.CodeChallenge == "" || g.CodeMethod == "" || g.AuthorizationCode == "" || g.CodeExpiresAt.IsZero() {
		return ErrInvalidAuthorizationGrantBody
	}
	return nil
}

func GenerateGrantID() (string, error) {
	key, err := common.GenerateRandomCharsKey(29)
	if err != nil {
		return "", err
	}
	return "grt_" + key, nil
}
