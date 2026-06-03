package conformance

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
)

const (
	ContractVersion     = "2026-06"
	ClientID            = "client_cherry_mock"
	TenantID            = "tenant_demo"
	AccountID           = "acct_demo"
	SkillResourceID     = "res_skill_demo"
	KnowledgeResourceID = "res_knowledge_demo"
	AgentResourceID     = "res_agent_demo"
	ModelID             = "model_demo_default"
	ProviderStableID    = "provider_demo"
)

type Fixture struct {
	Name          string
	Surface       string
	Method        string
	Path          string
	Description   string
	Payload       any
	PendingReason string
}

type OAuthContractErrorResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func FixtureCatalog() []Fixture {
	return []Fixture{
		{
			Name:        "oauth_authorize_success",
			Surface:     "oauth",
			Method:      "GET",
			Path:        "/api/agent-platform/oauth/authorize",
			Description: "PKCE authorize succeeds and records consent.",
			Payload: dtoagentplatform.OAuthAuthorizeResponse{
				ClientId:          ClientID,
				ContractVersion:   ContractVersion,
				Scope:             "ap.resources.read ap.skills.invoke",
				State:             "state_demo",
				AuthorizationCode: "code_synthetic_authorization",
				RedirectURI:       "cherrystudio://oauth/callback",
				ConsentRecorded:   true,
			},
		},
		{
			Name:        "oauth_token_success",
			Surface:     "oauth",
			Method:      "POST",
			Path:        "/api/agent-platform/oauth/token",
			Description: "Token exchange succeeds with synthetic non-secret token placeholders.",
			Payload: dtoagentplatform.OAuthTokenResponse{
				AccessToken:      "access_token_synthetic_fixture",
				TokenType:        "Bearer",
				ExpiresIn:        300,
				RefreshToken:     "refresh_token_synthetic_fixture",
				RefreshExpiresIn: 2592000,
				Scope:            "ap.resources.read ap.skills.invoke",
				ContractVersion:  ContractVersion,
				GrantId:          "grant_demo",
			},
		},
		{
			Name:        "oauth_refresh_rotation_success",
			Surface:     "oauth",
			Method:      "POST",
			Path:        "/api/agent-platform/oauth/token",
			Description: "Refresh token exchange rotates the refresh token and revokes the prior token.",
			Payload: dtoagentplatform.OAuthTokenResponse{
				AccessToken:      "access_token_synthetic_fixture_rotated",
				TokenType:        "Bearer",
				ExpiresIn:        300,
				RefreshToken:     "refresh_token_synthetic_fixture_rotated",
				RefreshExpiresIn: 2592000,
				Scope:            "ap.resources.read ap.skills.invoke",
				ContractVersion:  ContractVersion,
				GrantId:          "grant_demo",
			},
		},
		{
			Name:        "oauth_revoke_success",
			Surface:     "oauth",
			Method:      "POST",
			Path:        "/api/agent-platform/oauth/revoke",
			Description: "Token revoke succeeds with synthetic token input.",
			Payload:     dtoagentplatform.OAuthRevokeResponse{Revoked: true},
		},
		errorFixture("oauth_expired_token", "oauth", "GET", "/api/open-capabilities/discovery", apservice.OpenCapabilityCodePermissionDenied, "expired token is rejected", false),
		errorFixture("oauth_revoked_grant_token", "oauth", "GET", "/api/open-capabilities/discovery", apservice.OpenCapabilityCodePermissionDenied, "revoked grant/token is rejected", false),
		errorFixture("oauth_missing_scope_permission_denied", "oauth", "GET", "/api/open-capabilities/agents/res_skill_demo", apservice.OpenCapabilityCodePermissionDenied, "missing scope is permission denied", false),
		oauthErrorFixture("oauth_redirect_mismatch", "GET", "/api/agent-platform/oauth/authorize", "redirect URI is not registered for the integration client", false),
		oauthErrorFixture("oauth_invalid_pkce", "POST", "/api/agent-platform/oauth/token", "PKCE verifier or challenge is invalid", false),
		oauthErrorFixture("oauth_invalid_client", "GET", "/api/agent-platform/oauth/authorize", "client is not found or unauthorized", false),
		oauthErrorFixture("oauth_invalid_integration", "GET", "/api/agent-platform/oauth/authorize", "integration client is inactive or missing contract/capabilities", false),
		{
			Name:        "discovery_empty",
			Surface:     "open-capabilities",
			Method:      "GET",
			Path:        "/api/open-capabilities/discovery",
			Description: "Discovery returns an empty result set.",
			Payload: dtoagentplatform.OpenCapabilityDiscoveryResponse{
				Items: []dtoagentplatform.OpenCapabilityDiscoveryItem{},
				Total: 0,
			},
		},
		{
			Name:        "discovery_success_multi_resource",
			Surface:     "open-capabilities",
			Method:      "GET",
			Path:        "/api/open-capabilities/discovery",
			Description: "Discovery returns skill, knowledge, and agent resources.",
			Payload: dtoagentplatform.OpenCapabilityDiscoveryResponse{
				Items: []dtoagentplatform.OpenCapabilityDiscoveryItem{
					discoveryItem(SkillResourceID, "skill", "Demo Skill", "etag-skill-demo", raw(`{"cherry_studio":{"ui_variant":"enterprise"}}`)),
					discoveryItem(KnowledgeResourceID, "knowledge", "Demo Knowledge", "etag-knowledge-demo", nil),
					discoveryItem(AgentResourceID, "agent", "Demo Agent", "etag-agent-demo", nil),
				},
				Total: 3,
			},
		},
		errorFixture("discovery_contract_mismatch_filtered_or_rejected", "open-capabilities", "GET", "/api/open-capabilities/discovery", apservice.OpenCapabilityCodeContractInvalid, "contract mismatch is filtered from discovery or rejected by detail", false),
		detailFixture("detail_visible_callable", SkillResourceID, "skill", "published", "visible", "enabled", true, dtoagentplatform.OpenCapabilityDiagnostics{Reason: "in_sync", Converged: true}),
		detailFixture("detail_visible_not_callable", AgentResourceID, "agent", "published", "visible", "contract_invalid", false, dtoagentplatform.OpenCapabilityDiagnostics{Reason: "dependency_not_callable", Converged: false}),
		errorFixture("detail_contract_invalid", "open-capabilities", "GET", "/api/open-capabilities/resources/"+SkillResourceID, apservice.OpenCapabilityCodeContractInvalid, "contract invalid detail is rejected", false),
		errorFixture("detail_resource_revoked", "open-capabilities", "GET", "/api/open-capabilities/resources/"+SkillResourceID, apservice.OpenCapabilityCodeResourceRevoked, "revoked resource is rejected", false),
		errorFixture("detail_resource_offline", "open-capabilities", "GET", "/api/open-capabilities/resources/"+SkillResourceID, apservice.OpenCapabilityCodeResourceOffline, "offline resource is rejected", true),
		refreshFixture("refresh_fresh", "fresh", "etag-skill-demo", dtoagentplatform.OpenCapabilityDiagnostics{Reason: "in_sync", Converged: true}),
		refreshFixture("refresh_stale", "stale", "etag-skill-demo", dtoagentplatform.OpenCapabilityDiagnostics{Reason: "projection_stale", Converged: false}),
		errorFixture("refresh_revoked", "open-capabilities", "POST", "/api/open-capabilities/refresh", apservice.OpenCapabilityCodeResourceRevoked, "revoked resource refresh is rejected", false),
		errorFixture("refresh_offline", "open-capabilities", "POST", "/api/open-capabilities/refresh", apservice.OpenCapabilityCodeResourceOffline, "offline resource refresh is rejected", true),
		refreshFixture("refresh_observed_etag_version_mismatch", "fresh", "etag-skill-demo-v2", dtoagentplatform.OpenCapabilityDiagnostics{Reason: "client_version_mismatch", Converged: false, ObservedETag: "etag-skill-demo", ObservedVersion: "0.9.0"}),
		refreshFixture("refresh_ttl_over_300_non_compliance", "stale", "etag-skill-demo", dtoagentplatform.OpenCapabilityDiagnostics{Reason: "client_non_compliant_stale", Converged: false, ClientNonCompliant: true}),
		{
			Name:        "skill_invoke_sync_success",
			Surface:     "open-capabilities",
			Method:      "POST",
			Path:        "/api/open-capabilities/skills/" + SkillResourceID + "/invoke",
			Description: "Synchronous skill invocation returns provider output under output.",
			Payload: dtoagentplatform.OpenCapabilitySkillInvokeResponse{
				ResourceId:      SkillResourceID,
				ResourceVersion: "1.0.0",
				ContractVersion: ContractVersion,
				Output:          map[string]any{"ok": true},
			},
		},
		errorFixture("skill_invoke_contract_invalid", "open-capabilities", "POST", "/api/open-capabilities/skills/"+SkillResourceID+"/invoke", apservice.OpenCapabilityCodeContractInvalid, "invalid skill invocation contract is rejected", false),
		errorFixture("skill_invoke_timeout", "open-capabilities", "POST", "/api/open-capabilities/skills/"+SkillResourceID+"/invoke", apservice.OpenCapabilityCodeTimeout, "skill timeout is retryable", true),
		errorFixture("skill_invoke_upstream_provider_failure", "open-capabilities", "POST", "/api/open-capabilities/skills/"+SkillResourceID+"/invoke", apservice.OpenCapabilityCodeUpstreamFailed, "upstream provider failure is retryable", true),
		errorFixture("open_capability_quota_or_rate_limited", "open-capabilities", "POST", "/api/open-capabilities/skills/"+SkillResourceID+"/invoke", apservice.OpenCapabilityCodeQuotaLimited, "quota or rate limit is exposed as a stable error code", true),
		{
			Name:        "knowledge_query_retrieval_success_items_citations",
			Surface:     "open-capabilities",
			Method:      "POST",
			Path:        "/api/open-capabilities/knowledge-bases/" + KnowledgeResourceID + "/query",
			Description: "Knowledge query returns standardized items and citations without provider-native leakage.",
			Payload: dtoagentplatform.OpenCapabilityKnowledgeQueryResponse{
				ResourceId:      KnowledgeResourceID,
				ResourceVersion: "1.0.0",
				ContractVersion: ContractVersion,
				Items: []dtoagentplatform.OpenCapabilityKnowledgeResultItem{
					{ID: "doc_demo", Score: 0.91, Snippet: "Synthetic retrieval result", Metadata: map[string]any{"source_type": "fixture"}},
				},
				Citations: []dtoagentplatform.OpenCapabilityKnowledgeCitation{
					{SourceID: "doc_demo", Title: "Synthetic Doc"},
				},
			},
		},
		errorFixture("knowledge_query_provider_offline", "open-capabilities", "POST", "/api/open-capabilities/knowledge-bases/"+KnowledgeResourceID+"/query", apservice.OpenCapabilityCodeResourceOffline, "offline knowledge provider is rejected", true),
		errorFixture("knowledge_query_upstream_failure", "open-capabilities", "POST", "/api/open-capabilities/knowledge-bases/"+KnowledgeResourceID+"/query", apservice.OpenCapabilityCodeUpstreamFailed, "knowledge provider failure is retryable", true),
		modelDiscoveryFixture("model_discovery_default_model", "resolved", []dtoagentplatform.OpenCapabilityModelDiscoveryItem{
			modelDiscoveryItem("gpt-4o-mini", "openai", "GPT-4o Mini", true, "available", "", map[string]any{"chat": true}, AccountID, TenantID),
			modelDiscoveryItem("claude-3-5-sonnet", "anthropic", "Claude 3.5 Sonnet", false, "available", "", map[string]any{"chat": true}, AccountID, TenantID),
		}),
		modelDiscoveryFixture("model_discovery_no_default_model", "no_default", []dtoagentplatform.OpenCapabilityModelDiscoveryItem{
			modelDiscoveryItem("gpt-4o-mini", "openai", "GPT-4o Mini", false, "available", "", map[string]any{"chat": true}, AccountID, TenantID),
		}),
		modelDiscoveryFixture("model_discovery_multiple_default_models", "multiple_defaults", []dtoagentplatform.OpenCapabilityModelDiscoveryItem{
			modelDiscoveryItem("gpt-4o-mini", "openai", "GPT-4o Mini", true, "available", "", map[string]any{"chat": true}, AccountID, TenantID),
			modelDiscoveryItem("claude-3-5-sonnet", "anthropic", "Claude 3.5 Sonnet", true, "available", "", map[string]any{"chat": true}, AccountID, TenantID),
		}),
		modelDiscoveryFixture("model_discovery_default_disabled", "default_disabled", []dtoagentplatform.OpenCapabilityModelDiscoveryItem{
			modelDiscoveryItem("gpt-4o-mini", "openai", "GPT-4o Mini", true, "disabled", "default_model_disabled", map[string]any{"chat": true}, AccountID, TenantID),
		}),
		modelDiscoveryFixture("model_discovery_provider_offline", "resolved", []dtoagentplatform.OpenCapabilityModelDiscoveryItem{
			modelDiscoveryItem("claude-3-5-sonnet", "anthropic", "Claude 3.5 Sonnet", true, "provider_offline", "provider_offline", map[string]any{"chat": true}, AccountID, TenantID),
		}),
		modelDiscoveryFixture("model_discovery_model_unavailable", "resolved", []dtoagentplatform.OpenCapabilityModelDiscoveryItem{
			modelDiscoveryItem(ModelID, ProviderStableID, "Default Model", true, "unavailable", "model_unavailable", map[string]any{"chat": true}, AccountID, TenantID),
		}),
		modelDiscoveryFixture("model_discovery_account_tenant_mismatch", "resolved", []dtoagentplatform.OpenCapabilityModelDiscoveryItem{
			modelDiscoveryItem("gemini-1.5-pro", "gemini", "Gemini 1.5 Pro", true, "account_tenant_mismatch", "account_tenant_mismatch", map[string]any{"chat": true}, "acct_other", TenantID),
		}),
		pendingFixture("error_client_state_matrix", "error-matrix", "AP-6.6 error code and client state matrix artifact is missing; state mapping assertions are pending."),
	}
}

