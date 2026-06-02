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
		Slug:            "cherry-studio",
		DisplayName:     "Cherry Studio",
		ClientType:      "desktop",
		ContractVersion: "2026-06",
		Capabilities:    json.RawMessage(`{"discovery":true}`),
		AllowedScopes:   json.RawMessage(`["skills.read"]`),
	})
	require.NoError(t, err)
	require.Equal(t, "active", item.Status)
	require.NotEmpty(t, item.ClientId)

	fetched, err := svc.Get(item.ClientId)
	require.NoError(t, err)
	require.Equal(t, item.ClientId, fetched.ClientId)

	list, err := svc.List(ClientQuery{})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
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
