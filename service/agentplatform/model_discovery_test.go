package agentplatform

import (
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newModelDiscoveryServiceForTest(t *testing.T, extensionsJSON string) (*ModelDiscoveryService, *gorm.DB, string) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:            "model-discovery-client",
		DisplayName:     "Model Discovery Client",
		ClientType:      "desktop",
		Status:          "active",
		ContractVersion: "2026-06",
		CapabilitiesJSON: `{"model_discovery":true}`,
		ExtensionsJSON:  extensionsJSON,
	}
	require.NoError(t, db.Create(&client).Error)
	return NewModelDiscoveryService(db), db, client.ClientId
}

func TestModelDiscoveryServiceResolvesDefaultModel(t *testing.T) {
	svc, _, clientID := newModelDiscoveryServiceForTest(t, `{"model_discovery.config":{"default_model":"gpt-4o-mini","account_id":"acct_demo","tenant_id":"tenant_demo","models":[{"model_id":"gpt-4o-mini","provider_stable_id":"openai","display_name":"GPT-4o Mini","capabilities":{"chat":true}},{"model_id":"claude-3-5-sonnet","provider_stable_id":"anthropic","display_name":"Claude 3.5 Sonnet","capabilities":{"chat":true}}]}}`)

	result, err := svc.List(clientID)
	require.NoError(t, err)
	require.Equal(t, "2026-06", result.ContractVersion)
	require.Equal(t, OpenCapabilityModelDefaultResolved, result.DefaultState)
	require.Len(t, result.Items, 2)
	require.Equal(t, "gpt-4o-mini", result.Items[0].ModelID)
	require.True(t, result.Items[0].IsDefault)
	require.Equal(t, OpenCapabilityModelStatusAvailable, result.Items[0].Status)
	require.Equal(t, "acct_demo", result.Items[0].AccountID)
	require.Equal(t, "tenant_demo", result.Items[0].TenantID)
}

func TestModelDiscoveryServiceHandlesNoDefaultAndMultipleDefaults(t *testing.T) {
	noDefaultSvc, _, noDefaultClientID := newModelDiscoveryServiceForTest(t, `{"model_discovery.config":{"models":[{"model_id":"gpt-4o-mini","provider_stable_id":"openai","display_name":"GPT-4o Mini"}]}}`)
	noDefault, err := noDefaultSvc.List(noDefaultClientID)
	require.NoError(t, err)
	require.Equal(t, OpenCapabilityModelDefaultNoDefault, noDefault.DefaultState)

	multiSvc, _, multiClientID := newModelDiscoveryServiceForTest(t, `{"model_discovery.config":{"models":[{"model_id":"gpt-4o-mini","provider_stable_id":"openai","display_name":"GPT-4o Mini","is_default":true},{"model_id":"claude-3-5-sonnet","provider_stable_id":"anthropic","display_name":"Claude 3.5 Sonnet","is_default":true}]}}`)
	multi, err := multiSvc.List(multiClientID)
	require.NoError(t, err)
	require.Equal(t, OpenCapabilityModelDefaultMultipleDefaults, multi.DefaultState)
}

func TestModelDiscoveryServiceHandlesDisabledOfflineUnavailableAndMismatch(t *testing.T) {
	svc, _, clientID := newModelDiscoveryServiceForTest(t, `{"model_discovery.config":{"default_model":"gpt-4o-mini","account_id":"acct_demo","tenant_id":"tenant_demo","models":[{"model_id":"gpt-4o-mini","provider_stable_id":"openai","display_name":"GPT-4o Mini","status":"disabled","disabled_reason":"default_model_disabled"},{"model_id":"claude-3-5-sonnet","provider_stable_id":"anthropic","display_name":"Claude 3.5 Sonnet","status":"provider_offline","disabled_reason":"provider_offline"},{"model_id":"gemini-1.5-pro","provider_stable_id":"gemini","display_name":"Gemini 1.5 Pro","account_id":"acct_other","tenant_id":"tenant_demo"}]}}`)

	result, err := svc.List(clientID)
	require.NoError(t, err)
	require.Equal(t, OpenCapabilityModelDefaultDisabled, result.DefaultState)
	require.Len(t, result.Items, 3)
	require.Equal(t, OpenCapabilityModelStatusDisabled, result.Items[0].Status)
	require.Equal(t, "default_model_disabled", result.Items[0].DisabledReason)
	require.Equal(t, OpenCapabilityModelStatusProviderOffline, result.Items[1].Status)
	require.Equal(t, OpenCapabilityModelStatusAccountTenantMismatch, result.Items[2].Status)
	require.Equal(t, "account_tenant_mismatch", result.Items[2].DisabledReason)
}

func TestModelDiscoveryServiceSynthesizesUnavailableConfiguredDefault(t *testing.T) {
	svc, _, clientID := newModelDiscoveryServiceForTest(t, `{"model_discovery.config":{"default_model":"gpt-4o-mini","models":[{"model_id":"claude-3-5-sonnet","provider_stable_id":"anthropic","display_name":"Claude 3.5 Sonnet"}]}}`)

	result, err := svc.List(clientID)
	require.NoError(t, err)
	require.Equal(t, OpenCapabilityModelDefaultUnavailable, result.DefaultState)
	require.Len(t, result.Items, 2)

	last := result.Items[1]
	require.Equal(t, "gpt-4o-mini", last.ModelID)
	require.True(t, last.IsDefault)
	require.Equal(t, OpenCapabilityModelStatusUnavailable, last.Status)
	require.Equal(t, "model_unavailable", last.DisabledReason)
}