func oauthErrorFixture(name string, method string, path string, message string, retryable bool) Fixture {
	return Fixture{
		Name:        name,
		Surface:     "oauth",
		Method:      method,
		Path:        path,
		Description: message,
		Payload: OAuthContractErrorResponse{
			Success:   false,
			Message:   message,
			Retryable: retryable,
		},
	}
}

func discoveryItem(resourceID string, resourceType string, displayName string, etag string, extensions json.RawMessage) dtoagentplatform.OpenCapabilityDiscoveryItem {
	return dtoagentplatform.OpenCapabilityDiscoveryItem{
		ResourceId:          resourceID,
		ResourceType:        resourceType,
		DisplayName:         displayName,
		ResourceVersion:     "1.0.0",
		ContractVersion:     ContractVersion,
		VisibilityState:     "visible",
		CallableState:       "enabled",
		FreshnessTTLSeconds: 300,
		Freshness:           "fresh",
		ETag:                etag,
		Extensions:          extensions,
	}
}

func detailFixture(name string, resourceID string, resourceType string, status string, visibility string, callable string, compatible bool, diagnostics dtoagentplatform.OpenCapabilityDiagnostics) Fixture {
	return Fixture{
		Name:        name,
		Surface:     "open-capabilities",
		Method:      "GET",
		Path:        "/api/open-capabilities/resources/" + resourceID,
		Description: "Detail response for " + name + ".",
		Payload: dtoagentplatform.OpenCapabilityDetailResponse{
			ResourceId:          resourceID,
			ResourceType:        resourceType,
			DisplayName:         "Demo " + resourceType,
			ResourceVersion:     "1.0.0",
			ContractVersion:     ContractVersion,
			Status:              status,
			VisibilityState:     visibility,
			CallableState:       callable,
			FreshnessTTLSeconds: 300,
			Freshness:           "fresh",
			ETag:                "etag-" + resourceID,
			Schema:              raw(`{"type":"object"}`),
			Detail:              raw(`{"mode":"fixture"}`),
			Extensions:          raw(`{"cherry_studio":{"fixture":true}}`),
			SupportedExtensions: []string{"cherry_studio"},
			ContractCompatible:  compatible,
			Diagnostics:         diagnostics,
		},
	}
}

