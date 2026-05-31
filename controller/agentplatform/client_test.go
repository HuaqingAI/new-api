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
		Slug:            "cherry-studio",
		DisplayName:     "Cherry Studio",
		ClientType:      "desktop",
		ContractVersion: "2026-06",
		Capabilities:    json.RawMessage(`{"discovery":true}`),
		AllowedScopes:   json.RawMessage(`["skills.read"]`),
	})
	createResp := decodeClientAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)
	client := decodeClientData[dtoagentplatform.ClientItem](t, createResp)
	require.Equal(t, "active", client.Status)

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
