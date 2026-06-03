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

type adminActionsAPIResponse struct {
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

func TestEnterpriseDepartmentMembersAPIChecksTenantScopedDepartmentPermission(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 2001, Username: "member", Password: "password123", Group: "vip", AffCode: "member-api"}).Error)
	require.NoError(t, fixture.db.Create(&model.User{Id: 2002, Username: "new-member", Password: "password123", Group: "vip", AffCode: "new-member-api"}).Error)
	require.NoError(t, fixture.db.Create(&[]modelenterprise.Department{
		enterpriseDepartment(1, nil, "Tenant zero Engineering", constant.DepartmentStatusEnabled, constant.DepartmentSourceTypeManual, constant.DepartmentSyncStatusOK),
		{
			Id:          101,
			TenantId:    1,
			Name:        "Tenant one Engineering",
			Status:      constant.DepartmentStatusEnabled,
			SourceType:  constant.DepartmentSourceTypeManual,
			SyncStatus:  constant.DepartmentSyncStatusOK,
			NameHistory: "[]",
		},
	}).Error)
	require.NoError(t, fixture.db.Create(&[]modelenterprise.UserDepartment{
		{
			TenantId:       1,
			UserId:         2001,
			DepartmentId:   101,
			ExternalSource: constant.EnterpriseExternalSourceManual,
			Status:         constant.EnterpriseMembershipStatusActive,
		},
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		TenantId:     1,
		UserId:       1001,
		DepartmentId: 101,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	withoutTenant := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/101/members", cookies)
	withoutTenantPayload := decodeDepartmentMembersAPIResponse(t, withoutTenant)
	require.False(t, withoutTenantPayload.Success)
	require.Contains(t, withoutTenantPayload.Message, "error.enterprise.permission.dept_admin_required")

	withTenant := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/101/members?tenant_id=1", cookies)
	withTenantPayload := decodeDepartmentMembersAPIResponse(t, withTenant)
	require.True(t, withTenantPayload.Success, withTenantPayload.Message)
	require.Contains(t, string(withTenantPayload.Data), "member")

	add := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/101/members?tenant_id=1", cookies, map[string]any{"user_id": 2002})
	addPayload := decodeDepartmentMembersAPIResponse(t, add)
	require.True(t, addPayload.Success, addPayload.Message)

	deactivate := fixture.performEnterpriseRequest(t, http.MethodDelete, "/api/enterprise/departments/101/members/2001?tenant_id=1", cookies)
	deactivatePayload := decodeDepartmentMembersAPIResponse(t, deactivate)
	require.True(t, deactivatePayload.Success, deactivatePayload.Message)

	var membership modelenterprise.UserDepartment
	require.NoError(t, fixture.db.Where("tenant_id = ? AND department_id = ? AND user_id = ?", 1, 101, 2001).First(&membership).Error)
	require.Equal(t, constant.EnterpriseMembershipStatusInactive, membership.Status)

	var action modelenterprise.AdminAction
	require.NoError(t, fixture.db.Where("tenant_id = ? AND object_id = ?", 1, "101:2001").First(&action).Error)
	require.Equal(t, "enterprise.organization.membership.disable", action.ActionType)
}

func TestEnterpriseAdminActionsAPIRequiresEnterpriseAdmin(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.Create(&modelenterprise.AdminAction{
		ActorId:     1001,
		ActionType:  "enterprise.organization.membership.add",
		ObjectType:  "enterprise_department_member",
		ObjectId:    "1:2001",
		DiffSummary: "Added department member",
		Payload:     `{"user_id":2001}`,
	}).Error)

	commonUser := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions", fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled))
	commonUserPayload := decodeAdminActionsAPIResponse(t, commonUser)
	require.False(t, commonUserPayload.Success)
	require.Contains(t, commonUserPayload.Message, "error.enterprise.permission.admin_required")
	require.NotContains(t, string(commonUserPayload.Data), "enterprise_department_member")

	admin := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions?page=1&page_size=20&object_type=enterprise_department_member", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	adminPayload := decodeAdminActionsAPIResponse(t, admin)
	require.True(t, adminPayload.Success, adminPayload.Message)
	require.Contains(t, string(adminPayload.Data), `"action_id":1`)
	require.Contains(t, string(adminPayload.Data), `"actor_id":1001`)
	require.Contains(t, string(adminPayload.Data), `"object_type":"enterprise_department_member"`)
	require.Contains(t, string(adminPayload.Data), `"diff_summary":"Added department member"`)
	require.NotContains(t, string(adminPayload.Data), `"payload"`)

	detail := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions/1", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	detailPayload := decodeAdminActionsAPIResponse(t, detail)
	require.True(t, detailPayload.Success, detailPayload.Message)
	require.Contains(t, string(detailPayload.Data), `"payload":"{\"user_id\":2001}"`)
}

func TestEnterpriseDingTalkConfigAPIRequiresRootAndMasksSecret(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)
	adminRecorder := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/dingtalk/config", adminCookies, map[string]any{
		"corp_id":       "corp-id",
		"app_key":       "app-key",
		"app_secret":    "plain-secret",
		"callback_url":  "https://example.com/api/oauth/dingtalk",
		"login_enabled": true,
	})
	adminPayload := decodeAdminActionsAPIResponse(t, adminRecorder)
	require.False(t, adminPayload.Success)
	require.Contains(t, adminPayload.Message, "auth.insufficient_privilege")
	require.NotContains(t, adminRecorder.Body.String(), "plain-secret")

	rootCookies := fixture.login(t, common.RoleRootUser, common.UserStatusEnabled)
	rootRecorder := fixture.performEnterpriseRequestWithBody(t, http.MethodPut, "/api/enterprise/dingtalk/config", rootCookies, map[string]any{
		"corp_id":       "corp-id",
		"app_key":       "app-key",
		"app_secret":    "plain-secret",
		"callback_url":  "https://example.com/api/oauth/dingtalk",
		"login_enabled": true,
	})
	rootPayload := decodeAdminActionsAPIResponse(t, rootRecorder)
	require.True(t, rootPayload.Success, rootPayload.Message)
	require.Contains(t, string(rootPayload.Data), `"has_app_secret":true`)
	require.NotContains(t, rootRecorder.Body.String(), "plain-secret")
	require.NotContains(t, rootRecorder.Body.String(), `"app_secret":`)

	connectivityAdmin := fixture.performEnterpriseRequest(t, http.MethodPost, "/api/enterprise/dingtalk/connectivity-test", adminCookies)
	connectivityAdminPayload := decodeAdminActionsAPIResponse(t, connectivityAdmin)
	require.False(t, connectivityAdminPayload.Success)
	require.Contains(t, connectivityAdminPayload.Message, "auth.insufficient_privilege")
}