func refreshFixture(name string, freshness string, etag string, diagnostics dtoagentplatform.OpenCapabilityDiagnostics) Fixture {
	return Fixture{
		Name:        name,
		Surface:     "open-capabilities",
		Method:      "POST",
		Path:        "/api/open-capabilities/refresh",
		Description: "Refresh response for " + name + ".",
		Payload: dtoagentplatform.OpenCapabilityRefreshResponse{
			ResourceId:          SkillResourceID,
			ResourceVersion:     "1.0.0",
			ContractVersion:     ContractVersion,
			FreshnessTTLSeconds: 300,
			Freshness:           freshness,
			ETag:                etag,
			VisibilityState:     "visible",
			CallableState:       "enabled",
			ContractCompatible:  true,
			Diagnostics:         diagnostics,
		},
	}
}

func modelDiscoveryFixture(name string, defaultState string, items []dtoagentplatform.OpenCapabilityModelDiscoveryItem) Fixture {
	return Fixture{
		Name:        name,
		Surface:     "model-discovery",
		Method:      "GET",
		Path:        "/api/open-capabilities/models",
		Description: "Enterprise model discovery response for " + name + ".",
		Payload: dtoagentplatform.OpenCapabilityModelDiscoveryResponse{
			ContractVersion: ContractVersion,
			DefaultState:    defaultState,
			Items:           items,
			Total:           len(items),
		},
	}
}

