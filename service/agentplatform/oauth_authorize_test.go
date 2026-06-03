package agentplatform

import (
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthAuthorizeServiceForTest(t *testing.T) (*OAuthAuthorizeService, *gorm.DB, apmodel.Client) {
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
		AllowedScopesJSON:     `["skills.read","knowledge.read"]`,
		ContractVersion:       "2026-06",
		CapabilitiesJSON:      `{"discovery":true}`,
	}
	require.NoError(t, db.Create(&client).Error)
	return NewOAuthAuthorizeService(db), db, client
}

func TestOAuthAuthorizeServiceCreatesGrantForValidPKCERequest(t *testing.T) {
	svc, db, client := newOAuthAuthorizeServiceForTest(t)

	result, err := svc.Authorize(AuthorizeInput{
		ClientId:            client.ClientId,
		RedirectURI:         "https://example.com/callback",
		Scope:               "skills.read knowledge.read",
		State:               "state-1",
		CodeChallenge:       pkceChallenge("verifier-1"),
		CodeChallengeMethod: "S256",
		UserId:              100,
	})
	require.NoError(t, err)
	require.Equal(t, client.ClientId, result.ClientId)
	require.True(t, result.ConsentRecorded)
	require.NotEmpty(t, result.AuthorizationCode)

	var grants []apmodel.AuthorizationGrant
	require.NoError(t, db.Find(&grants).Error)
	require.Len(t, grants, 1)
	require.Equal(t, "authorization_code", grants[0].GrantType)
}

func TestOAuthAuthorizeServiceRejectsInvalidScopeOrRedirect(t *testing.T) {
	svc, _, client := newOAuthAuthorizeServiceForTest(t)

	_, err := svc.Authorize(AuthorizeInput{
		ClientId:            client.ClientId,
		RedirectURI:         "https://wrong.example.com/callback",
		Scope:               "skills.read",
		State:               "state-2",
		CodeChallenge:       pkceChallenge("verifier-2"),
		CodeChallengeMethod: "S256",
		UserId:              100,
	})
	require.ErrorIs(t, err, ErrUnauthorizedClient)

	_, err = svc.Authorize(AuthorizeInput{
		ClientId:            client.ClientId,
		RedirectURI:         "https://example.com/callback",
		Scope:               "skills.write",
		State:               "state-3",
		CodeChallenge:       pkceChallenge("verifier-3"),
		CodeChallengeMethod: "S256",
		UserId:              100,
	})
	require.ErrorIs(t, err, ErrUnauthorizedClient)
}

func TestOAuthAuthorizeServiceRejectsPlainPKCEAndInactiveOrInvalidClient(t *testing.T) {
	svc, db, client := newOAuthAuthorizeServiceForTest(t)

	_, err := svc.Authorize(AuthorizeInput{
		ClientId:            client.ClientId,
		RedirectURI:         "https://example.com/callback",
		Scope:               "skills.read",
		State:               "state-4",
		CodeChallenge:       pkceChallenge("verifier-4"),
		CodeChallengeMethod: "plain",
		UserId:              100,
	})
	require.ErrorIs(t, err, ErrInvalidAuthorizeInput)

	inactive := client
	inactive.Id = 0
	inactive.ClientId = ""
	inactive.Slug = "inactive-client"
	inactive.Status = "inactive"
	require.NoError(t, db.Create(&inactive).Error)
	_, err = svc.Authorize(AuthorizeInput{
		ClientId:            inactive.ClientId,
		RedirectURI:         "https://example.com/callback",
		Scope:               "skills.read",
		State:               "state-5",
		CodeChallenge:       pkceChallenge("verifier-5"),
		CodeChallengeMethod: "S256",
		UserId:              100,
	})
	require.ErrorIs(t, err, ErrUnauthorizedClient)

	invalid := client
	invalid.Id = 0
	invalid.ClientId = ""
	invalid.Slug = "invalid-integration-client"
	invalid.CapabilitiesJSON = ""
	require.NoError(t, db.Create(&invalid).Error)
	_, err = svc.Authorize(AuthorizeInput{
		ClientId:            invalid.ClientId,
		RedirectURI:         "https://example.com/callback",
		Scope:               "skills.read",
		State:               "state-6",
		CodeChallenge:       pkceChallenge("verifier-6"),
		CodeChallengeMethod: "S256",
		UserId:              100,
	})
	require.ErrorIs(t, err, ErrUnauthorizedClient)
}
