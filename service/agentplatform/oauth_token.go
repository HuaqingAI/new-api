package agentplatform

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const (
	accessTokenTTL  = 5 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

var (
	ErrInvalidTokenExchangeInput = errors.New("agent platform token exchange input invalid")
	ErrAuthorizationCodeInvalid  = errors.New("agent platform authorization code invalid")
	ErrRefreshTokenInvalid       = errors.New("agent platform refresh token invalid")
	ErrGrantRevoked              = errors.New("agent platform grant revoked")
)

type TokenExchangeInput struct {
	ClientId     string
	GrantType    string
	Code         string
	CodeVerifier string
	RedirectURI  string
	RefreshToken string
	Scope        string
}

type RevokeTokenInput struct {
	ClientId      string
	Token         string
	TokenTypeHint string
	ActorUserID   int
	RequestID     string
}

type TokenExchangeResult struct {
	AccessToken      string
	TokenType        string
	ExpiresIn        int
	RefreshToken     string
	RefreshExpiresIn int
	Scope            string
	ContractVersion  string
	GrantId          string
}

type TokenClaims struct {
	ClientId        string `json:"client_id"`
	UserId          *int   `json:"user_id,omitempty"`
	Scope           string `json:"scope"`
	ContractVersion string `json:"contract_version"`
	TokenVersion    int    `json:"token_version"`
	GrantId         string `json:"grant_id"`
	jwt.RegisteredClaims
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
	input.GrantType = normalizeGrantType(input.GrantType)
	input.Code = strings.TrimSpace(input.Code)
	input.CodeVerifier = strings.TrimSpace(input.CodeVerifier)
	input.RedirectURI = strings.TrimSpace(input.RedirectURI)
	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	input.Scope = strings.TrimSpace(input.Scope)
	if input.ClientId == "" {
		return TokenExchangeResult{}, ErrInvalidTokenExchangeInput
	}

	var client apmodel.Client
	if err := s.db.Where("client_id = ?", input.ClientId).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TokenExchangeResult{}, ErrUnauthorizedClient
		}
		return TokenExchangeResult{}, err
	}
	if !clientAuthorized(client) {
		return TokenExchangeResult{}, ErrUnauthorizedClient
	}

	switch input.GrantType {
	case "", "authorization_code":
		return s.exchangeAuthorizationCode(client, input)
	case "refresh_token":
		return s.exchangeRefreshToken(client, input)
	case "client_credentials":
		return s.exchangeClientCredentials(client, input)
	default:
		return TokenExchangeResult{}, ErrInvalidTokenExchangeInput
	}
}

func (s *OAuthTokenService) Revoke(input RevokeTokenInput) error {
	if s == nil || s.db == nil {
		return ErrInvalidTokenExchangeInput
	}
	input.ClientId = strings.TrimSpace(input.ClientId)
	input.Token = strings.TrimSpace(input.Token)
	input.TokenTypeHint = strings.TrimSpace(strings.ToLower(input.TokenTypeHint))
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.ClientId == "" || input.Token == "" {
		return ErrInvalidTokenExchangeInput
	}

	var client apmodel.Client
	if err := s.db.Where("client_id = ?", input.ClientId).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUnauthorizedClient
		}
		return err
	}

	now := time.Now().UTC()
	if input.TokenTypeHint == "refresh_token" || strings.HasPrefix(input.Token, "rt_") {
		hash := hashRefreshToken(input.Token)
		var refresh apmodel.RefreshToken
		if err := s.db.Where("refresh_token_hash = ?", hash).First(&refresh).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRefreshTokenInvalid
			}
			return err
		}
		var grant apmodel.AuthorizationGrant
		if err := s.db.Where("grant_id = ? AND client_id = ?", refresh.GrantId, input.ClientId).First(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRefreshTokenInvalid
			}
			return err
		}
		return s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&apmodel.RefreshToken{}).
				Where("id = ? AND revoked_at IS NULL", refresh.Id).
				Update("revoked_at", now).Error; err != nil {
				return err
			}
			return s.revokeGrant(tx, grant, input.ActorUserID, input.RequestID, now, false)
		})
	}

	claims, err := parseAccessToken(input.Token)
	if err != nil {
		return ErrInvalidTokenExchangeInput
	}
	if claims.ClientId != input.ClientId || strings.TrimSpace(claims.GrantId) == "" {
		return ErrInvalidTokenExchangeInput
	}
	var grant apmodel.AuthorizationGrant
	if err := s.db.Where("grant_id = ? AND client_id = ?", claims.GrantId, input.ClientId).First(&grant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAuthorizationCodeInvalid
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.revokeGrant(tx, grant, input.ActorUserID, input.RequestID, now, true)
	})
}

