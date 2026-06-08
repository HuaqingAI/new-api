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
		"oauth_refresh_rotation_success",
		"oauth_revoke_success",
		"oauth_expired_token",
		"oauth_revoked_grant_token",
		"oauth_missing_scope_permission_denied",
		"oauth_redirect_mismatch",
		"oauth_invalid_pkce",
		"oauth_invalid_client",
		"oauth_invalid_integration",
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
		if (fixture.Surface == "model-discovery" || fixture.Surface == "error-matrix") && fixture.PendingReason == "" {
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
			for _, item := range decoded.Items {
				itemBytes, err := common.Marshal(item)
				require.NoError(t, err, fixture.Name)
				var itemFields map[string]any
				require.NoError(t, common.Unmarshal(itemBytes, &itemFields), fixture.Name)
				assertJSONFields(t, fixture.Name, itemFields, "resource_id", "resource_type", "display_name", "resource_version", "contract_version", "visibility_state", "callable_state", "freshness_ttl_seconds", "freshness", "etag")
				assertFrozenFreshnessEnum(t, fixture.Name, item.Freshness)
				require.LessOrEqual(t, item.FreshnessTTLSeconds, 300, fixture.Name)
				require.NotContains(t, itemFields, "diagnostics", fixture.Name)
				assertResourceP0Excluded(t, fixture.Name, itemFields)
			}
		case dtoagentplatform.OpenCapabilityDetailResponse:
			var decoded dtoagentplatform.OpenCapabilityDetailResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.Equal(t, payload.ResourceId, decoded.ResourceId, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_type", "display_name", "resource_version", "contract_version", "status", "visibility_state", "callable_state", "freshness_ttl_seconds", "freshness", "etag", "contract_compatible", "diagnostics")
			assertFrozenFreshnessEnum(t, fixture.Name, decoded.Freshness)
			require.LessOrEqual(t, decoded.FreshnessTTLSeconds, 300, fixture.Name)
			assertFrozenDiagnosticsShape(t, fixture.Name, fields)
			assertResourceP0Excluded(t, fixture.Name, fields)
		case dtoagentplatform.OpenCapabilityRefreshResponse:
			var decoded dtoagentplatform.OpenCapabilityRefreshResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.Equal(t, payload.ResourceId, decoded.ResourceId, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_version", "contract_version", "freshness_ttl_seconds", "freshness", "etag", "visibility_state", "callable_state", "contract_compatible", "diagnostics")
			assertFrozenFreshnessEnum(t, fixture.Name, decoded.Freshness)
			require.LessOrEqual(t, decoded.FreshnessTTLSeconds, 300, fixture.Name)
			assertFrozenDiagnosticsShape(t, fixture.Name, fields)
			assertResourceP0Excluded(t, fixture.Name, fields)
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
		case OAuthContractErrorResponse:
			var decoded OAuthContractErrorResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			require.False(t, decoded.Success, fixture.Name)
			require.NotEmpty(t, decoded.Message, fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "success", "message", "retryable")
		case dtoagentplatform.OpenCapabilitySkillInvokeResponse:
			var decoded dtoagentplatform.OpenCapabilitySkillInvokeResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_version", "contract_version", "output")
			require.NotContains(t, fields, "task_id", fixture.Name)
			require.NotContains(t, fields, "status", fixture.Name)
			require.NotContains(t, fields, "provider_config", fixture.Name)
		case dtoagentplatform.OpenCapabilityKnowledgeQueryResponse:
			var decoded dtoagentplatform.OpenCapabilityKnowledgeQueryResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_version", "contract_version", "items", "citations")
			require.NotEmpty(t, decoded.Items, fixture.Name)
			require.NotContains(t, fields, "provider_config", fixture.Name)
			require.NotContains(t, fields, "provider_native", fixture.Name)
		case dtoagentplatform.OpenCapabilityModelDiscoveryResponse:
			var decoded dtoagentplatform.OpenCapabilityModelDiscoveryResponse
			require.NoError(t, common.Unmarshal(bytes, &decoded), fixture.Name)
			assertJSONFields(t, fixture.Name, fields, "contract_version", "default_state", "items", "total")
			require.NotEmpty(t, decoded.Items, fixture.Name)
			require.NotContains(t, fields, "data", fixture.Name)
		case map[string]any:
			if fixture.Surface == "error-matrix" {
				assertJSONFields(t, fixture.Name, fields, "error_code_matrix", "payload_state_matrix")
			} else {
				assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_version", "contract_version")
			}
		default:
			t.Fatalf("fixture %s uses unsupported payload type %T", fixture.Name, fixture.Payload)
		}
	}
}

