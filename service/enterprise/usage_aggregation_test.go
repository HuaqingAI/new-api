package enterprise

import (
	"bytes"
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
		Sort:     DefaultUsageSummarySort(),
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
		Sort:     DefaultUsageSummarySort(),
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
		Sort:     DefaultUsageSummarySort(),
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

func TestUsageExportBuildsCSVFromSummaryAndParentMappings(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	parentID := 100
	deptID := 101
	seedUsageTestDepartment(t, db, parentID, "Platform")
	require.NoError(t, db.Create(&entmodel.Department{
		Id:          deptID,
		TenantId:    0,
		Name:        "Engineering",
		ParentId:    &parentID,
		Status:      constant.DepartmentStatusEnabled,
		SourceType:  constant.DepartmentSourceTypeManual,
		SyncStatus:  constant.DepartmentSyncStatusOK,
		NameHistory: "[]",
	}).Error)

	snapshotA := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &deptID,
		DeptName:         "Engineering",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     5,
		PromptTokens:     60,
		CompletionTokens: 30,
		Quota:            150,
	}
	require.NoError(t, snapshotA.SetModelDistribution(nil))
	require.NoError(t, snapshotA.SetUserIds([]int{101, 102}))
	snapshotB := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           nil,
		DeptName:         "",
		WindowStart:      1714521600,
		WindowEnd:        1714608000,
		RequestCount:     3,
		PromptTokens:     30,
		CompletionTokens: 10,
		Quota:            50,
	}
	require.NoError(t, snapshotB.SetModelDistribution(nil))
	require.NoError(t, snapshotB.SetUserIds([]int{999}))
	require.NoError(t, db.Create(&snapshotA).Error)
	require.NoError(t, db.Create(&snapshotB).Error)

	service := NewUsageExportService(db)
	result, err := service.ExportDepartmentUsageCSV(DepartmentUsageExportQuery{
		TenantId: 0,
		From:     1714521600,
		To:       1714608000,
		Sort: UsageSummarySort{
			Field: UsageSummarySortByRequests,
			Order: UsageSortOrderDesc,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "usage-department-20240501-20240501.csv", result.FileName)
	require.Len(t, result.Rows, 2)
	require.Equal(t, "Engineering", result.Rows[0].DeptName)
	require.Equal(t, "Platform", result.Rows[0].ParentDepartment)
	require.Equal(t, "未归属", result.Rows[1].DeptName)
	require.Equal(t, "", result.Rows[1].ParentDepartment)

	var buffer bytes.Buffer
	require.NoError(t, service.WriteDepartmentUsageCSV(&buffer, result))
	csvText := buffer.String()
	require.Contains(t, csvText, "# 注意：用量按用户当前所属部门重复计入，部门间数值不可加和")
	require.Contains(t, csvText, "部门 ID,部门名称,父部门,周期开始,周期结束,请求数,prompt_tokens,completion_tokens,quota,用户数")
	require.Contains(t, csvText, "101,Engineering,Platform,1714521600,1714608000,5,60,30,150,2")
	require.Contains(t, csvText, ",未归属,,1714521600,1714608000,3,30,10,50,1")
	require.NotContains(t, csvText, "null")
	require.NotContains(t, csvText, "<nil>")

	lines := strings.Split(strings.TrimSpace(csvText), "\n")
	require.GreaterOrEqual(t, len(lines), 4)
	require.Equal(t, "# 注意：用量按用户当前所属部门重复计入，部门间数值不可加和", lines[0])
	require.Equal(t, "部门 ID,部门名称,父部门,周期开始,周期结束,请求数,prompt_tokens,completion_tokens,quota,用户数", lines[1])
	require.Equal(t, "101,Engineering,Platform,1714521600,1714608000,5,60,30,150,2", lines[2])
	require.Equal(t, ",未归属,,1714521600,1714608000,3,30,10,50,1", lines[3])
}

func TestUsageExportSortsDepartmentNamesWithStableTieBreakers(t *testing.T) {
	items := []UsageDepartmentSummaryItem{
		{
			DeptId:       intPtr(5),
			DeptName:     "Alpha",
			RequestCount: 10,
			Quota:        100,
			UserCount:    2,
		},
		{
			DeptId:       intPtr(3),
			DeptName:     "Alpha",
			RequestCount: 10,
			Quota:        100,
			UserCount:    2,
		},
		{
			DeptId:       nil,
			DeptName:     "",
			RequestCount: 10,
			Quota:        100,
			UserCount:    2,
		},
	}

	sorted := SortUsageDepartmentSummaryItems(items, UsageSummarySort{
		Field: UsageSummarySortByName,
		Order: UsageSortOrderAsc,
	})

	require.Len(t, sorted, 3)
	require.NotNil(t, sorted[0].DeptId)
	require.Equal(t, 3, *sorted[0].DeptId)
	require.NotNil(t, sorted[1].DeptId)
	require.Equal(t, 5, *sorted[1].DeptId)
	require.Nil(t, sorted[2].DeptId)
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
		Sort:     DefaultUsageSummarySort(),
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
		Sort:     DefaultUsageSummarySort(),
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
		Sort:     DefaultUsageSummarySort(),
	})
	require.NoError(t, err)
	require.Len(t, rows.Items, 1)
	require.Equal(t, int64(2), rows.Items[0].UserCount)
	require.Equal(t, int64(3), rows.Items[0].RequestCount)
}

