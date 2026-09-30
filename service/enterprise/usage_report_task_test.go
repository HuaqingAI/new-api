package enterprise

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUsageReportServiceTest(t *testing.T, now time.Time, sender usageReportEmailSender) (*UsageReportService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))
	require.NoError(t, entmodel.Migrate(db))
	return NewUsageReportServiceForTest(db, func() time.Time { return now }, sender), db
}

func TestUsageReportServiceSaveConfigValidatesInput(t *testing.T) {
	service, _ := setupUsageReportServiceTest(t, time.Unix(1717117200, 0), nil)

	_, err := service.SaveConfig(UsageReportConfigInput{
		TenantId:  0,
		Receivers: []string{"bad-email"},
		Frequency: entmodel.UsageReportFrequencyDaily,
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   boolPtr(true),
	})
	require.ErrorIs(t, err, ErrUsageReportInvalidEmail)

	_, err = service.SaveConfig(UsageReportConfigInput{
		TenantId:  0,
		Receivers: []string{"ops@example.com"},
		Frequency: "hourly",
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   boolPtr(true),
	})
	require.ErrorIs(t, err, ErrUsageReportInvalidInput)
}

func TestUsageReportServiceSaveConfigAndGetConfig(t *testing.T) {
	now := time.Unix(1717117200, 0)
	service, _ := setupUsageReportServiceTest(t, now, nil)

	saved, err := service.SaveConfig(UsageReportConfigInput{
		TenantId:  0,
		Receivers: []string{"ops@example.com", "cto@example.com"},
		Frequency: entmodel.UsageReportFrequencyWeekly,
		RangeType: entmodel.UsageReportRangeLast30Days,
		Enabled:   boolPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"ops@example.com", "cto@example.com"}, saved.Receivers)
	require.Equal(t, entmodel.UsageReportFrequencyWeekly, saved.Frequency)
	require.Equal(t, entmodel.UsageReportRangeLast30Days, saved.RangeType)
	require.True(t, saved.NextRunAt > now.Unix())

	got, err := service.GetConfig(0, nil)
	require.NoError(t, err)
	require.Equal(t, saved.Receivers, got.Receivers)
	require.Equal(t, saved.Frequency, got.Frequency)
}

func TestUsageReportServiceSaveConfigKeepsIndependentDepartmentScopes(t *testing.T) {
	now := time.Unix(1717117200, 0)
	service, _ := setupUsageReportServiceTest(t, now, nil)

	departmentId := 9
	tenantSaved, err := service.SaveConfig(UsageReportConfigInput{
		TenantId:  0,
		Receivers: []string{"tenant@example.com"},
		Frequency: entmodel.UsageReportFrequencyDaily,
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   boolPtr(true),
	})
	require.NoError(t, err)
	departmentSaved, err := service.SaveConfig(UsageReportConfigInput{
		TenantId:           0,
		DepartmentId:       &departmentId,
		IncludeDescendants: true,
		Receivers:          []string{"dept@example.com"},
		Frequency:          entmodel.UsageReportFrequencyMonthly,
		RangeType:          entmodel.UsageReportRangeLast30Days,
		Enabled:            boolPtr(true),
	})
	require.NoError(t, err)

	tenantGot, err := service.GetConfig(0, nil)
	require.NoError(t, err)
	departmentGot, err := service.GetConfig(0, &departmentId)
	require.NoError(t, err)
	require.Equal(t, tenantSaved.Id, tenantGot.Id)
	require.Equal(t, departmentSaved.Id, departmentGot.Id)
	require.Nil(t, tenantGot.DepartmentId)
	require.NotNil(t, departmentGot.DepartmentId)
	require.Equal(t, departmentId, *departmentGot.DepartmentId)
	require.Equal(t, []string{"tenant@example.com"}, tenantGot.Receivers)
	require.Equal(t, []string{"dept@example.com"}, departmentGot.Receivers)
	require.True(t, departmentGot.IncludeDescendants)
}

