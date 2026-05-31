package enterprise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type enterpriseAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupEnterpriseControllerTest(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.SubscriptionPlan{}, &model.UserSubscription{}, &model.Option{}, &model.Log{}))
	require.NoError(t, entmodel.AutoMigrate(db))

	require.NoError(t, db.Create(&model.User{Id: 100, Username: "alice", Password: "password123", DisplayName: "Alice", Group: "vip", AffCode: "alice-api"}).Error)
	require.NoError(t, db.Create(&model.User{Id: 101, Username: "bob", Password: "password123", DisplayName: "Bob", Group: "default", AffCode: "bob-api"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 1, TenantId: 0, Name: "Engineering", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 2, TenantId: 0, Name: "Security", Status: constant.EnterpriseDepartmentStatusActive}).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 999)
		c.Set("role", common.RoleAdminUser)
		c.Next()
	})
	router.GET("/api/enterprise/users/:id/departments", ListUserDepartments)
	router.PUT("/api/enterprise/users/:id/departments", ReplaceUserDepartments)
	router.GET("/api/enterprise/departments/:id/members", ListDepartmentMembers)
	router.POST("/api/enterprise/departments/:id/members", AddDepartmentMember)
	router.DELETE("/api/enterprise/departments/:id/members/:user_id", DeactivateDepartmentMember)
	router.POST("/api/enterprise/departments/:id/members/:user_id/restore", RestoreDepartmentMember)
	router.GET("/api/enterprise/usage/department-summary", GetDepartmentUsageSummary)
	router.GET("/api/enterprise/usage/department-detail", GetDepartmentUsageDetail)
	router.GET("/api/enterprise/usage/export", ExportDepartmentUsageCSV)
	router.GET("/api/enterprise/usage/reports", GetDepartmentUsageReportConfig)
	router.PUT("/api/enterprise/usage/reports", SaveDepartmentUsageReportConfig)
	router.GET("/api/enterprise/alerts/events", ListAlertEvents)
	router.GET("/api/enterprise/alerts/department-summary", GetDepartmentRiskSummary)
	router.GET("/api/enterprise/alerts/deliveries", ListAlertDeliveries)
	router.POST("/api/enterprise/alerts/deliveries/:id/resend", ResendAlertDelivery)
	router.GET("/api/enterprise/alerts/rules", ListAlertRules)
	router.GET("/api/enterprise/alerts/rules/:id", GetAlertRule)
	router.PUT("/api/enterprise/alerts/rules", SaveAlertRule)
	router.DELETE("/api/enterprise/alerts/rules/:id", DeleteAlertRule)

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return router, db
}

func performEnterpriseRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		payload, err := common.Marshal(body)
		require.NoError(t, err)
		requestBody = bytes.NewReader(payload)
	}
	request := httptest.NewRequest(method, target, requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func performEnterpriseRequestWithHeaders(t *testing.T, router *gin.Engine, method string, target string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		payload, err := common.Marshal(body)
		require.NoError(t, err)
		requestBody = bytes.NewReader(payload)
	}
	request := httptest.NewRequest(method, target, requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeEnterpriseAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) enterpriseAPIResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var response enterpriseAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func decodeEnterpriseData[T any](t *testing.T, response enterpriseAPIResponse) T {
	t.Helper()

	var data T
	require.NoError(t, common.Unmarshal(response.Data, &data))
	return data
}

func TestEnterpriseMembershipAPIUserDepartmentWorkflow(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	replaceRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/users/100/departments", dtoenterprise.ReplaceUserDepartmentsRequest{
		DepartmentIds:   []int{1, 2},
		ExternalSource:  "manual",
		DeactivateStale: boolPtr(true),
	})
	replaceResponse := decodeEnterpriseAPIResponse(t, replaceRecorder)
	require.True(t, replaceResponse.Success, replaceResponse.Message)
	replaceData := decodeEnterpriseData[dtoenterprise.UserDepartmentsResponse](t, replaceResponse)
	require.False(t, replaceData.IsUnassigned)
	require.Len(t, replaceData.Items, 2)
	require.Equal(t, []int{1, 2}, []int{replaceData.Items[0].DepartmentId, replaceData.Items[1].DepartmentId})

	listRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/users/100/departments", nil)
	listResponse := decodeEnterpriseAPIResponse(t, listRecorder)
	require.True(t, listResponse.Success, listResponse.Message)
	listData := decodeEnterpriseData[dtoenterprise.UserDepartmentsResponse](t, listResponse)
	require.False(t, listData.IsUnassigned)
	require.Equal(t, 2, listData.Total)
	require.Equal(t, "Engineering", listData.Items[0].DepartmentName)
	require.Equal(t, "Security", listData.Items[1].DepartmentName)

	unassignedRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/users/101/departments", nil)
	unassignedResponse := decodeEnterpriseAPIResponse(t, unassignedRecorder)
	require.True(t, unassignedResponse.Success, unassignedResponse.Message)
	require.Contains(t, unassignedRecorder.Body.String(), `"items":[]`)
	unassignedData := decodeEnterpriseData[dtoenterprise.UserDepartmentsResponse](t, unassignedResponse)
	require.True(t, unassignedData.IsUnassigned)
	require.NotNil(t, unassignedData.Items)
	require.Empty(t, unassignedData.Items)

	var user model.User
	require.NoError(t, db.First(&user, 100).Error)
	require.Equal(t, "vip", user.Group)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, 999, actions[0].ActorId)
	require.Equal(t, "enterprise.organization.membership.replace", actions[0].ActionType)
	require.Equal(t, "100", actions[0].ObjectId)
}