func (s *OAuthTokenService) ValidateAccessToken(accessToken string) (TokenClaims, error) {
	claims, err := parseAccessToken(accessToken)
	if err != nil {
		return TokenClaims{}, err
	}

	var client apmodel.Client
	if err := s.db.Where("client_id = ?", claims.ClientId).First(&client).Error; err != nil {
		return TokenClaims{}, err
	}
	if !clientAuthorized(client) {
		return TokenClaims{}, ErrUnauthorizedClient
	}

	var grant apmodel.AuthorizationGrant
	if err := s.db.Where("grant_id = ?", claims.GrantId).First(&grant).Error; err != nil {
		return TokenClaims{}, err
	}
	if grant.RevokedAt != nil || grant.TokenVersion != claims.TokenVersion {
		return TokenClaims{}, ErrGrantRevoked
	}
	return claims, nil
}

func (s *OAuthTokenService) exchangeAuthorizationCode(client apmodel.Client, input TokenExchangeInput) (TokenExchangeResult, error) {
	if input.Code == "" || input.CodeVerifier == "" || input.RedirectURI == "" {
		return TokenExchangeResult{}, ErrInvalidTokenExchangeInput
	}
	if !redirectAllowed(client.RedirectURIsJSON, input.RedirectURI) || !grantAllowed(client.AllowedGrantTypesJSON, "authorization_code") {
		return TokenExchangeResult{}, ErrUnauthorizedClient
	}

	var grant apmodel.AuthorizationGrant
	if err := s.db.Where("client_id = ? AND authorization_code = ?", input.ClientId, input.Code).First(&grant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TokenExchangeResult{}, ErrAuthorizationCodeInvalid
		}
		return TokenExchangeResult{}, err
	}
	if time.Now().UTC().After(grant.CodeExpiresAt) || grant.Status != "authorized" || grant.RevokedAt != nil {
		return TokenExchangeResult{}, ErrAuthorizationCodeInvalid
	}
	if grant.CodeMethod != "S256" || pkceChallenge(input.CodeVerifier) != grant.CodeChallenge {
		return TokenExchangeResult{}, ErrAuthorizationCodeInvalid
	}
	return s.issueGrantTokens(client, grant, nil)
}

func (s *OAuthTokenService) exchangeRefreshToken(client apmodel.Client, input TokenExchangeInput) (TokenExchangeResult, error) {
	if input.RefreshToken == "" {
		return TokenExchangeResult{}, ErrInvalidTokenExchangeInput
	}

	refreshHash := hashRefreshToken(input.RefreshToken)
	var refresh apmodel.RefreshToken
	if err := s.db.Where("refresh_token_hash = ?", refreshHash).First(&refresh).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TokenExchangeResult{}, ErrRefreshTokenInvalid
		}
		return TokenExchangeResult{}, err
	}

	var grant apmodel.AuthorizationGrant
	if err := s.db.Where("grant_id = ? AND client_id = ?", refresh.GrantId, client.ClientId).First(&grant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TokenExchangeResult{}, ErrRefreshTokenInvalid
		}
		return TokenExchangeResult{}, err
	}
	now := time.Now().UTC()
	if refresh.RevokedAt != nil || refresh.ExpiresAt.Before(now) || grant.RevokedAt != nil || grant.Status == "revoked" {
		return TokenExchangeResult{}, ErrRefreshTokenInvalid
	}

	var result TokenExchangeResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		newToken, newHash, err := generateOpaqueToken("rt_")
		if err != nil {
			return err
		}
		newRefresh := apmodel.RefreshToken{
			GrantId:          grant.GrantId,
			RefreshTokenHash: newHash,
			ExpiresAt:        now.Add(refreshTokenTTL),
			RotatedFromId:    &refresh.Id,
		}
		if err := tx.Create(&newRefresh).Error; err != nil {
			return err
		}
		if err := tx.Model(&apmodel.RefreshToken{}).
			Where("id = ?", refresh.Id).
			Updates(map[string]any{
				"revoked_at":   now,
				"last_used_at": now,
			}).Error; err != nil {
			return err
		}

		issued, issueErr := s.issueGrantTokensWithRefresh(tx, client, grant, &newRefresh, newToken)
		if issueErr != nil {
			return issueErr
		}
		result = issued
		return nil
	})
	return result, err
}

