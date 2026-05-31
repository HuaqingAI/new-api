package agentplatform

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthTokenServiceForTest(t *testing.T) (*OAuthTokenService, *gorm.DB, apmodel.Client, string) {
	t.Helper()

	common.CryptoSecret = "agent-platform-test-secret"
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:                   "cherry-studio",
		DisplayName:            "Cherry Studio",
		ClientType:             "desktop",
		Status:                 "active",
		AllowedGrantTypesJSON:  `["authorization_code","refresh_token"]`,
		RedirectURIsJSON:       `["https://example.com/callback"]`,
		AllowedScopesJSON:      `["skills.read"]`,
		ContractVersion:        "2026-06",
		CapabilitiesJSON:       `{"discovery":true}`,
		AllowClientCredentials: false,
	}
	require.NoError(t, db.Create(&client).Error)

	authorizeService := NewOAuthAuthorizeService(db)
	verifier := "verifier-1"
	result, err := authorizeService.Authorize(AuthorizeInput{
		ClientId:            client.ClientId,
		RedirectURI:         "https://example.com/callback",
		Scope:               "skills.read",
		State:               "state-1",
		CodeChallenge:       pkceChallenge(verifier),
		CodeChallengeMethod: "S256",
		UserId:              100,
	})
	require.NoError(t, err)
	return NewOAuthTokenService(db), db, client, result.AuthorizationCode + "|" + verifier
}

func TestOAuthTokenServiceExchangesAuthorizationCodeWithJWTAndHashedRefreshToken(t *testing.T) {
	svc, db, client, payload := newOAuthTokenServiceForTest(t)
	parts := strings.Split(payload, "|")

	result, err := svc.Exchange(TokenExchangeInput{
		ClientId:     client.ClientId,
		GrantType:    "authorization_code",
		Code:         parts[0],
		CodeVerifier: parts[1],
		RedirectURI:  "https://example.com/callback",
	})
	require.NoError(t, err)
	require.Equal(t, "Bearer", result.TokenType)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
	require.Equal(t, 300, result.ExpiresIn)

	claims, err := svc.ValidateAccessToken(result.AccessToken)
	require.NoError(t, err)
	require.Equal(t, client.ClientId, claims.ClientId)
	require.Equal(t, "skills.read", claims.Scope)
	require.Equal(t, result.GrantId, claims.GrantId)

	var grant apmodel.AuthorizationGrant
	require.NoError(t, db.Where("authorization_code = ?", parts[0]).First(&grant).Error)
	require.Equal(t, "issued", grant.Status)

	var refreshTokens []apmodel.RefreshToken
	require.NoError(t, db.Where("grant_id = ?", grant.GrantId).Find(&refreshTokens).Error)
	require.Len(t, refreshTokens, 1)
	require.NotEqual(t, result.RefreshToken, refreshTokens[0].RefreshTokenHash)
	require.Equal(t, hashRefreshToken(result.RefreshToken), refreshTokens[0].RefreshTokenHash)
}

func TestOAuthTokenServiceRotatesRefreshToken(t *testing.T) {
	svc, db, client, payload := newOAuthTokenServiceForTest(t)
	parts := strings.Split(payload, "|")

	initial, err := svc.Exchange(TokenExchangeInput{
		ClientId:     client.ClientId,
		Code:         parts[0],
		CodeVerifier: parts[1],
		RedirectURI:  "https://example.com/callback",
	})
	require.NoError(t, err)

	rotated, err := svc.Exchange(TokenExchangeInput{
		ClientId:     client.ClientId,
		GrantType:    "refresh_token",
		RefreshToken: initial.RefreshToken,
	})
	require.NoError(t, err)
	require.NotEqual(t, initial.RefreshToken, rotated.RefreshToken)
	require.NotEmpty(t, rotated.AccessToken)

	_, err = svc.Exchange(TokenExchangeInput{
		ClientId:     client.ClientId,
		GrantType:    "refresh_token",
		RefreshToken: initial.RefreshToken,
	})
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)

	var refreshTokens []apmodel.RefreshToken
	require.NoError(t, db.Order("id ASC").Find(&refreshTokens).Error)
	require.Len(t, refreshTokens, 2)
	require.NotNil(t, refreshTokens[0].RevokedAt)
	require.NotNil(t, refreshTokens[0].LastUsedAt)
	require.NotNil(t, refreshTokens[1].RotatedFromId)
}

