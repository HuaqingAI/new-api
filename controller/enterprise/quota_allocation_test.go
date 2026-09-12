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
		Id:          2001,
		Username:    "quota-member-one",
		DisplayName: "Quota Member One",
		Password:    "pwd",
		Group:       "default",
		AffCode:     "quota-member-one-aff",
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
	require.Contains(t, string(listResponse.Data), `"target_username":"quota-member-one"`)
	require.Contains(t, string(listResponse.Data), `"target_display_name":"Quota Member One"`)
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

func TestQuotaAllocationAPIReportsSpecificBudgetReason(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/quota-allocations", CreateQuotaAllocation)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:             1,
		TenantId:       0,
		DepartmentId:   1,
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		Status:         entmodel.DepartmentBudgetStatusActive,
		CycleQuota:     100,
		Remaining:      10,
		AllocatedTotal: 90,
		CycleType:      "monthly",
		CycleStartedAt: 1700000000,
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Id:       2003,
		Username: "quota-member-three",
		Password: "pwd",
		Group:    "default",
		AffCode:  "quota-member-three-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       2003,
		DepartmentId: 1,
		Status:       1,
	}).Error)

	create := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations", dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2003,
		CommittedQuota:     int64Ptr(20),
	})
	response := decodeEnterpriseAPIResponse(t, create)
	require.False(t, response.Success)
	require.Equal(t, "enterprise.organization.enterprise_budget_insufficient", response.Message)
	require.Contains(t, string(response.Data), `"reason":"enterprise.organization.subscription_cycle_allocated_exceeded"`)
}

func TestQuotaAllocationAPIReportsSpecificBalanceBudgetReason(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/quota-allocations", CreateQuotaAllocation)

	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   100,
		Remaining:    10,
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Id:       2004,
		Username: "quota-member-four",
		Password: "pwd",
		Group:    "default",
		AffCode:  "quota-member-four-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       2004,
		DepartmentId: 1,
		Status:       1,
	}).Error)

	create := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations", dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2004,
		CommittedQuota:     int64Ptr(20),
	})
	response := decodeEnterpriseAPIResponse(t, create)
	require.False(t, response.Success)
	require.Equal(t, "enterprise.organization.enterprise_budget_insufficient", response.Message)
	require.Contains(t, string(response.Data), `"reason":"enterprise.organization.balance_remaining_insufficient"`)
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

func TestQuotaAllocationAPIRevokeWorkflow(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/quota-allocations", CreateQuotaAllocation)
	router.GET("/api/enterprise/quota-allocations", ListQuotaAllocations)
	router.POST("/api/enterprise/quota-allocations/:id/revoke", RevokeQuotaAllocation)

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
		Id:       2006,
		Username: "quota-member-six",
		Password: "pwd",
		Group:    "default",
		AffCode:  "quota-member-six-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       2006,
		DepartmentId: 1,
		Status:       1,
	}).Error)

	create := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations", dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2006,
		CommittedQuota:     int64Ptr(300),
	})
	createResponse := decodeEnterpriseAPIResponse(t, create)
	require.True(t, createResponse.Success, createResponse.Message)

	revoke := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations/1/revoke", dtoenterprise.RevokeQuotaAllocationRequest{
		DepartmentId: 1,
		Reason:       "cleanup",
	})
	revokeResponse := decodeEnterpriseAPIResponse(t, revoke)
	require.True(t, revokeResponse.Success, revokeResponse.Message)
	require.Contains(t, string(revokeResponse.Data), `"status":"revoked"`)
	require.Contains(t, string(revokeResponse.Data), `"processed_at":`)

	list := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/quota-allocations?department_budget_id=1&department_id=1", nil)
	listResponse := decodeEnterpriseAPIResponse(t, list)
	require.True(t, listResponse.Success, listResponse.Message)
	require.Contains(t, string(listResponse.Data), `"status":"revoked"`)
}

