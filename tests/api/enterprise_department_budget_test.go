package api_test

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/model"
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
)

func TestEnterpriseDepartmentBudgetAPIWorkflowWritesAuditAndHidesPayloadInList(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	totalQuota := int64(1000)
	create := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budget", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:       modelenterprise.DepartmentBudgetTypeBalance,
		TotalQuota: &totalQuota,
	})
	createPayload := decodeDepartmentMembersAPIResponse(t, create)
	require.True(t, createPayload.Success, createPayload.Message)
	require.Contains(t, string(createPayload.Data), `"type":"balance"`)
	require.Contains(t, string(createPayload.Data), `"remaining":1000`)

	get := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/1/budget", cookies)
	getPayload := decodeDepartmentMembersAPIResponse(t, get)
	require.True(t, getPayload.Success, getPayload.Message)
	require.Contains(t, string(getPayload.Data), `"total_quota":1000`)

	actions := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions?page=1&page_size=20&object_type=enterprise_department_budget", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	actionsPayload := decodeAdminActionsAPIResponse(t, actions)
	require.True(t, actionsPayload.Success, actionsPayload.Message)
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.create")
	require.NotContains(t, string(actionsPayload.Data), `"payload"`)

	detail := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions/1", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	detailPayload := decodeAdminActionsAPIResponse(t, detail)
	require.True(t, detailPayload.Success, detailPayload.Message)
	require.Contains(t, string(detailPayload.Data), `"object_type":"enterprise_department_budget"`)
	require.Contains(t, string(detailPayload.Data), `"payload":"{`)
	require.Contains(t, string(detailPayload.Data), `\"type\":\"balance\"`)
	require.Contains(t, string(detailPayload.Data), `\"total_quota\":1000`)
}

func TestEnterpriseDepartmentBudgetAPIRejectsInvalidSubscriptionAndAuditsFailure(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	cycleQuota := int64(200)
	startedAt := int64(1700000000)
	create := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budget", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:           modelenterprise.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "custom",
		CycleStartedAt: &startedAt,
	})
	createPayload := decodeDepartmentMembersAPIResponse(t, create)
	require.False(t, createPayload.Success)
	require.Contains(t, createPayload.Message, "enterprise.organization.department_budget_invalid_custom_seconds")

	actions := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions?page=1&page_size=20&object_type=enterprise_department_budget", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	actionsPayload := decodeAdminActionsAPIResponse(t, actions)
	require.True(t, actionsPayload.Success, actionsPayload.Message)
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.reject")
}

func TestEnterpriseDepartmentBudgetAPITenantScopedDepartmentAdminFlow(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          101,
		TenantId:    1,
		Name:        "Tenant One Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		TenantId:     1,
		UserId:       1001,
		DepartmentId: 101,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	cycleQuota := int64(300)
	startedAt := int64(1700000000)
	customSeconds := int64(7200)
	withoutTenant := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/101/budget", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:           modelenterprise.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "custom",
		CycleStartedAt: &startedAt,
		CustomSeconds:  &customSeconds,
	})
	withoutTenantPayload := decodeDepartmentMembersAPIResponse(t, withoutTenant)
	require.False(t, withoutTenantPayload.Success)
	require.Contains(t, withoutTenantPayload.Message, "common.database_error")

	withTenant := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/101/budget?tenant_id=1", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:           modelenterprise.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "custom",
		CycleStartedAt: &startedAt,
		CustomSeconds:  &customSeconds,
	})
	withTenantPayload := decodeDepartmentMembersAPIResponse(t, withTenant)
	require.True(t, withTenantPayload.Success, withTenantPayload.Message)
	require.Contains(t, string(withTenantPayload.Data), `"tenant_id":1`)
	require.Contains(t, string(withTenantPayload.Data), `"cycle_type":"custom"`)
	require.Contains(t, string(withTenantPayload.Data), `"custom_seconds":7200`)

	get := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/101/budget?tenant_id=1", cookies)
	getPayload := decodeDepartmentMembersAPIResponse(t, get)
	require.True(t, getPayload.Success, getPayload.Message)
	require.Contains(t, string(getPayload.Data), `"department_id":101`)
}

func TestEnterpriseDepartmentBudgetAPIRejectsBudgetTypeSwitchForDepartment(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	totalQuota := int64(1000)
	firstCreate := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budget", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:       modelenterprise.DepartmentBudgetTypeBalance,
		TotalQuota: &totalQuota,
	})
	firstPayload := decodeDepartmentMembersAPIResponse(t, firstCreate)
	require.True(t, firstPayload.Success, firstPayload.Message)

	cycleQuota := int64(200)
	startedAt := int64(1700000000)
	typeSwitch := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budget", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:           modelenterprise.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "monthly",
		CycleStartedAt: &startedAt,
	})
	switchPayload := decodeDepartmentMembersAPIResponse(t, typeSwitch)
	require.False(t, switchPayload.Success)
	require.Contains(t, switchPayload.Message, "enterprise.organization.department_budget_type_immutable")
}

func TestEnterpriseDepartmentBudgetListAndDetailAPI(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.Option{}, &model.User{}, &model.UserSubscription{}))
	require.NoError(t, fixture.db.Create(&model.User{
		Id:          2001,
		Username:    "alice",
		DisplayName: "Alice",
		Password:    "password123",
		AffCode:     "alice-aff",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentBudget{
		Id:             8,
		TenantId:       0,
		DepartmentId:   1,
		Type:           modelenterprise.DepartmentBudgetTypeSubscription,
		Status:         modelenterprise.DepartmentBudgetStatusActive,
		CycleQuota:     500,
		Remaining:      100,
		AllocatedTotal: 400,
		CycleType:      "monthly",
		CycleStartedAt: 1700000000,
		ParentStatus:   modelenterprise.DepartmentBudgetStatusPaused,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.QuotaAllocation{
		Id:                 9,
		TenantId:           0,
		DepartmentBudgetId: 8,
		DepartmentId:       1,
		TargetUserId:       2001,
		WalletId:           19,
		CommittedQuota:     300,
		BudgetTypeSnapshot: modelenterprise.DepartmentBudgetTypeSubscription,
		CycleTypeSnapshot:  "monthly",
		Status:             modelenterprise.QuotaAllocationStatusExpired,
		ProcessedAt:        1700000200,
		CreatedAt:          1700000000,
	}).Error)
	require.NoError(t, fixture.db.Create(&model.UserSubscription{
		Id:                 19,
		UserId:             2001,
		AmountTotal:        300,
		AmountUsed:         30,
		Status:             "expired",
		SourceType:         model.SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 9,
		NextResetTime:      1700000300,
		EndTime:            1700000400,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	list := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/1/budgets?sort_by=usage_ratio&sort_order=desc", cookies)
	listPayload := decodeDepartmentMembersAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	require.Contains(t, string(listPayload.Data), `"usage_ratio":80`)
	require.Contains(t, string(listPayload.Data), `"threshold_state":"warning"`)

	detail := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/1/budgets/8", cookies)
	detailPayload := decodeDepartmentMembersAPIResponse(t, detail)
	require.True(t, detailPayload.Success, detailPayload.Message)
	require.Contains(t, string(detailPayload.Data), `"target_username":"alice"`)
	require.Contains(t, string(detailPayload.Data), `"remain_quota":270`)
	require.Contains(t, string(detailPayload.Data), `"wallet_status":"expired"`)
	require.Contains(t, string(detailPayload.Data), `"source_parent_budget_status":"active"`)
}
