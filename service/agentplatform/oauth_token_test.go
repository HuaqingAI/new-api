package agentplatform

import (
	"strings"
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthTokenServiceForTest(t *testing.T) (*OAuthTokenService, *gorm.DB, apmodel.Client, string) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:                  "cherry-studio",
		DisplayName:           "Cherry Studio",
		ClientType:            "desktop",
		Status:                "active",
		AllowedGrantTypesJSON: `["authorization_code"]`,
		RedirectURIsJSON:      `["https://example.com/callback"]`,
		AllowedScopesJSON:     `["skills.read"]`,
		ContractVersion:       "2026-06",
		CapabilitiesJSON:      `{"discovery":true}`,
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

func TestOAuthTokenServiceExchangesAuthorizationCode(t *testing.T) {
	svc, db, client, payload := newOAuthTokenServiceForTest(t)
	parts := strings.Split(payload, "|")
	result, err := svc.Exchange(TokenExchangeInput{
		ClientId:     client.ClientId,
		Code:         parts[0],
		CodeVerifier: parts[1],
		RedirectURI:  "https://example.com/callback",
	})
	require.NoError(t, err)
	require.Equal(t, "Bearer", result.TokenType)
	require.NotEmpty(t, result.AccessToken)

	var grant apmodel.AuthorizationGrant
	require.NoError(t, db.Where("authorization_code = ?", parts[0]).First(&grant).Error)
	require.Equal(t, "issued", grant.Status)
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
