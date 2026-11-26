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

type resourceAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupAgentPlatformControllerTest(t *testing.T) (*gin.Engine, *gorm.DB) {
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
	router.GET("/api/agent-platform/resources", ListResources)
	router.POST("/api/agent-platform/resources", CreateResource)
	router.GET("/api/agent-platform/resources/:id", GetResource)

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return router, db
}

func performAgentPlatformRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
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

func decodeResourceAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) resourceAPIResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var response resourceAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func decodeResourceData[T any](t *testing.T, response resourceAPIResponse) T {
	t.Helper()

	var data T
	require.NoError(t, common.Unmarshal(response.Data, &data))
	return data
}

func TestResourceAPIWorkflow(t *testing.T) {
	router, _ := setupAgentPlatformControllerTest(t)

	createRecorder := performAgentPlatformRequest(t, router, http.MethodPost, "/api/agent-platform/resources", dtoagentplatform.CreateResourceRequest{
		ResourceType: apmodel.ResourceTypeSkill,
		DisplayName:  "API Skill",
		OwnerUserId:  100,
	})
	createResponse := decodeResourceAPIResponse(t, createRecorder)
	require.True(t, createResponse.Success, createResponse.Message)
	created := decodeResourceData[dtoagentplatform.ResourceItem](t, createResponse)
	require.Equal(t, apmodel.ResourceTypeSkill, created.ResourceType)
	require.Equal(t, apmodel.ResourceStatusDraft, created.Status)
	require.NotEmpty(t, created.ResourceId)

	getRecorder := performAgentPlatformRequest(t, router, http.MethodGet, "/api/agent-platform/resources/"+created.ResourceId, nil)
	getResponse := decodeResourceAPIResponse(t, getRecorder)
	require.True(t, getResponse.Success, getResponse.Message)
	fetched := decodeResourceData[dtoagentplatform.ResourceItem](t, getResponse)
	require.Equal(t, created.ResourceId, fetched.ResourceId)
	require.Equal(t, created.ResourceType, fetched.ResourceType)

	listRecorder := performAgentPlatformRequest(t, router, http.MethodGet, "/api/agent-platform/resources?page=1&page_size=20", nil)
	listResponse := decodeResourceAPIResponse(t, listRecorder)
	require.True(t, listResponse.Success, listResponse.Message)
	listData := decodeResourceData[dtoagentplatform.ResourceListResponse](t, listResponse)
	require.Equal(t, 1, listData.Total)
	require.Len(t, listData.Items, 1)
	require.Equal(t, created.ResourceId, listData.Items[0].ResourceId)
}

func TestResourceAPIRejectsInvalidType(t *testing.T) {
	router, _ := setupAgentPlatformControllerTest(t)

	recorder := performAgentPlatformRequest(t, router, http.MethodPost, "/api/agent-platform/resources", dtoagentplatform.CreateResourceRequest{
		ResourceType: "invalid",
		DisplayName:  "Bad",
		OwnerUserId:  100,
	})
	response := decodeResourceAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "invalid request params", response.Message)
}

func TestResourceAPIRejectsInvalidListFilter(t *testing.T) {
	router, _ := setupAgentPlatformControllerTest(t)

	recorder := performAgentPlatformRequest(t, router, http.MethodGet, "/api/agent-platform/resources?resource_type=workflow", nil)
	response := decodeResourceAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "invalid request params", response.Message)
}