func TestUsageReportServiceSaveConfigPreservesPendingDueTimeForEnabledJobs(t *testing.T) {
	now := time.Unix(1717117200, 0)
	service, db := setupUsageReportServiceTest(t, now, nil)

	job := entmodel.UsageReportJob{
		TenantId:  0,
		Frequency: entmodel.UsageReportFrequencyDaily,
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   true,
		Status:    entmodel.UsageReportStatusPending,
		NextRunAt: now.Unix() + 300,
	}
	require.NoError(t, job.SetReceivers([]string{"ops@example.com"}))
	require.NoError(t, job.SetLastSnapshot(nil))
	require.NoError(t, db.Create(&job).Error)

	saved, err := service.SaveConfig(UsageReportConfigInput{
		TenantId:  0,
		Receivers: []string{"ops@example.com", "cto@example.com"},
		Frequency: entmodel.UsageReportFrequencyDaily,
		RangeType: entmodel.UsageReportRangeLast30Days,
		Enabled:   boolPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, now.Unix()+300, saved.NextRunAt)
	require.Equal(t, []string{"ops@example.com", "cto@example.com"}, saved.Receivers)
	require.Equal(t, entmodel.UsageReportRangeLast30Days, saved.RangeType)
}

func boolPtr(value bool) *bool {
	return &value
}

func TestComputeUsageReportNextRunAtUsesFixedNineAMSchedule(t *testing.T) {
	mondayBeforeNine := time.Date(2026, 6, 1, 8, 30, 0, 0, time.Local)
	mondayAfterNine := time.Date(2026, 6, 1, 9, 30, 0, 0, time.Local)
	wednesday := time.Date(2026, 6, 3, 10, 0, 0, 0, time.Local)
	firstBeforeNine := time.Date(2026, 6, 1, 8, 30, 0, 0, time.Local)
	firstAfterNine := time.Date(2026, 6, 1, 9, 30, 0, 0, time.Local)

	require.Equal(t, time.Date(2026, 6, 1, 9, 0, 0, 0, time.Local).Unix(), computeUsageReportNextRunAt(entmodel.UsageReportFrequencyDaily, mondayBeforeNine.Unix()))
	require.Equal(t, time.Date(2026, 6, 2, 9, 0, 0, 0, time.Local).Unix(), computeUsageReportNextRunAt(entmodel.UsageReportFrequencyDaily, mondayAfterNine.Unix()))
	require.Equal(t, time.Date(2026, 6, 1, 9, 0, 0, 0, time.Local).Unix(), computeUsageReportNextRunAt(entmodel.UsageReportFrequencyWeekly, mondayBeforeNine.Unix()))
	require.Equal(t, time.Date(2026, 6, 8, 9, 0, 0, 0, time.Local).Unix(), computeUsageReportNextRunAt(entmodel.UsageReportFrequencyWeekly, mondayAfterNine.Unix()))
	require.Equal(t, time.Date(2026, 6, 8, 9, 0, 0, 0, time.Local).Unix(), computeUsageReportNextRunAt(entmodel.UsageReportFrequencyWeekly, wednesday.Unix()))
	require.Equal(t, time.Date(2026, 6, 1, 9, 0, 0, 0, time.Local).Unix(), computeUsageReportNextRunAt(entmodel.UsageReportFrequencyMonthly, firstBeforeNine.Unix()))
	require.Equal(t, time.Date(2026, 7, 1, 9, 0, 0, 0, time.Local).Unix(), computeUsageReportNextRunAt(entmodel.UsageReportFrequencyMonthly, firstAfterNine.Unix()))
}

func TestUsageReportWindowUsesCompletedPreviousDays(t *testing.T) {
	now := time.Date(2026, 9, 23, 9, 0, 0, 0, time.Local)
	start, end := usageReportWindow(entmodel.UsageReportRangeToday, now.Unix())
	require.Equal(t, time.Date(2026, 9, 22, 0, 0, 0, 0, time.Local).Unix(), start)
	require.Equal(t, time.Date(2026, 9, 23, 0, 0, 0, 0, time.Local).Unix(), end)

	start, end = usageReportWindow(entmodel.UsageReportRangeLast7Days, now.Unix())
	require.Equal(t, time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local).Unix(), start)
	require.Equal(t, time.Date(2026, 9, 23, 0, 0, 0, 0, time.Local).Unix(), end)

	start, end = usageReportWindow(entmodel.UsageReportRangeLast30Days, now.Unix())
	require.Equal(t, time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local).Unix(), start)
	require.Equal(t, time.Date(2026, 9, 23, 0, 0, 0, 0, time.Local).Unix(), end)
}