func modelDiscoveryItem(modelID string, providerStableID string, displayName string, isDefault bool, status string, disabledReason string, capabilities map[string]any, accountID string, tenantID string) dtoagentplatform.OpenCapabilityModelDiscoveryItem {
	capabilitiesJSON, _ := common.Marshal(capabilities)
	return dtoagentplatform.OpenCapabilityModelDiscoveryItem{
		ModelID:          modelID,
		ProviderStableID: providerStableID,
		DisplayName:      displayName,
		IsDefault:        isDefault,
		Status:           status,
		DisabledReason:   disabledReason,
		Capabilities:     json.RawMessage(capabilitiesJSON),
		AccountID:        accountID,
		TenantID:         tenantID,
	}
}

func errorFixture(name string, surface string, method string, path string, code string, message string, retryable bool) Fixture {
	return Fixture{
		Name:        name,
		Surface:     surface,
		Method:      method,
		Path:        path,
		Description: message,
		Payload: apservice.OpenCapabilityErrorResponse{
			Success: false,
			Error: apservice.OpenCapabilityError{
				Code:            code,
				Message:         message,
				Retryable:       retryable,
				RequestID:       "req_fixture_demo",
				ResourceID:      SkillResourceID,
				ResourceVersion: "1.0.0",
			},
		},
	}
}

func pendingFixture(name string, surface string, reason string) Fixture {
	if surface == "model-discovery" {
		reason += " This pending case must not be satisfied by /api/models, /api/user/models, or /v1/models."
	}
	return Fixture{
		Name:          name,
		Surface:       surface,
		Method:        "PENDING",
		Path:          "pending",
		Description:   "Pending until prerequisite contract artifact is frozen.",
		PendingReason: reason,
	}
}

func raw(text string) json.RawMessage {
	return json.RawMessage(text)
}
