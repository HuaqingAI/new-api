package agentplatform

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stubOpenCapabilityHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (s stubOpenCapabilityHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

type stubOpenCapabilityKnowledgeProvider struct {
	query func(ctx context.Context, binding apservice.KnowledgeProviderBinding, req apservice.KnowledgeProviderQueryRequest) (apservice.KnowledgeProviderQueryResponse, error)
}

func (s stubOpenCapabilityKnowledgeProvider) ValidateBinding(ctx context.Context, binding apservice.KnowledgeProviderBinding) error {
	return nil
}

func (s stubOpenCapabilityKnowledgeProvider) Query(ctx context.Context, binding apservice.KnowledgeProviderBinding, req apservice.KnowledgeProviderQueryRequest) (apservice.KnowledgeProviderQueryResponse, error) {
	return s.query(ctx, binding, req)
}

func (s stubOpenCapabilityKnowledgeProvider) Refresh(ctx context.Context, binding apservice.KnowledgeProviderBinding) (apservice.KnowledgeProviderRefreshState, error) {
	return apservice.KnowledgeProviderRefreshState{Status: "ready"}, nil
}

func (s stubOpenCapabilityKnowledgeProvider) Health(ctx context.Context, binding apservice.KnowledgeProviderBinding) (apservice.KnowledgeProviderHealthState, error) {
	return apservice.KnowledgeProviderHealthState{Status: "healthy"}, nil
}

type openCapabilityAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func setupOpenCapabilityControllerTest(t *testing.T) (*gin.Engine, *gorm.DB, string, apmodel.Resource, apmodel.Resource, apmodel.Resource) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.CryptoSecret = "agent-platform-test-secret"

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:                   "cherry-studio",
		DisplayName:            "Cherry Studio",
		ClientType:             "desktop",
		Status:                 "active",
		AllowedGrantTypesJSON:  `["client_credentials"]`,
		AllowedScopesJSON:      `["ap.resources.read","ap.skills.invoke","ap.knowledge.query","ap.agents.read"]`,
		ContractVersion:        "2026-06",
		CapabilitiesJSON:       `{"discovery":true}`,
		ExtensionsJSON:         `{"model_discovery.config":{"default_model":"gpt-4o-mini","account_id":"acct_demo","tenant_id":"tenant_demo","models":[{"model_id":"gpt-4o-mini","provider_stable_id":"openai","display_name":"GPT-4o Mini","capabilities":{"chat":true},"is_default":true},{"model_id":"claude-3-5-sonnet","provider_stable_id":"anthropic","display_name":"Claude 3.5 Sonnet","status":"provider_offline","disabled_reason":"provider_offline","capabilities":{"chat":true}},{"model_id":"gemini-1.5-pro","provider_stable_id":"gemini","display_name":"Gemini 1.5 Pro","account_id":"acct_other","tenant_id":"tenant_demo","capabilities":{"chat":true}}]}}`,
		AllowClientCredentials: true,
	}
	require.NoError(t, db.Create(&client).Error)

	resource := apmodel.Resource{
		ResourceType:  apmodel.ResourceTypeSkill,
		DisplayName:   "Discovery Skill",
		OwnerUserId:   100,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}
	knowledge := apmodel.Resource{
		ResourceType:  apmodel.ResourceTypeKnowledge,
		DisplayName:   "Discovery Knowledge",
		OwnerUserId:   100,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}
	agent := apmodel.Resource{
		ResourceType:  apmodel.ResourceTypeAgent,
		DisplayName:   "Discovery Agent",
		OwnerUserId:   100,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}
	require.NoError(t, db.Create(&resource).Error)
	require.NoError(t, db.Create(&knowledge).Error)
	require.NoError(t, db.Create(&agent).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      resource.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      knowledge.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      agent.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)
	require.NoError(t, db.Create(&apmodel.SkillDef{
		ResourceId: resource.ResourceId,
		FileName:   "skill.zip",
		FilePath:   "oss://bucket/skill.zip",
		Sha256:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SizeBytes:  10,
	}).Error)
	require.NoError(t, db.Create(&apmodel.KnowledgeDef{
		ResourceId:          knowledge.ResourceId,
		ExternalKnowledgeId: "kb_demo",
	}).Error)
	require.NoError(t, db.Create(&apmodel.AgentDef{
		ResourceId:      agent.ResourceId,
		ResourceVersion: "1.0.0",
		CliType:         apmodel.AgentCliTypeOpenCode,
		Name:            "Discovery Agent",
		Description:     "Agent description",
	}).Error)
	require.NoError(t, db.Create(&apmodel.AgentDependency{
		AgentResourceId:  agent.ResourceId,
		ResourceVersion:  "1.0.0",
		TargetType:       apmodel.AgentDependencyTypeSkill,
		TargetResourceId: resource.ResourceId,
		SortOrder:        0,
	}).Error)
	require.NoError(t, db.Create(&apmodel.AgentDependency{
		AgentResourceId:  agent.ResourceId,
		ResourceVersion:  "1.0.0",
		TargetType:       apmodel.AgentDependencyTypeKnowledge,
		TargetResourceId: knowledge.ResourceId,
		SortOrder:        1,
	}).Error)
	now := time.Now().UTC()
	require.NoError(t, db.Create(&apmodel.Exposure{
		ResourceId:          resource.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           client.ClientId,
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-open",
		ExtensionsJSON:      `{"cherry_studio":{"ui_variant":"enterprise"}}`,
		PublishedAt:         &now,
	}).Error)
	require.NoError(t, db.Create(&apmodel.Exposure{
		ResourceId:          knowledge.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           client.ClientId,
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-knowledge",
		PublishedAt:         &now,
	}).Error)
	require.NoError(t, db.Create(&apmodel.Exposure{
		ResourceId:          agent.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           client.ClientId,
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-agent",
		PublishedAt:         &now,
	}).Error)

	tokenResult, err := apservice.NewOAuthTokenService(db).Exchange(apservice.TokenExchangeInput{
		ClientId:  client.ClientId,
		GrantType: "client_credentials",
		Scope:     "ap.resources.read ap.skills.invoke ap.knowledge.query ap.agents.read",
	})
	require.NoError(t, err)

	router := gin.New()
	router.Use(middleware.RequestId())
	router.GET("/api/open-capabilities/discovery", middleware.AgentPlatformBearer("ap.resources.read"), OpenCapabilityDiscovery)
	router.GET("/api/open-capabilities/resources/:id", middleware.AgentPlatformBearer("ap.resources.read"), OpenCapabilityResourceDetail)
	router.GET("/api/open-capabilities/models", middleware.AgentPlatformBearer("ap.resources.read"), OpenCapabilityModelDiscovery)
	router.POST("/api/open-capabilities/refresh", middleware.AgentPlatformBearer("ap.resources.read"), OpenCapabilityRefresh)
	router.POST("/api/open-capabilities/skills/:id/invoke", middleware.AgentPlatformBearer("ap.skills.invoke"), OpenCapabilitySkillInvoke)
	router.POST("/api/open-capabilities/knowledge-bases/:id/query", middleware.AgentPlatformBearer("ap.knowledge.query"), OpenCapabilityKnowledgeQuery)
	router.GET("/api/open-capabilities/agents/:id", middleware.AgentPlatformBearer("ap.agents.read"), OpenCapabilityAgentDetail)
	return router, db, tokenResult.AccessToken, resource, knowledge, agent
}

