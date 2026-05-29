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

func TestEnterpriseQuotaAllocationAPICreatesWalletWithoutAdminActionDoubleWrite(t *testing.T) {
	fixture := newEnterpriseDepartmentTreeAPIFixture(t)
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	require.NoError(t, fixture.db.Create(&model.User{Id: 2001, Username: "member", Password: "password123", Group: "vip", AffCode: "member-api"}).Error)
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
	require.NoError(t, fixture.db.Create(&modelenterprise.UserDepartment{
		TenantId:     0,
		UserId:       2001,
		DepartmentId: 1,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)
	require.NoError(t, fixture.db.Create(&modelenterprise.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         modelenterprise.DepartmentBudgetTypeBalance,
		Status:       modelenterprise.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	cookies := fixture.login(t, common.RoleCommonUser, common.UserStatusEnabled)

	create := fixture.performEnterpriseRequestWithBody(t, http.MethodPost, "/api/enterprise/quota-allocations", cookies, dtoenterprise.CreateQuotaAllocationRequest{
		DepartmentBudgetId: 1,
		DepartmentId:       1,
		TargetUserId:       2001,
		CommittedQuota:     int64PtrValue(300),
	})
	payload := decodeDepartmentMembersAPIResponse(t, create)
	require.True(t, payload.Success, payload.Message)
	require.Contains(t, string(payload.Data), `"committed_quota":300`)

	list := fixture.performEnterpriseRequest(t, http.MethodGet, "/api/enterprise/quota-allocations?department_budget_id=1&department_id=1", cookies)
	listPayload := decodeDepartmentMembersAPIResponse(t, list)
	require.True(t, listPayload.Success, listPayload.Message)
	require.Contains(t, string(listPayload.Data), `"wallet_id":`)

	var wallet model.UserSubscription
	require.NoError(t, fixture.db.Where("user_id = ? AND source_type = ?", 2001, "enterprise_allocation").First(&wallet).Error)
	require.Equal(t, 300, int(wallet.AmountTotal))

	var adminActionCount int64
	require.NoError(t, fixture.db.Model(&modelenterprise.AdminAction{}).Count(&adminActionCount).Error)
	require.Equal(t, int64(0), adminActionCount)
}

func int64PtrValue(value int64) *int64 {
	return &value
}
