package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetSelfIncludesEnterpriseOrganizationPermission(t *testing.T) {
	db := setupUserSelfControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{Id: 100, Username: "dept-owner", Password: "password123", AffCode: "dept-owner-aff", Role: common.RoleCommonUser}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 1, TenantId: 0, Name: "Engineering", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       100,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualGrant,
		Effect:       constant.EnterpriseDepartmentRoleEffectAllow,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	recorder := performGetSelfRequest(100, common.RoleCommonUser)
	response := decodeUserSelfResponse(t, recorder)

	permissions := response.Data["permissions"].(map[string]any)
	require.Equal(t, true, permissions["enterprise_organization"])
}

func TestGetSelfKeepsEnterpriseOrganizationHiddenForOrdinaryUsers(t *testing.T) {
	db := setupUserSelfControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{Id: 101, Username: "member", Password: "password123", AffCode: "member-aff", Role: common.RoleCommonUser}).Error)

	recorder := performGetSelfRequest(101, common.RoleCommonUser)
	response := decodeUserSelfResponse(t, recorder)

	permissions := response.Data["permissions"].(map[string]any)
	require.Equal(t, false, permissions["enterprise_organization"])
}

type userSelfAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func setupUserSelfControllerTestDB(t *testing.T) *gorm.DB {
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
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, entmodel.AutoMigrate(db))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func performGetSelfRequest(userId int, role int) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", userId)
	ctx.Set("role", role)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	GetSelf(ctx)
	return recorder
}

func decodeUserSelfResponse(t *testing.T, recorder *httptest.ResponseRecorder) userSelfAPIResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var response userSelfAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	return response
}
