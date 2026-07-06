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

func TestReorderUserSubscriptionMovesAdjacentWhenSortOrdersTie(t *testing.T) {
	truncateTables(t)
	ensureEnterpriseAllocationTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       507,
		Username: "subscription-reorder-user",
		Password: "pwd",
		AffCode:  "subscription-reorder-aff",
	}).Error)
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&SubscriptionPlan{
		Id:            71,
		Title:         "Small",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   100,
	}).Error)
	require.NoError(t, DB.Create(&SubscriptionPlan{
		Id:            72,
		Title:         "Large",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   500,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:          71,
		UserId:      507,
		PlanId:      71,
		Status:      "active",
		Source:      SubscriptionSourceTypeBalance,
		SourceType:  SubscriptionSourceTypeBalance,
		SortOrder:   100,
		IsPrimary:   true,
		StartTime:   now,
		EndTime:     now + 86400,
		AmountTotal: 100,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:          72,
		UserId:      507,
		PlanId:      72,
		Status:      "active",
		Source:      SubscriptionSourceTypeBalance,
		SourceType:  SubscriptionSourceTypeBalance,
		SortOrder:   100,
		IsPrimary:   true,
		StartTime:   now,
		EndTime:     now + 86400,
		AmountTotal: 500,
	}).Error)

	before, err := GetAllUserSubscriptions(507)
	require.NoError(t, err)
	require.Equal(t, 71, before[0].Subscription.Id)
	require.Equal(t, 72, before[1].Subscription.Id)

	require.NoError(t, ReorderUserSubscription(507, 71, 200))

	after, err := GetAllUserSubscriptions(507)
	require.NoError(t, err)
	require.Equal(t, 72, after[0].Subscription.Id)
	require.Equal(t, 71, after[1].Subscription.Id)
	require.Equal(t, 100, after[0].Subscription.SortOrder)
	require.Equal(t, 101, after[1].Subscription.SortOrder)

	result, err := PreConsumeUserSubscription("subscription-reorder-request-1", 507, "gpt-4o-mini", 0, 300)
	require.NoError(t, err)
	require.Equal(t, 72, result.UserSubscriptionId)
}

func TestAdminReorderUserSubscriptionMovesWithinOwnerList(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       508,
		Username: "subscription-admin-reorder-user",
		Password: "pwd",
		AffCode:  "subscription-admin-reorder-aff",
	}).Error)
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&UserSubscription{
		Id:          81,
		UserId:      508,
		PlanId:      81,
		Status:      "active",
		Source:      SubscriptionSourceTypeAdmin,
		SourceType:  SubscriptionSourceTypeAdmin,
		SortOrder:   100,
		IsPrimary:   true,
		StartTime:   now,
		EndTime:     now + 86400,
		AmountTotal: 100,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:          82,
		UserId:      508,
		PlanId:      82,
		Status:      "active",
		Source:      SubscriptionSourceTypeAdmin,
		SourceType:  SubscriptionSourceTypeAdmin,
		SortOrder:   200,
		IsPrimary:   true,
		StartTime:   now,
		EndTime:     now + 86400,
		AmountTotal: 100,
	}).Error)

	require.NoError(t, AdminReorderUserSubscription(82, 100))

	items, err := GetAllUserSubscriptions(508)
	require.NoError(t, err)
	require.Equal(t, 82, items[0].Subscription.Id)
	require.Equal(t, 81, items[1].Subscription.Id)
	require.Equal(t, 99, items[0].Subscription.SortOrder)
	require.Equal(t, 100, items[1].Subscription.SortOrder)
}

func TestReorderUserSubscriptionMovesAcrossSamePriorityGroupByTargetSortOrder(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       510,
		Username: "subscription-long-reorder-user",
		Password: "pwd",
		AffCode:  "subscription-long-reorder-aff",
	}).Error)
	now := time.Now().Unix()
	for _, sub := range []UserSubscription{
		{
			Id:          101,
			UserId:      510,
			PlanId:      101,
			Status:      "active",
			Source:      SubscriptionSourceTypeAdmin,
			SourceType:  SubscriptionSourceTypeAdmin,
			SortOrder:   100,
			IsPrimary:   true,
			StartTime:   now,
			EndTime:     now + 86400,
			AmountTotal: 100,
		},
		{
			Id:          102,
			UserId:      510,
			PlanId:      102,
			Status:      "active",
			Source:      SubscriptionSourceTypeAdmin,
			SourceType:  SubscriptionSourceTypeAdmin,
			SortOrder:   200,
			IsPrimary:   true,
			StartTime:   now,
			EndTime:     now + 86400,
			AmountTotal: 100,
		},
		{
			Id:          103,
			UserId:      510,
			PlanId:      103,
			Status:      "active",
			Source:      SubscriptionSourceTypeAdmin,
			SourceType:  SubscriptionSourceTypeAdmin,
			SortOrder:   300,
			IsPrimary:   true,
			StartTime:   now,
			EndTime:     now + 86400,
			AmountTotal: 100,
		},
	} {
		require.NoError(t, DB.Create(&sub).Error)
	}

	require.NoError(t, ReorderUserSubscription(510, 101, 300))

	items, err := GetAllUserSubscriptions(510)
	require.NoError(t, err)
	require.Equal(t, 102, items[0].Subscription.Id)
	require.Equal(t, 103, items[1].Subscription.Id)
	require.Equal(t, 101, items[2].Subscription.Id)
	require.Equal(t, 200, items[0].Subscription.SortOrder)
	require.Equal(t, 300, items[1].Subscription.SortOrder)
	require.Equal(t, 301, items[2].Subscription.SortOrder)
}