func TestOpenCapabilityResourceContractFrozenFields(t *testing.T) {
	for _, fixture := range FixtureCatalog() {
		if fixture.PendingReason != "" {
			continue
		}

		bytes, err := common.Marshal(fixture.Payload)
		require.NoError(t, err, fixture.Name)

		var fields map[string]any
		require.NoError(t, common.Unmarshal(bytes, &fields), fixture.Name)

		switch fixture.Payload.(type) {
		case dtoagentplatform.OpenCapabilityDiscoveryResponse:
			items, ok := fields["items"].([]any)
			require.True(t, ok, fixture.Name)
			for _, item := range items {
				itemFields, ok := item.(map[string]any)
				require.True(t, ok, fixture.Name)
				assertJSONFields(t, fixture.Name, itemFields, "resource_id", "resource_type", "display_name", "resource_version", "contract_version", "visibility_state", "callable_state", "freshness_ttl_seconds", "freshness", "etag")
				require.NotContains(t, itemFields, "diagnostics", fixture.Name)
				assertResourceP0Excluded(t, fixture.Name, itemFields)
				assertFrozenFreshnessTTL(t, fixture.Name, itemFields)
				assertFrozenFreshnessEnum(t, fixture.Name, itemFields["freshness"])
			}
		case dtoagentplatform.OpenCapabilityDetailResponse:
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_type", "display_name", "resource_version", "contract_version", "visibility_state", "callable_state", "freshness_ttl_seconds", "freshness", "etag", "status", "contract_compatible", "diagnostics")
			assertResourceP0Excluded(t, fixture.Name, fields)
			assertFrozenFreshnessTTL(t, fixture.Name, fields)
			assertFrozenFreshnessEnum(t, fixture.Name, fields["freshness"])
			assertFrozenDiagnosticsShape(t, fixture.Name, fields)
		case dtoagentplatform.OpenCapabilityRefreshResponse:
			assertJSONFields(t, fixture.Name, fields, "resource_id", "resource_version", "contract_version", "freshness_ttl_seconds", "freshness", "etag", "visibility_state", "callable_state", "contract_compatible", "diagnostics")
			require.NotContains(t, fields, "resource_type", fixture.Name)
			require.NotContains(t, fields, "display_name", fixture.Name)
			assertResourceP0Excluded(t, fixture.Name, fields)
			assertFrozenFreshnessTTL(t, fixture.Name, fields)
			assertFrozenFreshnessEnum(t, fixture.Name, fields["freshness"])
			assertFrozenDiagnosticsShape(t, fixture.Name, fields)
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
		"/api/agent-platform/clients",
		"/api/agent-platform/clients/{id}",
		"/api/agent-platform/oauth/authorize",
		"/api/agent-platform/oauth/token",
		"/api/agent-platform/oauth/revoke",
		"/api/open-capabilities/discovery",
		"/api/open-capabilities/models",
		"/api/open-capabilities/resources/{id}",
		"/api/open-capabilities/refresh",
		"/api/open-capabilities/skills/{id}/invoke",
		"/api/open-capabilities/knowledge-bases/{id}/query",
		"/api/open-capabilities/agents/{id}",
	}
	for _, path := range expectedPaths {
		require.Contains(t, spec.Paths, path, "OpenAPI path missing for conformance coverage: %s", path)
	}

	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/agent-platform/clients", "id", "client_id", "slug", "display_name", "client_type", "status", "contract_version", "allow_client_credentials", "created_at", "updated_at")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "get", "/api/agent-platform/clients/{id}", "id", "client_id", "slug", "display_name", "client_type", "status", "contract_version", "allow_client_credentials", "created_at", "updated_at")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "get", "/api/agent-platform/oauth/authorize", "client_id", "contract_version", "scope", "authorization_code", "redirect_uri", "consent_recorded")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/agent-platform/oauth/token", "access_token", "token_type", "expires_in", "scope", "contract_version")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/agent-platform/oauth/revoke", "revoked")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "get", "/api/open-capabilities/discovery", "items", "total")
	assertOpenAPIDiscoveryItemSchema(t, spec.Paths, "/api/open-capabilities/discovery")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "get", "/api/open-capabilities/resources/{id}", "resource_id", "resource_type", "display_name", "resource_version", "contract_version", "status", "visibility_state", "callable_state", "freshness_ttl_seconds", "freshness", "etag", "contract_compatible", "diagnostics")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/open-capabilities/refresh", "resource_id", "resource_version", "contract_version", "freshness_ttl_seconds", "freshness", "etag", "visibility_state", "callable_state", "contract_compatible", "diagnostics")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/open-capabilities/skills/{id}/invoke", "resource_id", "resource_version", "contract_version", "output")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "post", "/api/open-capabilities/knowledge-bases/{id}/query", "resource_id", "resource_version", "contract_version", "items", "citations")
	assertOpenAPIRequiredFields(t, spec.Paths, spec.Components.Schemas, "get", "/api/open-capabilities/models", "contract_version", "default_state", "items", "total")
	assertOpenAPIResourceContractSchema(t, spec.Paths, spec.Components.Schemas, "get", "/api/open-capabilities/resources/{id}")
	assertOpenAPIResourceContractSchema(t, spec.Paths, spec.Components.Schemas, "post", "/api/open-capabilities/refresh")
	assertOpenAPISkillInvokeSchema(t, spec.Paths)
	assertOpenAPIKnowledgeQuerySchema(t, spec.Paths)
	assertOpenAPIModelDiscoverySchema(t, spec.Paths)
	require.Contains(t, spec.Components.Schemas, "AgentPlatformClientCreateRequest")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformClientUpdateRequest")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformClientItem")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformClientListResponse")
	assertOpenAPIRequiredSchemaFields(t, spec.Components.Schemas, "AgentPlatformClientCreateRequest", "slug", "display_name", "client_type")
	assertOpenAPIOptionalSchemaFields(t, spec.Components.Schemas, "AgentPlatformClientCreateRequest", "status", "allowed_grant_types", "redirect_uris", "allowed_scopes", "contract_version", "capabilities", "extensions", "allow_client_credentials")
	assertOpenAPIRequiredSchemaFields(t, spec.Components.Schemas, "AgentPlatformClientItem", "id", "client_id", "slug", "display_name", "client_type", "status", "contract_version", "allow_client_credentials", "created_at", "updated_at")
	assertClientRegistrationOpenAPISchema(t, spec.Components.Schemas, "AgentPlatformClientCreateRequest")
	assertClientRegistrationOpenAPISchema(t, spec.Components.Schemas, "AgentPlatformClientUpdateRequest")
	assertClientRegistrationOpenAPISchema(t, spec.Components.Schemas, "AgentPlatformClientItem")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformOAuthAuthorizeResponse")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformOAuthTokenResponse")
	require.Contains(t, spec.Components.Schemas, "AgentPlatformOAuthRevokeResponse")
	assertOpenAPIRequiredSchemaFields(t, spec.Components.Schemas, "AgentPlatformOAuthAuthorizeResponse", "client_id", "contract_version", "scope", "state", "authorization_code", "redirect_uri", "consent_recorded")
	assertOpenAPIRequiredSchemaFields(t, spec.Components.Schemas, "AgentPlatformOAuthTokenRequest", "client_id")
	assertOpenAPIOptionalSchemaFields(t, spec.Components.Schemas, "AgentPlatformOAuthTokenRequest", "grant_type")
	assertOpenAPIRequiredSchemaFields(t, spec.Components.Schemas, "AgentPlatformOAuthTokenResponse", "access_token", "token_type", "expires_in", "refresh_token", "refresh_expires_in", "scope", "contract_version", "grant_id")
}

