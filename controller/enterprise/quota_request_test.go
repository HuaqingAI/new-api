package enterprise

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestQuotaRequestAPIWorkflow(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/quota-requests", ListQuotaRequests)
	router.POST("/api/enterprise/quota-requests", SubmitQuotaRequest)
	router.GET("/api/enterprise/quota-requests/:id", GetQuotaRequest)
	router.POST("/api/enterprise/quota-requests/:id/decision", DecideQuotaRequest)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       999,
		DepartmentId: 1,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Id:          3001,
		Username:    "owner-3001",
		DisplayName: "Owner 3001",
		Password:    "pwd",
		Group:       "default",
		AffCode:     "owner-3001-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       3001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualGrant,
		Effect:       constant.EnterpriseDepartmentRoleEffectAllow,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	submit := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-requests", dtoenterprise.SubmitQuotaRequestRequest{
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         "department_budget",
		RequestedQuota:     int64Ptr(220),
		RequestReason:      "need more quota for delivery",
	})
	submitResponse := decodeEnterpriseAPIResponse(t, submit)
	require.True(t, submitResponse.Success, submitResponse.Message)
	require.Contains(t, string(submitResponse.Data), `"status":"submitted"`)

	list := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/quota-requests?department_id=1", nil)
	listResponse := decodeEnterpriseAPIResponse(t, list)
	require.True(t, listResponse.Success, listResponse.Message)
	require.Contains(t, string(listResponse.Data), `"budget_mode":"department_budget"`)

	router.Use(func(c *gin.Context) {
		c.Set("id", 3001)
		c.Set("role", common.RoleCommonUser)
		c.Next()
	})

	decision := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-requests/1/decision", dtoenterprise.DecideQuotaRequestRequest{
		Action:         "approve",
		ApprovedQuota:  int64Ptr(180),
		ApprovalReason: "approve smaller amount",
	})
	decisionResponse := decodeEnterpriseAPIResponse(t, decision)
	require.True(t, decisionResponse.Success, decisionResponse.Message)
	require.Contains(t, string(decisionResponse.Data), `"status":"fulfilled"`)
	require.Contains(t, string(decisionResponse.Data), `"allocation"`)
	require.Contains(t, string(decisionResponse.Data), `"approved_quota":180`)
}

func TestQuotaRequestAPIRejectWorkflow(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/quota-requests", SubmitQuotaRequest)
	router.POST("/api/enterprise/quota-requests/:id/decision", DecideQuotaRequest)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       999,
		DepartmentId: 1,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)

	submit := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-requests", dtoenterprise.SubmitQuotaRequestRequest{
		DepartmentId:       1,
		DepartmentBudgetId: 1,
		BudgetMode:         "department_budget",
		RequestedQuota:     int64Ptr(120),
	})
	submitResponse := decodeEnterpriseAPIResponse(t, submit)
	require.True(t, submitResponse.Success, submitResponse.Message)

	reject := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-requests/1/decision", dtoenterprise.DecideQuotaRequestRequest{
		Action:         "reject",
		RejectedReason: "budget unavailable",
	})
	rejectResponse := decodeEnterpriseAPIResponse(t, reject)
	require.True(t, rejectResponse.Success, rejectResponse.Message)
	require.Contains(t, string(rejectResponse.Data), `"status":"rejected"`)
	require.Contains(t, string(rejectResponse.Data), `"approval_reason":"budget unavailable"`)
}

func TestQuotaRequestCapabilityRouteDoesNotCollideWithRequestDetail(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/quota-requests/capability/:department_id", GetQuotaRequestCapability)
	router.GET("/api/enterprise/quota-requests/:id", GetQuotaRequest)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       999,
		DepartmentId: 1,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)

	recorder := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/quota-requests/capability/1", nil)
	response := decodeEnterpriseAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	require.Contains(t, string(response.Data), `"can_submit":true`)
	require.Contains(t, string(response.Data), `"budgets":[`)
}
