package conformance

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/stretchr/testify/require"
)

func TestFixtureCatalogCoversRequiredCasesWithSyntheticData(t *testing.T) {
	fixtures := FixtureCatalog()
	require.NotEmpty(t, fixtures)

	required := []string{
		"oauth_authorize_success",
		"oauth_token_success",
		"oauth_revoke_success",
		"oauth_expired_token",
		"oauth_revoked_grant_token",
		"oauth_missing_scope_permission_denied",
		"discovery_empty",
		"discovery_success_multi_resource",
		"discovery_contract_mismatch_filtered_or_rejected",
		"detail_visible_callable",
		"detail_visible_not_callable",
		"detail_contract_invalid",
		"detail_resource_revoked",
		"detail_resource_offline",
		"refresh_fresh",
		"refresh_stale",
		"refresh_revoked",
		"refresh_offline",
		"refresh_observed_etag_version_mismatch",
		"refresh_ttl_over_300_non_compliance",
		"skill_invoke_sync_success",
		"skill_invoke_contract_invalid",
		"skill_invoke_timeout",
		"skill_invoke_upstream_provider_failure",
		"open_capability_quota_or_rate_limited",
		"knowledge_query_retrieval_success_items_citations",
		"knowledge_query_provider_offline",
		"knowledge_query_upstream_failure",
		"model_discovery_default_model",
		"model_discovery_no_default_model",
		"model_discovery_multiple_default_models",
		"model_discovery_default_disabled",
		"model_discovery_provider_offline",
		"model_discovery_model_unavailable",
		"model_discovery_account_tenant_mismatch",
		"error_client_state_matrix",
	}

	byName := map[string]Fixture{}
	for _, fixture := range fixtures {
		require.NotEmpty(t, fixture.Name)
		require.NotEmpty(t, fixture.Surface, fixture.Name)
		require.NotEmpty(t, fixture.Method, fixture.Name)
		require.NotEmpty(t, fixture.Path, fixture.Name)
		require.NotEmpty(t, fixture.Description, fixture.Name)
		byName[fixture.Name] = fixture
		assertFixtureHasNoSensitiveMaterial(t, fixture)
	}

	for _, name := range required {
		require.Contains(t, byName, name, "missing fixture %s", name)
	}
}

func TestPendingPrerequisiteFixturesHaveExplicitReasons(t *testing.T) {
	for _, fixture := range FixtureCatalog() {
		if fixture.Surface != "model-discovery" && fixture.Surface != "error-matrix" {
			continue
		}

		require.Equal(t, "PENDING", fixture.Method, fixture.Name)
		require.NotEmpty(t, fixture.PendingReason, fixture.Name)
		switch fixture.Surface {
		case "model-discovery":
			require.Contains(t, fixture.PendingReason, "AP-6.5", fixture.Name)
			require.Contains(t, fixture.PendingReason, "/v1/models", fixture.Name)
		case "error-matrix":
			require.Contains(t, fixture.PendingReason, "AP-6.6", fixture.Name)
		}
	}
}

func TestErrorFixtureCodesMatchServiceConstants(t *testing.T) {
	expectedCodes := map[string]struct{}{
		apservice.OpenCapabilityCodePermissionDenied: {},
		apservice.OpenCapabilityCodeResourceRevoked:  {},
		apservice.OpenCapabilityCodeResourceOffline:  {},
		apservice.OpenCapabilityCodeQuotaLimited:     {},
		apservice.OpenCapabilityCodeTimeout:          {},
		apservice.OpenCapabilityCodeUpstreamFailed:   {},
		apservice.OpenCapabilityCodeContractInvalid:  {},
	}

	seenCodes := map[string]struct{}{}
	for _, fixture := range FixtureCatalog() {
		payload, ok := fixture.Payload.(apservice.OpenCapabilityErrorResponse)
		if !ok {
			continue
		}
		require.False(t, payload.Success, fixture.Name)
		require.NotEmpty(t, payload.Error.Message, fixture.Name)
		require.Contains(t, expectedCodes, payload.Error.Code, fixture.Name)
		seenCodes[payload.Error.Code] = struct{}{}
	}

	for code := range expectedCodes {
		require.Contains(t, seenCodes, code, "missing error fixture for %s", code)
	}
}

