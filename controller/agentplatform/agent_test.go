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

type agentAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupAgentControllerTest(t *testing.T) (*gin.Engine, *gorm.DB) {
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
	router.GET("/api/agent-platform/agents", ListAgents)
	router.POST("/api/agent-platform/agents", CreateAgent)
	router.GET("/api/agent-platform/agents/:id", GetAgent)
	router.PUT("/api/agent-platform/agents/:id", UpdateAgent)
	return router, db
}

func performAgentRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
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

func decodeAgentAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) agentAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response agentAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestAgentAPIWorkflow(t *testing.T) {
	router, _ := setupAgentControllerTest(t)

	create := performAgentRequest(t, router, http.MethodPost, "/api/agent-platform/agents", dtoagentplatform.CreateAgentRequest{
		CliType:            apmodel.AgentCliTypeOpenCode,
		DisplayName:        "Agent A",
		Categories:         []string{apmodel.AgentCategoryGeneral},
		RecommendedPrompts: []string{"问题一", "问题二"},
		OwnerUserId:        100,
	})
	createResp := decodeAgentAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)

	var created dtoagentplatform.AgentDetailItem
	require.NoError(t, common.Unmarshal(createResp.Data, &created))
	require.Equal(t, apmodel.ResourceTypeAgent, created.ResourceType)
	require.Equal(t, []string{apmodel.AgentCategoryGeneral}, created.Categories)
	require.Equal(t, []string{"问题一", "问题二"}, created.RecommendedPrompts)

	list := performAgentRequest(t, router, http.MethodGet, "/api/agent-platform/agents", nil)
	listResp := decodeAgentAPIResponse(t, list)
	require.True(t, listResp.Success, listResp.Message)

	var listData dtoagentplatform.AgentListResponse
	require.NoError(t, common.Unmarshal(listResp.Data, &listData))
	require.Equal(t, 1, listData.Total)
	require.Equal(t, []string{apmodel.AgentCategoryGeneral}, listData.Items[0].Categories)
	require.Equal(t, []string{"问题一", "问题二"}, listData.Items[0].RecommendedPrompts)

	update := performAgentRequest(t, router, http.MethodPut, "/api/agent-platform/agents/"+created.ResourceId, dtoagentplatform.UpdateAgentRequest{
		CliType:            apmodel.AgentCliTypeOpenCode,
		DisplayName:        "Agent A V2",
		Categories:         []string{apmodel.AgentCategoryGeneral},
		RecommendedPrompts: []string{"更新问题"},
	})
	updateResp := decodeAgentAPIResponse(t, update)
	require.True(t, updateResp.Success, updateResp.Message)
	var updated dtoagentplatform.AgentDetailItem
	require.NoError(t, common.Unmarshal(updateResp.Data, &updated))
	require.Equal(t, []string{"更新问题"}, updated.RecommendedPrompts)
}
