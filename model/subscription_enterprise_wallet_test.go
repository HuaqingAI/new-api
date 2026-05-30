package model

import (
	"strings"
	"testing"
	"time"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func ensureEnterpriseAllocationTables(t *testing.T) {
	t.Helper()
	require.NoError(t, entmodel.AutoMigrate(DB))
	require.NoError(t, DB.AutoMigrate(&SubscriptionPreConsumeRecord{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM enterprise_quota_allocations")
		DB.Exec("DELETE FROM subscription_pre_consume_records")
	})
}

func TestGetAllUserSubscriptionsOrdersEnterpriseWalletFirst(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       501,
		Username: "enterprise-order-user",
		Password: "pwd",
		AffCode:  "enterprise-order-aff",
	}).Error)

	now := time.Now().Unix()
	require.NoError(t, DB.Create(&UserSubscription{
		Id:         1,
		UserId:     501,
		PlanId:     10,
		Status:     "active",
		Source:     "admin",
		SourceType: SubscriptionSourceTypeAdmin,
		SortOrder:  100,
		IsPrimary:  true,
		StartTime:  now,
		EndTime:    now + 86400,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:                 2,
		UserId:             501,
		PlanId:             0,
		Status:             "active",
		Source:             SubscriptionSourceTypeEnterprise,
		SourceType:         SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 33,
		SortOrder:          -100,
		IsPrimary:          false,
		StartTime:          now,
		EndTime:            now + 86400,
	}).Error)

	items, err := GetAllUserSubscriptions(501)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, SubscriptionSourceTypeEnterprise, items[0].Subscription.SourceType)
	require.Equal(t, 33, items[0].Subscription.SourceAllocationId)
	require.Equal(t, -100, items[0].Subscription.SortOrder)
}

func TestAdminDeleteUserSubscriptionRejectsEnterpriseWallet(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       502,
		Username: "enterprise-delete-user",
		Password: "pwd",
		AffCode:  "enterprise-delete-aff",
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:                 3,
		UserId:             502,
		Status:             "active",
		Source:             SubscriptionSourceTypeEnterprise,
		SourceType:         SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 44,
		SortOrder:          -100,
		StartTime:          time.Now().Unix(),
		EndTime:            time.Now().Unix() + 86400,
	}).Error)

	_, err := AdminDeleteUserSubscription(3)
	require.ErrorIs(t, err, ErrEnterpriseSubscriptionDeletion)
}

func TestGetAllActiveUserSubscriptionsIncludesNonExpiringEnterpriseWallet(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       503,
		Username: "enterprise-active-user",
		Password: "pwd",
		AffCode:  "enterprise-active-aff",
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:                 4,
		UserId:             503,
		Status:             "active",
		Source:             SubscriptionSourceTypeEnterprise,
		SourceType:         SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 45,
		SortOrder:          -100,
		StartTime:          time.Now().Unix(),
		EndTime:            0,
		AmountTotal:        500,
		AmountUsed:         10,
	}).Error)

	items, err := GetAllActiveUserSubscriptions(503)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, 4, items[0].Subscription.Id)
}

func TestPreConsumeUserSubscriptionMaintainsEnterpriseWalletQuotaInvariant(t *testing.T) {
	truncateTables(t)
	ensureEnterpriseAllocationTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       504,
		Username: "enterprise-preconsume-user",
		Password: "pwd",
		AffCode:  "enterprise-preconsume-aff",
	}).Error)
	require.NoError(t, DB.Create(&entmodel.QuotaAllocation{
		Id:                   88,
		DepartmentBudgetId:   1,
		DepartmentId:         1,
		TargetUserId:         504,
		WalletId:             5,
		ActorId:              1001,
		CommittedQuota:       500,
		BudgetTypeSnapshot:   entmodel.DepartmentBudgetTypeSubscription,
		CycleTypeSnapshot:    SubscriptionResetNever,
		BeforeBudgetSnapshot: "{}",
		AfterBudgetSnapshot:  "{}",
		Status:               entmodel.QuotaAllocationStatusActive,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:                 5,
		UserId:             504,
		Status:             "active",
		Source:             SubscriptionSourceTypeEnterprise,
		SourceType:         SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 88,
		SortOrder:          -100,
		StartTime:          time.Now().Unix(),
		EndTime:            time.Now().Unix() + 86400,
		AmountTotal:        500,
		AmountUsed:         0,
	}).Error)

	result, err := PreConsumeUserSubscription("enterprise-wallet-request-1", 504, "gpt-4o-mini", 0, 120)
	require.NoError(t, err)
	require.Equal(t, 5, result.UserSubscriptionId)
	require.Equal(t, int64(500), result.AmountTotal)
	require.Equal(t, int64(0), result.AmountUsedBefore)
	require.Equal(t, int64(120), result.AmountUsedAfter)

	duplicate, err := PreConsumeUserSubscription("enterprise-wallet-request-1", 504, "gpt-4o-mini", 0, 120)
	require.NoError(t, err)
	require.Equal(t, int64(120), duplicate.AmountUsedAfter)

	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", 5).First(&sub).Error)
	require.Equal(t, int64(120), sub.AmountUsed)
	require.Equal(t, int64(380), sub.AmountTotal-sub.AmountUsed)
}