func TestUsageReportEmailCopyUsesScopeNameAndReadableRange(t *testing.T) {
	windowStart := time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local).Unix()
	windowEnd := time.Date(2026, 9, 23, 0, 0, 0, 0, time.Local).Unix()

	subject := buildUsageReportSubject("信息系统中心", entmodel.UsageReportRangeLast7Days, windowStart, windowEnd)
	require.Equal(t, "部门用量报告 信息系统中心 最近7天 (2026-09-16 ~ 2026-09-22)", subject)
	require.NotContains(t, subject, "last7d")

	departmentId := 14
	job := entmodel.UsageReportJob{
		DepartmentId:       &departmentId,
		IncludeDescendants: true,
		RangeType:          entmodel.UsageReportRangeLast7Days,
	}
	content := buildUsageReportEmailHTML(&job, windowStart, windowEnd, "部门用量-信息系统中心-20260916-20260922.csv", "信息系统中心")
	require.Contains(t, content, "范围：信息系统中心及其子组织")
	require.Contains(t, content, "报告范围：最近7天")
	require.NotContains(t, content, "组织 ID")
	require.NotContains(t, content, "last7d")
}

func TestBuildUsageReportSnapshotDetectsGrowth(t *testing.T) {
	deptID := 9
	current := []UsageDepartmentSummaryItem{
		{
			DeptId:       &deptID,
			DeptName:     "Engineering",
			RequestCount: 15,
			Quota:        500,
			UserCount:    3,
		},
	}
	previous := []UsageDepartmentSummaryItem{
		{
			DeptId:       &deptID,
			DeptName:     "Engineering",
			RequestCount: 5,
			Quota:        100,
			UserCount:    2,
		},
	}
	snapshot := buildUsageReportSnapshot(current, previous, 100, 200, 0, 100)
	require.NotNil(t, snapshot)
	require.Len(t, snapshot.TopDepartments, 1)
	require.Len(t, snapshot.GrowthDepartments, 1)
	require.InDelta(t, 2.0, snapshot.GrowthDepartments[0].RequestGrowthRate, 0.001)
	require.InDelta(t, 4.0, snapshot.GrowthDepartments[0].QuotaGrowthRate, 0.001)
}

func TestBuildUsageReportSnapshotSkipsGrowthWithoutPreviousWindowData(t *testing.T) {
	deptID := 9
	current := []UsageDepartmentSummaryItem{
		{
			DeptId:       &deptID,
			DeptName:     "Engineering",
			RequestCount: 15,
			Quota:        500,
			UserCount:    3,
		},
	}

	snapshot := buildUsageReportSnapshot(current, nil, 100, 200, 0, 100)
	require.NotNil(t, snapshot)
	require.Len(t, snapshot.TopDepartments, 1)
	require.Empty(t, snapshot.GrowthDepartments)
}