func TestConsumerSignoffArtifactAlignsWithContractSources(t *testing.T) {
	signoffBytes, err := os.ReadFile("../../../docs/agent-platform-consumer-signoff.md")
	require.NoError(t, err)
	signoff := string(signoffBytes)

	contractBytes, err := os.ReadFile("../../../docs/agent-platform-downstream-contract-spec.md")
	require.NoError(t, err)
	contract := string(contractBytes)

	for _, required := range []string{
		"Status: signed off",
		"Client registration / onboarding",
		"contract_version: 2026-06",
		"docs/openapi/api.json",
		"tests/agentplatform/conformance/",
		"controller/agentplatform/open_capabilities_test.go",
		"controller/agentplatform/oauth_test.go",
		"service/agentplatform/oauth_authorize_test.go",
		"service/agentplatform/oauth_token_test.go",
		"extensions.cherry_studio",
		"extensions.codex",
		"AP-6.2 client registration schema freeze",
		"AP-6.5",
		"AP-6.6",
		"no parallel core protocol",
	} {
		require.Contains(t, signoff, required)
	}
	assertAllowedSignoffStatuses(t, signoff)
	assertAllowedSignoffStatuses(t, markdownSection(contract, "### 11.8 Consumer Signoff"))

	require.Contains(t, signoff, "Cherry Studio OAuth subdomain signoff: `signed off`")
	require.Contains(t, signoff, "Cherry Studio first-consumer signoff: `signed off`")
	require.Contains(t, signoff, "Resource discovery/detail/refresh signoff: `signed off`")
	require.Contains(t, signoff, "Codex second-consumer review: `signed off`")
	require.Contains(t, signoff, "OpenAPI change: OAuth schema freeze")
	require.NotContains(t, signoff, "AP-6.5 model discovery 与 AP-6.6 error/client state matrix 尚未冻结")
	require.NotContains(t, signoff, "AP-6.5 enterprise model discovery contract 与 AP-6.6 error/client state matrix 仍未冻结")
	require.NotContains(t, signoff, "AP-6.5 与 AP-6.6 阻塞")
	require.NotContains(t, signoff, "只要 AP-6.5 / AP-6.6 仍是 pending fixture")

	require.Contains(t, contract, "docs/agent-platform-consumer-signoff.md")
	require.Contains(t, contract, "Status: frozen")
	require.Contains(t, contract, "AP-6.3 resource discovery/detail/refresh contract freeze")
	require.Contains(t, contract, "Status: frozen")
	require.Contains(t, contract, "sync-only")
	require.Contains(t, contract, "provider-native")
	require.Contains(t, contract, "`source`、`accountId`、`tenantId`、`disabledReason`、`fetchedAt`、`expiresAt` 不进入 AP-6.3 P0")
	require.Contains(t, contract, "下一次 refresh 或 300 秒 TTL 上限内收敛")
	require.Contains(t, contract, "API / fixture 支撑")
	require.Contains(t, contract, "`invalid_integration`")
	require.Contains(t, contract, "controller/agentplatform/oauth_test.go")
	require.Contains(t, contract, "service/agentplatform/oauth_authorize_test.go")
	require.Contains(t, contract, "service/agentplatform/oauth_token_test.go")
	require.Contains(t, contract, "extensions.cherry_studio")
	require.Contains(t, contract, "extensions.codex")
	require.Contains(t, contract, "no parallel core protocol")
	require.Contains(t, contract, "Cherry Studio overall first-consumer signoff is `signed off`")
	require.Contains(t, contract, "Codex second-consumer review")
	require.NotContains(t, markdownSection(contract, "### 11.7 Mock / Fixture / Conformance"), "pending with AP-6.5")
	require.NotContains(t, markdownSection(contract, "### 11.7 Mock / Fixture / Conformance"), "pending with AP-6.6")
	require.NotContains(t, markdownSection(contract, "### 11.8 Consumer Signoff"), "AP-6.5 is not frozen")
	require.NotContains(t, markdownSection(contract, "### 11.8 Consumer Signoff"), "AP-6.6 is not frozen")

	fixtures := FixtureCatalog()
	require.NotEmpty(t, fixtures)
	require.False(t, hasPendingSurface(fixtures, "model-discovery"), "AP-6.5 should freeze model-discovery fixtures once contract is implemented")
	require.False(t, hasPendingSurface(fixtures, "error-matrix"), "AP-6.6 should freeze error-matrix fixtures once contract is implemented")
}

