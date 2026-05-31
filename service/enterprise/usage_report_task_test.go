package enterprise

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
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
		Enabled:   usageReportBoolPtr(true),
	})
	require.ErrorIs(t, err, ErrUsageReportInvalidEmail)

	_, err = service.SaveConfig(UsageReportConfigInput{
		TenantId:  0,
		Receivers: []string{"ops@example.com"},
		Frequency: "hourly",
		RangeType: entmodel.UsageReportRangeLast7Days,
		Enabled:   usageReportBoolPtr(true),
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
		Enabled:   usageReportBoolPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"ops@example.com", "cto@example.com"}, saved.Receivers)
	require.Equal(t, entmodel.UsageReportFrequencyWeekly, saved.Frequency)
	require.Equal(t, entmodel.UsageReportRangeLast30Days, saved.RangeType)
	require.True(t, saved.NextRunAt > now.Unix())

	got, err := service.GetConfig(0)
	require.NoError(t, err)
	require.Equal(t, saved.Receivers, got.Receivers)
	require.Equal(t, saved.Frequency, got.Frequency)
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
		Enabled:   usageReportBoolPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, now.Unix()+300, saved.NextRunAt)
	require.Equal(t, []string{"ops@example.com", "cto@example.com"}, saved.Receivers)
	require.Equal(t, entmodel.UsageReportRangeLast30Days, saved.RangeType)
}

func usageReportBoolPtr(value bool) *bool {
	return &value
}

func TestUsageReportWindowResolvesExpectedRanges(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 30, 0, 0, time.Local)
	startOfDay := time.Date(2026, 5, 29, 0, 0, 0, 0, time.Local)

	todayStart, todayEnd := usageReportWindow(entmodel.UsageReportRangeToday, now.Unix())
	require.Equal(t, startOfDay.Unix(), todayStart)
	require.Equal(t, startOfDay.Add(24*time.Hour).Unix(), todayEnd)

	last7Start, last7End := usageReportWindow(entmodel.UsageReportRangeLast7Days, now.Unix())
	require.Equal(t, startOfDay.AddDate(0, 0, -6).Unix(), last7Start)
	require.Equal(t, startOfDay.Add(24*time.Hour).Unix(), last7End)

	last30Start, last30End := usageReportWindow(entmodel.UsageReportRangeLast30Days, now.Unix())
	require.Equal(t, startOfDay.AddDate(0, 0, -29).Unix(), last30Start)
	require.Equal(t, startOfDay.Add(24*time.Hour).Unix(), last30End)

	previousStart, previousEnd := previousUsageReportWindow(last7Start, last7End)
	require.Equal(t, last7Start-(last7End-last7Start), previousStart)
	require.Equal(t, last7Start, previousEnd)
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

func TestRunDueReportsSelectsOnlyEnabledDueJobs(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.Local)
	received := make([]string, 0, 3)
	service, db := setupUsageReportServiceTest(t, now, func(subject string, receiver string, content string) error {
		received = append(received, receiver)
		return nil
	})

	jobs := []entmodel.UsageReportJob{
		{
			TenantId:  0,
			Frequency: entmodel.UsageReportFrequencyDaily,
			RangeType: entmodel.UsageReportRangeLast7Days,
			Enabled:   true,
			Status:    entmodel.UsageReportStatusPending,
			NextRunAt: now.Unix() - 1,
		},
		{
			TenantId:  1,
			Frequency: entmodel.UsageReportFrequencyDaily,
			RangeType: entmodel.UsageReportRangeLast7Days,
			Enabled:   true,
			Status:    entmodel.UsageReportStatusPending,
			NextRunAt: now.Unix() + 300,
		},
		{
			TenantId:  2,
			Frequency: entmodel.UsageReportFrequencyDaily,
			RangeType: entmodel.UsageReportRangeLast7Days,
			Enabled:   false,
			Status:    entmodel.UsageReportStatusPending,
			NextRunAt: now.Unix() - 1,
		},
	}
	for i := range jobs {
		require.NoError(t, jobs[i].SetReceivers([]string{time.Unix(int64(i), 0).UTC().Format("enabled-20060102-150405@example.com")}))
		require.NoError(t, jobs[i].SetLastSnapshot(nil))
		require.NoError(t, db.Create(&jobs[i]).Error)
	}

	windowStart, windowEnd := usageReportWindow(entmodel.UsageReportRangeLast7Days, now.Unix())
	for tenantID := 0; tenantID < 3; tenantID++ {
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
	}

	result, err := service.RunDueReports(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	require.Zero(t, result.Failed)
	require.Equal(t, []string{"enabled-19700101-000000@example.com"}, received)

	var futureJob entmodel.UsageReportJob
	require.NoError(t, db.First(&futureJob, "tenant_id = ?", 1).Error)
	require.Equal(t, entmodel.UsageReportStatusPending, futureJob.Status)

	var disabledJob entmodel.UsageReportJob
	require.NoError(t, db.First(&disabledJob, "tenant_id = ?", 2).Error)
	require.Equal(t, entmodel.UsageReportStatusPending, disabledJob.Status)
}

