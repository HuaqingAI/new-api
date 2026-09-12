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
	cookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

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
	cookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

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
	cookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

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
	require.Contains(t, withoutTenantPayload.Message, "enterprise.organization.department_not_found")

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

func TestEnterpriseDepartmentBudgetAPIAllowsMixedTypeCreatesForDepartment(t *testing.T) {
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
	cookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	totalQuota := int64(1000)
	firstCreate := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budget", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:       modelenterprise.DepartmentBudgetTypeBalance,
		TotalQuota: &totalQuota,
	})
	firstPayload := decodeDepartmentMembersAPIResponse(t, firstCreate)
	require.True(t, firstPayload.Success, firstPayload.Message)
	require.Contains(t, string(firstPayload.Data), `"type":"balance"`)

	cycleQuota := int64(200)
	startedAt := int64(1700000000)
	secondCreate := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budget", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:           modelenterprise.DepartmentBudgetTypeSubscription,
		CycleQuota:     &cycleQuota,
		CycleType:      "monthly",
		CycleStartedAt: &startedAt,
	})
	secondPayload := decodeDepartmentMembersAPIResponse(t, secondCreate)
	require.True(t, secondPayload.Success, secondPayload.Message)
	require.Contains(t, string(secondPayload.Data), `"type":"subscription"`)

	list := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/1/budgets?sort_by=type&sort_order=asc", cookies)
	listPayload := decodeDepartmentMembersAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	require.Contains(t, string(listPayload.Data), `"type":"balance"`)
	require.Contains(t, string(listPayload.Data), `"type":"subscription"`)

	actions := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions?page=1&page_size=20&object_type=enterprise_department_budget", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	actionsPayload := decodeAdminActionsAPIResponse(t, actions)
	require.True(t, actionsPayload.Success, actionsPayload.Message)
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.create")
	require.NotContains(t, string(actionsPayload.Data), "enterprise.organization.department_budget_type_immutable")
}

func TestEnterpriseDepartmentBudgetAPIDepartmentAdminCanReadButCannotCreate(t *testing.T) {
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
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentBudget{
		Id:           31,
		TenantId:     0,
		DepartmentId: 1,
		Type:         modelenterprise.DepartmentBudgetTypeBalance,
		Status:       modelenterprise.DepartmentBudgetStatusActive,
		TotalQuota:   500,
		Remaining:    500,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	get := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/1/budget", cookies)
	getPayload := decodeDepartmentMembersAPIResponse(t, get)
	require.True(t, getPayload.Success, getPayload.Message)
	require.Contains(t, string(getPayload.Data), `"total_quota":500`)

	list := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/departments/1/budgets", cookies)
	listPayload := decodeDepartmentMembersAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	require.Contains(t, string(listPayload.Data), `"id":31`)

	totalQuota := int64(1000)
	create := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budget", cookies, dtoenterprise.CreateDepartmentBudgetRequest{
		Type:       modelenterprise.DepartmentBudgetTypeBalance,
		TotalQuota: &totalQuota,
	})
	createPayload := decodeDepartmentMembersAPIResponse(t, create)
	require.False(t, createPayload.Success)
	require.Equal(t, "error.enterprise.permission.admin_required", createPayload.Message)

	actions := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions?page=1&page_size=20&object_type=enterprise_department_budget", fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled))
	actionsPayload := decodeAdminActionsAPIResponse(t, actions)
	require.True(t, actionsPayload.Success, actionsPayload.Message)
	require.NotContains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.create")
	require.NotContains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.reject")
}