func TestConsumerSignoffCoverageMatrixMatchesFixtures(t *testing.T) {
	signoffBytes, err := os.ReadFile("../../../docs/agent-platform-consumer-signoff.md")
	require.NoError(t, err)
	signoff := string(signoffBytes)

	requiredRows := []struct {
		domain   string
		status   string
		evidence []string
	}{
		{
			domain: "OAuth authorize/token/revoke/callback/allowlist",
			status: "signed off",
			evidence: []string{
				"oauth_authorize_success",
				"oauth_token_success",
				"oauth_refresh_rotation_success",
				"oauth_revoke_success",
				"oauth_expired_token",
				"oauth_revoked_grant_token",
				"oauth_missing_scope_permission_denied",
				"oauth_redirect_mismatch",
				"oauth_invalid_pkce",
				"oauth_invalid_client",
				"oauth_invalid_integration",
			},
		},
		{
			domain: "Resource discovery/detail/refresh",
			status: "signed off",
			evidence: []string{
				"discovery_empty",
				"discovery_success_multi_resource",
				"detail_visible_callable",
				"detail_resource_revoked",
				"detail_resource_offline",
				"refresh_fresh",
				"refresh_stale",
				"refresh_revoked",
				"refresh_offline",
			},
		},
		{
			domain: "Skill invoke",
			status: "signed off",
			evidence: []string{
				"skill_invoke_sync_success",
				"skill_invoke_contract_invalid",
				"skill_invoke_timeout",
				"skill_invoke_upstream_provider_failure",
			},
		},
		{
			domain: "Knowledge query",
			status: "signed off",
			evidence: []string{
				"knowledge_query_retrieval_success_items_citations",
				"knowledge_query_provider_offline",
				"knowledge_query_upstream_failure",
			},
		},
		{
			domain: "Enterprise model discovery",
			status: "signed off",
			evidence: []string{
				"model_discovery_default_model",
				"model_discovery_default_disabled",
				"model_discovery_model_unavailable",
				"model_discovery_account_tenant_mismatch",
			},
		},
		{
			domain:   "Error matrix and client state matrix",
			status:   "signed off",
			evidence: []string{"error_client_state_matrix"},
		},
	}

	byName := fixturesByName(FixtureCatalog())
	for _, row := range requiredRows {
		require.Contains(t, signoff, row.domain)
		require.Contains(t, signoff, "| "+row.domain+" | "+row.status+" |")
		for _, fixtureName := range row.evidence {
			require.Contains(t, byName, fixtureName)
		}
	}
}

func assertOpenAPIRequiredSchemaFields(t *testing.T, schemas map[string]any, schemaName string, requiredFields ...string) {
	t.Helper()

	schema, ok := schemas[schemaName].(map[string]any)
	require.True(t, ok, "OpenAPI schema %s must exist", schemaName)
	required, ok := schema["required"].([]any)
	require.True(t, ok, "OpenAPI schema %s must define required fields", schemaName)
	requiredSet := map[string]struct{}{}
	for _, value := range required {
		if text, ok := value.(string); ok {
			requiredSet[text] = struct{}{}
		}
	}
	for _, field := range requiredFields {
		require.Contains(t, requiredSet, field, "OpenAPI schema %s missing required field %s", schemaName, field)
	}
}

func assertOpenAPIOptionalSchemaFields(t *testing.T, schemas map[string]any, schemaName string, optionalFields ...string) {
	t.Helper()

	schema, ok := schemas[schemaName].(map[string]any)
	require.True(t, ok, "OpenAPI schema %s must exist", schemaName)
	required, _ := schema["required"].([]any)
	requiredSet := map[string]struct{}{}
	for _, value := range required {
		if text, ok := value.(string); ok {
			requiredSet[text] = struct{}{}
		}
	}
	for _, field := range optionalFields {
		require.NotContains(t, requiredSet, field, "OpenAPI schema %s field %s must remain optional", schemaName, field)
	}
}