func TestQuotaAllocationAPISupersedeCancelAndReclaimWorkflow(t *testing.T) {
	router, db := setupEnterpriseControllerTest(t)
	router.POST("/api/enterprise/quota-allocations", CreateQuotaAllocation)
	router.GET("/api/enterprise/quota-allocations", ListQuotaAllocations)
	router.POST("/api/enterprise/quota-allocations/:id/supersede", SupersedeQuotaAllocation)
	router.POST("/api/enterprise/quota-allocations/:id/cancel", CancelQuotaAllocation)
	router.POST("/api/enterprise/quota-allocations/:id/reclaim", ReclaimQuotaAllocation)

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
		Id:          2010,
		Username:    "quota-member-ten",
		DisplayName: "Quota Member Ten",
		Password:    "pwd",
		Group:       "default",
		AffCode:     "quota-member-ten-aff",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Id:          2011,
		Username:    "quota-member-eleven",
		DisplayName: "Quota Member Eleven",
		Password:    "pwd",
		Group:       "default",
		AffCode:     "quota-member-eleven-aff",
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       2010,
		DepartmentId: 1,
		Status:       1,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:     0,
		UserId:       2011,
		DepartmentId: 1,
		Status:       1,
	}).Error)

	createA := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations", dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2010,
		CommittedQuota:     int64Ptr(300),
	})
	createAResponse := decodeEnterpriseAPIResponse(t, createA)
	require.True(t, createAResponse.Success, createAResponse.Message)

	supersede := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations/1/supersede", dtoenterprise.SupersedeQuotaAllocationRequest{
		DepartmentId:      1,
		NewCommittedQuota: int64Ptr(240),
		Reason:            "resize",
	})
	supersedeResponse := decodeEnterpriseAPIResponse(t, supersede)
	require.True(t, supersedeResponse.Success, supersedeResponse.Message)
	require.Contains(t, string(supersedeResponse.Data), `"status":"active"`)
	require.Contains(t, string(supersedeResponse.Data), `"supersedes_allocation_id":1`)

	createB := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations", dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2011,
		CommittedQuota:     int64Ptr(180),
	})
	createBResponse := decodeEnterpriseAPIResponse(t, createB)
	require.True(t, createBResponse.Success, createBResponse.Message)

	cancel := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations/3/cancel", dtoenterprise.CancelQuotaAllocationRequest{
		DepartmentId: 1,
		Reason:       "member moved",
	})
	cancelResponse := decodeEnterpriseAPIResponse(t, cancel)
	require.True(t, cancelResponse.Success, cancelResponse.Message)
	require.Contains(t, string(cancelResponse.Data), `"status":"revoked"`)
	require.Contains(t, string(cancelResponse.Data), `"processed_at":`)

	reclaim := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations/2/reclaim", dtoenterprise.ReclaimQuotaAllocationRequest{
		DepartmentId: 1,
		Reason:       "wallet cleanup",
	})
	reclaimResponse := decodeEnterpriseAPIResponse(t, reclaim)
	require.True(t, reclaimResponse.Success, reclaimResponse.Message)
	require.Contains(t, string(reclaimResponse.Data), `"status":"closed"`)
	require.Contains(t, string(reclaimResponse.Data), `"reclaimed_quota":240`)

	reclaimAgain := performEnterpriseRequest(t, router, http.MethodPost, "/api/enterprise/quota-allocations/2/reclaim", dtoenterprise.ReclaimQuotaAllocationRequest{
		DepartmentId: 1,
		Reason:       "retry",
	})
	reclaimAgainResponse := decodeEnterpriseAPIResponse(t, reclaimAgain)
	require.True(t, reclaimAgainResponse.Success, reclaimAgainResponse.Message)
	require.Contains(t, string(reclaimAgainResponse.Data), `"status":"closed"`)
	require.Contains(t, string(reclaimAgainResponse.Data), `"reclaimed_quota":240`)

	list := performEnterpriseRequest(t, router, http.MethodGet, "/api/enterprise/quota-allocations?department_budget_id=1&department_id=1", nil)
	listResponse := decodeEnterpriseAPIResponse(t, list)
	require.True(t, listResponse.Success, listResponse.Message)
	require.Contains(t, string(listResponse.Data), `"status":"superseded"`)
	require.Contains(t, string(listResponse.Data), `"status":"closed"`)
	require.Contains(t, string(listResponse.Data), `"status":"revoked"`)
}

func int64Ptr(value int64) *int64 {
	return &value
}
