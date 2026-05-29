package enterprise

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUsageAggregationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldOptionMap := common.OptionMap

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.OptionMap = map[string]string{}
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Option{}))
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	require.NoError(t, entmodel.AutoMigrate(db))

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.OptionMap = oldOptionMap
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func seedUsageTestUser(t *testing.T, db *gorm.DB, userId int, username string) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:       userId,
		Username: username,
		Password: "pwd",
		Group:    "default",
		AffCode:  username + "-aff",
	}).Error)
}

func seedUsageTestDepartment(t *testing.T, db *gorm.DB, id int, name string) {
	t.Helper()
	require.NoError(t, db.Create(&entmodel.Department{
		Id:          id,
		TenantId:    0,
		Name:        name,
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)
}

func TestUsageAggregationExpandsLogsAcrossMultipleDepartmentsAndUnassignedBucket(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	seedUsageTestUser(t, db, 101, "alice")
	seedUsageTestUser(t, db, 102, "bob")
	seedUsageTestDepartment(t, db, 1, "Engineering")
	seedUsageTestDepartment(t, db, 2, "Security")

	require.NoError(t, db.Create(&[]entmodel.UserDepartment{
		{
			TenantId:       0,
			UserId:         101,
			DepartmentId:   1,
			ExternalSource: constant.EnterpriseExternalSourceManual,
			Status:         constant.EnterpriseMembershipStatusActive,
			JoinedAt:       0,
			LeftAt:         0,
		},
		{
			TenantId:       0,
			UserId:         101,
			DepartmentId:   2,
			ExternalSource: constant.EnterpriseExternalSourceManual,
			Status:         constant.EnterpriseMembershipStatusActive,
			JoinedAt:       0,
			LeftAt:         0,
		},
	}).Error)
	require.NoError(t, db.Create(&[]model.Log{
		{
			Id:               1,
			UserId:           101,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			ModelName:        "gpt-4o",
			Quota:            90,
			PromptTokens:     30,
			CompletionTokens: 10,
			CreatedAt:        1700000100,
		},
		{
			Id:               2,
			UserId:           102,
			Username:         "bob",
			Type:             model.LogTypeConsume,
			ModelName:        "gpt-4o-mini",
			Quota:            60,
			PromptTokens:     20,
			CompletionTokens: 5,
			CreatedAt:        1700000200,
		},
	}).Error)

	service := NewUsageAggregationService(db)
	processed, err := service.AggregateWindow(UsageAggregationWindow{
		TenantId:    0,
		WindowStart: 1700000000,
		WindowEnd:   1700003600,
	})
	require.NoError(t, err)
	require.Equal(t, 2, processed)

	rows, err := service.GetDepartmentSummary(UsageSummaryQuery{
		TenantId: 0,
		From:     1700000000,
		To:       1700003600,
	})
	require.NoError(t, err)
	require.Len(t, rows.Items, 3)

	rowByName := map[string]UsageDepartmentSummaryItem{}
	for _, item := range rows.Items {
		rowByName[item.DeptName] = item
	}

	require.Equal(t, int64(1), rowByName["Engineering"].RequestCount)
	require.Equal(t, int64(30), rowByName["Engineering"].PromptTokens)
	require.Equal(t, int64(10), rowByName["Engineering"].CompletionTokens)
	require.Equal(t, int64(90), rowByName["Engineering"].Quota)
	require.Equal(t, int64(1), rowByName["Engineering"].UserCount)
	require.Len(t, rowByName["Engineering"].ModelDistribution, 1)

	require.Equal(t, int64(1), rowByName["Security"].RequestCount)
	require.Equal(t, int64(1), rowByName["Security"].UserCount)

	require.Nil(t, rowByName["未归属"].DeptId)
	require.Equal(t, int64(1), rowByName["未归属"].RequestCount)
	require.Equal(t, int64(1), rowByName["未归属"].UserCount)
	require.NotNil(t, rowByName["未归属"].ModelDistribution)
}

func TestUsageAggregationRespectsMembershipEffectiveWindow(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	seedUsageTestUser(t, db, 101, "alice")
	seedUsageTestDepartment(t, db, 1, "Engineering")
	seedUsageTestDepartment(t, db, 2, "Security")

	require.NoError(t, db.Create(&[]entmodel.UserDepartment{
		{
			TenantId:       0,
			UserId:         101,
			DepartmentId:   1,
			ExternalSource: constant.EnterpriseExternalSourceManual,
			Status:         constant.EnterpriseMembershipStatusInactive,
			JoinedAt:       1700000000,
			LeftAt:         1700000500,
		},
		{
			TenantId:       0,
			UserId:         101,
			DepartmentId:   2,
			ExternalSource: constant.EnterpriseExternalSourceManual,
			Status:         constant.EnterpriseMembershipStatusActive,
			JoinedAt:       1700000501,
			LeftAt:         0,
		},
	}).Error)
	require.NoError(t, db.Create(&[]model.Log{
		{
			Id:               1,
			UserId:           101,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			ModelName:        "gpt-4o",
			Quota:            50,
			PromptTokens:     10,
			CompletionTokens: 5,
			CreatedAt:        1700000400,
		},
		{
			Id:               2,
			UserId:           101,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			ModelName:        "gpt-4o-mini",
			Quota:            70,
			PromptTokens:     15,
			CompletionTokens: 6,
			CreatedAt:        1700000600,
		},
	}).Error)

	service := NewUsageAggregationService(db)
	_, err := service.AggregateWindow(UsageAggregationWindow{
		TenantId:    0,
		WindowStart: 1700000000,
		WindowEnd:   1700003600,
	})
	require.NoError(t, err)

	rows, err := service.GetDepartmentSummary(UsageSummaryQuery{
		TenantId: 0,
		From:     1700000000,
		To:       1700003600,
	})
	require.NoError(t, err)
	require.Len(t, rows.Items, 2)

	rowByName := map[string]UsageDepartmentSummaryItem{}
	for _, item := range rows.Items {
		rowByName[item.DeptName] = item
	}
	require.Equal(t, int64(1), rowByName["Engineering"].RequestCount)
	require.Equal(t, int64(50), rowByName["Engineering"].Quota)
	require.Equal(t, int64(1), rowByName["Security"].RequestCount)
	require.Equal(t, int64(70), rowByName["Security"].Quota)
}

func TestUsageAggregationIsIdempotentAndSortsModelDistribution(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	seedUsageTestUser(t, db, 101, "alice")
	seedUsageTestDepartment(t, db, 1, "Engineering")
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         101,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
		JoinedAt:       0,
		LeftAt:         0,
	}).Error)
	require.NoError(t, db.Create(&[]model.Log{
		{
			Id:               1,
			UserId:           101,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			ModelName:        "z-model",
			Quota:            30,
			PromptTokens:     5,
			CompletionTokens: 2,
			CreatedAt:        1700000100,
		},
		{
			Id:               2,
			UserId:           101,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			ModelName:        "a-model",
			Quota:            20,
			PromptTokens:     4,
			CompletionTokens: 1,
			CreatedAt:        1700000200,
		},
	}).Error)

	service := NewUsageAggregationService(db)
	firstProcessed, err := service.AggregateWindow(UsageAggregationWindow{
		TenantId:    0,
		WindowStart: 1700000000,
		WindowEnd:   1700003600,
	})
	require.NoError(t, err)
	require.Equal(t, 2, firstProcessed)

	secondProcessed, err := service.AggregateWindow(UsageAggregationWindow{
		TenantId:    0,
		WindowStart: 1700000000,
		WindowEnd:   1700003600,
	})
	require.NoError(t, err)
	require.Equal(t, 0, secondProcessed)

	rows, err := service.GetDepartmentSummary(UsageSummaryQuery{
		TenantId: 0,
		From:     1700000000,
		To:       1700003600,
	})
	require.NoError(t, err)
	require.Len(t, rows.Items, 1)
	require.Equal(t, int64(2), rows.Items[0].RequestCount)
	require.Equal(t, []string{"a-model", "z-model"}, []string{
		rows.Items[0].ModelDistribution[0].ModelName,
		rows.Items[0].ModelDistribution[1].ModelName,
	})

	var watermark model.Option
	require.NoError(t, db.Where("key = ?", UsageAggregationWatermarkOptionKey(0)).First(&watermark).Error)
	require.Equal(t, "1700003600", watermark.Value)
}