func TestOAuthTokenServiceRevokeInvalidatesGrantAndAccessToken(t *testing.T) {
	svc, _, client, payload := newOAuthTokenServiceForTest(t)
	parts := strings.Split(payload, "|")

	tokenResult, err := svc.Exchange(TokenExchangeInput{
		ClientId:     client.ClientId,
		Code:         parts[0],
		CodeVerifier: parts[1],
		RedirectURI:  "https://example.com/callback",
	})
	require.NoError(t, err)

	require.NoError(t, svc.Revoke(RevokeTokenInput{
		ClientId:      client.ClientId,
		Token:         tokenResult.AccessToken,
		TokenTypeHint: "access_token",
		ActorUserID:   100,
		RequestID:     "req-1",
	}))

	_, err = svc.ValidateAccessToken(tokenResult.AccessToken)
	require.ErrorIs(t, err, ErrGrantRevoked)

	_, err = svc.Exchange(TokenExchangeInput{
		ClientId:     client.ClientId,
		GrantType:    "refresh_token",
		RefreshToken: tokenResult.RefreshToken,
	})
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)
}

func TestOAuthTokenServiceSupportsClientCredentialsOnlyWhenEnabledPerClient(t *testing.T) {
	common.CryptoSecret = "agent-platform-test-secret"
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	disabledClient := apmodel.Client{
		Slug:                  "disabled-bot",
		DisplayName:           "Disabled Bot",
		ClientType:            "server",
		Status:                "active",
		AllowedGrantTypesJSON: `["client_credentials"]`,
		AllowedScopesJSON:     `["ap.resources.read"]`,
		ContractVersion:       "2026-06",
		CapabilitiesJSON:      `{"discovery":true}`,
	}
	enabledClient := apmodel.Client{
		Slug:                   "enabled-bot",
		DisplayName:            "Enabled Bot",
		ClientType:             "server",
		Status:                 "active",
		AllowedGrantTypesJSON:  `["client_credentials"]`,
		AllowedScopesJSON:      `["ap.resources.read"]`,
		ContractVersion:        "2026-06",
		CapabilitiesJSON:       `{"discovery":true}`,
		AllowClientCredentials: true,
	}
	require.NoError(t, db.Create(&disabledClient).Error)
	require.NoError(t, db.Create(&enabledClient).Error)

	svc := NewOAuthTokenService(db)
	_, err = svc.Exchange(TokenExchangeInput{
		ClientId:  disabledClient.ClientId,
		GrantType: "client_credentials",
		Scope:     "ap.resources.read",
	})
	require.ErrorIs(t, err, ErrUnauthorizedClient)

	result, err := svc.Exchange(TokenExchangeInput{
		ClientId:  enabledClient.ClientId,
		GrantType: "client_credentials",
		Scope:     "ap.resources.read",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)

	claims, err := svc.ValidateAccessToken(result.AccessToken)
	require.NoError(t, err)
	require.Equal(t, enabledClient.ClientId, claims.ClientId)
	require.Nil(t, claims.UserId)
}

func TestOAuthTokenServiceRejectsWrongVerifier(t *testing.T) {
	svc, _, client, payload := newOAuthTokenServiceForTest(t)
	code := strings.Split(payload, "|")[0]

	_, err := svc.Exchange(TokenExchangeInput{
		ClientId:     client.ClientId,
		Code:         code,
		CodeVerifier: "wrong-verifier",
		RedirectURI:  "https://example.com/callback",
	})
	require.ErrorIs(t, err, ErrAuthorizationCodeInvalid)
}
