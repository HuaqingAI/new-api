package enterprise

import (
	"net/http"
	"strings"
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

func TestDepartmentBudgetListAndDetailAPI(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/departments/:id/budgets", ListDepartmentBudgets)
	router.GET("/api/enterprise/departments/:id/budgets/:budget_id", GetDepartmentBudgetDetail)

	require.NoError(t, db.Create(&model.User{
		Id:          2001,
		Username:    "budget-alice",
		DisplayName: "Alice",
		Password:    "password123",
		AffCode:     "budget-alice-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:             1,
		TenantId:       0,
		DepartmentId:   1,
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		Status:         entmodel.DepartmentBudgetStatusActive,
		CycleQuota:     500,
		Remaining:      100,
		AllocatedTotal: 400,
		CycleType:      "monthly",
		CycleStartedAt: 1700000000,
		ParentStatus:   entmodel.DepartmentBudgetStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           2,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusPaused,
		TotalQuota:   1000,
		Remaining:    900,
		ParentStatus: entmodel.DepartmentBudgetStatusPaused,
	}).Error)
	require.NoError(t, db.Create(&entmodel.QuotaAllocation{
		Id:                 3,
		TenantId:           0,
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		WalletId:           9,
		CommittedQuota:     300,
		BudgetTypeSnapshot: entmodel.DepartmentBudgetTypeSubscription,
		CycleTypeSnapshot:  "monthly",
		Status:             entmodel.QuotaAllocationStatusActive,
		CreatedAt:          1700000000,
	}).Error)
	require.NoError(t, db.Create(&model.UserSubscription{
		Id:                 9,
		UserId:             2001,
		AmountTotal:        300,
		AmountUsed:         25,
		Status:             "active",
		SourceType:         model.SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 3,
		NextResetTime:      1700000500,
		EndTime:            1700000800,
	}).Error)

	listRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/1/budgets?sort_by=usage_ratio&sort_order=desc", nil)
	listResponse := decodeEnterpriseAPIResponse(t, listRecorder)
	require.True(t, listResponse.Success, listResponse.Message)
	listData := decodeEnterpriseData[dtoenterprise.DepartmentBudgetListResponse](t, listResponse)
	require.Len(t, listData.Items, 2)
	require.Equal(t, 1, listData.Items[0].Id)
	require.Equal(t, float64(80), listData.Items[0].UsageRatio)
	require.Equal(t, "warning", listData.Items[0].ThresholdState)
	require.Equal(t, 80, listData.Thresholds.Warning)
	require.Equal(t, 95, listData.Thresholds.Critical)

	detailRecorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/1/budgets/1", nil)
	detailResponse := decodeEnterpriseAPIResponse(t, detailRecorder)
	require.True(t, detailResponse.Success, detailResponse.Message)
	detailData := decodeEnterpriseData[dtoenterprise.DepartmentBudgetDetailResponse](t, detailResponse)
	require.NotNil(t, detailData.Budget)
	require.Equal(t, 1, detailData.Budget.Id)
	require.Len(t, detailData.Wallets, 1)
	require.Equal(t, 2001, detailData.Wallets[0].TargetUserId)
	require.Equal(t, "budget-alice", detailData.Wallets[0].TargetUsername)
	require.Equal(t, int64(275), detailData.Wallets[0].RemainQuota)
	require.Equal(t, entmodel.DepartmentBudgetStatusActive, detailData.Wallets[0].SourceParentBudgetStatus)
}

func TestDepartmentBudgetLifecycleAPIWritesAdminActions(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/departments/:id/budgets/:budget_id/pause", PauseDepartmentBudget)
	router.POST("/api/enterprise/departments/:id/budgets/:budget_id/resume", ResumeDepartmentBudget)
	router.POST("/api/enterprise/departments/:id/budgets/:budget_id/resize", ResizeDepartmentBudget)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           20,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    700,
	}).Error)

	pauseRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budgets/20/pause", dtoenterprise.DepartmentBudgetLifecycleRequest{})
	pauseResponse := decodeEnterpriseAPIResponse(t, pauseRecorder)
	require.True(t, pauseResponse.Success, pauseResponse.Message)
	pauseData := decodeEnterpriseData[dtoenterprise.DepartmentBudgetResponse](t, pauseResponse)
	require.Equal(t, entmodel.DepartmentBudgetStatusPaused, pauseData.Item.Status)

	resumeRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budgets/20/resume", dtoenterprise.DepartmentBudgetLifecycleRequest{})
	resumeResponse := decodeEnterpriseAPIResponse(t, resumeRecorder)
	require.True(t, resumeResponse.Success, resumeResponse.Message)
	resumeData := decodeEnterpriseData[dtoenterprise.DepartmentBudgetResponse](t, resumeResponse)
	require.Equal(t, entmodel.DepartmentBudgetStatusActive, resumeData.Item.Status)

	total := int64(1200)
	resizeRecorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budgets/20/resize", dtoenterprise.ResizeDepartmentBudgetRequest{TotalQuota: &total})
	resizeResponse := decodeEnterpriseAPIResponse(t, resizeRecorder)
	require.True(t, resizeResponse.Success, resizeResponse.Message)
	resizeData := decodeEnterpriseData[dtoenterprise.DepartmentBudgetResponse](t, resizeResponse)
	require.Equal(t, int64(1200), resizeData.Item.TotalQuota)
	require.Equal(t, int64(900), resizeData.Item.Remaining)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 3)
	require.Equal(t, "enterprise.organization.department_budget.pause", actions[0].ActionType)
	require.Contains(t, actions[0].Payload, `"before_status":"active"`)
	require.Contains(t, actions[0].Payload, `"after_status":"paused"`)
	require.Equal(t, "enterprise.organization.department_budget.resume", actions[1].ActionType)
	require.Equal(t, "enterprise.organization.department_budget.resize", actions[2].ActionType)
	require.Contains(t, actions[2].Payload, `"before_total_quota":1000`)
	require.Contains(t, actions[2].Payload, `"after_total_quota":1200`)
}