func TestUsageSummaryReturnsEmptyArrayForModelDistribution(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	require.NoError(t, db.Create(&entmodel.UsageSnapshot{
		TenantId:          0,
		DeptId:            nil,
		DeptName:          "",
		WindowStart:       1700000000,
		WindowEnd:         1700003600,
		RequestCount:      0,
		PromptTokens:      0,
		CompletionTokens:  0,
		Quota:             0,
		UserCount:         0,
		ModelDistribution: "",
	}).Error)

	rows, err := NewUsageAggregationService(db).GetDepartmentSummary(UsageSummaryQuery{
		TenantId: 0,
		From:     1700000000,
		To:       1700003600,
	})
	require.NoError(t, err)
	require.Len(t, rows.Items, 1)
	require.Nil(t, rows.Items[0].DeptId)
	require.Equal(t, "未归属", rows.Items[0].DeptName)
	require.NotNil(t, rows.Items[0].ModelDistribution)
	require.Empty(t, rows.Items[0].ModelDistribution)
}

func TestUsageAggregationRecomputesWindowAndRemovesStaleBuckets(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	seedUsageTestUser(t, db, 101, "alice")
	seedUsageTestDepartment(t, db, 1, "Engineering")
	seedUsageTestDepartment(t, db, 2, "Security")

	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         101,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
		JoinedAt:       0,
		LeftAt:         0,
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		Id:               1,
		UserId:           101,
		Username:         "alice",
		Type:             model.LogTypeConsume,
		ModelName:        "gpt-4o",
		Quota:            90,
		PromptTokens:     30,
		CompletionTokens: 10,
		CreatedAt:        1700000100,
	}).Error)

	service := NewUsageAggregationService(db)
	_, err := service.AggregateWindow(UsageAggregationWindow{
		TenantId:    0,
		WindowStart: 1700000000,
		WindowEnd:   1700003600,
	})
	require.NoError(t, err)

	require.NoError(t, db.Model(&entmodel.UserDepartment{}).
		Where("tenant_id = ? AND user_id = ? AND department_id = ?", 0, 101, 1).
		Updates(map[string]any{
			"status":  constant.EnterpriseMembershipStatusInactive,
			"left_at": int64(1700000001),
		}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         101,
		DepartmentId:   2,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
		JoinedAt:       1700000002,
		LeftAt:         0,
	}).Error)
	common.OptionMap[UsageAggregationWatermarkOptionKey(0)] = "1700000000"

	processed, err := service.AggregateWindow(UsageAggregationWindow{
		TenantId:    0,
		WindowStart: 1700000000,
		WindowEnd:   1700003600,
	})
	require.NoError(t, err)
	require.Equal(t, 1, processed)

	rows, err := service.GetDepartmentSummary(UsageSummaryQuery{
		TenantId: 0,
		From:     1700000000,
		To:       1700003600,
	})
	require.NoError(t, err)
	require.Len(t, rows.Items, 1)
	require.Equal(t, "Security", rows.Items[0].DeptName)
	require.Equal(t, int64(1), rows.Items[0].RequestCount)
}