func TestOpenCapabilityModelDiscoveryReturnsEnterpriseProjection(t *testing.T) {
	router, _, token, _, _, _ := setupOpenCapabilityControllerTest(t)

	response := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/models", token, nil)
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.True(t, apiResponse.Success)

	var data struct {
		ContractVersion string `json:"contract_version"`
		DefaultState    string `json:"default_state"`
		Items           []struct {
			ModelID          string         `json:"model_id"`
			ProviderStableID string         `json:"provider_stable_id"`
			DisplayName      string         `json:"display_name"`
			IsDefault        bool           `json:"is_default"`
			Status           string         `json:"status"`
			DisabledReason   string         `json:"disabled_reason"`
			Capabilities     map[string]any `json:"capabilities"`
			AccountID        string         `json:"account_id"`
			TenantID         string         `json:"tenant_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	require.NoError(t, common.Unmarshal(apiResponse.Data, &data))
	require.Equal(t, "2026-06", data.ContractVersion)
	require.Equal(t, "resolved", data.DefaultState)
	require.Equal(t, 3, data.Total)
	require.Len(t, data.Items, 3)
	require.Equal(t, "gpt-4o-mini", data.Items[0].ModelID)
	require.True(t, data.Items[0].IsDefault)
	require.Equal(t, "available", data.Items[0].Status)
	require.Equal(t, "acct_demo", data.Items[0].AccountID)
	require.Equal(t, "tenant_demo", data.Items[0].TenantID)
	require.Equal(t, "provider_offline", data.Items[1].Status)
	require.Equal(t, "account_tenant_mismatch", data.Items[2].Status)
	require.NotContains(t, string(apiResponse.Data), "/v1/models")
}

func performOpenCapabilityRequest(t *testing.T, router *gin.Engine, method string, target string, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = common.Marshal(body)
		require.NoError(t, err)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeOpenCapabilityAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) openCapabilityAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response openCapabilityAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestOpenCapabilityDiscoveryDetailAndRefreshWorkflow(t *testing.T) {
	router, _, token, resource, _, _ := setupOpenCapabilityControllerTest(t)

	discovery := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/discovery", token, nil)
	discoveryResp := decodeOpenCapabilityAPIResponse(t, discovery)
	require.True(t, discoveryResp.Success)

	var discoveryData struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	require.NoError(t, common.Unmarshal(discoveryResp.Data, &discoveryData))
	require.Equal(t, 3, discoveryData.Total)

	detail := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/resources/"+resource.ResourceId, token, nil)
	detailResp := decodeOpenCapabilityAPIResponse(t, detail)
	require.True(t, detailResp.Success)

	var detailData struct {
		ResourceId         string `json:"resource_id"`
		ETag               string `json:"etag"`
		ContractCompatible bool   `json:"contract_compatible"`
	}
	require.NoError(t, common.Unmarshal(detailResp.Data, &detailData))
	require.Equal(t, resource.ResourceId, detailData.ResourceId)
	require.NotEmpty(t, detailData.ETag)
	require.True(t, detailData.ContractCompatible)

	refresh := performOpenCapabilityRequest(t, router, http.MethodPost, "/api/open-capabilities/refresh", token, map[string]any{
		"resource_id":               resource.ResourceId,
		"observed_etag":             detailData.ETag,
		"observed_resource_version": "1.0.0",
	})
	refreshResp := decodeOpenCapabilityAPIResponse(t, refresh)
	require.True(t, refreshResp.Success)
}

func TestOpenCapabilityConformanceCoversStateAndRefreshDiagnostics(t *testing.T) {
	router, db, token, resource, knowledge, _ := setupOpenCapabilityControllerTest(t)

	staleAt := time.Now().UTC().Add(-10 * time.Minute)
	require.NoError(t, db.Model(&apmodel.Exposure{}).
		Where("resource_id = ?", resource.ResourceId).
		Updates(map[string]any{
			"published_at":          staleAt,
			"freshness_ttl_seconds": 300,
			"e_tag":                 "etag-stale",
			"resource_version":      "1.0.0",
			"visibility_state":      apmodel.ExposureVisibilityVisible,
			"callable_state":        apmodel.ExposureCallableEnabled,
		}).Error)

	observedAt := time.Now().UTC().Add(-20 * time.Minute).Unix()
	refresh := performOpenCapabilityRequest(t, router, http.MethodPost, "/api/open-capabilities/refresh", token, map[string]any{
		"resource_id":               resource.ResourceId,
		"observed_etag":             "etag-old",
		"observed_resource_version": "0.9.0",
		"observed_at":               observedAt,
	})
	refreshResp := decodeOpenCapabilityAPIResponse(t, refresh)
	require.True(t, refreshResp.Success)

	var refreshData struct {
		Freshness   string `json:"freshness"`
		Diagnostics struct {
			Reason             string `json:"reason"`
			Converged          bool   `json:"converged"`
			ClientNonCompliant bool   `json:"client_non_compliant"`
			ObservedETag       string `json:"observed_etag"`
			ObservedVersion    string `json:"observed_version"`
		} `json:"diagnostics"`
	}
	require.NoError(t, common.Unmarshal(refreshResp.Data, &refreshData))
	require.Equal(t, apservice.OpenCapabilityFreshnessStale, refreshData.Freshness)
	require.Equal(t, "client_non_compliant_stale", refreshData.Diagnostics.Reason)
	require.False(t, refreshData.Diagnostics.Converged)
	require.True(t, refreshData.Diagnostics.ClientNonCompliant)
	require.Equal(t, "etag-old", refreshData.Diagnostics.ObservedETag)
	require.Equal(t, "0.9.0", refreshData.Diagnostics.ObservedVersion)

	require.NoError(t, db.Model(&apmodel.Exposure{}).
		Where("resource_id = ?", resource.ResourceId).
		Updates(map[string]any{
			"visibility_state": apmodel.ExposureVisibilityRevoked,
			"callable_state":   apmodel.ExposureCallableRevoked,
		}).Error)
	revoked := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/resources/"+resource.ResourceId, token, nil)
	revokedResp := decodeOpenCapabilityAPIResponse(t, revoked)
	require.False(t, revokedResp.Success)
	assertOpenCapabilityErrorCode(t, revokedResp, apservice.OpenCapabilityCodeResourceRevoked)

	require.NoError(t, db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", knowledge.ResourceId).
		Update("status", apmodel.ResourceStatusOffline).Error)
	offline := performOpenCapabilityRequest(t, router, http.MethodPost, "/api/open-capabilities/refresh", token, map[string]any{
		"resource_id": knowledge.ResourceId,
	})
	offlineResp := decodeOpenCapabilityAPIResponse(t, offline)
	require.False(t, offlineResp.Success)
	assertOpenCapabilityErrorCode(t, offlineResp, apservice.OpenCapabilityCodeResourceOffline)
}

func TestOpenCapabilityReturnsStableErrorEnvelope(t *testing.T) {
	router, _, token, _, _, _ := setupOpenCapabilityControllerTest(t)

	response := performOpenCapabilityRequest(t, router, http.MethodPost, "/api/open-capabilities/skills/res_missing/invoke", token, map[string]any{
		"input": "demo",
	})
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.False(t, apiResponse.Success)

	var errorPayload struct {
		Code       string `json:"code"`
		Message    string `json:"message"`
		Retryable  bool   `json:"retryable"`
		RequestID  string `json:"request_id"`
		ResourceID string `json:"resource_id"`
	}
	require.NoError(t, common.Unmarshal(apiResponse.Error, &errorPayload))
	require.Equal(t, apservice.OpenCapabilityCodePermissionDenied, errorPayload.Code)
	require.NotEmpty(t, errorPayload.RequestID)
	require.Equal(t, "res_missing", errorPayload.ResourceID)
}

func TestOpenCapabilitySkillInvokeReturnsContractInvalidWithoutConfig(t *testing.T) {
	router, _, token, resource, _, _ := setupOpenCapabilityControllerTest(t)

	response := performOpenCapabilityRequest(t, router, http.MethodPost, "/api/open-capabilities/skills/"+resource.ResourceId+"/invoke", token, map[string]any{
		"input": "demo",
	})
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.False(t, apiResponse.Success)
	assertOpenCapabilityErrorCode(t, apiResponse, apservice.OpenCapabilityCodeContractInvalid)
}

func TestOpenCapabilityKnowledgeQueryReturnsContractInvalidWithoutProviderConfig(t *testing.T) {
	router, _, token, _, knowledge, _ := setupOpenCapabilityControllerTest(t)

	response := performOpenCapabilityRequest(t, router, http.MethodPost, "/api/open-capabilities/knowledge-bases/"+knowledge.ResourceId+"/query", token, map[string]any{
		"query": "what is the answer?",
	})
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.False(t, apiResponse.Success)
	assertOpenCapabilityErrorCode(t, apiResponse, apservice.OpenCapabilityCodeContractInvalid)
}

func TestOpenCapabilityBearerRejectsMissingScope(t *testing.T) {
	router, db, _, resource, _, _ := setupOpenCapabilityControllerTest(t)

	client := apmodel.Client{
		Slug:                   "narrow-client",
		DisplayName:            "Narrow Client",
		ClientType:             "desktop",
		Status:                 "active",
		AllowedGrantTypesJSON:  `["client_credentials"]`,
		AllowedScopesJSON:      `["ap.resources.read"]`,
		ContractVersion:        "2026-06",
		CapabilitiesJSON:       `{"discovery":true}`,
		AllowClientCredentials: true,
	}
	require.NoError(t, db.Create(&client).Error)
	tokenResult, err := apservice.NewOAuthTokenService(db).Exchange(apservice.TokenExchangeInput{
		ClientId:  client.ClientId,
		GrantType: "client_credentials",
		Scope:     "ap.resources.read",
	})
	require.NoError(t, err)

	response := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/agents/"+resource.ResourceId, tokenResult.AccessToken, nil)
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.False(t, apiResponse.Success)

	var errorPayload struct {
		Code string `json:"code"`
	}
	require.NoError(t, common.Unmarshal(apiResponse.Error, &errorPayload))
	require.Equal(t, apservice.OpenCapabilityCodePermissionDenied, errorPayload.Code)
}

func TestOpenCapabilityBearerRejectsExpiredToken(t *testing.T) {
	router, db, _, _, _, _ := setupOpenCapabilityControllerTest(t)
	token := expiredOpenCapabilityAccessToken(t, db)

	response := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/discovery", token, nil)
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.False(t, apiResponse.Success)
	assertOpenCapabilityErrorCode(t, apiResponse, apservice.OpenCapabilityCodePermissionDenied)
}

func TestOpenCapabilityBearerRejectsRevokedGrantToken(t *testing.T) {
	router, db, token, _, _, _ := setupOpenCapabilityControllerTest(t)
	revokedAt := time.Now().UTC()
	require.NoError(t, db.Model(&apmodel.AuthorizationGrant{}).
		Where("status = ?", "issued").
		Updates(map[string]any{
			"status":     "revoked",
			"revoked_at": revokedAt,
		}).Error)

	response := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/discovery", token, nil)
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.False(t, apiResponse.Success)
	assertOpenCapabilityErrorCode(t, apiResponse, apservice.OpenCapabilityCodePermissionDenied)
}

func TestOpenCapabilityDetailRejectsContractVersionMismatch(t *testing.T) {
	router, db, token, resource, _, _ := setupOpenCapabilityControllerTest(t)
	require.NoError(t, db.Model(&apmodel.Client{}).Where("slug = ?", "cherry-studio").Update("contract_version", "2026-07").Error)

	response := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/resources/"+resource.ResourceId, token, nil)
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.False(t, apiResponse.Success)

	var errorPayload struct {
		Code string `json:"code"`
	}
	require.NoError(t, common.Unmarshal(apiResponse.Error, &errorPayload))
	require.Equal(t, apservice.OpenCapabilityCodeContractInvalid, errorPayload.Code)
}

func TestOpenCapabilityAgentDetailReturnsDependencyBoundary(t *testing.T) {
	router, _, token, _, _, agent := setupOpenCapabilityControllerTest(t)

	response := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/agents/"+agent.ResourceId, token, nil)
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.True(t, apiResponse.Success)

	var data struct {
		CallableState string         `json:"callable_state"`
		Detail        map[string]any `json:"detail"`
	}
	require.NoError(t, common.Unmarshal(apiResponse.Data, &data))
	require.Equal(t, "enabled", data.CallableState)
	_, ok := data.Detail["dependencies"]
	require.True(t, ok)
}

func TestOpenCapabilityAgentDetailBecomesVisibleButNotCallableWhenDependencyBreaks(t *testing.T) {
	router, db, token, skill, _, agent := setupOpenCapabilityControllerTest(t)
	require.NoError(t, db.Model(&apmodel.Exposure{}).
		Where("resource_id = ?", skill.ResourceId).
		Update("callable_state", apmodel.ExposureCallableRevoked).Error)

	response := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/agents/"+agent.ResourceId, token, nil)
	apiResponse := decodeOpenCapabilityAPIResponse(t, response)
	require.True(t, apiResponse.Success)

	var data struct {
		CallableState string `json:"callable_state"`
		Diagnostics   struct {
			Reason string `json:"reason"`
		} `json:"diagnostics"`
	}
	require.NoError(t, common.Unmarshal(apiResponse.Data, &data))
	require.Equal(t, "contract_invalid", data.CallableState)
	require.Equal(t, "dependency_not_callable", data.Diagnostics.Reason)
}

func assertOpenCapabilityErrorCode(t *testing.T, apiResponse openCapabilityAPIResponse, expected string) {
	t.Helper()
	var errorPayload struct {
		Code string `json:"code"`
	}
	require.NoError(t, common.Unmarshal(apiResponse.Error, &errorPayload))
	require.Equal(t, expected, errorPayload.Code)
}

func expiredOpenCapabilityAccessToken(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var grant apmodel.AuthorizationGrant
	require.NoError(t, db.Where("status = ?", "issued").First(&grant).Error)

	now := time.Now().UTC()
	claims := apservice.TokenClaims{
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
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Minute)),
			ID:        "jti_expired_fixture",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(common.CryptoSecret))
	require.NoError(t, err)
	return token
}
