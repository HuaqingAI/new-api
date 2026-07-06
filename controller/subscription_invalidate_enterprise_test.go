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
	"github.com/gin-gonic/gin/binding"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAdminInvalidateUserSubscriptionRejectsEnterpriseAllocationWallet(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	require.NoError(t, db.Create(&model.User{
		Id:       1001,
		Username: "admin",
		Password: "password123",
		AffCode:  "admin-aff",
		Role:     common.RoleAdminUser,
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Id:       2001,
		Username: "member",
		Password: "password123",
		AffCode:  "member-aff",
		Role:     common.RoleCommonUser,
	}).Error)
	require.NoError(t, db.Create(&entmodel.Department{
		Id:       1,
		TenantId: 0,
		Name:     "Engineering",
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           1,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    700,
	}).Error)
	require.NoError(t, db.Create(&entmodel.QuotaAllocation{
		Id:                   88,
		TenantId:             0,
		DepartmentBudgetId:   1,
		DepartmentId:         1,
		TargetUserId:         2001,
		WalletId:             5,
		ActorId:              1001,
		CommittedQuota:       300,
		BudgetTypeSnapshot:   entmodel.DepartmentBudgetTypeBalance,
		BeforeBudgetSnapshot: `{"remaining":1000}`,
		AfterBudgetSnapshot:  `{"remaining":700}`,
		Status:               entmodel.QuotaAllocationStatusActive,
	}).Error)
	require.NoError(t, db.Create(&model.UserSubscription{
		Id:                 5,
		UserId:             2001,
		Status:             "active",
		Source:             model.SubscriptionSourceTypeEnterprise,
		SourceType:         model.SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 88,
		SortOrder:          -100,
		StartTime:          common.GetTimestamp(),
		EndTime:            0,
		AmountTotal:        300,
		AmountUsed:         120,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1001)
	ctx.Params = append(ctx.Params, gin.Param{Key: "id", Value: "5"})
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/admin/user_subscriptions/5/invalidate", nil)
	ctx.Request.Header.Set("Content-Type", binding.MIMEJSON)

	AdminInvalidateUserSubscription(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)

	var allocation entmodel.QuotaAllocation
	require.NoError(t, db.Where("id = ?", 88).First(&allocation).Error)
	require.Equal(t, entmodel.QuotaAllocationStatusActive, allocation.Status)
	require.Equal(t, int64(0), allocation.ReclaimedQuota)

	var wallet model.UserSubscription
	require.NoError(t, db.Where("id = ?", 5).First(&wallet).Error)
	require.Equal(t, "active", wallet.Status)

	var budget entmodel.DepartmentBudget
	require.NoError(t, db.Where("id = ?", 1).First(&budget).Error)
	require.Equal(t, int64(700), budget.Remaining)
}

func setupSubscriptionControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.SubscriptionPlan{}, &model.UserSubscription{}))
	require.NoError(t, entmodel.AutoMigrate(db))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}