func TestEnterpriseMembershipAPIDepartmentMemberLifecycle(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)

	addRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/members", dtoenterprise.AddDepartmentMemberRequest{
		UserId:         100,
		ExternalSource: "manual",
	})
	addResponse := decodeEnterpriseAPIResponse(t, addRecorder)
	require.True(t, addResponse.Success, addResponse.Message)
	added := decodeEnterpriseData[dtoenterprise.DepartmentMemberItem](t, addResponse)
	require.Equal(t, 100, added.UserId)
	require.Equal(t, "alice", added.Username)
	require.Equal(t, constant.EnterpriseMembershipStatusActive, added.Status)

	duplicateRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/members", dtoenterprise.AddDepartmentMemberRequest{
		UserId:         100,
		ExternalSource: "manual",
	})
	duplicateResponse := decodeEnterpriseAPIResponse(t, duplicateRecorder)
	require.False(t, duplicateResponse.Success)
	require.Equal(t, "enterprise.organization.membership_already_exists", duplicateResponse.Message)

	membersRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/1/members", nil)
	membersResponse := decodeEnterpriseAPIResponse(t, membersRecorder)
	require.True(t, membersResponse.Success, membersResponse.Message)
	members := decodeEnterpriseData[dtoenterprise.DepartmentMembersResponse](t, membersResponse)
	require.Equal(t, 1, members.Total)
	require.Len(t, members.Items, 1)

	deactivateRecorder := performEnterpriseRequest(t, router, http.MethodDelete, "/api/enterprise/departments/1/members/100", nil)
	deactivateResponse := decodeEnterpriseAPIResponse(t, deactivateRecorder)
	require.True(t, deactivateResponse.Success, deactivateResponse.Message)

	userAfterDeactivateRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/users/100/departments", nil)
	userAfterDeactivateResponse := decodeEnterpriseAPIResponse(t, userAfterDeactivateRecorder)
	require.True(t, userAfterDeactivateResponse.Success, userAfterDeactivateResponse.Message)
	userAfterDeactivate := decodeEnterpriseData[dtoenterprise.UserDepartmentsResponse](t, userAfterDeactivateResponse)
	require.True(t, userAfterDeactivate.IsUnassigned)
	require.Empty(t, userAfterDeactivate.Items)

	inactiveRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/1/members?status=2", nil)
	inactiveResponse := decodeEnterpriseAPIResponse(t, inactiveRecorder)
	require.True(t, inactiveResponse.Success, inactiveResponse.Message)
	inactive := decodeEnterpriseData[dtoenterprise.DepartmentMembersResponse](t, inactiveResponse)
	require.Len(t, inactive.Items, 1)
	require.Equal(t, constant.EnterpriseMembershipStatusInactive, inactive.Items[0].Status)

	restoreRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/members/100/restore", nil)
	restoreResponse := decodeEnterpriseAPIResponse(t, restoreRecorder)
	require.True(t, restoreResponse.Success, restoreResponse.Message)
	restored := decodeEnterpriseData[dtoenterprise.DepartmentMemberItem](t, restoreResponse)
	require.Equal(t, constant.EnterpriseMembershipStatusActive, restored.Status)
	require.Equal(t, int64(0), restored.LeftAt)

	var user model.User
	require.NoError(t, db.First(&user, 100).Error)
	require.Equal(t, "vip", user.Group)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 3)
	require.Equal(t, "enterprise.organization.membership.add", actions[0].ActionType)
	require.Equal(t, "enterprise.organization.membership.disable", actions[1].ActionType)
	require.Equal(t, "enterprise.organization.membership.restore", actions[2].ActionType)
}

func TestEnterpriseMembershipAPIRejectsInvalidRequests(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)

	duplicateDepartmentsRecorder := performEnterpriseRequest(t, router, http.MethodPut, "/api/enterprise/users/100/departments", dtoenterprise.ReplaceUserDepartmentsRequest{
		DepartmentIds: []int{1, 1},
	})
	duplicateDepartmentsResponse := decodeEnterpriseAPIResponse(t, duplicateDepartmentsRecorder)
	require.False(t, duplicateDepartmentsResponse.Success)
	require.Equal(t, "enterprise.organization.duplicate_department", duplicateDepartmentsResponse.Message)

	missingDepartmentRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/999/members", dtoenterprise.AddDepartmentMemberRequest{
		UserId: 100,
	})
	missingDepartmentResponse := decodeEnterpriseAPIResponse(t, missingDepartmentRecorder)
	require.False(t, missingDepartmentResponse.Success)
	require.Equal(t, "enterprise.organization.department_not_found", missingDepartmentResponse.Message)

	invalidPathRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/users/not-an-id/departments", nil)
	invalidPathResponse := decodeEnterpriseAPIResponse(t, invalidPathRecorder)
	require.False(t, invalidPathResponse.Success)
	require.Equal(t, "common.invalid_params", invalidPathResponse.Message)
}

func boolPtr(v bool) *bool {
	return &v
}