func assertClientRegistrationOpenAPISchema(t *testing.T, schemas map[string]any, schemaName string) {
	t.Helper()

	schema, ok := schemas[schemaName].(map[string]any)
	require.True(t, ok, "OpenAPI schema %s must exist", schemaName)
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "OpenAPI schema %s must define properties", schemaName)

	status, ok := properties["status"].(map[string]any)
	require.True(t, ok, "OpenAPI schema %s must define status", schemaName)
	assertOpenAPIEnum(t, status, "active", "disabled", "invalid_integration")

	for _, field := range []string{"allowed_grant_types", "redirect_uris", "allowed_scopes"} {
		property, ok := properties[field].(map[string]any)
		require.True(t, ok, "OpenAPI schema %s must define %s", schemaName, field)
		require.Equal(t, "array", property["type"], "OpenAPI schema %s field %s must remain an array", schemaName, field)
		items, ok := property["items"].(map[string]any)
		require.True(t, ok, "OpenAPI schema %s field %s must define array items", schemaName, field)
		require.Equal(t, "string", items["type"], "OpenAPI schema %s field %s items must remain strings", schemaName, field)
	}

	grantTypes := properties["allowed_grant_types"].(map[string]any)
	assertOpenAPIEnum(t, grantTypes["items"].(map[string]any), "authorization_code", "refresh_token", "client_credentials")
	assertOpenAPIExampleContains(t, grantTypes, "authorization_code", "refresh_token")
	assertOpenAPIExampleContains(t, properties["redirect_uris"].(map[string]any), "cherrystudio://oauth/callback")
	assertOpenAPIExampleContains(t, properties["allowed_scopes"].(map[string]any), "ap.resources.read")
	assertOpenAPIPropertyExample(t, properties, "contract_version", ContractVersion)
}

func assertOpenAPIEnum(t *testing.T, schema map[string]any, expectedValues ...string) {
	t.Helper()

	values, ok := schema["enum"].([]any)
	require.True(t, ok, "OpenAPI schema must define enum")
	valueSet := map[string]struct{}{}
	for _, value := range values {
		text, ok := value.(string)
		require.True(t, ok, "OpenAPI enum value must be a string")
		valueSet[text] = struct{}{}
	}
	for _, expected := range expectedValues {
		require.Contains(t, valueSet, expected, "OpenAPI enum missing %s", expected)
	}
}

func assertOpenAPIExampleContains(t *testing.T, schema map[string]any, expectedValues ...string) {
	t.Helper()

	example, ok := schema["example"].([]any)
	require.True(t, ok, "OpenAPI schema must define array example")
	valueSet := map[string]struct{}{}
	for _, value := range example {
		text, ok := value.(string)
		require.True(t, ok, "OpenAPI example value must be a string")
		valueSet[text] = struct{}{}
	}
	for _, expected := range expectedValues {
		require.Contains(t, valueSet, expected, "OpenAPI example missing %s", expected)
	}
}

func assertOpenAPIPropertyExample(t *testing.T, properties map[string]any, propertyName string, expectedValue string) {
	t.Helper()

	property, ok := properties[propertyName].(map[string]any)
	require.True(t, ok, "OpenAPI property %s must exist", propertyName)
	require.Equal(t, expectedValue, property["example"], "OpenAPI property %s example drifted", propertyName)
}

func assertResourceP0Excluded(t *testing.T, fixtureName string, fields map[string]any) {
	t.Helper()

	for _, field := range []string{"source", "accountId", "tenantId", "disabledReason", "fetchedAt", "expiresAt"} {
		require.NotContains(t, fields, field, "fixture %s must not promote %s into AP-6.3 P0 core fields", fixtureName, field)
	}
}

func assertFrozenFreshnessTTL(t *testing.T, fixtureName string, fields map[string]any) {
	t.Helper()

	value, ok := fields["freshness_ttl_seconds"].(float64)
	require.True(t, ok, "fixture %s freshness_ttl_seconds must be numeric", fixtureName)
	require.LessOrEqual(t, int(value), 300, "fixture %s TTL must not exceed 300 seconds", fixtureName)
}

func assertFrozenFreshnessEnum(t *testing.T, fixtureName string, value any) {
	t.Helper()

	text, ok := value.(string)
	require.True(t, ok, "fixture %s freshness must be a string", fixtureName)
	require.Contains(t, []string{"fresh", "stale", "offline", "revoked"}, text, "fixture %s freshness enum drifted", fixtureName)
}

func assertFrozenDiagnosticsShape(t *testing.T, fixtureName string, fields map[string]any) {
	t.Helper()

	diagnostics, ok := fields["diagnostics"].(map[string]any)
	require.True(t, ok, "fixture %s diagnostics must be an object", fixtureName)
	assertJSONFields(t, fixtureName, diagnostics, "reason", "converged", "client_non_compliant")
	_, ok = diagnostics["reason"].(string)
	require.True(t, ok, "fixture %s diagnostics.reason must be string", fixtureName)
	for _, field := range []string{"converged", "client_non_compliant"} {
		_, ok := diagnostics[field].(bool)
		require.True(t, ok, "fixture %s diagnostics.%s must be boolean", fixtureName, field)
	}
}

func assertOpenCapabilityFreshness(t *testing.T, fixtureName string, value string) {
	t.Helper()

	assertFrozenFreshnessEnum(t, fixtureName, value)
}

func assertDiagnosticsShape(t *testing.T, fixtureName string, fields map[string]any) {
	t.Helper()

	assertFrozenDiagnosticsShape(t, fixtureName, fields)
}

