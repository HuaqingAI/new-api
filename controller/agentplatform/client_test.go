package agentplatform

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type clientAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupClientControllerTest(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, apmodel.Migrate(db))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 999)
		c.Set("role", common.RoleAdminUser)
		c.Next()
	})
	router.GET("/api/agent-platform/clients", ListClients)
	router.POST("/api/agent-platform/clients", CreateClient)
	router.GET("/api/agent-platform/clients/:id", GetClient)
	router.PUT("/api/agent-platform/clients/:id", UpdateClient)
	return router, db
}

func performClientRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = common.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeClientAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) clientAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response clientAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func decodeClientData[T any](t *testing.T, response clientAPIResponse) T {
	t.Helper()
	var data T
	require.NoError(t, common.Unmarshal(response.Data, &data))
	return data
}

func TestClientAPIWorkflow(t *testing.T) {
	router, _ := setupClientControllerTest(t)

	create := performClientRequest(t, router, http.MethodPost, "/api/agent-platform/clients", dtoagentplatform.CreateClientRequest{
		Slug:                   "cherry-studio",
		DisplayName:            "Cherry Studio",
		ClientType:             "desktop",
		ContractVersion:        "2026-06",
		Capabilities:           json.RawMessage(`{"discovery":true}`),
		AllowedGrantTypes:      json.RawMessage(`["authorization_code","refresh_token"]`),
		RedirectURIs:           json.RawMessage(`["cherrystudio://oauth/callback"]`),
		AllowedScopes:          json.RawMessage(`["skills.read"]`),
		Extensions:             json.RawMessage(`{"cherry_studio.fixture":true}`),
		AllowClientCredentials: true,
	})
	createResp := decodeClientAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)
	client := decodeClientData[dtoagentplatform.ClientItem](t, createResp)
	require.Equal(t, "active", client.Status)
	require.True(t, client.AllowClientCredentials)
	require.JSONEq(t, `["authorization_code","refresh_token"]`, string(client.AllowedGrantTypes))
	require.JSONEq(t, `["cherrystudio://oauth/callback"]`, string(client.RedirectURIs))
	require.JSONEq(t, `["skills.read"]`, string(client.AllowedScopes))

	get := performClientRequest(t, router, http.MethodGet, "/api/agent-platform/clients/"+client.ClientId, nil)
	getResp := decodeClientAPIResponse(t, get)
	require.True(t, getResp.Success, getResp.Message)

	update := performClientRequest(t, router, http.MethodPut, "/api/agent-platform/clients/"+client.ClientId, dtoagentplatform.UpdateClientRequest{
		Status: "disabled",
	})
	updateResp := decodeClientAPIResponse(t, update)
	require.True(t, updateResp.Success, updateResp.Message)
	updated := decodeClientData[dtoagentplatform.ClientItem](t, updateResp)
	require.Equal(t, "disabled", updated.Status)

	list := performClientRequest(t, router, http.MethodGet, "/api/agent-platform/clients", nil)
	listResp := decodeClientAPIResponse(t, list)
	require.True(t, listResp.Success, listResp.Message)
	listData := decodeClientData[dtoagentplatform.ClientListResponse](t, listResp)
	require.Equal(t, 1, listData.Total)
}

func TestClientAPIOnboardingRegistrationRoundTrip(t *testing.T) {
	router, _ := setupClientControllerTest(t)

	create := performClientRequest(t, router, http.MethodPost, "/api/agent-platform/clients", map[string]any{
		"slug":                     "Codex",
		"display_name":             "Codex CLI",
		"client_type":              "CLI",
		"allowed_grant_types":      []string{"authorization_code", "refresh_token"},
		"redirect_uris":            []string{"http://127.0.0.1:1455/oauth/callback"},
		"allowed_scopes":           []string{"ap.resources.read", "ap.skills.invoke"},
		"contract_version":         "2026-06",
		"capabilities":             map[string]any{"discovery": true, "skills": true},
		"extensions":               map[string]any{"codex.owner": "platform-ops"},
		"allow_client_credentials": false,
	})
	createResp := decodeClientAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)
	client := decodeClientData[dtoagentplatform.ClientItem](t, createResp)
	require.NotEmpty(t, client.ClientId)
	require.Equal(t, "codex", client.Slug)
	require.Equal(t, "cli", client.ClientType)
	require.Equal(t, "active", client.Status)
	require.False(t, client.AllowClientCredentials)
	require.JSONEq(t, `["authorization_code","refresh_token"]`, string(client.AllowedGrantTypes))
	require.JSONEq(t, `["ap.resources.read","ap.skills.invoke"]`, string(client.AllowedScopes))
	require.JSONEq(t, `{"codex.owner":"platform-ops"}`, string(client.Extensions))

	list := performClientRequest(t, router, http.MethodGet, "/api/agent-platform/clients?contract_version=2026-06", nil)
	listResp := decodeClientAPIResponse(t, list)
	require.True(t, listResp.Success, listResp.Message)
	listData := decodeClientData[dtoagentplatform.ClientListResponse](t, listResp)
	require.Equal(t, 1, listData.Total)
	require.Equal(t, client.ClientId, listData.Items[0].ClientId)
	require.Equal(t, "2026-06", listData.Items[0].ContractVersion)

	allow := true
	update := performClientRequest(t, router, http.MethodPut, "/api/agent-platform/clients/"+client.ClientId, dtoagentplatform.UpdateClientRequest{
		DisplayName:            "Codex CLI managed",
		Status:                 "disabled",
		AllowedScopes:          json.RawMessage(`["ap.resources.read"]`),
		AllowClientCredentials: &allow,
	})
	updateResp := decodeClientAPIResponse(t, update)
	require.True(t, updateResp.Success, updateResp.Message)
	updated := decodeClientData[dtoagentplatform.ClientItem](t, updateResp)
	require.Equal(t, "Codex CLI managed", updated.DisplayName)
	require.Equal(t, "disabled", updated.Status)
	require.True(t, updated.AllowClientCredentials)
	require.JSONEq(t, `["ap.resources.read"]`, string(updated.AllowedScopes))
}

