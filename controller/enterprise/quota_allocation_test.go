package enterprise

import (
	"net/http"
	"testing"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestQuotaAllocationAPIWorkflow(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/quota-allocations", CreateQuotaAllocation)
	router.GET("/api/enterprise/quota-allocations", ListQuotaAllocations)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Id:       2001,
		Username: "quota-member-one",
		Password: "pwd",
		Group:    "default",
		AffCode:  "quota-member-one-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       2001,
		DepartmentId: 1,
		Status:       1,
	}).Error)

	create := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations", dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		CommittedQuota:     int64Ptr(300),
	})
	response := decodeEnterpriseAPIResponse(t, create)
	require.True(t, response.Success, response.Message)

	list := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/quota-allocations?department_budget_id=1&department_id=1", nil)
	listResponse := decodeEnterpriseAPIResponse(t, list)
	require.True(t, listResponse.Success, listResponse.Message)
	require.Contains(t, string(listResponse.Data), `"wallet_id":`)
}

func TestQuotaAllocationAPIRejectsUserOutsideDepartment(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/quota-allocations", CreateQuotaAllocation)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Id:       2002,
		Username: "quota-member-two",
		Password: "pwd",
		Group:    "default",
		AffCode:  "quota-member-two-aff",
	}).Error)

	create := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations", dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2002,
		CommittedQuota:     int64Ptr(300),
	})
	response := decodeEnterpriseAPIResponse(t, create)
	require.False(t, response.Success)
	require.Equal(t, "enterprise.organization.quota_allocation_user_out_of_department", response.Message)
}

func TestQuotaAllocationListRejectsMismatchedDepartmentID(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.GET("/api/enterprise/quota-allocations", ListQuotaAllocations)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)

	list := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/quota-allocations?department_budget_id=1&department_id=2", nil)
	response := decodeEnterpriseAPIResponse(t, list)
	require.False(t, response.Success)
	require.Equal(t, "common.invalid_params", response.Message)
}

func int64Ptr(value int64) *int64 {
	return &value
}