func assertNoAP63ExcludedCoreFields(t *testing.T, fixtureName string, fields map[string]any) {
	t.Helper()

	assertResourceP0Excluded(t, fixtureName, fields)
}

func assertOpenAPIDiscoveryItemSchema(t *testing.T, paths map[string]any, path string) {
	t.Helper()

	dataSchema := openAPIResponseDataSchema(t, paths, nil, "get", path)
	properties, ok := dataSchema["properties"].(map[string]any)
	require.True(t, ok, "OpenAPI discovery response must define properties")
	items, ok := properties["items"].(map[string]any)
	require.True(t, ok, "OpenAPI discovery response must define items")
	itemSchema, ok := items["items"].(map[string]any)
	require.True(t, ok, "OpenAPI discovery items must define item schema")
	required, ok := itemSchema["required"].([]any)
	require.True(t, ok, "OpenAPI discovery item schema must define required fields")
	requiredSet := stringSetFromAny(required)
	for _, field := range []string{"resource_id", "resource_type", "display_name", "resource_version", "contract_version", "visibility_state", "callable_state", "freshness_ttl_seconds", "freshness", "etag"} {
		require.Contains(t, requiredSet, field, "OpenAPI discovery item missing required field %s", field)
	}
	require.NotContains(t, requiredSet, "diagnostics", "discovery list summary must not require diagnostics")
	itemProperties, ok := itemSchema["properties"].(map[string]any)
	require.True(t, ok, "OpenAPI discovery item must define properties")
	assertOpenAPIFreshnessAndTTL(t, itemProperties)
	for _, field := range []string{"source", "accountId", "tenantId", "disabledReason", "fetchedAt", "expiresAt"} {
		require.NotContains(t, itemProperties, field, "OpenAPI discovery item must not define AP-6.5/6.6 candidate field %s", field)
	}
}

func assertOpenAPIResourceContractSchema(t *testing.T, paths map[string]any, schemas map[string]any, method string, path string) {
	t.Helper()

	dataSchema := openAPIResponseDataSchema(t, paths, schemas, method, path)
	properties, ok := dataSchema["properties"].(map[string]any)
	require.True(t, ok, "OpenAPI %s %s data schema must define properties", method, path)
	assertOpenAPIFreshnessAndTTL(t, properties)

	diagnostics, ok := properties["diagnostics"].(map[string]any)
	require.True(t, ok, "OpenAPI %s %s must define diagnostics", method, path)
	diagnosticsProperties, ok := diagnostics["properties"].(map[string]any)
	require.True(t, ok, "OpenAPI diagnostics must define properties")
	assertOpenAPIRequiredSchemaNodeFields(t, diagnostics, "reason", "converged", "client_non_compliant")
	require.Equal(t, "string", diagnosticsProperties["reason"].(map[string]any)["type"])
	require.Equal(t, "boolean", diagnosticsProperties["converged"].(map[string]any)["type"])
	require.Equal(t, "boolean", diagnosticsProperties["client_non_compliant"].(map[string]any)["type"])

	for _, field := range []string{"source", "accountId", "tenantId", "disabledReason", "fetchedAt", "expiresAt"} {
		require.NotContains(t, properties, field, "OpenAPI %s %s must not define AP-6.5/6.6 candidate field %s", method, path, field)
	}
}

func assertOpenAPIFreshnessAndTTL(t *testing.T, properties map[string]any) {
	t.Helper()

	freshness, ok := properties["freshness"].(map[string]any)
	require.True(t, ok, "OpenAPI freshness property must exist")
	assertOpenAPIEnum(t, freshness, "fresh", "stale", "offline", "revoked")

	ttl, ok := properties["freshness_ttl_seconds"].(map[string]any)
	require.True(t, ok, "OpenAPI freshness_ttl_seconds property must exist")
	require.Equal(t, float64(300), ttl["maximum"], "OpenAPI TTL maximum must remain 300 seconds")
}

