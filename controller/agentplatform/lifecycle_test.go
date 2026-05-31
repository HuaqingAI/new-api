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
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type lifecycleAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupLifecycleControllerTest(t *testing.T) (*gin.Engine, *gorm.DB, apmodel.Resource) {
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
	versionService := apservice.NewResourceVersionService(db)
	_, err = versionService.Create(resource.ResourceId, apservice.CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Skill: &apservice.SkillDetailInput{
			InvokeSchema:   json.RawMessage(`{"type":"object"}`),
			OutputSchema:   json.RawMessage(`{"type":"object"}`),
			InvokeMode:     "sync",
			TimeoutSeconds: lifecycleIntPtr(30),
			BindingConfig:  json.RawMessage(`{"provider":"demo"}`),
		},
		CreatedBy: 100,
	})
	require.NoError(t, err)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 999)
		c.Set("role", common.RoleAdminUser)
		c.Next()
	})
	router.POST("/api/agent-platform/resources/:id/publish", PublishResource)
	router.POST("/api/agent-platform/resources/:id/disable", DisableResource)
	router.POST("/api/agent-platform/resources/:id/revoke", RevokeResource)
	router.POST("/api/agent-platform/resources/:id/offline", OfflineResource)
	router.POST("/api/agent-platform/resources/:id/versions/:version/rollback", RollbackResourceVersion)
	return router, db, resource
}

func performLifecycleRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
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

func decodeLifecycleAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) lifecycleAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response lifecycleAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func decodeLifecycleData[T any](t *testing.T, response lifecycleAPIResponse) T {
	t.Helper()
	var data T
	require.NoError(t, common.Unmarshal(response.Data, &data))
	return data
}

func TestLifecycleAPIWorkflow(t *testing.T) {
	router, _, resource := setupLifecycleControllerTest(t)

	publish := performLifecycleRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/publish", dtoagentplatform.LifecycleActionRequest{
		Version:   "1.0.0",
		RequestId: "req-publish",
	})
	publishResp := decodeLifecycleAPIResponse(t, publish)
	require.True(t, publishResp.Success, publishResp.Message)
	published := decodeLifecycleData[dtoagentplatform.LifecycleActionResponse](t, publishResp)
	require.Equal(t, apmodel.ResourceStatusPublished, published.CurrentStatus)

	disable := performLifecycleRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/disable", dtoagentplatform.LifecycleActionRequest{
		RequestId: "req-disable",
	})
	disableResp := decodeLifecycleAPIResponse(t, disable)
	require.True(t, disableResp.Success, disableResp.Message)
	disabled := decodeLifecycleData[dtoagentplatform.LifecycleActionResponse](t, disableResp)
	require.Equal(t, apmodel.ResourceStatusDisabled, disabled.CurrentStatus)

	rollback := performLifecycleRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/versions/1.0.0/rollback", dtoagentplatform.LifecycleActionRequest{
		RequestId: "req-rollback",
	})
	rollbackResp := decodeLifecycleAPIResponse(t, rollback)
	require.True(t, rollbackResp.Success, rollbackResp.Message)
	rolledBack := decodeLifecycleData[dtoagentplatform.LifecycleActionResponse](t, rollbackResp)
	require.Equal(t, apmodel.ResourceStatusPublished, rolledBack.CurrentStatus)
	require.Equal(t, "1.0.0", rolledBack.CurrentVersion)
}

func TestLifecycleAPIRejectsInvalidRollbackRequest(t *testing.T) {
	router, _, resource := setupLifecycleControllerTest(t)

	recorder := performLifecycleRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/versions/9.9.9/rollback", dtoagentplatform.LifecycleActionRequest{
		RequestId: "req-invalid-rollback",
	})
	response := decodeLifecycleAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "invalid request params", response.Message)
}

func lifecycleIntPtr(v int) *int {
	return &v
}
