package agentplatform

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type mcpAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupMcpControllerTest(t *testing.T) (*gin.Engine, *gorm.DB) {
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
	router.POST("/api/agent-platform/mcps", CreateMcp)
	router.GET("/api/agent-platform/mcps", ListMcps)
	router.DELETE("/api/agent-platform/mcps/:id", DeleteMcp)
	router.POST("/api/agent-platform/agents", CreateAgent)
	return router, db
}

func performMcpRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
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

func decodeMcpAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) mcpAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response mcpAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestDeleteMcpRejectsLatestAgentVersionDependency(t *testing.T) {
	router, db := setupMcpControllerTest(t)

	create := performMcpRequest(t, router, http.MethodPost, "/api/agent-platform/mcps", dtoagentplatform.CreateMcpRequest{
		DisplayName: "MCP A",
		Config:      json.RawMessage(`{"mcpServers":{"local":{"command":"node"}}}`),
	})
	createResp := decodeMcpAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)

	var mcp dtoagentplatform.McpItem
	require.NoError(t, common.Unmarshal(createResp.Data, &mcp))
	agent := performMcpRequest(t, router, http.MethodPost, "/api/agent-platform/agents", dtoagentplatform.CreateAgentRequest{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "Agent A",
		McpIds:      []string{mcp.ResourceId},
	})
	agentResp := decodeMcpAPIResponse(t, agent)
	require.True(t, agentResp.Success, agentResp.Message)
	var agentItem dtoagentplatform.AgentDetailItem
	require.NoError(t, common.Unmarshal(agentResp.Data, &agentItem))
	publishedAt := time.Now().UTC()
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      agentItem.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "agent-platform/v1",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       999,
		PublishedAt:     &publishedAt,
	}).Error)
	require.NoError(t, db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", agentItem.ResourceId).
		Update("latest_version", "1.0.0").Error)
	require.NoError(t, db.Create(&apmodel.AgentDependency{
		AgentResourceId:  agentItem.ResourceId,
		ResourceVersion:  "1.0.0",
		TargetType:       apmodel.AgentDependencyTypeMCP,
		TargetResourceId: mcp.ResourceId,
	}).Error)

	deleted := performMcpRequest(t, router, http.MethodDelete, "/api/agent-platform/mcps/"+mcp.ResourceId, nil)
	deleteResp := decodeMcpAPIResponse(t, deleted)

	require.False(t, deleteResp.Success)
	require.Equal(t, "该资源已被 Agent 关联，请先在 Agent 中解除关联后再删除", deleteResp.Message)
}

func TestDeleteMcpPhysicallyRemovesResourceFromList(t *testing.T) {
	router, db := setupMcpControllerTest(t)

	create := performMcpRequest(t, router, http.MethodPost, "/api/agent-platform/mcps", dtoagentplatform.CreateMcpRequest{
		DisplayName: "MCP A",
		Config:      json.RawMessage(`{"mcpServers":{"local":{"command":"node"}}}`),
	})
	createResp := decodeMcpAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)

	var mcp dtoagentplatform.McpItem
	require.NoError(t, common.Unmarshal(createResp.Data, &mcp))
	deleted := performMcpRequest(t, router, http.MethodDelete, "/api/agent-platform/mcps/"+mcp.ResourceId, nil)
	deleteResp := decodeMcpAPIResponse(t, deleted)
	require.True(t, deleteResp.Success, deleteResp.Message)

	list := performMcpRequest(t, router, http.MethodGet, "/api/agent-platform/mcps", nil)
	listResp := decodeMcpAPIResponse(t, list)
	require.True(t, listResp.Success, listResp.Message)
	var listData dtoagentplatform.McpListResponse
	require.NoError(t, common.Unmarshal(listResp.Data, &listData))
	require.Equal(t, 0, listData.Total)

	var count int64
	require.NoError(t, db.Model(&apmodel.Resource{}).Where("resource_id = ?", mcp.ResourceId).Count(&count).Error)
	require.Equal(t, int64(0), count)
}