func TestFixturePayloadsRoundTripWithRequiredContractFields(t *testing.T) {
	for _, fixture := range FixtureCatalog() {
		if fixture.PendingReason != "" {
			require.Nil(t, fixture.Payload, fixture.Name)
			continue
		}

		bytes, err := common.Marshal(fixture.Payload)
		require.NoError(t, err, fixture.Name)
		require.NotEmpty(t, bytes, fixture.Name)

		var fields map[string]any
		require.NoError(t, common.Unmarshal(bytes, &fields), fixture.Name)

		switch payload := fixture.Payload.(type) {
		case dtoagentplatform.OAuthAuthorizeResponse:
			var decoded dtoagentplatform.OAuthAuthorizeResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.Equal(t, payload.ClientId, decoded.ClientId, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "client_id", "contract_version", "scope", "state", "authorization_code", "redirect_uri", "consent_recorded")
		case dtoagentplatform.OAuthTokenResponse:
			var decoded dtoagentplatform.OAuthTokenResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.Equal(t, payload.AccessToken, decoded.AccessToken, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "access_token", "token_type", "expires_in", "scope", "contract_version")
		case dtoagentplatform.OAuthRevokeResponse:
			var decoded dtoagentplatform.OAuthRevokeResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.Equal(t, payload.Revoked, decoded.Revoked, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "revoked")
		case dtoagentplatform.OpenCapabilityDiscoveryResponse:
			var decoded dtoagentplatform.OpenCapabilityDiscoveryResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.Equal(t, payload.Total, decoded.Total, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "items", "total")
		case dtoagentplatform.OpenCapabilityDetailResponse:
			var decoded dtoagentplatform.OpenCapabilityDetailResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.Equal(t, payload.ResourceId, decoded.ResourceId, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_type", "display_name", "resource_version", "contract_version", "status", "visibility_state", "callable_state", "freshness_ttl_seconds", "freshness", "etag", "contract_compatible", "diagnostics")
		case dtoagentplatform.OpenCapabilityRefreshResponse:
			var decoded dtoagentplatform.OpenCapabilityRefreshResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.Equal(t, payload.ResourceId, decoded.ResourceId, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_version", "contract_version", "freshness_ttl_seconds", "freshness", "etag", "visibility_state", "callable_state", "contract_compatible", "diagnostics")
		case apservice.OpenCapabilityErrorResponse:
			var decoded apservice.OpenCapabilityErrorResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.False(t, decoded.Success, fixture.Name)
			require.NotEmpty(t, decoded.Error.Code, fixture.Name)
			require.NotEmpty(t, decoded.Error.Message, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "success", "error")
			errorFields, ok := fields["error"].(map[string]any)
			require.True(t, ok, "fixture %s error must be an object", fixture.Name)
			assertJSONFields(t, fixture.Name, errorFields, "code", "message", "retryable", "request_id", "resource_id", "resource_version")
		case map[string]any:
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_version", "contract_version")
		default:
			t.Fatalf("fixture %s uses unsupported payload type %T", fixture.Name, fixture.Payload)
		}
	}
}

func TestCoveredOpenAPIPathsExist(t *testing.T) {
	bytes, err := os.ReadFile("../../../docs/openapi/api.json")
	require.NoError(t, err)

	var spec struct {
		Paths      map[string]any `json:"paths"`
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, common.Unmarshal(bytes, &spec))

	expectedPaths := []string{
		"/api/agent-platform/oauth/authorize",
		"/api/agent-platform/oauth/token",
		"/api/agent-platform/oauth/revoke",
		"/api/open-capabilities/discovery",
		"/api/open-capabilities/resources/{id}",
		"/api/open-capabilities/refresh",
		"/api/open-capabilities/skills/{id}/invoke",
		"/api/open-capabilities/knowledge-bases/{id}/query",
		"/api/open-capabilities/agents/{id}",
	}
	for _, path := range expectedPaths {
		require.Contains(t, spec.Paths, path, "OpenAPI path missing for conformance coverage: %s", path)
	}

	for _, fixture := range FixtureCatalog() {
		if fixture.Surface == "model-discovery" {
			require.Equal(t, "pending", fixture.Path, fixture.Name)
		}
	}

	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "get", "/api/agent-platform/oauth/authorize", "client_id", "contract_version", "scope", "authorization_code", "redirect_uri", "consent_recorded")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/agent-platform/oauth/token", "access_token", "token_type", "expires_in", "scope", "contract_version")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/agent-platform/oauth/revoke", "revoked")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "get", "/api/open-capabilities/resources/{id}", "resource_id", "resource_type", "display_name", "resource_version", "contract_version", "status", "visibility_state", "callable_state", "freshness_ttl_seconds", "freshness", "etag", "contract_compatible", "diagnostics")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/open-capabilities/refresh", "resource_id", "resource_version", "contract_version", "freshness_ttl_seconds", "freshness", "etag", "visibility_state", "callable_state", "contract_compatible", "diagnostics")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformOAuthAuthorizeResponse")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformOAuthTokenResponse")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformOAuthRevokeResponse")
}