func TestDepartmentBudgetResizeAPIRejectsBelowCommittedAndAudits(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/departments/:id/budgets/:budget_id/resize", ResizeDepartmentBudget)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           21,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    600,
	}).Error)

	total := int64(399)
	recorder := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/departments/1/budgets/21/resize", dtoenterprise.ResizeDepartmentBudgetRequest{TotalQuota: &total})
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "enterprise.organization.department_budget_resize_below_committed", response.Message)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 21).First(&budget).Error)
	require.Equal(t, int64(1000), budget.TotalQuota)
	require.Equal(t, int64(600), budget.Remaining)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 1)
	require.Equal(t, "enterprise.organization.department_budget.resize.reject", actions[0].ActionType)
	require.True(t, strings.Contains(actions[0].Payload, "resize below committed"))
}

func TestDepartmentBudgetListAPIIncludesDescendantsAndScopeMetadata(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/departments/:id/budgets", ListDepartmentBudgets)
	parentID := 1
	require.NoError(t, db.Create(&entmodel.Department{
		Id:       3,
		TenantId: 0,
		Name:     "Platform",
		ParentId: &parentID,
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           5,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   100,
		Remaining:    80,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           6,
		TenantId:     0,
		DepartmentId: 3,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   50,
		Remaining:    10,
	}).Error)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/1/budgets?include_descendants=true", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	data := decodeEnterpriseData[dtoenterprise.DepartmentBudgetListResponse](t, response)
	require.Len(t, data.Items, 2)
	require.True(t, data.IncludeDescendants)
	require.Equal(t, "Engineering", data.ScopeDepartmentName)
	require.Equal(t, []int{1, 3}, data.ScopeDepartmentIds)
}

func TestDepartmentBudgetDetailReturnsBudgetNotFound(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/departments/:id/budgets/:budget_id", GetDepartmentBudgetDetail)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/1/budgets/999", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, "enterprise.organization.department_budget_not_found", response.Message)
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

func TestDepartmentBudgetAPIAllowsMixedTypeCreates(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
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
	require.True(t, createSubscriptionResponse.Success, createSubscriptionResponse.Message)

	var budgets []entmodel.DepartmentBudget
	require.NoError(t, db.Where("department_id = ?", 1).Order("id ASC").Find(&budgets).Error)
	require.Len(t, budgets, 2)
	require.Equal(t, entmodel.DepartmentBudgetTypeBalance, budgets[0].Type)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, budgets[1].Type)

	var actions []entmodel.AdminAction
	require.NoError(t, db.Order("action_id ASC").Find(&actions).Error)
	require.Len(t, actions, 2)
	require.Equal(t, "enterprise.organization.department_budget.create", actions[0].ActionType)
	require.Equal(t, "enterprise.organization.department_budget.create", actions[1].ActionType)
}

func TestDepartmentBudgetAPIRequiresValidPath(t *testing.T) {
	router, _ := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/departments/:id/budget", GetDepartmentBudget)

	request := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/departments/not-valid/budget", nil)
	response := decodeEnterpriseAPIResponse(t, request)
	require.False(t, response.Success)
	require.Equal(t, "common.invalid_params", response.Message)
}

func TestDepartmentBudgetAPIDeniesNonEnterpriseAdmin(t *testing.T) {
	for _, tc := range []struct {
		name      string
		role      int
		deptAdmin bool
	}{
		{name: "common user", role: common.RoleCommonUser},
		{name: "department admin", role: common.RoleCommonUser, deptAdmin: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			db := setupEnterpriseBudgetPermissionDB(t)
			model.DB = db
			if tc.deptAdmin {
				require.NoError(t, db.Create(&entmodel.DepartmentRole{
					UserId:       200,
					DepartmentId: 1,
					Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
					Status:       constant.EnterpriseDepartmentRoleStatusActive,
				}).Error)
			}
			router.Use(func(c *gin.Context) {
				c.Set("id", 200)
				c.Set("role", tc.role)
				c.Next()
			})
			router.POST(
				"/api/enterprise/departments/:id/budget",
				middleware.EnterpriseAdmin(),
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
			require.Equal(t, "error.enterprise.permission.admin_required", response.Message)
		})
	}
}

func setupEnterpriseBudgetPermissionDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Option{}))
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