func TestClientAPIExposesInvalidIntegrationForMinimalOnboarding(t *testing.T) {
	router, _ := setupClientControllerTest(t)

	create := performClientRequest(t, router, http.MethodPost, "/api/agent-platform/clients", dtoagentplatform.CreateClientRequest{
		Slug:        "draft-client",
		DisplayName: "Draft Client",
		ClientType:  "desktop",
	})
	createResp := decodeClientAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)
	client := decodeClientData[dtoagentplatform.ClientItem](t, createResp)
	require.Equal(t, "invalid_integration", client.Status)

	list := performClientRequest(t, router, http.MethodGet, "/api/agent-platform/clients?status=invalid_integration", nil)
	listResp := decodeClientAPIResponse(t, list)
	require.True(t, listResp.Success, listResp.Message)
	listData := decodeClientData[dtoagentplatform.ClientListResponse](t, listResp)
	require.Equal(t, 1, listData.Total)
	require.Equal(t, client.ClientId, listData.Items[0].ClientId)
}

func TestClientAPIRejectsInvalidExtensions(t *testing.T) {
	router, _ := setupClientControllerTest(t)

	recorder := performClientRequest(t, router, http.MethodPost, "/api/agent-platform/clients", dtoagentplatform.CreateClientRequest{
		Slug:            "bad-client",
		DisplayName:     "Bad Client",
		ClientType:      "desktop",
		ContractVersion: "2026-06",
		Capabilities:    json.RawMessage(`{"discovery":true}`),
		Extensions:      json.RawMessage(`{"plain":"value"}`),
	})
	response := decodeClientAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "invalid request params", response.Message)
}

func TestClientAPIRejectsInvalidRegistrationSchema(t *testing.T) {
	router, _ := setupClientControllerTest(t)

	cases := []struct {
		name string
		body any
	}{
		{
			name: "missing required slug",
			body: map[string]any{
				"display_name": "Missing Slug",
				"client_type":  "desktop",
			},
		},
		{
			name: "unknown status",
			body: dtoagentplatform.CreateClientRequest{
				Slug:        "bad-status",
				DisplayName: "Bad Status",
				ClientType:  "desktop",
				Status:      "pending",
			},
		},
		{
			name: "grant types must be array",
			body: dtoagentplatform.CreateClientRequest{
				Slug:              "bad-grants",
				DisplayName:       "Bad Grants",
				ClientType:        "desktop",
				AllowedGrantTypes: json.RawMessage(`{"grant":"authorization_code"}`),
			},
		},
		{
			name: "redirect uris cannot contain blank values",
			body: dtoagentplatform.CreateClientRequest{
				Slug:         "bad-redirects",
				DisplayName:  "Bad Redirects",
				ClientType:   "desktop",
				RedirectURIs: json.RawMessage(`[" "]`),
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := performClientRequest(t, router, http.MethodPost, "/api/agent-platform/clients", testCase.body)
			response := decodeClientAPIResponse(t, recorder)
			require.False(t, response.Success)
			require.Equal(t, "invalid request params", response.Message)
		})
	}
}

func TestClientAPIReturnsClearNotFoundForUnknownClient(t *testing.T) {
	router, _ := setupClientControllerTest(t)

	get := performClientRequest(t, router, http.MethodGet, "/api/agent-platform/clients/cli_missing", nil)
	getResp := decodeClientAPIResponse(t, get)
	require.False(t, getResp.Success)
	require.Equal(t, "client not found", getResp.Message)

	update := performClientRequest(t, router, http.MethodPut, "/api/agent-platform/clients/cli_missing", dtoagentplatform.UpdateClientRequest{
		Status: "disabled",
	})
	updateResp := decodeClientAPIResponse(t, update)
	require.False(t, updateResp.Success)
	require.Equal(t, "client not found", updateResp.Message)
}
