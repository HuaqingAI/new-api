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

type resourceVersionAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupAgentPlatformVersionControllerTest(t *testing.T) (*gin.Engine, *gorm.DB, apmodel.Resource) {
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

	resource := apmodel.Resource{ResourceType: apmodel.ResourceTypeSkill, DisplayName: "Skill", OwnerUserId: 100}
	require.NoError(t, db.Create(&resource).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 999)
		c.Set("role", common.RoleAdminUser)
		c.Next()
	})
	router.POST("/api/agent-platform/resources/:id/versions", CreateResourceVersion)
	router.GET("/api/agent-platform/resources/:id/versions/:version", GetResourceVersion)

	return router, db, resource
}

func performResourceVersionRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	var err error
	if body != nil {
		payload, err = common.Marshal(body)
		require.NoError(t, err)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(payload))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeResourceVersionAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) resourceVersionAPIResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var response resourceVersionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func decodeResourceVersionData[T any](t *testing.T, response resourceVersionAPIResponse) T {
	t.Helper()

	var data T
	require.NoError(t, common.Unmarshal(response.Data, &data))
	return data
}

func TestResourceVersionAPIWorkflow(t *testing.T) {
	router, _, resource := setupAgentPlatformVersionControllerTest(t)

	recorder := performResourceVersionRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/versions", dtoagentplatform.CreateResourceVersionRequest{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Summary:         "first release",
	})
	response := decodeResourceVersionAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	version := decodeResourceVersionData[dtoagentplatform.ResourceVersionItem](t, response)
	require.Equal(t, resource.ResourceId, version.ResourceId)
	require.Equal(t, "1.0.0", version.Version)
	require.Equal(t, "first release", version.Summary)

	getRecorder := performResourceVersionRequest(t, router, http.MethodGet, "/api/agent-platform/resources/"+resource.ResourceId+"/versions/1.0.0", nil)
	getResponse := decodeResourceVersionAPIResponse(t, getRecorder)
	require.True(t, getResponse.Success, getResponse.Message)
	fetched := decodeResourceVersionData[dtoagentplatform.ResourceVersionItem](t, getResponse)
	require.Equal(t, resource.ResourceId, fetched.ResourceId)
	require.Equal(t, "1.0.0", fetched.Version)
}

func TestResourceVersionAPIRejectsInvalidMetadata(t *testing.T) {
	router, _, resource := setupAgentPlatformVersionControllerTest(t)

	recorder := performResourceVersionRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/versions", dtoagentplatform.CreateResourceVersionRequest{
		ContractVersion: "2026-06",
	})
	response := decodeResourceVersionAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "invalid request params", response.Message)
}