func TestEnterpriseDepartmentBudgetLifecycleAPIWorkflow(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 2001, Username: "member", Password: "password123", Group: "vip", AffCode: "budget-lifecycle-member"}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.Department{
		Id:          1,
		TenantId:    0,
		Name:        "Engineering",
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.UserDepartment{
		TenantId:     0,
		UserId:       2001,
		DepartmentId: 1,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentRole{
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentBudget{
		Id:           11,
		TenantId:     0,
		DepartmentId: 1,
		Type:         modelenterprise.DepartmentBudgetTypeBalance,
		Status:       modelenterprise.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    700,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.QuotaAllocation{
		Id:                     21,
		TenantId:               0,
		DepartmentBudgetId:     11,
		DepartmentId:           1,
		TargetUserId:           2001,
		WalletId:               31,
		CommittedQuota:         300,
		BudgetTypeSnapshot:     modelenterprise.DepartmentBudgetTypeBalance,
		CycleTypeSnapshot:      "never",
		CycleStartedAtSnapshot: 1700000000,
		Status:                 modelenterprise.QuotaAllocationStatusActive,
		ProcessedAt:            1700000000,
		CreatedAt:              1700000000,
	}).Error)
	require.NoError(t, fixture.db.Create(&model.UserSubscription{
		Id:          30,
		UserId:      2001,
		AmountTotal: 1,
		Status:      "active",
		Source:      model.SubscriptionSourceTypeOrder,
		SourceType:  model.SubscriptionSourceTypeOrder,
		IsPrimary:   true,
	}).Error)
	require.NoError(t, fixture.db.Create(&model.UserSubscription{
		Id:                 31,
		UserId:             2001,
		AmountTotal:        300,
		AmountUsed:         300,
		Status:             "active",
		Source:             model.SubscriptionSourceTypeEnterprise,
		SourceType:         model.SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 21,
	}).Error)
	userCookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)
	adminCookies := fixture.login(t, common.RoleAdminUser, common.UserStatusEnabled)

	forbiddenPause := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budgets/11/pause", userCookies, dtoenterprise.DepartmentBudgetLifecycleRequest{})
	forbiddenPayload := decodeDepartmentMembersAPIResponse(t, forbiddenPause)
	require.False(t, forbiddenPayload.Success)
	require.Equal(t, "error.enterprise.permission.admin_required", forbiddenPayload.Message)

	pause := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budgets/11/pause", adminCookies, dtoenterprise.DepartmentBudgetLifecycleRequest{})
	pausePayload := decodeDepartmentMembersAPIResponse(t, pause)
	require.True(t, pausePayload.Success, pausePayload.Message)
	require.Contains(t, string(pausePayload.Data), `"status":"paused"`)

	var pausedBudget modelenterprise.DepartmentBudget
	require.NoError(t, fixture.db.First(&pausedBudget, 11).Error)
	require.Equal(t, modelenterprise.DepartmentBudgetStatusPaused, pausedBudget.Status)
	var pausedAllocation modelenterprise.QuotaAllocation
	require.NoError(t, fixture.db.First(&pausedAllocation, 21).Error)
	require.Equal(t, modelenterprise.QuotaAllocationStatusPaused, pausedAllocation.Status)
	var pausedWallet model.UserSubscription
	require.NoError(t, fixture.db.First(&pausedWallet, 31).Error)
	require.Equal(t, "paused", pausedWallet.Status)

	allocationQuota := int64(50)
	createWhilePaused := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/quota-allocations", userCookies, dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 11,
		DepartmentId:       1,
		TargetUserId:       2001,
		CommittedQuota:     &allocationQuota,
	})
	createWhilePausedPayload := decodeDepartmentMembersAPIResponse(t, createWhilePaused)
	require.False(t, createWhilePausedPayload.Success)
	require.Equal(t, "enterprise.organization.quota_allocation_budget_inactive", createWhilePausedPayload.Message)

	resume := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budgets/11/resume", adminCookies, dtoenterprise.DepartmentBudgetLifecycleRequest{})
	resumePayload := decodeDepartmentMembersAPIResponse(t, resume)
	require.True(t, resumePayload.Success, resumePayload.Message)
	require.Contains(t, string(resumePayload.Data), `"status":"active"`)
	require.NoError(t, fixture.db.First(&pausedAllocation, 21).Error)
	require.Equal(t, modelenterprise.QuotaAllocationStatusActive, pausedAllocation.Status)
	require.NoError(t, fixture.db.First(&pausedWallet, 31).Error)
	require.Equal(t, "active", pausedWallet.Status)

	tooSmallTotal := int64(200)
	rejectedResize := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budgets/11/resize", adminCookies, dtoenterprise.ResizeDepartmentBudgetRequest{TotalQuota: &tooSmallTotal})
	rejectedResizePayload := decodeDepartmentMembersAPIResponse(t, rejectedResize)
	require.False(t, rejectedResizePayload.Success)
	require.Equal(t, "enterprise.organization.department_budget_resize_below_committed", rejectedResizePayload.Message)
	require.NoError(t, fixture.db.First(&pausedBudget, 11).Error)
	require.Equal(t, int64(1000), pausedBudget.TotalQuota)
	require.Equal(t, int64(700), pausedBudget.Remaining)

	resizedTotal := int64(1200)
	resize := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/departments/1/budgets/11/resize", adminCookies, dtoenterprise.ResizeDepartmentBudgetRequest{TotalQuota: &resizedTotal})
	resizePayload := decodeDepartmentMembersAPIResponse(t, resize)
	require.True(t, resizePayload.Success, resizePayload.Message)
	require.Contains(t, string(resizePayload.Data), `"total_quota":1200`)
	require.Contains(t, string(resizePayload.Data), `"remaining":900`)

	createAfterResume := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/quota-allocations", userCookies, dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 11,
		DepartmentId:       1,
		TargetUserId:       2001,
		CommittedQuota:     &allocationQuota,
	})
	createAfterResumePayload := decodeDepartmentMembersAPIResponse(t, createAfterResume)
	require.True(t, createAfterResumePayload.Success, createAfterResumePayload.Message)
	require.Contains(t, string(createAfterResumePayload.Data), `"committed_quota":50`)

	actions := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/admin-actions?page=1&page_size=20&object_type=enterprise_department_budget", adminCookies)
	actionsPayload := decodeAdminActionsAPIResponse(t, actions)
	require.True(t, actionsPayload.Success, actionsPayload.Message)
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.pause")
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.resize.reject")
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.resize")
	require.Contains(t, string(actionsPayload.Data), "enterprise.organization.department_budget.resume")
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
