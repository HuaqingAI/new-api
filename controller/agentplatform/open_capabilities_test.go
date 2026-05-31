package agentplatform

import (
	"bytes"
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
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type openCapabilityAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func setupOpenCapabilityControllerTest(t *testing.T) (*gin.Engine, *gorm.DB, string, apmodel.Resource) {
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
		AllowedScopesJSON:      `["ap.resources.read","ap.skills.invoke","ap.agents.read"]`,
		ContractVersion:        "2026-06",
		CapabilitiesJSON:       `{"discovery":true}`,
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
	require.NoError(t, db.Create(&resource).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      resource.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		SchemaJSON:      `{"type":"object"}`,
		DetailJSON:      `{"skill":{"invoke_mode":"sync"}}`,
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
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

	tokenResult, err := apservice.NewOAuthTokenService(db).Exchange(apservice.TokenExchangeInput{
		ClientId:  client.ClientId,
		GrantType: "client_credentials",
		Scope:     "ap.resources.read ap.skills.invoke ap.agents.read",
	})
	require.NoError(t, err)

	router := gin.New()
	router.Use(middleware.RequestId())
	router.GET("/api/open-capabilities/discovery", middleware.AgentPlatformBearer("ap.resources.read"), OpenCapabilityDiscovery)
	router.GET("/api/open-capabilities/resources/:id", middleware.AgentPlatformBearer("ap.resources.read"), OpenCapabilityResourceDetail)
	router.POST("/api/open-capabilities/refresh", middleware.AgentPlatformBearer("ap.resources.read"), OpenCapabilityRefresh)
	router.POST("/api/open-capabilities/skills/:id/invoke", middleware.AgentPlatformBearer("ap.skills.invoke"), OpenCapabilitySkillInvoke)
	router.GET("/api/open-capabilities/agents/:id", middleware.AgentPlatformBearer("ap.agents.read"), OpenCapabilityAgentDetail)
	return router, db, tokenResult.AccessToken, resource
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
	router, _, token, resource := setupOpenCapabilityControllerTest(t)

	discovery := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/discovery", token, nil)
	discoveryResp := decodeOpenCapabilityAPIResponse(t, discovery)
	require.True(t, discoveryResp.Success)

	var discoveryData struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	require.NoError(t, common.Unmarshal(discoveryResp.Data, &discoveryData))
	require.Equal(t, 1, discoveryData.Total)

	detail := performOpenCapabilityRequest(t, router, http.MethodGet, "/api/open-capabilities/resources/"+resource.ResourceId, token, nil)
	detailResp := decodeOpenCapabilityAPIResponse(t, detail)
	require.True(t, detailResp.Success)

	var detailData struct {
		ResourceId string `json:"resource_id"`
		ETag       string `json:"etag"`
	}
	require.NoError(t, common.Unmarshal(detailResp.Data, &detailData))
	require.Equal(t, resource.ResourceId, detailData.ResourceId)
	require.NotEmpty(t, detailData.ETag)

	refresh := performOpenCapabilityRequest(t, router, http.MethodPost, "/api/open-capabilities/refresh", token, map[string]any{
		"resource_id": resource.ResourceId,
	})
	refreshResp := decodeOpenCapabilityAPIResponse(t, refresh)
	require.True(t, refreshResp.Success)
}

func TestOpenCapabilityReturnsStableErrorEnvelope(t *testing.T) {
	router, _, token, _ := setupOpenCapabilityControllerTest(t)

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
	require.Equal(t, apservice.OpenCapabilityCodeContractInvalid, errorPayload.Code)
	require.NotEmpty(t, errorPayload.RequestID)
	require.Equal(t, "res_missing", errorPayload.ResourceID)
}

func TestOpenCapabilityBearerRejectsMissingScope(t *testing.T) {
	router, db, _, resource := setupOpenCapabilityControllerTest(t)

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