func TestUsageDetailBuildsRankingTrendAndAllowsMultiDepartmentDuplication(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	seedUsageTestUser(t, db, 101, "alice")
	seedUsageTestUser(t, db, 102, "bob")
	seedUsageTestDepartment(t, db, 1, "Engineering")
	seedUsageTestDepartment(t, db, 2, "Operations")

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
		{
			TenantId:       0,
			UserId:         102,
			DepartmentId:   1,
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
			Quota:            100,
			PromptTokens:     40,
			CompletionTokens: 20,
			CreatedAt:        1700000100,
		},
		{
			Id:               2,
			UserId:           101,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			ModelName:        "claude-sonnet-4",
			Quota:            200,
			PromptTokens:     80,
			CompletionTokens: 40,
			CreatedAt:        1700003900,
		},
		{
			Id:               3,
			UserId:           102,
			Username:         "bob",
			Type:             model.LogTypeConsume,
			ModelName:        "gpt-4o",
			Quota:            50,
			PromptTokens:     20,
			CompletionTokens: 10,
			CreatedAt:        1700000200,
		},
	}).Error)

	service := NewUsageAggregationService(db)
	_, err := service.AggregateWindow(UsageAggregationWindow{
		TenantId:    0,
		WindowStart: 1700000000,
		WindowEnd:   1700003600,
	})
	require.NoError(t, err)
	_, err = service.AggregateWindow(UsageAggregationWindow{
		TenantId:    0,
		WindowStart: 1700003600,
		WindowEnd:   1700007200,
	})
	require.NoError(t, err)

	deptId := 1
	detail, err := service.GetDepartmentDetail(UsageDetailQuery{
		TenantId: 0,
		DeptId:   &deptId,
		From:     1700000000,
		To:       1700007200,
	})
	require.NoError(t, err)
	require.Equal(t, "Engineering", detail.DeptName)
	require.Equal(t, int64(3), detail.RequestCount)
	require.Equal(t, int64(140), detail.PromptTokens)
	require.Equal(t, int64(70), detail.CompletionTokens)
	require.Equal(t, int64(210), detail.TokenCount)
	require.Equal(t, int64(350), detail.Quota)
	require.Equal(t, int64(2), detail.UserCount)
	require.Len(t, detail.UserRanking, 2)
	require.Equal(t, "alice", detail.UserRanking[0].Username)
	require.Equal(t, int64(300), detail.UserRanking[0].Quota)
	require.Equal(t, int64(2), detail.UserRanking[0].RequestCount)
	require.Equal(t, "bob", detail.UserRanking[1].Username)
	require.Equal(t, int64(50), detail.UserRanking[1].Quota)
	require.Equal(t, int64(30), detail.UserRanking[1].TokenCount)
	require.Len(t, detail.ModelDistribution, 2)
	require.Equal(t, "claude-sonnet-4", detail.ModelDistribution[0].ModelName)
	require.Equal(t, int64(200), detail.ModelDistribution[0].Quota)
	require.Equal(t, "gpt-4o", detail.ModelDistribution[1].ModelName)
	require.Len(t, detail.Trend, 2)
	require.Equal(t, int64(1700000000), detail.Trend[0].WindowStart)
	require.Equal(t, int64(2), detail.Trend[0].RequestCount)
	require.Equal(t, int64(1700003600), detail.Trend[1].WindowStart)
	require.Equal(t, int64(1), detail.Trend[1].RequestCount)
	require.Equal(t, "/usage-logs/common", detail.RecentLogsLink.Path)
	require.Equal(t, "common", detail.RecentLogsLink.Section)
	require.Equal(t, "Engineering", detail.RecentLogsLink.DepartmentName)
	require.Equal(t, int64(1700000000), detail.RecentLogsLink.StartTimestamp)
	require.Equal(t, int64(1700007199), detail.RecentLogsLink.EndTimestamp)
	require.Equal(t, []string{"alice", "bob"}, detail.RecentLogsLink.Usernames)

	otherDeptId := 2
	otherDetail, err := service.GetDepartmentDetail(UsageDetailQuery{
		TenantId: 0,
		DeptId:   &otherDeptId,
		From:     1700000000,
		To:       1700007200,
	})
	require.NoError(t, err)
	require.Equal(t, "Operations", otherDetail.DeptName)
	require.Equal(t, int64(2), otherDetail.RequestCount)
	require.Equal(t, int64(1), otherDetail.UserCount)
	require.Len(t, otherDetail.UserRanking, 1)
	require.Equal(t, "alice", otherDetail.UserRanking[0].Username)
}