func assertOpenAPISkillInvokeSchema(t *testing.T, paths map[string]any) {
	t.Helper()

	dataSchema := openAPIResponseDataSchema(t, paths, nil, "post", "/api/open-capabilities/skills/{id}/invoke")
	properties, ok := dataSchema["properties"].(map[string]any)
	require.True(t, ok, "OpenAPI skill invoke response must define properties")
	for _, field := range []string{"resource_id", "resource_version", "contract_version", "output"} {
		require.Contains(t, properties, field, "OpenAPI skill invoke response missing %s", field)
	}
	require.NotContains(t, properties, "task_id", "AP-6.4 P0 must not imply async task protocol")
	require.NotContains(t, properties, "status", "AP-6.4 P0 must not imply async task status")

	pathNode := paths["/api/open-capabilities/skills/{id}/invoke"].(map[string]any)
	methodNode := pathNode["post"].(map[string]any)
	requestBody := methodNode["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	applicationJSON := content["application/json"].(map[string]any)
	requestSchema := applicationJSON["schema"].(map[string]any)
	requestProperties := requestSchema["properties"].(map[string]any)
	require.Contains(t, requestProperties, "input")
}

func assertOpenAPIKnowledgeQuerySchema(t *testing.T, paths map[string]any) {
	t.Helper()

	dataSchema := openAPIResponseDataSchema(t, paths, nil, "post", "/api/open-capabilities/knowledge-bases/{id}/query")
	properties, ok := dataSchema["properties"].(map[string]any)
	require.True(t, ok, "OpenAPI knowledge query response must define properties")
	for _, field := range []string{"resource_id", "resource_version", "contract_version", "items", "citations"} {
		require.Contains(t, properties, field, "OpenAPI knowledge query response missing %s", field)
	}
	require.NotContains(t, properties, "provider_config", "public response must not expose provider config")

	pathNode := paths["/api/open-capabilities/knowledge-bases/{id}/query"].(map[string]any)
	methodNode := pathNode["post"].(map[string]any)
	requestBody := methodNode["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	applicationJSON := content["application/json"].(map[string]any)
	requestSchema := applicationJSON["schema"].(map[string]any)
	requestProperties := requestSchema["properties"].(map[string]any)
	require.Contains(t, requestProperties, "query")
	requiredSet := stringSetFromAny(requestSchema["required"].([]any))
	require.Contains(t, requiredSet, "query")
}

func assertOpenAPIModelDiscoverySchema(t *testing.T, paths map[string]any) {
	t.Helper()

	dataSchema := openAPIResponseDataSchema(t, paths, nil, "get", "/api/open-capabilities/models")
	properties, ok := dataSchema["properties"].(map[string]any)
	require.True(t, ok, "OpenAPI model discovery response must define properties")
	for _, field := range []string{"contract_version", "default_state", "items", "total"} {
		require.Contains(t, properties, field, "OpenAPI model discovery response missing %s", field)
	}
	defaultState := properties["default_state"].(map[string]any)
	assertOpenAPIEnum(t, defaultState, "resolved", "no_default", "multiple_defaults", "default_disabled", "default_unavailable")

	items := properties["items"].(map[string]any)
	itemSchema := items["items"].(map[string]any)
	itemProperties := itemSchema["properties"].(map[string]any)
	for _, field := range []string{"model_id", "provider_stable_id", "display_name", "is_default", "status", "disabled_reason", "capabilities", "account_id", "tenant_id"} {
		require.Contains(t, itemProperties, field, "OpenAPI model discovery item missing %s", field)
	}
	status := itemProperties["status"].(map[string]any)
	assertOpenAPIEnum(t, status, "available", "disabled", "provider_offline", "unavailable", "account_tenant_mismatch")
}

func TestModelDiscoveryUnavailableDefaultIsNotResolved(t *testing.T) {
	fixtures := fixturesByName(FixtureCatalog())
	fixture, ok := fixtures["model_discovery_model_unavailable"]
	require.True(t, ok)

	payload, ok := fixture.Payload.(dtoagentplatform.OpenCapabilityModelDiscoveryResponse)
	require.True(t, ok)
	require.Equal(t, "default_unavailable", payload.DefaultState)
	require.Len(t, payload.Items, 1)
	require.True(t, payload.Items[0].IsDefault)
	require.Equal(t, "unavailable", payload.Items[0].Status)
	require.Equal(t, "model_unavailable", payload.Items[0].DisabledReason)
}

func TestErrorClientStateMatrixFixtureCoversRequiredStates(t *testing.T) {
	fixtures := fixturesByName(FixtureCatalog())
	fixture, ok := fixtures["error_client_state_matrix"]
	require.True(t, ok)

	payload, ok := fixture.Payload.(map[string]any)
	require.True(t, ok)
	errorRows, ok := payload["error_code_matrix"].([]map[string]any)
	require.True(t, ok)
	stateRows, ok := payload["payload_state_matrix"].([]map[string]any)
	require.True(t, ok)

	errorStates := map[string]string{}
	for _, row := range errorRows {
		code, _ := row["code"].(string)
		state, _ := row["recommended_client_state"].(string)
		boundary, _ := row["boundary"].(string)
		require.NotEmpty(t, code)
		require.NotEmpty(t, state)
		require.NotEmpty(t, boundary)
		errorStates[code+"->"+state] = boundary
	}
	require.Contains(t, errorStates, "permissionDenied->loginExpired")
	require.Contains(t, errorStates, "permissionDenied->noAssignedResource")
	require.Contains(t, errorStates, "resourceRevoked->revoked")
	require.Contains(t, errorStates, "resourceOffline->offline")
	require.Contains(t, errorStates, "quotaOrRateLimited->loadFailed")
	require.Contains(t, errorStates, "timeout->networkFailed")
	require.Contains(t, errorStates, "upstreamFailed->loadFailed")
	require.Contains(t, errorStates, "contractInvalid->visibleButNotCallable")

	payloadStates := map[string]string{}
	for _, row := range stateRows {
		sourceState, _ := row["source_state"].(string)
		state, _ := row["recommended_client_state"].(string)
		payloadStates[sourceState+"->"+state] = row["boundary"].(string)
	}
	require.Contains(t, payloadStates, "stale->stale")
	require.Contains(t, payloadStates, "revoked->revoked")
	require.Contains(t, payloadStates, "offline->offline")
	require.Contains(t, payloadStates, "provider_offline->offline")
	require.Contains(t, payloadStates, "account_tenant_mismatch->noAssignedResource")
	require.Contains(t, payloadStates, "empty->empty")
}

func assertOpenAPIRequiredSchemaNodeFields(t *testing.T, schema map[string]any, requiredFields ...string) {
	t.Helper()

	required, ok := schema["required"].([]any)
	require.True(t, ok, "OpenAPI schema node must define required fields")
	requiredSet := stringSetFromAny(required)
	for _, field := range requiredFields {
		require.Contains(t, requiredSet, field)
	}
}

func stringSetFromAny(values []any) map[string]struct{} {
	set := map[string]struct{}{}
	for _, value := range values {
		if text, ok := value.(string); ok {
			set[text] = struct{}{}
		}
	}
	return set
}

func TestConsumerSignoffExtensionGovernanceMatchesFixtures(t *testing.T) {
	signoffBytes, err := os.ReadFile("../../../docs/agent-platform-consumer-signoff.md")
	require.NoError(t, err)
	signoff := string(signoffBytes)

	require.Contains(t, signoff, "Cherry Studio 私有展示字段只能进入 `extensions.cherry_studio`")
	require.Contains(t, signoff, "`extensions.codex`")

	fixtures := FixtureCatalog()
	require.True(t, fixtureHasExtensionNamespace(t, fixtures, "cherry_studio"), "Cherry Studio signoff requires a namespaced extension fixture")
	require.False(t, fixtureHasExtensionNamespace(t, fixtures, "codex"), "Codex review reserves extensions.codex but must not add Codex-specific fixture fields before needed")

	for _, fixture := range fixtures {
		if fixture.PendingReason != "" || fixture.Payload == nil {
			continue
		}

		bytes, err := common.Marshal(fixture.Payload)
		require.NoError(t, err, fixture.Name)

		var fields map[string]any
		require.NoError(t, common.Unmarshal(bytes, &fields), fixture.Name)
		extensions, ok := fields["extensions"].(map[string]any)
		if !ok {
			continue
		}
		for namespace := range extensions {
			require.Contains(t, []string{"cherry_studio", "codex"}, namespace, "fixture %s uses an ungoverned extension namespace", fixture.Name)
		}
	}
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

func assertAllowedSignoffStatuses(t *testing.T, document string) {
	t.Helper()

	allowed := map[string]struct{}{
		"draft":             {},
		"ready for signoff": {},
		"signed off":        {},
		"blocked":           {},
	}
	for _, line := range strings.Split(document, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status: ") {
			status := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "Status: ")), "`")
			require.Contains(t, allowed, status, "unexpected signoff status %q", status)
			continue
		}
		if !strings.HasPrefix(line, "|") || !strings.Contains(line, "|") {
			continue
		}
		cells := markdownTableCells(line)
		if len(cells) < 3 || strings.EqualFold(cells[0], "domain") || strings.EqualFold(cells[0], "consumer / domain") || cells[1] == "---" {
			continue
		}
		status := strings.Trim(cells[1], "`")
		if _, ok := allowed[status]; ok {
			continue
		}
		require.Contains(t, allowed, status, "unexpected signoff status %q", status)
	}
}

