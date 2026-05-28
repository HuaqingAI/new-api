package api_test

import (
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
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/router"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type departmentTreeAPIResponse struct {
	Success bool                               `json:"success"`
	Message string                             `json:"message"`
	Data    []dtoenterprise.DepartmentTreeNode `json:"data"`
}

type departmentMembersAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestEnterpriseDepartmentTreeAPIRequiresBackendDepartmentPermission(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)

	noSession := fixture.performDepartmentTreeRequest(t, nil)
	require.Equal(t, http.StatusUnauthorized, noSession.Code)
	require.Contains(t, noSession.Body.String(), "auth.not_logged_in")

	commonUserCookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)
	commonUser := fixture.performDepartmentTreeRequest(t, commonUserCookies)
	require.Equal(t, http.StatusOK, commonUser.Code)
	require.Contains(t, commonUser.Body.String(), "error.enterprise.permission.dept_admin_required")
}

func TestEnterpriseDepartmentTreeAPIReturnsEmptyArray(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	recorder := fixture.performDepartmentTreeRequest(t, fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))

	payload := decodeDepartmentTreeAPIResponse(t, recorder)
	require.True(t, payload.Success)
	require.NotNil(t, payload.Data)
	require.Empty(t, payload.Data)
	require.Contains(t, recorder.Body.String(), `"data":[]`)
}

func TestEnterpriseDepartmentTreeAPIReturnsThreeLevelTreeWithStatuses(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	parentId := 1
	childId := 2
	root := enterpriseDepartment(1, nil, "Headquarters", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK)
	engineering := enterpriseDepartment(2, &parentId, "Engineering", constant.DepartmentStatusDisabled, constant.DepartmentSourceTypeDingTalk, constant.DepartmentSyncStatusWarning)
	platform := enterpriseDepartment(3, &childId, "Platform", constant.DepartmentStatusDeleted, constant.DepartmentSourceTypeDingTalk, constant.DepartmentSyncStatusFailed)
	platform.SyncError = "deleted upstream during sync"
	require.NoError(t, platform.SetNameHistory([]modelenterprise.DepartmentNameHistoryEntry{
		{Name: "Core Platform", ChangedAt: 1700000000},
	}))
	require.NoError(t, fixture.db.Create(&[]modelenterprise.Department{root, engineering, platform}).Error)

	recorder := fixture.performDepartmentTreeRequest(t, fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))

	payload := decodeDepartmentTreeAPIResponse(t, recorder)
	require.True(t, payload.Success)
	require.Len(t, payload.Data, 1)
	require.Equal(t, "Headquarters", payload.Data[0].Name)
	require.Equal(t, constant.DepartmentStatusEnabled, payload.Data[0].Status)
	require.Len(t, payload.Data[0].Children, 1)
	require.Equal(t, "Engineering", payload.Data[0].Children[0].Name)
	require.Equal(t, constant.DepartmentStatusDisabled, payload.Data[0].Children[0].Status)
	require.Equal(t, constant.DepartmentSyncStatusWarning, payload.Data[0].Children[0].SyncStatus)
	require.Len(t, payload.Data[0].Children[0].Children, 1)

	leaf := payload.Data[0].Children[0].Children[0]
	require.Equal(t, "Platform", leaf.Name)
	require.Equal(t, constant.DepartmentStatusDeleted, leaf.Status)
	require.Equal(t, constant.DepartmentSyncStatusFailed, leaf.SyncStatus)
	require.Equal(t, "deleted upstream during sync", leaf.SyncError)
	require.Equal(t, []dtoenterprise.DepartmentNameHistoryEntry{
		{Name: "Core Platform", ChangedAt: 1700000000},
	}, leaf.NameHistory)
	require.NotNil(t, leaf.Children)
	require.Empty(t, leaf.Children)
}