func TestRunDueReportsMarksSuccessAndFailureWithoutBlocking(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.Local)
	sent := 0
	service, db := setupUsageReportServiceTest(t, now, func(subject string, receiver string, content string, attachments []common.EmailAttachment) error {
		sent++
		require.Contains(t, receiver, "ops@example.com")
		require.Contains(t, content, "CSV 数据见附件")
		require.Len(t, attachments, 1)
		require.Contains(t, string(attachments[0].Content), "输入 Tokens")
		require.Contains(t, string(attachments[0].Content), "alice")
		return nil
	})

	job := entmodel.UsageReportJob{
		TenantId:  0,
		Frequency: entmodel.UsageReportFrequencyDaily,
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   true,
		Status:    entmodel.UsageReportStatusPending,
		NextRunAt: now.Unix() - 1,
	}
	require.NoError(t, job.SetReceivers([]string{"ops@example.com"}))
	require.NoError(t, job.SetLastSnapshot(nil))
	require.NoError(t, db.Create(&job).Error)

	deptID := 1
	windowStart, windowEnd := usageReportWindow(entmodel.UsageReportRangeLast7Days, now.Unix())
	snapshot := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           &deptID,
		DeptName:         "Engineering",
		WindowStart:      windowStart,
		WindowEnd:        windowEnd,
		RequestCount:     12,
		PromptTokens:     120,
		CompletionTokens: 60,
		Quota:            300,
	}
	require.NoError(t, snapshot.SetModelDistribution(nil))
	require.NoError(t, snapshot.SetUserIds([]int{101, 102}))
	require.NoError(t, db.Create(&snapshot).Error)
	createUsageReportLogFixture(t, db, windowStart, 101, "alice", 300)

	result, err := service.RunDueReports(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	require.Equal(t, 0, result.Failed)
	require.Equal(t, 1, sent)

	var saved entmodel.UsageReportJob
	require.NoError(t, db.First(&saved, job.Id).Error)
	require.Equal(t, entmodel.UsageReportStatusSuccess, saved.Status)
	require.NotZero(t, saved.LastSuccessAt)

	failingService, failingDB := setupUsageReportServiceTest(t, now, func(subject string, receiver string, content string, attachments []common.EmailAttachment) error {
		return errors.New("smtp down")
	})
	failingJob := entmodel.UsageReportJob{
		TenantId:  0,
		Frequency: entmodel.UsageReportFrequencyDaily,
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   true,
		Status:    entmodel.UsageReportStatusPending,
		NextRunAt: now.Unix() - 1,
	}
	require.NoError(t, failingJob.SetReceivers([]string{"ops@example.com"}))
	require.NoError(t, failingJob.SetLastSnapshot(nil))
	require.NoError(t, failingDB.Create(&failingJob).Error)
	require.NoError(t, failingDB.Create(&snapshot).Error)
	createUsageReportLogFixture(t, failingDB, windowStart, 101, "alice", 300)

	failingResult, err := failingService.RunDueReports(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, failingResult.Processed)
	require.Equal(t, 1, failingResult.Failed)
	require.NoError(t, failingDB.First(&saved, failingJob.Id).Error)
	require.Equal(t, entmodel.UsageReportStatusFailed, saved.Status)
	require.Contains(t, saved.ErrorReason, "smtp down")
}

func TestUsageReportServiceSendNowUsesSavedConfigWithoutAdvancingSchedule(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local)
	nextRunAt := now.Add(12 * time.Hour).Unix()
	sent := 0
	service, db := setupUsageReportServiceTest(t, now, func(subject string, receiver string, content string, attachments []common.EmailAttachment) error {
		sent++
		require.Contains(t, subject, "部门用量报告")
		require.Equal(t, "ops@example.com", receiver)
		require.Contains(t, content, "CSV 数据见附件")
		require.Len(t, attachments, 1)
		require.Contains(t, string(attachments[0].Content), "manual_user")
		require.Contains(t, string(attachments[0].Content), "输入 Tokens")
		return nil
	})

	job := entmodel.UsageReportJob{
		TenantId:  0,
		Frequency: entmodel.UsageReportFrequencyWeekly,
		RangeType: entmodel.UsageReportRangeToday,
		Enabled:   false,
		Status:    entmodel.UsageReportStatusPending,
		NextRunAt: nextRunAt,
	}
	require.NoError(t, job.SetReceivers([]string{"ops@example.com"}))
	require.NoError(t, job.SetLastSnapshot(nil))
	require.NoError(t, db.Create(&job).Error)
	require.NoError(t, db.Model(&job).Update("enabled", false).Error)

	windowStart, windowEnd := usageReportWindow(entmodel.UsageReportRangeToday, now.Unix())
	createUsageReportLogFixture(t, db, windowStart, 101, "manual_user", 420)

	result, err := service.SendNow(context.Background(), 0, nil)
	require.NoError(t, err)
	require.Equal(t, job.Id, result.Id)
	require.Equal(t, entmodel.UsageReportStatusSuccess, result.Status)
	require.Equal(t, windowStart, result.LastWindowStart)
	require.Equal(t, windowEnd, result.LastWindowEnd)
	require.Equal(t, nextRunAt, result.NextRunAt)
	require.Equal(t, 1, sent)

	var saved entmodel.UsageReportJob
	require.NoError(t, db.First(&saved, job.Id).Error)
	require.Equal(t, entmodel.UsageReportStatusSuccess, saved.Status)
	require.Equal(t, nextRunAt, saved.NextRunAt)
	require.False(t, saved.Enabled)
	require.Equal(t, int64(1), saved.RunCount)
	require.Zero(t, saved.FailureCount)
}

