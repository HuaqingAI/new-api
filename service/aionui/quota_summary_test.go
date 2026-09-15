package aionui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBuildQuotaSummaryGroupsEnterpriseAndPersonalSubscriptions(t *testing.T) {
	db := newQuotaSummaryTestDB(t)
	model.DB = db
	model.LOG_DB = db
	t.Setenv("HTH_QUOTA_APPLY_URL", "https://new-api.test/wallet#quota-request")
	require.NoError(t, db.Create(&model.User{
		Id:        301,
		Username:  "quota-user",
		Quota:     1200,
		UsedQuota: 300,
		Status:    common.UserStatusEnabled,
		AffCode:   "quota-user",
	}).Error)
	require.NoError(t, db.Create(&model.UserSubscription{
		UserId:      301,
		AmountTotal: 1000,
		AmountUsed:  250,
		Status:      "active",
		SourceType:  model.SubscriptionSourceTypeEnterprise,
		EndTime:     0,
	}).Error)
	require.NoError(t, db.Create(&model.UserSubscription{
		UserId:      301,
		PlanId:      9,
		AmountTotal: 600,
		AmountUsed:  100,
		Status:      "active",
		SourceType:  model.SubscriptionSourceTypeOrder,
		EndTime:     0,
	}).Error)
	require.NoError(t, db.Create(&model.UserSubscription{
		UserId:      301,
		PlanId:      10,
		AmountTotal: 300,
		AmountUsed:  20,
		Status:      "active",
		SourceType:  model.SubscriptionSourceTypeAdmin,
		EndTime:     0,
	}).Error)

	result, err := BuildQuotaSummary(301)

	require.NoError(t, err)
	require.Equal(t, 1200, result.Wallet.RemainQuota)
	require.Equal(t, 300, result.Wallet.UsedQuota)
	require.Equal(t, "https://new-api.test/wallet#quota-request", result.QuotaApplyURL)
	require.Len(t, result.Subscriptions, 2)
	require.Equal(t, SubscriptionGroupEnterprise, result.Subscriptions[0].GroupKey)
	require.Equal(t, int64(750), result.Subscriptions[0].AmountAvailable)
	require.Equal(t, quotaDisplayInt64(750), result.Subscriptions[0].AmountAvailableDisplay)
	require.Len(t, result.Subscriptions[0].Items, 1)
	require.Equal(t, quotaDisplayInt64(750), result.Subscriptions[0].Items[0].AmountAvailableDisplay)
	require.Equal(t, SubscriptionGroupPersonal, result.Subscriptions[1].GroupKey)
	require.Equal(t, int64(780), result.Subscriptions[1].AmountAvailable)
	require.Equal(t, quotaDisplayInt64(780), result.Subscriptions[1].AmountAvailableDisplay)
	require.Len(t, result.Subscriptions[1].Items, 2)
	require.Equal(t, int64(2730), result.TotalAvailable)
	require.Equal(t, quotaDisplayInt64(2730), result.TotalAvailableDisplay)
}

func TestBuildQuotaSummaryReturnsEmptyGroupsWhenNoSubscriptions(t *testing.T) {
	db := newQuotaSummaryTestDB(t)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.Create(&model.User{
		Id:       302,
		Username: "empty-quota-user",
		Quota:    10,
		Status:   common.UserStatusEnabled,
		AffCode:  "empty-quota-user",
	}).Error)

	result, err := BuildQuotaSummary(302)

	require.NoError(t, err)
	require.Len(t, result.Subscriptions, 2)
	require.Empty(t, result.Subscriptions[0].Items)
	require.Empty(t, result.Subscriptions[1].Items)
	require.Equal(t, int64(10), result.TotalAvailable)
}

func TestQuotaDisplayUsesTwoDecimalsForCurrency(t *testing.T) {
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	require.Equal(t, "$1.54", quotaDisplay(770384))

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeTokens
	require.Equal(t, "770384", quotaDisplay(770384))
}

func newQuotaSummaryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.RedisEnabled = oldRedisEnabled
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