func TestEnterpriseDepartmentAdminRoleMutationWritesAudit(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 2001, Username: "dept-admin", Password: "password123", Group: "default", AffCode: "dept-admin-api"}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	grant := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/admins", adminCookies, map[string]any{"user_id": 2001})
	grantPayload := decodeAdminActionsAPIResponse(t, grant)
	require.True(t, grantPayload.Success, grantPayload.Message)
	require.Contains(t, string(grantPayload.Data), `"user_id":2001`)

	revoke := fixture.performEnterpriseRequest(t, http.MethodDelete, "/api/enterprise/departments/1/admins/2001", adminCookies)
	revokePayload := decodeAdminActionsAPIResponse(t, revoke)
	require.True(t, revokePayload.Success, revokePayload.Message)

	actions := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions?page=1&page_size=20", adminCookies)
	actionsPayload := decodeAdminActionsAPIResponse(t, actions)
	require.True(t, actionsPayload.Success, actionsPayload.Message)
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_owner.manual_grant")
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_owner.manual_grant.revoke")
}

func TestEnterpriseAdminActionsAPIRejectsInvalidQuery(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)

	recorder := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions?page=bad", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	payload := decodeAdminActionsAPIResponse(t, recorder)

	require.False(t, payload.Success)
	require.Contains(t, payload.Message, "common.invalid_params")
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
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, modelenterprise.Migrate(db))
	require.NoError(t, db.Create(&model.User{
		Id:          1001,
		Username:    "enterprise-admin",
		DisplayName: "Enterprise Admin",
		Password:    "password123",
		Group:       "default",
		AffCode:     "enterprise-admin-api-fixture",
		Status:      common.UserStatusEnabled,
		Role:        common.RoleAdminUser,
	}).Error)
	model.DB = db
	model.LOG_DB = db

	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("enterprise-department-tree-api-test"))))
	engine.GET("/login/:role", func(c *gin.Context) {
		role := common.RoleAdminUser
		if c.Param("role") == "user" {
			role = common.RoleCommonUser
		} else if c.Param("role") == "root" {
			role = common.RoleRootUser
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
	} else if role == common.RoleRootUser {
		path = "/login/root"
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

func (f enterpriseDepartmentTreeAPIFixture) performEnterpriseRequestWithBody(t *testing.T, method string, path string, cookies []*http.Cookie, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
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

func decodeAdminActionsAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) adminActionsAPIResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload adminActionsAPIResponse
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
