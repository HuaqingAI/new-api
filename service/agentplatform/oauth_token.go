package agentplatform

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrInvalidTokenExchangeInput = errors.New("agent platform token exchange input invalid")
	ErrAuthorizationCodeInvalid  = errors.New("agent platform authorization code invalid")
)

type TokenExchangeInput struct {
	ClientId     string
	Code         string
	CodeVerifier string
	RedirectURI  string
}

type TokenExchangeResult struct {
	AccessToken     string
	TokenType       string
	ExpiresIn       int
	Scope           string
	ContractVersion string
}

type OAuthTokenService struct {
	db *gorm.DB
}

func NewOAuthTokenService(db *gorm.DB) *OAuthTokenService {
	return &OAuthTokenService{db: db}
}

func (s *OAuthTokenService) Exchange(input TokenExchangeInput) (TokenExchangeResult, error) {
	if s == nil || s.db == nil {
		return TokenExchangeResult{}, ErrInvalidTokenExchangeInput
	}
	input.ClientId = strings.TrimSpace(input.ClientId)
	input.Code = strings.TrimSpace(input.Code)
	input.CodeVerifier = strings.TrimSpace(input.CodeVerifier)
	input.RedirectURI = strings.TrimSpace(input.RedirectURI)
	if input.ClientId == "" || input.Code == "" || input.CodeVerifier == "" || input.RedirectURI == "" {
		return TokenExchangeResult{}, ErrInvalidTokenExchangeInput
	}

	var client apmodel.Client
	if err := s.db.Where("client_id = ?", input.ClientId).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TokenExchangeResult{}, ErrUnauthorizedClient
		}
		return TokenExchangeResult{}, err
	}
	if !redirectAllowed(client.RedirectURIsJSON, input.RedirectURI) {
		return TokenExchangeResult{}, ErrUnauthorizedClient
	}

	var grant apmodel.AuthorizationGrant
	if err := s.db.Where("client_id = ? AND authorization_code = ?", input.ClientId, input.Code).First(&grant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TokenExchangeResult{}, ErrAuthorizationCodeInvalid
		}
		return TokenExchangeResult{}, err
	}
	if time.Now().UTC().After(grant.CodeExpiresAt) || grant.Status != "authorized" {
		return TokenExchangeResult{}, ErrAuthorizationCodeInvalid
	}
	if grant.CodeMethod != "S256" || pkceChallenge(input.CodeVerifier) != grant.CodeChallenge {
		return TokenExchangeResult{}, ErrAuthorizationCodeInvalid
	}

	accessToken, err := generateAccessToken()
	if err != nil {
		return TokenExchangeResult{}, err
	}
	if err := s.db.Model(&grant).Updates(map[string]any{"status": "issued"}).Error; err != nil {
		return TokenExchangeResult{}, err
	}

	return TokenExchangeResult{
		AccessToken:     "at_" + accessToken,
		TokenType:       "Bearer",
		ExpiresIn:       300,
		Scope:           grant.ScopeText,
		ContractVersion: grant.ContractVersion,
	}, nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func generateAccessToken() (string, error) {
	key, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return "", err
	}
	return "at_" + key, nil
}
