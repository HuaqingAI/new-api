package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/QuantumNous/new-api/router"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type agentPlatformClientsAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type agentPlatformClientsAPIFixture struct {
	engine *gin.Engine
	db     *gorm.DB
}

func TestAgentPlatformClientsAPIRequiresAdminSession(t *testing.T) {
	fixture := newAgentPlatformClientsAPIFixture(t)

	noSession := fixture.performRequest(t, http.MethodGet, "/api/agent-platform/clients", nil, nil)
	require.Equal(t, http.StatusUnauthorized, noSession.Code)
	require.Contains(t, noSession.Body.String(), `"code":"AUTH_UNAUTHORIZED"`)
	require.Contains(t, noSession.Body.String(), "auth.access_token_invalid")

	commonUser := fixture.performRequest(t, http.MethodGet, "/api/agent-platform/clients", fixture.login(t, common.RoleCommonUser), nil)
	require.Equal(t, http.StatusForbidden, commonUser.Code)
	require.Contains(t, commonUser.Body.String(), `"code":"AUTH_INSUFFICIENT_PRIVILEGE"`)
	require.Contains(t, commonUser.Body.String(), "auth.insufficient_privilege")
}

func TestAgentPlatformClientsAPICompletesMinimalOnboardingWorkflow(t *testing.T) {
	fixture := newAgentPlatformClientsAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser)

	create := fixture.performRequest(t, http.MethodPost, "/api/agent-platform/clients", adminCookies, dtoagentplatform.CreateClientRequest{
		Slug:                   "Cherry-Studio",
		DisplayName:            "Cherry Studio",
		ClientType:             "Desktop",
		ContractVersion:        "2026-06",
		Capabilities:           json.RawMessage(`{"discovery":true,"oauth":true}`),
		AllowedGrantTypes:      json.RawMessage(`["authorization_code","refresh_token","client_credentials"]`),
		RedirectURIs:           json.RawMessage(`["cherrystudio://oauth/callback"]`),
		AllowedScopes:          json.RawMessage(`["skills.read","resources.read"]`),
		Extensions:             json.RawMessage(`{"cherry_studio.fixture":"ap-6.2"}`),
		AllowClientCredentials: true,
	})
	createPayload := decodeAgentPlatformClientsAPIResponse(t, create)
	require.True(t, createPayload.Success, createPayload.Message)

	client := decodeAgentPlatformClientsData[dtoagentplatform.ClientItem](t, createPayload)
	require.NotEmpty(t, client.ClientId)
	require.Equal(t, "cherry-studio", client.Slug)
	require.Equal(t, "desktop", client.ClientType)
	require.Equal(t, "active", client.Status)
	require.Equal(t, "2026-06", client.ContractVersion)
	require.True(t, client.AllowClientCredentials)
	require.JSONEq(t, `["authorization_code","refresh_token","client_credentials"]`, string(client.AllowedGrantTypes))
	require.JSONEq(t, `["cherrystudio://oauth/callback"]`, string(client.RedirectURIs))
	require.JSONEq(t, `["skills.read","resources.read"]`, string(client.AllowedScopes))
	require.JSONEq(t, `{"discovery":true,"oauth":true}`, string(client.Capabilities))
	require.JSONEq(t, `{"cherry_studio.fixture":"ap-6.2"}`, string(client.Extensions))

	get := fixture.performRequest(t, http.MethodGet, "/api/agent-platform/clients/"+client.ClientId, adminCookies, nil)
	getPayload := decodeAgentPlatformClientsAPIResponse(t, get)
	require.True(t, getPayload.Success, getPayload.Message)
	fetched := decodeAgentPlatformClientsData[dtoagentplatform.ClientItem](t, getPayload)
	require.Equal(t, client.ClientId, fetched.ClientId)

	allowClientCredentials := false
	update := fixture.performRequest(t, http.MethodPut, "/api/agent-platform/clients/"+client.ClientId, adminCookies, dtoagentplatform.UpdateClientRequest{
		DisplayName:            "Cherry Studio Desktop",
		Status:                 "disabled",
		AllowedScopes:          json.RawMessage(`["resources.read"]`),
		AllowClientCredentials: &allowClientCredentials,
	})
	updatePayload := decodeAgentPlatformClientsAPIResponse(t, update)
	require.True(t, updatePayload.Success, updatePayload.Message)
	updated := decodeAgentPlatformClientsData[dtoagentplatform.ClientItem](t, updatePayload)
	require.Equal(t, "Cherry Studio Desktop", updated.DisplayName)
	require.Equal(t, "disabled", updated.Status)
	require.False(t, updated.AllowClientCredentials)
	require.JSONEq(t, `["resources.read"]`, string(updated.AllowedScopes))

	list := fixture.performRequest(t, http.MethodGet, "/api/agent-platform/clients?contract_version=2026-06", adminCookies, nil)
	listPayload := decodeAgentPlatformClientsAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	listData := decodeAgentPlatformClientsData[dtoagentplatform.ClientListResponse](t, listPayload)
	require.Equal(t, 1, listData.Total)
	require.Equal(t, client.ClientId, listData.Items[0].ClientId)
	require.Equal(t, "disabled", listData.Items[0].Status)
}