func TestUsageDetailReturnsEmptyArraysForAdjacentWindow(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	deptId := 1

	detail, err := NewUsageAggregationService(db).GetDepartmentDetail(UsageDetailQuery{
		TenantId: 0,
		DeptId:   &deptId,
		From:     1700000000,
		To:       1700003600,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1700000000), detail.WindowStart)
	require.Equal(t, int64(1700003600), detail.WindowEnd)
	require.NotNil(t, detail.UserRanking)
	require.Empty(t, detail.UserRanking)
	require.NotNil(t, detail.ModelDistribution)
	require.Empty(t, detail.ModelDistribution)
	require.NotNil(t, detail.Trend)
	require.Empty(t, detail.Trend)
	require.Equal(t, "/usage-logs/common", detail.RecentLogsLink.Path)
	require.Equal(t, "common", detail.RecentLogsLink.Section)
	require.Equal(t, deptId, *detail.RecentLogsLink.DepartmentId)
	require.Equal(t, int64(1700003599), detail.RecentLogsLink.EndTimestamp)
	require.NotNil(t, detail.RecentLogsLink.Usernames)
	require.Empty(t, detail.RecentLogsLink.Usernames)
}

func TestUsageDetailRejectsInvalidQuery(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	deptId := 1

	_, err := NewUsageAggregationService(db).GetDepartmentDetail(UsageDetailQuery{
		TenantId: 0,
		DeptId:   nil,
		From:     1700000000,
		To:       1700003600,
	})
	require.ErrorIs(t, err, ErrInvalidUsageDetailQuery)

	_, err = NewUsageAggregationService(db).GetDepartmentDetail(UsageDetailQuery{
		TenantId: 0,
		DeptId:   &deptId,
		From:     1700003600,
		To:       1700000000,
	})
	require.ErrorIs(t, err, ErrInvalidUsageDetailQuery)
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