func assertFixtureHasNoSensitiveMaterial(t *testing.T, fixture Fixture) {
	t.Helper()

	bytes, err := common.Marshal(fixture)
	require.NoError(t, err, fixture.Name)
	text := strings.ToLower(string(bytes))

	for _, forbidden := range []string{
		"sk-",
		"bearer ey",
		"provider_secret",
		"tenant_secret",
		"real token",
		"real_token",
		"secret_access_key",
		"provider_config",
		"provider_native",
		"endpoint",
	} {
		require.NotContains(t, text, forbidden, fixture.Name)
	}
}

func assertJSONFields(t *testing.T, fixtureName string, fields map[string]any, requiredFields ...string) {
	t.Helper()
	for _, field := range requiredFields {
		require.Contains(t, fields, field, "fixture %s missing required JSON field %s", fixtureName, field)
	}
}

func assertOpenAPIRequiredFields(t *testing.T, paths map[string]any, schemas map[string]any, method string, path string, requiredFields ...string) {
	t.Helper()

	pathNode, ok := paths[path].(map[string]any)
	require.True(t, ok, "OpenAPI path %s must be an object", path)
	methodNode, ok := pathNode[method].(map[string]any)
	require.True(t, ok, "OpenAPI path %s must define method %s", path, method)
	responses, ok := methodNode["responses"].(map[string]any)
	require.True(t, ok, "OpenAPI path %s %s must define responses", method, path)
	status200, ok := responses["200"].(map[string]any)
	require.True(t, ok, "OpenAPI path %s %s must define 200 response", method, path)
	content, ok := status200["content"].(map[string]any)
	require.True(t, ok, "OpenAPI path %s %s 200 response must define content", method, path)
	applicationJSON, ok := content["application/json"].(map[string]any)
	require.True(t, ok, "OpenAPI path %s %s 200 response must define application/json", method, path)
	schema, ok := applicationJSON["schema"].(map[string]any)
	require.True(t, ok, "OpenAPI path %s %s 200 response must define schema", method, path)
	dataSchema := openAPIDataSchema(schema)
	require.NotNil(t, dataSchema, "OpenAPI path %s %s 200 response must define data schema", method, path)
	dataSchema = resolveOpenAPIRef(t, dataSchema, schemas)
	required, ok := dataSchema["required"].([]any)
	require.True(t, ok, "OpenAPI path %s %s data schema must define required fields", method, path)
	requiredSet := map[string]struct{}{}
	for _, value := range required {
		if text, ok := value.(string); ok {
			requiredSet[text] = struct{}{}
		}
	}
	for _, field := range requiredFields {
		require.Contains(t, requiredSet, field, "OpenAPI path %s %s data schema missing required field %s", method, path, field)
	}
}

func resolveOpenAPIRef(t *testing.T, schema map[string]any, schemas map[string]any) map[string]any {
	t.Helper()
	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}
	const prefix = "#/components/schemas/"
	require.True(t, strings.HasPrefix(ref, prefix), "unsupported OpenAPI ref %s", ref)
	name := strings.TrimPrefix(ref, prefix)
	resolved, ok := schemas[name].(map[string]any)
	require.True(t, ok, "OpenAPI ref %s must resolve to a schema object", ref)
	return resolved
}

func openAPIDataSchema(schema map[string]any) map[string]any {
	properties, ok := schema["properties"].(map[string]any)
	if ok {
		if data, ok := properties["data"].(map[string]any); ok {
			return data
		}
	}
	allOf, ok := schema["allOf"].([]any)
	if !ok {
		return nil
	}
	for _, entry := range allOf {
		node, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if data := openAPIDataSchema(node); data != nil {
			return data
		}
	}
	return nil
}
