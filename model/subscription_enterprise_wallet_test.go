package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

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
