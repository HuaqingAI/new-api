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

type knowledgeAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupKnowledgeControllerTest(t *testing.T) (*gin.Engine, *gorm.DB) {
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
	router.GET("/api/agent-platform/knowledge-bases", ListKnowledge)
	router.POST("/api/agent-platform/knowledge-bases", CreateKnowledge)
	router.GET("/api/agent-platform/knowledge-bases/:id", GetKnowledge)
	router.PUT("/api/agent-platform/knowledge-bases/:id", UpdateKnowledge)
	return router, db
}

func performKnowledgeRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
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

func decodeKnowledgeAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) knowledgeAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response knowledgeAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestKnowledgeAPIWorkflow(t *testing.T) {
	router, _ := setupKnowledgeControllerTest(t)

	create := performKnowledgeRequest(t, router, http.MethodPost, "/api/agent-platform/knowledge-bases", dtoagentplatform.CreateKnowledgeRequest{
		DisplayName:         "Knowledge A",
		ExternalKnowledgeId: "kb-001",
		OwnerUserId:         100,
	})
	createResp := decodeKnowledgeAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)

	var created dtoagentplatform.KnowledgeDetailItem
	require.NoError(t, common.Unmarshal(createResp.Data, &created))
	require.Equal(t, apmodel.ResourceTypeKnowledge, created.ResourceType)
	require.Equal(t, "kb-001", created.ExternalKnowledgeId)

	list := performKnowledgeRequest(t, router, http.MethodGet, "/api/agent-platform/knowledge-bases", nil)
	listResp := decodeKnowledgeAPIResponse(t, list)
	require.True(t, listResp.Success, listResp.Message)

	var listData dtoagentplatform.KnowledgeListResponse
	require.NoError(t, common.Unmarshal(listResp.Data, &listData))
	require.Equal(t, 1, listData.Total)
	require.Len(t, listData.Items, 1)
	require.Equal(t, "kb-001", listData.Items[0].ExternalKnowledgeId)

	get := performKnowledgeRequest(t, router, http.MethodGet, "/api/agent-platform/knowledge-bases/"+created.ResourceId, nil)
	getResp := decodeKnowledgeAPIResponse(t, get)
	require.True(t, getResp.Success, getResp.Message)
	var detail dtoagentplatform.KnowledgeDetailItem
	require.NoError(t, common.Unmarshal(getResp.Data, &detail))
	require.Equal(t, "kb-001", detail.ExternalKnowledgeId)

	update := performKnowledgeRequest(t, router, http.MethodPut, "/api/agent-platform/knowledge-bases/"+created.ResourceId, dtoagentplatform.UpdateKnowledgeRequest{
		DisplayName:         "Knowledge A V2",
		ExternalKnowledgeId: "kb-002",
	})
	updateResp := decodeKnowledgeAPIResponse(t, update)
	require.True(t, updateResp.Success, updateResp.Message)
	var updated dtoagentplatform.KnowledgeDetailItem
	require.NoError(t, common.Unmarshal(updateResp.Data, &updated))
	require.Equal(t, "kb-002", updated.ExternalKnowledgeId)
}