func TestAgentPlatformClientsAPIMapsIncompleteRegistrationToInvalidIntegration(t *testing.T) {
	fixture := newAgentPlatformClientsAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser)

	create := fixture.performRequest(t, http.MethodPost, "/api/agent-platform/clients", adminCookies, dtoagentplatform.CreateClientRequest{
		Slug:        "codex",
		DisplayName: "Codex",
		ClientType:  "cli",
		Extensions:  json.RawMessage(`{"codex.fixture":"ap-6.2"}`),
	})
	createPayload := decodeAgentPlatformClientsAPIResponse(t, create)
	require.True(t, createPayload.Success, createPayload.Message)
	client := decodeAgentPlatformClientsData[dtoagentplatform.ClientItem](t, createPayload)
	require.Equal(t, "invalid_integration", client.Status)

	list := fixture.performRequest(t, http.MethodGet, "/api/agent-platform/clients?status=invalid_integration", adminCookies, nil)
	listPayload := decodeAgentPlatformClientsAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	listData := decodeAgentPlatformClientsData[dtoagentplatform.ClientListResponse](t, listPayload)
	require.Equal(t, 1, listData.Total)
	require.Equal(t, client.ClientId, listData.Items[0].ClientId)
	require.Equal(t, "codex", listData.Items[0].Slug)
}

func TestAgentPlatformClientsAPIRejectsInvalidRegistrationSchema(t *testing.T) {
	fixture := newAgentPlatformClientsAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser)

	tests := []struct {
		name string
		body dtoagentplatform.CreateClientRequest
	}{
		{
			name: "non array grant types",
			body: dtoagentplatform.CreateClientRequest{
				Slug:              "bad-grants",
				DisplayName:       "Bad Grants",
				ClientType:        "desktop",
				AllowedGrantTypes: json.RawMessage(`{"grant":"authorization_code"}`),
			},
		},
		{
			name: "empty redirect item",
			body: dtoagentplatform.CreateClientRequest{
				Slug:         "bad-redirects",
				DisplayName:  "Bad Redirects",
				ClientType:   "desktop",
				RedirectURIs: json.RawMessage(`[" "]`),
			},
		},
		{
			name: "non namespaced extension",
			body: dtoagentplatform.CreateClientRequest{
				Slug:            "bad-extension",
				DisplayName:     "Bad Extension",
				ClientType:      "desktop",
				ContractVersion: "2026-06",
				Capabilities:    json.RawMessage(`{"discovery":true}`),
				Extensions:      json.RawMessage(`{"plain":"value"}`),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := fixture.performRequest(t, http.MethodPost, "/api/agent-platform/clients", adminCookies, tt.body)
			payload := decodeAgentPlatformClientsAPIResponse(t, recorder)
			require.False(t, payload.Success)
			require.Equal(t, "invalid request params", payload.Message)
		})
	}
}

func newAgentPlatformClientsAPIFixture(t *testing.T) agentPlatformClientsAPIFixture {
	t.Helper()

	gin.SetMode(gin.TestMode)
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldRedisEnabled := common.RedisEnabled

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, apmodel.Migrate(db))
	accessToken := "agent-platform-clients-api-test"
	require.NoError(t, db.Create(&model.User{
		Id:          1001,
		Username:    "agent-platform-admin",
		Password:    "password123",
		Group:       "default",
		Status:      common.UserStatusEnabled,
		AffCode:     "agent-platform-admin-api",
		AccessToken: &accessToken,
	}).Error)
	model.DB = db
	model.LOG_DB = db

	engine := gin.New()
	apiRouter := engine.Group("/api")
	router.RegisterAgentPlatformRouter(apiRouter)

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.RedisEnabled = oldRedisEnabled
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return agentPlatformClientsAPIFixture{engine: engine, db: db}
}

func (f agentPlatformClientsAPIFixture) login(t *testing.T, role int) []*http.Cookie {
	t.Helper()
	require.NoError(t, f.db.Model(&model.User{}).Where("id = ?", 1001).Update("role", role).Error)
	return []*http.Cookie{{Name: "test-access-token", Value: "agent-platform-clients-api-test"}}
}

func (f agentPlatformClientsAPIFixture) performRequest(t *testing.T, method string, path string, cookies []*http.Cookie, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := common.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(payload)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if len(cookies) > 0 {
		request.Header.Set("Authorization", "Bearer "+cookies[0].Value)
	}
	f.engine.ServeHTTP(recorder, request)
	return recorder
}

func decodeAgentPlatformClientsAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) agentPlatformClientsAPIResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload agentPlatformClientsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	return payload
}

func decodeAgentPlatformClientsData[T any](t *testing.T, response agentPlatformClientsAPIResponse) T {
	t.Helper()

	var data T
	require.NoError(t, common.Unmarshal(response.Data, &data))
	return data
}
