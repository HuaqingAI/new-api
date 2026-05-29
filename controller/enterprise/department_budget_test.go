package enterprise

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDepartmentBudgetAPIWorkflow(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/departments/:id/budget", CreateDepartmentBudget)
	router.GET("/api/enterprise/departments/:id/budget", GetDepartmentBudget)

	total := int64(1000)
	createRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budget", dtoenterprise.CreateDepartmentBudgetRequest{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	})
	createResponse := decodeEnterpriseAPIResponse(t, createRecorder)
	require.True(t, createResponse.Success, createResponse.Message)

	getRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/1/budget", nil)
	getResponse := decodeEnterpriseAPIResponse(t, getRecorder)
	require.True(t, getResponse.Success, getResponse.Message)
	data := decodeEnterpriseData[dtoenterprise.DepartmentBudgetResponse](t, getResponse)
	require.NotNil(t, data.Item)
	require.Equal(t, entmodel.DepartmentBudgetTypeBalance, data.Item.Type)
	require.Equal(t, int64(1000), data.Item.TotalQuota)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, "enterprise.organization.department_budget.create", actions[0].ActionType)
	require.Equal(t, "enterprise_department_budget", actions[0].ObjectType)
}

func TestDepartmentBudgetAPIRejectsInvalidPayload(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/departments/:id/budget", CreateDepartmentBudget)

	total := int64(0)
	recorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budget", dtoenterprise.CreateDepartmentBudgetRequest{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	})
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "enterprise.organization.department_budget_invalid_quota", response.Message)

	var actions []entmodel.AdminAction
	require.NoError(t, model.DB.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, "enterprise.organization.department_budget.reject", actions[0].ActionType)
}

func TestDepartmentBudgetAPIRejectsTypeSwitch(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/departments/:id/budget", CreateDepartmentBudget)

	total := int64(1000)
	createBalance := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budget", dtoenterprise.CreateDepartmentBudgetRequest{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	})
	createBalanceResponse := decodeEnterpriseAPIResponse(t, createBalance)
	require.True(t, createBalanceResponse.Success, createBalanceResponse.Message)

	cycleQuota := int64(200)
	startedAt := int64(1700000000)
	createSubscription := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budget", dtoenterprise.CreateDepartmentBudgetRequest{
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "monthly",
		CycleStartedAt: &startedAt,
	})
	createSubscriptionResponse := decodeEnterpriseAPIResponse(t, createSubscription)
	require.False(t, createSubscriptionResponse.Success)
	require.Equal(t, "enterprise.organization.department_budget_type_immutable", createSubscriptionResponse.Message)
}

func TestDepartmentBudgetAPIRequiresValidPath(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/departments/:id/budget", GetDepartmentBudget)

	request := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/not-valid/budget", nil)
	response := decodeEnterpriseAPIResponse(t, request)
	require.False(t, response.Success)
	require.Equal(t, "common.invalid_params", response.Message)
}

func TestDepartmentBudgetAPIDeniesNonDepartmentAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	model.DB = setupEnterpriseBudgetPermissionDB(t)
	router.Use(func(c *gin.Context) {
		c.Set("id", 200)
		c.Set("role", common.RoleCommonUser)
		c.Next()
	})
	router.POST(
		"/api/enterprise/departments/:id/budget",
		middleware.EnterpriseDepartmentAdmin("id"),
		CreateDepartmentBudget,
	)

	total := int64(100)
	body := dtoenterprise.CreateDepartmentBudgetRequest{
		Type:       entmodel.DepartmentBudgetTypeBalance,
		TotalQuota: &total,
	}
	recorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budget", body)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "error.enterprise.permission.dept_admin_required", response.Message)
}

func setupEnterpriseBudgetPermissionDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, entmodel.AutoMigrate(db))
	require.NoError(t, db.Create(&model.User{Id: 200, Username: "viewer", Password: "password123", AffCode: "viewer-aff"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 1, TenantId: 0, Name: "Engineering", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