func TestUsageReportServiceSendNowRequiresSavedConfig(t *testing.T) {
	service, _ := setupUsageReportServiceTest(t, time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local), nil)

	_, err := service.SendNow(context.Background(), 0, nil)
	require.ErrorIs(t, err, ErrUsageReportNotConfigured)
}

func TestRunDueReportsContinuesAcrossJobsAndJoinsMultipleReceivers(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.Local)
	received := make([]string, 0, 2)
	service, db := setupUsageReportServiceTest(t, now, func(subject string, receiver string, content string, attachments []common.EmailAttachment) error {
		received = append(received, receiver)
		if receiver == "fail@example.com" {
			return errors.New("smtp down")
		}
		require.Equal(t, "ops@example.com;cto@example.com", receiver)
		require.Contains(t, subject, "部门用量报告")
		require.Contains(t, content, "CSV 数据见附件")
		require.Len(t, attachments, 1)
		return nil
	})

	successJob := entmodel.UsageReportJob{
		TenantId:  0,
		Frequency: entmodel.UsageReportFrequencyDaily,
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   true,
		Status:    entmodel.UsageReportStatusPending,
		NextRunAt: now.Unix() - 1,
	}
	require.NoError(t, successJob.SetReceivers([]string{"ops@example.com", "cto@example.com"}))
	require.NoError(t, successJob.SetLastSnapshot(nil))
	require.NoError(t, db.Create(&successJob).Error)

	failureJob := entmodel.UsageReportJob{
		TenantId:  1,
		Frequency: entmodel.UsageReportFrequencyDaily,
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   true,
		Status:    entmodel.UsageReportStatusPending,
		NextRunAt: now.Unix() - 1,
	}
	require.NoError(t, failureJob.SetReceivers([]string{"fail@example.com"}))
	require.NoError(t, failureJob.SetLastSnapshot(nil))
	require.NoError(t, db.Create(&failureJob).Error)

	windowStart, windowEnd := usageReportWindow(entmodel.UsageReportRangeLast7Days, now.Unix())
	for _, tenantID := range []int{0, 1} {
		deptID := tenantID + 1
		snapshot := entmodel.UsageSnapshot{
			TenantId:         tenantID,
			DeptId:           &deptID,
			DeptName:         "Engineering",
			WindowStart:      windowStart,
			WindowEnd:        windowEnd,
			RequestCount:     12,
			PromptTokens:     120,
			CompletionTokens: 60,
			Quota:            300,
		}
		require.NoError(t, snapshot.SetModelDistribution(nil))
		require.NoError(t, snapshot.SetUserIds([]int{101, 102}))
		require.NoError(t, db.Create(&snapshot).Error)
		createUsageReportLogFixture(t, db, windowStart, 101+tenantID, "user"+string(rune('a'+tenantID)), 300)
	}

	result, err := service.RunDueReports(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, []string{"ops@example.com;cto@example.com", "fail@example.com"}, received)

	var savedSuccess entmodel.UsageReportJob
	require.NoError(t, db.First(&savedSuccess, successJob.Id).Error)
	require.Equal(t, entmodel.UsageReportStatusSuccess, savedSuccess.Status)
	require.Zero(t, savedSuccess.FailureCount)

	var savedFailure entmodel.UsageReportJob
	require.NoError(t, db.First(&savedFailure, failureJob.Id).Error)
	require.Equal(t, entmodel.UsageReportStatusFailed, savedFailure.Status)
	require.Equal(t, int64(1), savedFailure.FailureCount)
	require.Contains(t, savedFailure.ErrorReason, "smtp down")
}

func createUsageReportLogFixture(t *testing.T, db *gorm.DB, createdAt int64, userId int, username string, quota int) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:          userId,
		Username:    username,
		DisplayName: username,
		Password:    "password",
		AffCode:     username,
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		UserId:           userId,
		Username:         username,
		Type:             model.LogTypeConsume,
		ModelName:        "gpt-4o",
		Quota:            quota,
		PromptTokens:     120,
		CompletionTokens: 60,
		CreatedAt:        createdAt + 60,
	}).Error)
}