func TestReorderUserSubscriptionMovesAdjacentWithoutChangingOtherSortOrders(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       511,
		Username: "subscription-adjacent-reorder-user",
		Password: "pwd",
		AffCode:  "subscription-adjacent-reorder-aff",
	}).Error)
	now := time.Now().Unix()
	for _, sub := range []UserSubscription{
		{
			Id:          111,
			UserId:      511,
			PlanId:      111,
			Status:      "active",
			Source:      SubscriptionSourceTypeAdmin,
			SourceType:  SubscriptionSourceTypeAdmin,
			SortOrder:   100,
			IsPrimary:   true,
			StartTime:   now,
			EndTime:     now + 86400,
			AmountTotal: 100,
		},
		{
			Id:          112,
			UserId:      511,
			PlanId:      112,
			Status:      "active",
			Source:      SubscriptionSourceTypeAdmin,
			SourceType:  SubscriptionSourceTypeAdmin,
			SortOrder:   200,
			IsPrimary:   true,
			StartTime:   now,
			EndTime:     now + 86400,
			AmountTotal: 100,
		},
		{
			Id:          113,
			UserId:      511,
			PlanId:      113,
			Status:      "active",
			Source:      SubscriptionSourceTypeAdmin,
			SourceType:  SubscriptionSourceTypeAdmin,
			SortOrder:   300,
			IsPrimary:   true,
			StartTime:   now,
			EndTime:     now + 86400,
			AmountTotal: 100,
		},
	} {
		require.NoError(t, DB.Create(&sub).Error)
	}

	require.NoError(t, ReorderUserSubscription(511, 113, 200))

	items, err := GetAllUserSubscriptions(511)
	require.NoError(t, err)
	require.Equal(t, 111, items[0].Subscription.Id)
	require.Equal(t, 113, items[1].Subscription.Id)
	require.Equal(t, 112, items[2].Subscription.Id)
	require.Equal(t, 100, items[0].Subscription.SortOrder)
	require.Equal(t, 150, items[1].Subscription.SortOrder)
	require.Equal(t, 200, items[2].Subscription.SortOrder)
}

func TestReorderUserSubscriptionPreservesEnterpriseWalletSortOrder(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       509,
		Username: "subscription-enterprise-boundary-user",
		Password: "pwd",
		AffCode:  "subscription-enterprise-boundary-aff",
	}).Error)
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&UserSubscription{
		Id:                 91,
		UserId:             509,
		Status:             "active",
		Source:             SubscriptionSourceTypeEnterprise,
		SourceType:         SubscriptionSourceTypeEnterprise,
		SourceAllocationId: 91,
		SortOrder:          -100,
		IsPrimary:          false,
		StartTime:          now,
		EndTime:            now + 86400,
		AmountTotal:        100,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:          92,
		UserId:      509,
		PlanId:      92,
		Status:      "active",
		Source:      SubscriptionSourceTypeAdmin,
		SourceType:  SubscriptionSourceTypeAdmin,
		SortOrder:   100,
		IsPrimary:   true,
		StartTime:   now,
		EndTime:     now + 86400,
		AmountTotal: 100,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:          93,
		UserId:      509,
		PlanId:      93,
		Status:      "active",
		Source:      SubscriptionSourceTypeAdmin,
		SourceType:  SubscriptionSourceTypeAdmin,
		SortOrder:   200,
		IsPrimary:   true,
		StartTime:   now,
		EndTime:     now + 86400,
		AmountTotal: 100,
	}).Error)

	require.NoError(t, ReorderUserSubscription(509, 92, 200))

	items, err := GetAllUserSubscriptions(509)
	require.NoError(t, err)
	require.Equal(t, 91, items[0].Subscription.Id)
	require.Equal(t, -100, items[0].Subscription.SortOrder)
	require.Equal(t, 93, items[1].Subscription.Id)
	require.Equal(t, 200, items[1].Subscription.SortOrder)
	require.Equal(t, 92, items[2].Subscription.Id)
	require.Equal(t, 201, items[2].Subscription.SortOrder)
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

func TestAdminInvalidateUserSubscriptionRejectsSourceOnlyEnterpriseWallet(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       520,
		Username: "enterprise-invalidate-source-user",
		Password: "pwd",
		AffCode:  "enterprise-invalidate-source-aff",
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:                 30,
		UserId:             520,
		Status:             "active",
		Source:             SubscriptionSourceTypeEnterprise,
		SourceType:         "",
		SourceAllocationId: 430,
		SortOrder:          -100,
		StartTime:          time.Now().Unix(),
		EndTime:            time.Now().Unix() + 86400,
	}).Error)

	_, err := AdminInvalidateUserSubscription(30)
	require.ErrorIs(t, err, ErrEnterpriseSubscriptionInvalid)
}

func TestAdminDeleteUserSubscriptionRejectsSourceOnlyEnterpriseWallet(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       521,
		Username: "enterprise-delete-source-user",
		Password: "pwd",
		AffCode:  "enterprise-delete-source-aff",
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:                 31,
		UserId:             521,
		Status:             "active",
		Source:             SubscriptionSourceTypeEnterprise,
		SourceType:         "",
		SourceAllocationId: 431,
		SortOrder:          -100,
		StartTime:          time.Now().Unix(),
		EndTime:            time.Now().Unix() + 86400,
	}).Error)

	_, err := AdminDeleteUserSubscription(31)
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
