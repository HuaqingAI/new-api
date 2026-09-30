package enterprise

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestBudgetDelegationAPIWorkflow(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/budget-delegations", CreateBudgetDelegation)
	router.GET("/api/enterprise/budget-delegations", ListBudgetDelegations)
	router.POST("/api/enterprise/budget-delegations/:id/supersede", SupersedeBudgetDelegation)

	parentID := 1
	require.NoError(t, db.Create(&entmodel.Department{
		Id:       3,
		TenantId: 0,
		Name:     "Platform",
		ParentId: &parentID,
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           2,
		TenantId:     0,
		DepartmentId: 3,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   100,
		Remaining:    100,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       999,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualGrant,
		Effect:       constant.EnterpriseDepartmentRoleEffectAllow,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	create := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/budget-delegations", dtoenterprise.CreateBudgetDelegationRequest{
		SourceDepartmentId: 1,
		SourceBudgetId:     1,
		TargetDepartmentId: 3,
		TargetBudgetId:     2,
		CommittedQuota:     int64Ptr(250),
	})
	createResponse := decodeEnterpriseAPIResponse(t, create)
	require.True(t, createResponse.Success, createResponse.Message)
	require.Contains(t, string(createResponse.Data), `"source_department_id":1`)
	require.Contains(t, string(createResponse.Data), `"target_department_id":3`)
	require.Contains(t, string(createResponse.Data), `"before_source_budget_snapshot":`)

	list := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/budget-delegations?department_id=1", nil)
	listResponse := decodeEnterpriseAPIResponse(t, list)
	require.True(t, listResponse.Success, listResponse.Message)
	require.Contains(t, string(listResponse.Data), `"items":[`)

	supersede := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/budget-delegations/1/supersede", dtoenterprise.SupersedeBudgetDelegationRequest{
		SourceDepartmentId: 1,
		NewCommittedQuota:  int64Ptr(300),
		Reason:             "adjust",
	})
	supersedeResponse := decodeEnterpriseAPIResponse(t, supersede)
	require.True(t, supersedeResponse.Success, supersedeResponse.Message)
	require.Contains(t, string(supersedeResponse.Data), `"committed_quota":300`)

	var first entmodel.BudgetDelegation
	require.NoError(t, db.First(&first, 1).Error)
	require.Equal(t, entmodel.BudgetDelegationStatusSuperseded, first.Status)
	require.NotZero(t, first.SupersededById)
}

func TestBudgetDelegationAPIReportsTreeReason(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/budget-delegations", CreateBudgetDelegation)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           2,
		TenantId:     0,
		DepartmentId: 2,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   100,
		Remaining:    100,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       999,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualGrant,
		Effect:       constant.EnterpriseDepartmentRoleEffectAllow,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	create := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/budget-delegations", dtoenterprise.CreateBudgetDelegationRequest{
		SourceDepartmentId: 1,
		SourceBudgetId:     1,
		TargetDepartmentId: 2,
		TargetBudgetId:     2,
		CommittedQuota:     int64Ptr(100),
	})
	response := decodeEnterpriseAPIResponse(t, create)
	require.False(t, response.Success)
	require.Equal(t, "enterprise.organization.budget_delegation_rejected", response.Message)
	require.Contains(t, string(response.Data), `"reason":"enterprise.organization.budget_delegation_target_not_descendant"`)
}