func markdownTableCells(line string) []string {
	parts := strings.Split(strings.Trim(line, "|"), "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func markdownSection(document string, heading string) string {
	start := strings.Index(document, heading)
	if start == -1 {
		return document
	}
	rest := document[start+len(heading):]
	if end := strings.Index(rest, "\n## "); end != -1 {
		return document[start : start+len(heading)+end]
	}
	return document[start:]
}

func fixturesByName(fixtures []Fixture) map[string]Fixture {
	byName := map[string]Fixture{}
	for _, fixture := range fixtures {
		byName[fixture.Name] = fixture
	}
	return byName
}

func fixtureHasExtensionNamespace(t *testing.T, fixtures []Fixture, namespace string) bool {
	t.Helper()

	for _, fixture := range fixtures {
		if fixture.PendingReason != "" || fixture.Payload == nil {
			continue
		}

		bytes, err := common.Marshal(fixture.Payload)
		require.NoError(t, err, fixture.Name)

		var fields map[string]any
		require.NoError(t, common.Unmarshal(bytes, &fields), fixture.Name)
		extensions, ok := fields["extensions"].(map[string]any)
		if !ok {
			continue
		}
		if _, ok := extensions[namespace]; ok {
			return true
		}
	}
	return false
}

func hasPendingSurface(fixtures []Fixture, surface string) bool {
	for _, fixture := range fixtures {
		if fixture.Surface == surface && fixture.PendingReason != "" {
			return true
		}
	}
	return false
}

func assertJSONFields(t *testing.T, fixtureName string, fields map[string]any, requiredFields ...string) {
	t.Helper()
	for _, field := range requiredFields {
		require.Contains(t, fields, field, "fixture %s missing required JSON field %s", fixtureName, field)
	}
}

func assertOpenAPIRequiredFields(t *testing.T, paths map[string]any, schemas map[string]any, method string, path string, requiredFields ...string) {
	t.Helper()

	dataSchema := openAPIResponseDataSchema(t, paths, schemas, method, path)
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

func openAPIResponseDataSchema(t *testing.T, paths map[string]any, schemas map[string]any, method string, path string) map[string]any {
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
	if schemas != nil {
		dataSchema = resolveOpenAPIRef(t, dataSchema, schemas)
	}
	return dataSchema
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