func TestRunDueReportsMarksSuccessAndFailureWithoutBlocking(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.Local)
	sent := 0
	service, db := setupUsageReportServiceTest(t, now, func(subject string, receiver string, content string) error {
		sent++
		require.Contains(t, receiver, "ops@example.com")
		require.Contains(t, content, "部门间数值不可加和")
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

	result, err := service.RunDueReports(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	require.Equal(t, 0, result.Failed)
	require.Equal(t, 1, sent)

	var saved entmodel.UsageReportJob
	require.NoError(t, db.First(&saved, job.Id).Error)
	require.Equal(t, entmodel.UsageReportStatusSuccess, saved.Status)
	require.NotZero(t, saved.LastSuccessAt)

	failingService, failingDB := setupUsageReportServiceTest(t, now, func(subject string, receiver string, content string) error {
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

	failingResult, err := failingService.RunDueReports(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, failingResult.Processed)
	require.Equal(t, 1, failingResult.Failed)
	require.NoError(t, failingDB.First(&saved, failingJob.Id).Error)
	require.Equal(t, entmodel.UsageReportStatusFailed, saved.Status)
	require.Contains(t, saved.ErrorReason, "smtp down")
}

func TestRunDueReportsContinuesAcrossJobsAndJoinsMultipleReceivers(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.Local)
	received := make([]string, 0, 2)
	service, db := setupUsageReportServiceTest(t, now, func(subject string, receiver string, content string) error {
		received = append(received, receiver)
		if receiver == "fail@example.com" {
			return errors.New("smtp down")
		}
		require.Equal(t, "ops@example.com;cto@example.com", receiver)
		require.Contains(t, subject, "部门用量报告")
		require.Contains(t, content, "Top 部门")
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

func TestRunDueReportsFailureDoesNotBlockUsageAggregationTask(t *testing.T) {
	db := newUsageAggregationTestDB(t)
	now := time.Now()

	seedUsageTestUser(t, db, 101, "alice")
	seedUsageTestDepartment(t, db, 1, "Engineering")
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         101,
		DepartmentId:   1,
		ExternalSource: constant.EnterpriseExternalSourceManual,
		Status:         constant.EnterpriseMembershipStatusActive,
	}).Error)

	latestWindowEnd := now.Unix() - (now.Unix() % usageAggregationWindowSeconds)
	require.GreaterOrEqual(t, latestWindowEnd, usageAggregationWindowSeconds*2)
	logWindowStart := latestWindowEnd - 2*usageAggregationWindowSeconds
	require.NoError(t, db.Create(&model.Log{
		Id:               1,
		UserId:           101,
		Username:         "alice",
		Type:             model.LogTypeConsume,
		ModelName:        "gpt-4o",
		Quota:            42,
		PromptTokens:     12,
		CompletionTokens: 4,
		CreatedAt:        logWindowStart + 60,
	}).Error)

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

	reportWindowStart, reportWindowEnd := usageReportWindow(entmodel.UsageReportRangeLast7Days, now.Unix())
	reportSnapshot := entmodel.UsageSnapshot{
		TenantId:         0,
		DeptId:           intPtr(1),
		DeptName:         "Engineering",
		WindowStart:      reportWindowStart,
		WindowEnd:        reportWindowEnd,
		RequestCount:     8,
		PromptTokens:     80,
		CompletionTokens: 20,
		Quota:            160,
	}
	require.NoError(t, reportSnapshot.SetModelDistribution(nil))
	require.NoError(t, reportSnapshot.SetUserIds([]int{101}))
	require.NoError(t, db.Create(&reportSnapshot).Error)

	service := NewUsageReportServiceForTest(db, func() time.Time { return now }, func(subject string, receiver string, content string) error {
		return errors.New("smtp down")
	})
	reportResult, err := service.RunDueReports(context.Background())
	require.NoError(t, err)
	require.Zero(t, reportResult.Processed)
	require.Equal(t, 1, reportResult.Failed)

	processed, err := RunUsageAggregationTaskOnce(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, processed, 1)

	var snapshots []entmodel.UsageSnapshot
	require.NoError(t, db.Where("window_start = ? AND window_end = ?", logWindowStart, logWindowStart+usageAggregationWindowSeconds).Find(&snapshots).Error)
	require.NotEmpty(t, snapshots)
}
