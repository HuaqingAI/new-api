package agentplatform

import (
	"encoding/json"
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newClientServiceForTest(t *testing.T) (*ClientService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))
	return NewClientService(db), db
}

func TestClientServiceCreatesAndListsValidClient(t *testing.T) {
	svc, _ := newClientServiceForTest(t)

	item, err := svc.Create(CreateClientInput{
		Slug:                   "cherry-studio",
		DisplayName:            "Cherry Studio",
		ClientType:             "desktop",
		ContractVersion:        "2026-06",
		Capabilities:           json.RawMessage(`{"discovery":true}`),
		AllowedGrantTypes:      json.RawMessage(`["authorization_code","refresh_token","client_credentials"]`),
		RedirectURIs:           json.RawMessage(`["cherrystudio://oauth/callback"]`),
		AllowedScopes:          json.RawMessage(`["skills.read"]`),
		Extensions:             json.RawMessage(`{"cherry_studio.owner":"ops"}`),
		AllowClientCredentials: true,
	})
	require.NoError(t, err)
	require.Equal(t, "active", item.Status)
	require.NotEmpty(t, item.ClientId)
	require.True(t, item.AllowClientCredentials)
	require.JSONEq(t, `["authorization_code","refresh_token","client_credentials"]`, item.AllowedGrantTypesJSON)
	require.JSONEq(t, `["cherrystudio://oauth/callback"]`, item.RedirectURIsJSON)
	require.JSONEq(t, `["skills.read"]`, item.AllowedScopesJSON)
	require.JSONEq(t, `{"cherry_studio.owner":"ops"}`, item.ExtensionsJSON)

	fetched, err := svc.Get(item.ClientId)
	require.NoError(t, err)
	require.Equal(t, item.ClientId, fetched.ClientId)

	list, err := svc.List(ClientQuery{})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
}

func TestClientServiceValidatesRegistrationSchema(t *testing.T) {
	svc, _ := newClientServiceForTest(t)

	invalidArrays := []CreateClientInput{
		{
			Slug:              "bad-grants",
			DisplayName:       "Bad Grants",
			ClientType:        "desktop",
			AllowedGrantTypes: json.RawMessage(`{"grant":"authorization_code"}`),
		},
		{
			Slug:         "bad-redirects",
			DisplayName:  "Bad Redirects",
			ClientType:   "desktop",
			RedirectURIs: json.RawMessage(`[""]`),
		},
		{
			Slug:          "bad-scopes",
			DisplayName:   "Bad Scopes",
			ClientType:    "desktop",
			AllowedScopes: json.RawMessage(`["ap.resources.read", " "]`),
		},
	}
	for _, input := range invalidArrays {
		_, err := svc.Create(input)
		require.ErrorIs(t, err, ErrInvalidClientInput)
	}

	_, err := svc.Create(CreateClientInput{
		Slug:        "bad-status",
		DisplayName: "Bad Status",
		ClientType:  "desktop",
		Status:      "pending",
	})
	require.ErrorIs(t, err, ErrInvalidClientInput)

	_, err = svc.Create(CreateClientInput{
		Slug:            "bad-capabilities",
		DisplayName:     "Bad Capabilities",
		ClientType:      "desktop",
		ContractVersion: "2026-06",
		Capabilities:    json.RawMessage(`["discovery"]`),
	})
	require.ErrorIs(t, err, ErrInvalidClientInput)

	_, err = svc.Create(CreateClientInput{
		Slug:            "bad-extensions",
		DisplayName:     "Bad Extensions",
		ClientType:      "desktop",
		ContractVersion: "2026-06",
		Capabilities:    json.RawMessage(`{"discovery":true}`),
		Extensions:      json.RawMessage(`["codex.owner"]`),
	})
	require.ErrorIs(t, err, ErrInvalidClientInput)
}

func TestClientServiceMarksIncompleteConfigAsInvalidIntegration(t *testing.T) {
	svc, _ := newClientServiceForTest(t)

	item, err := svc.Create(CreateClientInput{
		Slug:        "draft-client",
		DisplayName: "Draft Client",
		ClientType:  "desktop",
	})
	require.NoError(t, err)
	require.Equal(t, "invalid_integration", item.Status)

	explicitActive, err := svc.Create(CreateClientInput{
		Slug:            "explicit-active",
		DisplayName:     "Explicit Active",
		ClientType:      "desktop",
		Status:          "active",
		ContractVersion: "2026-06",
		Capabilities:    json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	require.Equal(t, "invalid_integration", explicitActive.Status)

	list, err := svc.List(ClientQuery{Status: "invalid_integration"})
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
}

func TestClientServiceRejectsNonNamespacedExtensions(t *testing.T) {
	svc, _ := newClientServiceForTest(t)

	_, err := svc.Create(CreateClientInput{
		Slug:            "bad-client",
		DisplayName:     "Bad Client",
		ClientType:      "desktop",
		ContractVersion: "2026-06",
		Capabilities:    json.RawMessage(`{"discovery":true}`),
		Extensions:      json.RawMessage(`{"plain":"value"}`),
	})
	require.ErrorIs(t, err, ErrInvalidClientInput)
}

func TestClientServiceUpdatesAllowClientCredentialsAndInvalidIntegration(t *testing.T) {
	svc, _ := newClientServiceForTest(t)

	item, err := svc.Create(CreateClientInput{
		Slug:            "codex",
		DisplayName:     "Codex",
		ClientType:      "cli",
		ContractVersion: "2026-06",
		Capabilities:    json.RawMessage(`{"discovery":true}`),
	})
	require.NoError(t, err)
	require.False(t, item.AllowClientCredentials)

	allow := true
	updated, err := svc.Update(item.ClientId, UpdateClientInput{
		Status:                 "disabled",
		AllowClientCredentials: &allow,
	})
	require.NoError(t, err)
	require.Equal(t, "disabled", updated.Status)
	require.True(t, updated.AllowClientCredentials)

	_, err = svc.Update(item.ClientId, UpdateClientInput{Status: "pending"})
	require.ErrorIs(t, err, ErrInvalidClientInput)

	updated, err = svc.Update(item.ClientId, UpdateClientInput{
		Status:       "active",
		Capabilities: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	require.Equal(t, "invalid_integration", updated.Status)
}