func TestPreConsumeUserSubscriptionDoesNotMaskDatabaseErrorsAsNoActiveSubscription(t *testing.T) {
	truncateTables(t)
	ensureEnterpriseAllocationTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       506,
		Username: "enterprise-db-error-user",
		Password: "pwd",
		AffCode:  "enterprise-db-error-aff",
	}).Error)
	require.NoError(t, DB.Create(&entmodel.QuotaAllocation{
		Id:                   90,
		DepartmentBudgetId:   1,
		DepartmentId:         1,
		TargetUserId:         506,
		WalletId:             7,
		ActorId:              1001,
		CommittedQuota:       200,
		BudgetTypeSnapshot:   entmodel.DepartmentBudgetTypeSubscription,
		CycleTypeSnapshot:    SubscriptionResetNever,
		BeforeBudgetSnapshot: "{}",
		AfterBudgetSnapshot:  "{}",
		Status:               entmodel.QuotaAllocationStatusActive,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:                 7,
		UserId:             506,
		Status:             "active",
		Source:             SubscriptionSourceTypeEnterprise,
		SourceType:         SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 90,
		SortOrder:          -100,
		StartTime:          time.Now().Unix(),
		EndTime:            time.Now().Unix() + 86400,
		AmountTotal:        200,
		AmountUsed:         0,
	}).Error)

	require.NoError(t, DB.Migrator().DropTable(&SubscriptionPreConsumeRecord{}))

	_, err := PreConsumeUserSubscription("enterprise-wallet-db-error", 506, "gpt-4o-mini", 0, 50)
	require.Error(t, err)
	require.False(t, strings.Contains(strings.ToLower(err.Error()), "no active subscription"))
}

func TestResetDueSubscriptionsResetsEnterpriseWalletConsumedAmount(t *testing.T) {
	truncateTables(t)
	ensureEnterpriseAllocationTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       505,
		Username: "enterprise-reset-user",
		Password: "pwd",
		AffCode:  "enterprise-reset-aff",
	}).Error)
	cycleStartedAt := time.Now().Add(-2 * time.Minute).Unix()
	require.NoError(t, DB.Create(&entmodel.QuotaAllocation{
		Id:                     89,
		DepartmentBudgetId:     1,
		DepartmentId:           1,
		TargetUserId:           505,
		WalletId:               6,
		ActorId:                1001,
		CommittedQuota:         400,
		BudgetTypeSnapshot:     entmodel.DepartmentBudgetTypeSubscription,
		CycleTypeSnapshot:      SubscriptionResetCustom,
		CycleStartedAtSnapshot: cycleStartedAt,
		CustomSecondsSnapshot:  60,
		BeforeBudgetSnapshot:   "{}",
		AfterBudgetSnapshot:    "{}",
		Status:                 entmodel.QuotaAllocationStatusActive,
	}).Error)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := CreateEnterpriseAllocationSubscriptionTx(
			tx,
			505,
			89,
			400,
			SubscriptionResetCustom,
			cycleStartedAt,
			60,
			0,
		)
		return err
	}))

	var sub UserSubscription
	require.NoError(t, DB.Where("source_allocation_id = ?", 89).First(&sub).Error)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Updates(map[string]any{
		"amount_used":     int64(180),
		"last_reset_time": cycleStartedAt,
		"next_reset_time": time.Now().Add(-time.Minute).Unix(),
	}).Error)

	resetCount, err := ResetDueSubscriptions(10)
	require.NoError(t, err)
	require.Equal(t, 1, resetCount)

	require.NoError(t, DB.Where("id = ?", sub.Id).First(&sub).Error)
	require.Equal(t, int64(0), sub.AmountUsed)
	require.Greater(t, sub.NextResetTime, time.Now().Unix())
	require.Equal(t, int64(400), sub.AmountTotal)
}

func TestEnsureUserSubscriptionTableSQLiteAddsEnterpriseWalletColumns(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Migrator().DropTable(&UserSubscription{}))
	require.NoError(t, DB.Exec(`CREATE TABLE user_subscriptions (
		id integer primary key,
		user_id integer,
		plan_id integer,
		amount_total bigint,
		amount_used bigint,
		start_time bigint,
		end_time bigint,
		status varchar(32),
		source varchar(32),
		last_reset_time bigint,
		next_reset_time bigint,
		upgrade_group varchar(64),
		prev_user_group varchar(64),
		created_at bigint,
		updated_at bigint
	)`).Error)

	require.NoError(t, ensureUserSubscriptionTableSQLite())
	for _, column := range []string{
		"source_type",
		"source_allocation_id",
		"sort_order",
		"is_primary",
	} {
		require.True(t, DB.Migrator().HasColumn(&UserSubscription{}, column), column)
	}

	type legacyRow struct {
		Id                 int
		UserId             int
		PlanId             int
		Status             string
		Source             string
		SourceType         string
		SourceAllocationId int
		SortOrder          int
		IsPrimary          bool
	}
	require.NoError(t, DB.Exec(`INSERT INTO user_subscriptions
		(id, user_id, plan_id, status, source, source_type, source_allocation_id, sort_order, is_primary)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		9, 808, 0, "active", "enterprise_allocation", "enterprise_allocation", 77, -100, 0,
	).Error)

	var row legacyRow
	require.NoError(t, DB.Session(&gorm.Session{}).Table("user_subscriptions").Where("id = ?", 9).First(&row).Error)
	require.Equal(t, "enterprise_allocation", row.SourceType)
	require.Equal(t, 77, row.SourceAllocationId)
	require.Equal(t, -100, row.SortOrder)
	require.False(t, row.IsPrimary)
}