func TestUsageSummaryCountsDistinctUsersAcrossWindows(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	deptId := 1
	snapshotA := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &deptId,
		DeptName:         "Engineering",
		WindowStart:      1700000000,
		WindowEnd:        1700003600,
		RequestCount:     1,
		PromptTokens:     10,
		CompletionTokens: 5,
		Quota:            20,
	}
	require.NoError(t, snapshotA.SetModelDistribution([]entmodel.UsageSnapshotModelStat{{ModelName: "gpt-4o", RequestCount: 1, PromptTokens: 10, CompletionTokens: 5, Quota: 20}}))
	require.NoError(t, snapshotA.SetUserIds([]int{101}))

	snapshotB := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &deptId,
		DeptName:         "Engineering",
		WindowStart:      1700003600,
		WindowEnd:        1700007200,
		RequestCount:     2,
		PromptTokens:     20,
		CompletionTokens: 10,
		Quota:            40,
	}
	require.NoError(t, snapshotB.SetModelDistribution([]entmodel.UsageSnapshotModelStat{{ModelName: "gpt-4o", RequestCount: 2, PromptTokens: 20, CompletionTokens: 10, Quota: 40}}))
	require.NoError(t, snapshotB.SetUserIds([]int{101, 102}))

	require.NoError(t, db.Create(&snapshotA).Error)
	require.NoError(t, db.Create(&snapshotB).Error)

	rows, err := NewUsageAggregationService(db).GetDepartmentSummary(UsageSummaryQuery{
		TenantId: 0,
		From:     1700000000,
		To:       1700007200,
	})
	require.NoError(t, err)
	require.Len(t, rows.Items, 1)
	require.Equal(t, int64(2), rows.Items[0].UserCount)
	require.Equal(t, int64(3), rows.Items[0].RequestCount)
}

func TestRunUsageAggregationTaskOnceStartsFromEarliestMissingWindow(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	seedUsageTestUser(t, db, 101, "alice")
	seedUsageTestDepartment(t, db, 1, "Engineering")
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         101,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
	}).Error)

	now := time.Now().Unix()
	latestWindowEnd := now - (now % usageAggregationWindowSeconds)
	require.GreaterOrEqual(t, latestWindowEnd, usageAggregationWindowSeconds*2)
	oldWindowStart := latestWindowEnd - 2*usageAggregationWindowSeconds

	require.NoError(t, db.Create(&model.Log{
		Id:               1,
		UserId:           101,
		Username:         "alice",
		Type:             model.LogTypeConsume,
		ModelName:        "gpt-4o",
		Quota:            42,
		PromptTokens:     12,
		CompletionTokens: 4,
		CreatedAt:        oldWindowStart + 60,
	}).Error)

	processed, err := RunUsageAggregationTaskOnce(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, processed, 1)

	var snapshots []entmodel.UsageSnapshot
	require.NoError(t, db.Order("window_start ASC").Find(&snapshots).Error)
	require.NotEmpty(t, snapshots)
	require.Equal(t, oldWindowStart, snapshots[0].WindowStart)
}
