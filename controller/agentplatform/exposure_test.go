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

type exposureAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupExposureControllerTest(t *testing.T) (*gin.Engine, *gorm.DB, apmodel.Resource) {
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

	resource := apmodel.Resource{ResourceType: apmodel.ResourceTypeSkill, DisplayName: "Skill", OwnerUserId: 1}
	require.NoError(t, db.Create(&resource).Error)
	version := apmodel.ResourceVersion{
		ResourceId:      resource.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       1,
	}
	require.NoError(t, db.Create(&version).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 999)
		c.Set("role", common.RoleAdminUser)
		c.Next()
	})
	router.POST("/api/agent-platform/resources/:id/exposures", CreateExposure)
	router.GET("/api/agent-platform/resources/:id/exposures", ListExposures)
	router.GET("/api/agent-platform/resources/:id/exposures/:target", GetExposure)
	router.PUT("/api/agent-platform/resources/:id/exposures/:target", UpdateExposure)
	router.POST("/api/agent-platform/resources/:id/exposures/:target/revoke", RevokeExposure)
	return router, db, resource
}

func performExposureRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
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

func decodeExposureAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) exposureAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response exposureAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func decodeExposureData[T any](t *testing.T, response exposureAPIResponse) T {
	t.Helper()
	var data T
	require.NoError(t, common.Unmarshal(response.Data, &data))
	return data
}

func TestExposureAPIWorkflow(t *testing.T) {
	router, _, resource := setupExposureControllerTest(t)

	create := performExposureRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/exposures", dtoagentplatform.CreateExposureRequest{
		ResourceVersion: "1.0.0",
		ClientKey:       "client-a",
		ClientScope:     "placeholder",
		VisibilityState: apmodel.ExposureVisibilityVisible,
		CallableState:   apmodel.ExposureCallableEnabled,
		Extensions:      json.RawMessage(`{"placeholder":true}`),
	})
	createResp := decodeExposureAPIResponse(t, create)
	require.True(t, createResp.Success, createResp.Message)
	created := decodeExposureData[dtoagentplatform.ExposureItem](t, createResp)
	require.Equal(t, resource.ResourceId, created.ResourceId)
	require.Equal(t, apmodel.ExposureVisibilityVisible, created.VisibilityState)

	list := performExposureRequest(t, router, http.MethodGet, "/api/agent-platform/resources/"+resource.ResourceId+"/exposures", nil)
	listResp := decodeExposureAPIResponse(t, list)
	require.True(t, listResp.Success, listResp.Message)
	listData := decodeExposureData[dtoagentplatform.ExposureListResponse](t, listResp)
	require.Equal(t, 1, listData.Total)

	update := performExposureRequest(t, router, http.MethodPut, "/api/agent-platform/resources/"+resource.ResourceId+"/exposures/client-a", dtoagentplatform.UpdateExposureRequest{
		VisibilityState: apmodel.ExposureVisibilityHidden,
		CallableState:   apmodel.ExposureCallableDisabled,
	})
	updateResp := decodeExposureAPIResponse(t, update)
	require.True(t, updateResp.Success, updateResp.Message)
	updated := decodeExposureData[dtoagentplatform.ExposureItem](t, updateResp)
	require.Equal(t, apmodel.ExposureVisibilityHidden, updated.VisibilityState)

	recreate := performExposureRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/exposures", dtoagentplatform.CreateExposureRequest{
		ResourceVersion: "1.0.0",
		ClientKey:       "client-b",
		ClientScope:     "placeholder",
		VisibilityState: apmodel.ExposureVisibilityVisible,
		CallableState:   apmodel.ExposureCallableEnabled,
	})
	recreateResp := decodeExposureAPIResponse(t, recreate)
	require.True(t, recreateResp.Success, recreateResp.Message)

	revoke := performExposureRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/exposures/client-b/revoke", nil)
	revokeResp := decodeExposureAPIResponse(t, revoke)
	require.True(t, revokeResp.Success, revokeResp.Message)
	revoked := decodeExposureData[dtoagentplatform.ExposureItem](t, revokeResp)
	require.Equal(t, apmodel.ExposureVisibilityRevoked, revoked.VisibilityState)
	require.Equal(t, apmodel.ExposureCallableRevoked, revoked.CallableState)
}

func TestExposureAPIRejectsInvalidState(t *testing.T) {
	router, _, resource := setupExposureControllerTest(t)

	recorder := performExposureRequest(t, router, http.MethodPost, "/api/agent-platform/resources/"+resource.ResourceId+"/exposures", dtoagentplatform.CreateExposureRequest{
		ResourceVersion: "1.0.0",
		ClientKey:       "client-a",
		VisibilityState: "ghost",
		CallableState:   apmodel.ExposureCallableEnabled,
	})
	response := decodeExposureAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "invalid request params", response.Message)
}