func TestEnterpriseDepartmentAdminSeesOnlyManageableDepartmentTree(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	parentId := 1
	childId := 2
	require.NoError(t, fixture.db.Create(&[]modelenterprise.Department{
		enterpriseDepartment(1, nil, "Headquarters", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
		enterpriseDepartment(2, &parentId, "Engineering", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeDingTalk, constant.DepartmentSyncStatusOK),
		enterpriseDepartment(3, &childId, "Platform", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeDingTalk, constant.DepartmentSyncStatusOK),
		enterpriseDepartment(4, nil, "Finance", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 2,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	recorder := fixture.performDepartmentTreeRequest(t, fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))

	payload := decodeDepartmentTreeAPIResponse(t, recorder)
	require.True(t, payload.Success, payload.Message)
	require.Len(t, payload.Data, 1)
	require.Equal(t, "Engineering", payload.Data[0].Name)
	require.Len(t, payload.Data[0].Children, 1)
	require.Equal(t, "Platform", payload.Data[0].Children[0].Name)
	require.NotContains(t, recorder.Body.String(), "Headquarters")
	require.NotContains(t, recorder.Body.String(), "Finance")
}

func TestEnterpriseDepartmentMembersAPIUsesBackendDepartmentPermission(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 2001, Username: "member", Password: "password123", Group: "vip", AffCode: "member-api"}).Error)
	require.NoError(t, fixture.db.Create(&[]modelenterprise.Department{
		enterpriseDepartment(1, nil, "Engineering", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
		enterpriseDepartment(2, nil, "Finance", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.UserDepartment{
		UserId:         2001,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	allowed := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/1/members", cookies)
	allowedPayload := decodeDepartmentMembersAPIResponse(t, allowed)
	require.True(t, allowedPayload.Success, allowedPayload.Message)
	require.Contains(t, string(allowedPayload.Data), "member")

	denied := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/2/members", cookies)
	deniedPayload := decodeDepartmentMembersAPIResponse(t, denied)
	require.False(t, deniedPayload.Success)
	require.Contains(t, deniedPayload.Message, "error.enterprise.permission.dept_admin_required")
}

func TestEnterpriseDepartmentTreeAPIMapsInvalidNameHistoryToBusinessError(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	department := enterpriseDepartment(1, nil, "Broken history", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK)
	department.NameHistory = "{"
	require.NoError(t, fixture.db.Create(&department).Error)

	recorder := fixture.performDepartmentTreeRequest(t, fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload departmentTreeAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.False(t, payload.Success)
	require.Contains(t, payload.Message, "error.enterprise.organization.invalid_name_history")
}

type enterpriseDepartmentTreeAPIFixture struct {
	engine *gin.Engine
	db     *gorm.DB
}

func newEnterpriseDepartmentTreeAPIFixture(t *testing.T) enterpriseDepartmentTreeAPIFixture {
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
	require.NoError(t, modelenterprise.Migrate(db))
	model.DB = db
	model.LOG_DB = db

	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("enterprise-department-tree-api-test"))))
	engine.GET("/login/:role", func(c *gin.Context) {
		role := common.RoleAdminUser
		if c.Param("role") == "user" {
			role = common.RoleCommonUser
		}
		session := sessions.Default(c)
		session.Set("username", "enterprise-admin")
		session.Set("role", role)
		session.Set("id", 1001)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	apiRouter := engine.Group("/api")
	router.RegisterEnterpriseRouter(apiRouter)

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

	return enterpriseDepartmentTreeAPIFixture{engine: engine, db: db}
}

func (f enterpriseDepartmentTreeAPIFixture) login(t *testing.T, role int, status int) []*http.Cookie {
	t.Helper()

	path := "/login/admin"
	if role == common.RoleCommonUser {
		path = "/login/user"
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	f.engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusNoContent, recorder.Code)

	if status != common.UserStatusEnabled {
		t.Fatalf("test fixture only supports enabled login sessions")
	}
	return recorder.Result().Cookies()
}

func (f enterpriseDepartmentTreeAPIFixture) performDepartmentTreeRequest(t *testing.T, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	return f.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/tree", cookies)
}

func (f enterpriseDepartmentTreeAPIFixture) performEnterpriseRequest(t *testing.T, method string, path string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	if len(cookies) > 0 {
		request.Header.Set("New-Api-User", "1001")
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
	}
	f.engine.ServeHTTP(recorder, request)
	return recorder
}

func decodeDepartmentTreeAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) departmentTreeAPIResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload departmentTreeAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	return payload
}

func decodeDepartmentMembersAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) departmentMembersAPIResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload departmentMembersAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	return payload
}

func enterpriseDepartment(id int, parentId *int, name string, status int, sourceType int, syncStatus int) modelenterprise.Department {
	return modelenterprise.Department{
		Id:          id,
		TenantId:    0,
		Name:        name,
		ParentId:    parentId,
		Status:      status,
		SourceType:  sourceType,
		ExternalId:  strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		SyncStatus:  syncStatus,
		SyncError:   "",
		NameHistory: "[]",
		CreatedAt:   1700000000,
		UpdatedAt:   1700000000,
	}
}