func (s *OAuthTokenService) exchangeClientCredentials(client apmodel.Client, input TokenExchangeInput) (TokenExchangeResult, error) {
	if !client.AllowClientCredentials || !grantAllowed(client.AllowedGrantTypesJSON, "client_credentials") {
		return TokenExchangeResult{}, ErrUnauthorizedClient
	}
	scopeText := input.Scope
	if scopeText == "" {
		scopeText = defaultScopeFromAllowed(client.AllowedScopesJSON)
	}
	if scopeText == "" || !scopeAllowed(client.AllowedScopesJSON, scopeText) {
		return TokenExchangeResult{}, ErrUnauthorizedClient
	}

	now := time.Now().UTC()
	grant := apmodel.AuthorizationGrant{
		ClientId:          client.ClientId,
		GrantType:         "client_credentials",
		ScopeText:         scopeText,
		Status:            "issued",
		ContractVersion:   client.ContractVersion,
		CodeChallenge:     "n/a",
		CodeMethod:        "NONE",
		AuthorizationCode: "client_credentials",
		CodeExpiresAt:     now,
		ConsentedAt:       nil,
	}
	if err := s.db.Create(&grant).Error; err != nil {
		if errors.Is(err, apmodel.ErrInvalidAuthorizationGrantBody) {
			return TokenExchangeResult{}, ErrInvalidTokenExchangeInput
		}
		return TokenExchangeResult{}, err
	}
	return s.issueGrantTokens(client, grant, &now)
}

func (s *OAuthTokenService) issueGrantTokens(client apmodel.Client, grant apmodel.AuthorizationGrant, refreshCreatedAt *time.Time) (TokenExchangeResult, error) {
	var result TokenExchangeResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		if refreshCreatedAt == nil {
			refreshCreatedAt = &now
		}
		refreshToken, refreshHash, err := generateOpaqueToken("rt_")
		if err != nil {
			return err
		}
		refresh := apmodel.RefreshToken{
			GrantId:          grant.GrantId,
			RefreshTokenHash: refreshHash,
			ExpiresAt:        refreshCreatedAt.Add(refreshTokenTTL),
		}
		if err := tx.Create(&refresh).Error; err != nil {
			return err
		}
		if err := tx.Model(&apmodel.AuthorizationGrant{}).
			Where("id = ?", grant.Id).
			Updates(map[string]any{"status": "issued"}).Error; err != nil {
			return err
		}
		updatedGrant := grant
		updatedGrant.Status = "issued"
		issued, err := s.issueGrantTokensWithRefresh(tx, client, updatedGrant, &refresh, refreshToken)
		if err != nil {
			return err
		}
		result = issued
		return nil
	})
	return result, err
}

func (s *OAuthTokenService) issueGrantTokensWithRefresh(tx *gorm.DB, client apmodel.Client, grant apmodel.AuthorizationGrant, refresh *apmodel.RefreshToken, refreshToken string) (TokenExchangeResult, error) {
	accessToken, err := generateJWTAccessToken(grant)
	if err != nil {
		return TokenExchangeResult{}, err
	}
	return TokenExchangeResult{
		AccessToken:      accessToken,
		TokenType:        "Bearer",
		ExpiresIn:        int(accessTokenTTL / time.Second),
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int(time.Until(refresh.ExpiresAt).Seconds()),
		Scope:            grant.ScopeText,
		ContractVersion:  grant.ContractVersion,
		GrantId:          grant.GrantId,
	}, nil
}

func (s *OAuthTokenService) revokeGrant(tx *gorm.DB, grant apmodel.AuthorizationGrant, actorUserID int, requestID string, now time.Time, incrementVersion bool) error {
	updates := map[string]any{
		"status":     "revoked",
		"revoked_at": now,
	}
	if incrementVersion {
		updates["token_version"] = grant.TokenVersion + 1
	}
	if err := tx.Model(&apmodel.AuthorizationGrant{}).
		Where("id = ?", grant.Id).
		Updates(updates).Error; err != nil {
		return err
	}
	if err := tx.Model(&apmodel.RefreshToken{}).
		Where("grant_id = ? AND revoked_at IS NULL", grant.GrantId).
		Update("revoked_at", now).Error; err != nil {
		return err
	}
	if actorUserID > 0 {
		payload, err := common.Marshal(map[string]any{
			"grant_id":      grant.GrantId,
			"client_id":     grant.ClientId,
			"grant_type":    grant.GrantType,
			"token_version": grant.TokenVersion,
		})
		if err != nil {
			return err
		}
		action := apmodel.AdminAction{
			ActorUserId:   actorUserID,
			ActionType:    "agentplatform.oauth.revoke",
			ObjectType:    "grant",
			ObjectId:      grant.GrantId,
			BeforeStatus:  grant.Status,
			AfterStatus:   "revoked",
			TargetVersion: "",
			RequestId:     requestID,
			Result:        "success",
			ErrorSummary:  "",
			Payload:       string(payload),
		}
		if err := tx.Create(&action).Error; err != nil {
			return err
		}
	}
	return nil
}

func clientAuthorized(client apmodel.Client) bool {
	return client.Status == "active" && strings.TrimSpace(client.ContractVersion) != "" && strings.TrimSpace(client.CapabilitiesJSON) != ""
}

func normalizeGrantType(grantType string) string {
	grantType = strings.TrimSpace(strings.ToLower(grantType))
	if grantType == "" {
		return "authorization_code"
	}
	return grantType
}

func defaultScopeFromAllowed(raw string) string {
	values, err := parseStringArray(raw)
	if err != nil || len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " ")
}

func generateOpaqueToken(prefix string) (string, string, error) {
	key, err := common.GenerateRandomCharsKey(48)
	if err != nil {
		return "", "", err
	}
	token := prefix + key
	return token, hashRefreshToken(token), nil
}

func hashRefreshToken(token string) string {
	return common.GenerateHMAC(token)
}

func generateJWTAccessToken(grant apmodel.AuthorizationGrant) (string, error) {
	now := time.Now().UTC()
	jti, err := apmodel.GenerateGrantID()
	if err != nil {
		return "", err
	}
	claims := TokenClaims{
		ClientId:        grant.ClientId,
		UserId:          grant.UserId,
		Scope:           grant.ScopeText,
		ContractVersion: grant.ContractVersion,
		TokenVersion:    grant.TokenVersion,
		GrantId:         grant.GrantId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "new-api/agent-platform",
			Subject:   grant.ClientId,
			Audience:  jwt.ClaimStrings{grant.ClientId},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(common.CryptoSecret))
}

func parseAccessToken(accessToken string) (TokenClaims, error) {
	tokenText := strings.TrimSpace(accessToken)
	tokenText = strings.TrimPrefix(tokenText, "Bearer ")
	tokenText = strings.TrimPrefix(tokenText, "bearer ")
	parsed, err := jwt.ParseWithClaims(tokenText, &TokenClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(common.CryptoSecret), nil
	})
	if err != nil {
		return TokenClaims{}, err
	}
	claims, ok := parsed.Claims.(*TokenClaims)
	if !ok || !parsed.Valid {
		return TokenClaims{}, ErrInvalidTokenExchangeInput
	}
	return *claims, nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
