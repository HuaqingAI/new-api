package enterprise

import (
	"context"
	"fmt"
	"html"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const (
	usageReportGrowthThreshold = 0.5
	usageReportTopDepartments  = 5
)

type UsageReportConfigInput struct {
	TenantId  int
	Receivers []string
	Frequency string
	RangeType string
	Enabled   *bool
}

type UsageReportSummary struct {
	WindowStart      int64
	WindowEnd        int64
	DepartmentCount  int64
	RequestCount     int64
	PromptTokens     int64
	CompletionTokens int64
	Quota            int64
	UserCount        int64
}

type UsageReportJobResult struct {
	Id              int
	TenantId        int
	Receivers       []string
	Frequency       string
	RangeType       string
	Enabled         bool
	Status          string
	LastRunAt       int64
	NextRunAt       int64
	LastSuccessAt   int64
	LastWindowStart int64
	LastWindowEnd   int64
	RunCount        int64
	FailureCount    int64
	ErrorReason     string
	LastSnapshot    *entmodel.UsageReportSnapshot
	CreatedAt       int64
	UpdatedAt       int64
}

type UsageReportDispatchResult struct {
	Processed int
	Failed    int
}

type usageReportClock func() time.Time
type usageReportEmailSender func(subject string, receiver string, content string) error

type UsageReportService struct {
	db          *gorm.DB
	aggregation *UsageAggregationService
	now         usageReportClock
	sendEmail   usageReportEmailSender
}

func NewUsageReportService(db *gorm.DB) *UsageReportService {
	if db == nil {
		db = model.DB
	}
	return &UsageReportService{
		db:          db,
		aggregation: NewUsageAggregationService(db),
		now:         time.Now,
		sendEmail:   common.SendEmail,
	}
}

func NewUsageReportServiceForTest(db *gorm.DB, now usageReportClock, send usageReportEmailSender) *UsageReportService {
	service := NewUsageReportService(db)
	if now != nil {
		service.now = now
	}
	if send != nil {
		service.sendEmail = send
	}
	return service
}

func (s *UsageReportService) GetConfig(tenantId int) (UsageReportJobResult, error) {
	var job entmodel.UsageReportJob
	if err := s.db.Where("tenant_id = ?", tenantId).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return UsageReportJobResult{
				TenantId:     tenantId,
				Receivers:    []string{},
				Frequency:    entmodel.UsageReportFrequencyDaily,
				RangeType:    entmodel.UsageReportRangeLast7Days,
				Enabled:      false,
				Status:       entmodel.UsageReportStatusPending,
				LastSnapshot: nil,
			}, nil
		}
		return UsageReportJobResult{}, err
	}
	return mapUsageReportJob(job)
}

func (s *UsageReportService) SaveConfig(input UsageReportConfigInput) (UsageReportJobResult, error) {
	input = normalizeUsageReportConfigInput(input)
	if err := validateUsageReportConfigInput(input); err != nil {
		return UsageReportJobResult{}, err
	}

	now := s.now().Unix()

	var existing entmodel.UsageReportJob
	err := s.db.Where("tenant_id = ?", input.TenantId).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return UsageReportJobResult{}, err
	}

	if err == gorm.ErrRecordNotFound {
		enabled := false
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		nextRunAt := int64(0)
		if enabled {
			nextRunAt = computeUsageReportNextRunAt(input.Frequency, now)
		}
		job := entmodel.UsageReportJob{
			TenantId:  input.TenantId,
			Frequency: input.Frequency,
			RangeType: input.RangeType,
			Enabled:   enabled,
			Status:    entmodel.UsageReportStatusPending,
			NextRunAt: nextRunAt,
		}
		if err := job.SetReceivers(input.Receivers); err != nil {
			return UsageReportJobResult{}, err
		}
		if err := job.SetLastSnapshot(nil); err != nil {
			return UsageReportJobResult{}, err
		}
		if err := s.db.Create(&job).Error; err != nil {
			return UsageReportJobResult{}, err
		}
		return mapUsageReportJob(job)
	}

	enabled := existing.Enabled
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	updates := map[string]any{
		"frequency":  input.Frequency,
		"range_type": input.RangeType,
		"enabled":    enabled,
	}
	if err := existing.SetReceivers(input.Receivers); err != nil {
		return UsageReportJobResult{}, err
	}
	updates["receivers"] = existing.Receivers
	if enabled {
		if !existing.Enabled {
			updates["next_run_at"] = computeUsageReportNextRunAt(input.Frequency, now)
			updates["status"] = entmodel.UsageReportStatusPending
			updates["error_reason"] = ""
		} else if existing.Frequency != input.Frequency || existing.NextRunAt <= 0 {
			updates["next_run_at"] = computeUsageReportNextRunAt(input.Frequency, now)
		} else {
			// Keep the existing due time so editing receivers/range does not skip an imminent send.
			updates["next_run_at"] = existing.NextRunAt
		}
	} else {
		updates["status"] = entmodel.UsageReportStatusPending
		updates["error_reason"] = ""
		updates["next_run_at"] = int64(0)
	}
	if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
		return UsageReportJobResult{}, err
	}
	if err := s.db.Where("tenant_id = ?", input.TenantId).First(&existing).Error; err != nil {
		return UsageReportJobResult{}, err
	}
	return mapUsageReportJob(existing)
}

func (s *UsageReportService) RunDueReports(ctx context.Context) (UsageReportDispatchResult, error) {
	now := s.now().Unix()
	var jobs []entmodel.UsageReportJob
	if err := s.db.
		Where("enabled = ? AND next_run_at > 0 AND next_run_at <= ?", true, now).
		Order("next_run_at ASC, id ASC").
		Find(&jobs).Error; err != nil {
		return UsageReportDispatchResult{}, err
	}

	result := UsageReportDispatchResult{}
	for _, job := range jobs {
		if err := s.runJob(ctx, &job); err != nil {
			result.Failed++
			continue
		}
		result.Processed++
	}
	return result, nil
}

func (s *UsageReportService) runJob(ctx context.Context, job *entmodel.UsageReportJob) error {
	now := s.now().Unix()
	if err := s.db.Model(job).Updates(map[string]any{
		"status":       entmodel.UsageReportStatusRunning,
		"error_reason": "",
	}).Error; err != nil {
		return err
	}
	job.Status = entmodel.UsageReportStatusRunning
	job.ErrorReason = ""
	windowStart, windowEnd := usageReportWindow(job.RangeType, now)
	summary, err := s.aggregation.GetDepartmentSummary(UsageSummaryQuery{
		TenantId: job.TenantId,
		From:     windowStart,
		To:       windowEnd,
		Sort: UsageSummarySort{
			Field: UsageSummarySortByRequests,
			Order: UsageSortOrderDesc,
		},
	})
	if err != nil {
		return s.markJobFailure(job, now, err)
	}

	receivers, err := job.ParsedReceivers()
	if err != nil {
		return s.markJobFailure(job, now, err)
	}
	if len(receivers) == 0 {
		return s.markJobFailure(job, now, ErrUsageReportNotConfigured)
	}

	previousStart, previousEnd := previousUsageReportWindow(windowStart, windowEnd)
	previousSummary, err := s.aggregation.GetDepartmentSummary(UsageSummaryQuery{
		TenantId: job.TenantId,
		From:     previousStart,
		To:       previousEnd,
		Sort: UsageSummarySort{
			Field: UsageSummarySortByRequests,
			Order: UsageSortOrderDesc,
		},
	})
	if err != nil {
		return s.markJobFailure(job, now, err)
	}

	snapshot := buildUsageReportSnapshot(summary.Items, previousSummary.Items, windowStart, windowEnd, previousStart, previousEnd)
	subject := buildUsageReportSubject(job.RangeType, windowStart, windowEnd)
	content := buildUsageReportHTML(snapshot)
	if err := s.sendEmail(subject, strings.Join(receivers, ";"), content); err != nil {
		return s.markJobFailure(job, now, err)
	}

	if err := job.SetLastSnapshot(snapshot); err != nil {
		return s.markJobFailure(job, now, err)
	}
	job.Status = entmodel.UsageReportStatusSuccess
	job.ErrorReason = ""
	job.LastRunAt = now
	job.LastSuccessAt = now
	job.LastWindowStart = windowStart
	job.LastWindowEnd = windowEnd
	job.RunCount++
	job.NextRunAt = computeUsageReportNextRunAt(job.Frequency, now)
	if err := s.db.Model(job).Updates(map[string]any{
		"status":            job.Status,
		"error_reason":      job.ErrorReason,
		"last_run_at":       job.LastRunAt,
		"last_success_at":   job.LastSuccessAt,
		"last_window_start": job.LastWindowStart,
		"last_window_end":   job.LastWindowEnd,
		"run_count":         job.RunCount,
		"next_run_at":       job.NextRunAt,
		"last_snapshot":     job.LastSnapshot,
	}).Error; err != nil {
		return err
	}
	_ = ctx
	return nil
}

func (s *UsageReportService) markJobFailure(job *entmodel.UsageReportJob, now int64, failure error) error {
	job.Status = entmodel.UsageReportStatusFailed
	job.LastRunAt = now
	job.RunCount++
	job.FailureCount++
	job.ErrorReason = failure.Error()
	job.NextRunAt = computeUsageReportNextRunAt(job.Frequency, now)
	if err := s.db.Model(job).Updates(map[string]any{
		"status":        job.Status,
		"last_run_at":   job.LastRunAt,
		"run_count":     job.RunCount,
		"failure_count": job.FailureCount,
		"error_reason":  job.ErrorReason,
		"next_run_at":   job.NextRunAt,
	}).Error; err != nil {
		return err
	}
	return failure
}

func normalizeUsageReportConfigInput(input UsageReportConfigInput) UsageReportConfigInput {
	receivers := make([]string, 0, len(input.Receivers))
	for _, receiver := range input.Receivers {
		value := strings.TrimSpace(receiver)
		if value != "" {
			receivers = append(receivers, value)
		}
	}
	input.Receivers = receivers
	input.Frequency = strings.TrimSpace(strings.ToLower(input.Frequency))
	input.RangeType = strings.TrimSpace(strings.ToLower(input.RangeType))
	return input
}

func validateUsageReportConfigInput(input UsageReportConfigInput) error {
	if input.TenantId < 0 {
		return ErrUsageReportInvalidInput
	}
	switch input.Frequency {
	case entmodel.UsageReportFrequencyDaily, entmodel.UsageReportFrequencyWeekly, entmodel.UsageReportFrequencyMonthly:
	default:
		return ErrUsageReportInvalidInput
	}
	switch input.RangeType {
	case entmodel.UsageReportRangeToday, entmodel.UsageReportRangeLast7Days, entmodel.UsageReportRangeLast30Days:
	default:
		return ErrUsageReportInvalidInput
	}
	if len(input.Receivers) == 0 {
		return ErrUsageReportInvalidInput
	}
	for _, receiver := range input.Receivers {
		if _, err := mail.ParseAddress(receiver); err != nil {
			return ErrUsageReportInvalidEmail
		}
	}
	return nil
}

func computeUsageReportNextRunAt(frequency string, now int64) int64 {
	current := time.Unix(now, 0).In(time.Local)
	switch frequency {
	case entmodel.UsageReportFrequencyWeekly:
		next := current.AddDate(0, 0, 7)
		return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location()).Unix()
	case entmodel.UsageReportFrequencyMonthly:
		next := current.AddDate(0, 1, 0)
		return time.Date(next.Year(), next.Month(), 1, 9, 0, 0, 0, next.Location()).Unix()
	default:
		next := current.AddDate(0, 0, 1)
		return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location()).Unix()
	}
}

func usageReportWindow(rangeType string, now int64) (int64, int64) {
	current := time.Unix(now, 0).In(time.Local)
	startOfDay := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
	switch rangeType {
	case entmodel.UsageReportRangeToday:
		return startOfDay.Unix(), startOfDay.Add(24 * time.Hour).Unix()
	case entmodel.UsageReportRangeLast30Days:
		return startOfDay.AddDate(0, 0, -29).Unix(), startOfDay.Add(24 * time.Hour).Unix()
	default:
		return startOfDay.AddDate(0, 0, -6).Unix(), startOfDay.Add(24 * time.Hour).Unix()
	}
}

func previousUsageReportWindow(windowStart int64, windowEnd int64) (int64, int64) {
	duration := windowEnd - windowStart
	return windowStart - duration, windowStart
}

func buildUsageReportSnapshot(items []UsageDepartmentSummaryItem, previousItems []UsageDepartmentSummaryItem, windowStart int64, windowEnd int64, previousStart int64, previousEnd int64) *entmodel.UsageReportSnapshot {
	snapshot := &entmodel.UsageReportSnapshot{
		WindowStart:         windowStart,
		WindowEnd:           windowEnd,
		PreviousWindowStart: previousStart,
		PreviousWindowEnd:   previousEnd,
		TopDepartments:      []entmodel.UsageReportTopDepartment{},
		GrowthDepartments:   []entmodel.UsageReportGrowthDepartment{},
	}
	previousByBucket := make(map[string]UsageDepartmentSummaryItem, len(previousItems))
	for _, item := range previousItems {
		previousByBucket[usageBucketKey(item.DeptId)] = item
	}
	for _, item := range items {
		snapshot.DepartmentCount++
		snapshot.RequestCount += item.RequestCount
		snapshot.PromptTokens += item.PromptTokens
		snapshot.CompletionTokens += item.CompletionTokens
		snapshot.Quota += item.Quota
		snapshot.UserCount += item.UserCount
	}

	sortedTop := SortUsageDepartmentSummaryItems(items, UsageSummarySort{
		Field: UsageSummarySortByRequests,
		Order: UsageSortOrderDesc,
	})
	for i, item := range sortedTop {
		if i >= usageReportTopDepartments {
			break
		}
		snapshot.TopDepartments = append(snapshot.TopDepartments, entmodel.UsageReportTopDepartment{
			DeptId:       item.DeptId,
			DeptName:     usageSummarySortName(item),
			RequestCount: item.RequestCount,
			Quota:        item.Quota,
			UserCount:    item.UserCount,
		})
	}

	growthCandidates := make([]entmodel.UsageReportGrowthDepartment, 0)
	for _, item := range items {
		previous := previousByBucket[usageBucketKey(item.DeptId)]
		requestGrowth := calculateUsageGrowthRate(previous.RequestCount, item.RequestCount)
		quotaGrowth := calculateUsageGrowthRate(previous.Quota, item.Quota)
		if previous.RequestCount == 0 && previous.Quota == 0 {
			continue
		}
		if requestGrowth < usageReportGrowthThreshold && quotaGrowth < usageReportGrowthThreshold {
			continue
		}
		growthCandidates = append(growthCandidates, entmodel.UsageReportGrowthDepartment{
			DeptId:               item.DeptId,
			DeptName:             usageSummarySortName(item),
			RequestCount:         item.RequestCount,
			PreviousRequestCount: previous.RequestCount,
			Quota:                item.Quota,
			PreviousQuota:        previous.Quota,
			RequestGrowthRate:    requestGrowth,
			QuotaGrowthRate:      quotaGrowth,
		})
	}
	sort.SliceStable(growthCandidates, func(i, j int) bool {
		if growthCandidates[i].QuotaGrowthRate != growthCandidates[j].QuotaGrowthRate {
			return growthCandidates[i].QuotaGrowthRate > growthCandidates[j].QuotaGrowthRate
		}
		return growthCandidates[i].RequestGrowthRate > growthCandidates[j].RequestGrowthRate
	})
	if len(growthCandidates) > usageReportTopDepartments {
		growthCandidates = growthCandidates[:usageReportTopDepartments]
	}
	snapshot.GrowthDepartments = growthCandidates
	return snapshot
}

func calculateUsageGrowthRate(previous int64, current int64) float64 {
	if previous <= 0 {
		return 0
	}
	return float64(current-previous) / float64(previous)
}

func buildUsageReportSubject(rangeType string, windowStart int64, windowEnd int64) string {
	return fmt.Sprintf("部门用量报告 %s (%s ~ %s)", rangeType, time.Unix(windowStart, 0).Format("2006-01-02"), time.Unix(windowEnd-1, 0).Format("2006-01-02"))
}

func buildUsageReportHTML(snapshot *entmodel.UsageReportSnapshot) string {
	var builder strings.Builder
	builder.WriteString("<div>")
	builder.WriteString(fmt.Sprintf("<p>%s</p>", html.EscapeString(usageExportDisclaimer)))
	builder.WriteString(fmt.Sprintf("<p>统计周期：%s ~ %s</p>", time.Unix(snapshot.WindowStart, 0).Format("2006-01-02"), time.Unix(snapshot.WindowEnd-1, 0).Format("2006-01-02")))
	builder.WriteString(fmt.Sprintf("<p>部门总览：部门数 %d，请求数 %d，Prompt Tokens %d，Completion Tokens %d，Quota %d，用户数 %d。</p>", snapshot.DepartmentCount, snapshot.RequestCount, snapshot.PromptTokens, snapshot.CompletionTokens, snapshot.Quota, snapshot.UserCount))
	builder.WriteString("<p>Top 部门：</p><ul>")
	for _, item := range snapshot.TopDepartments {
		builder.WriteString(fmt.Sprintf("<li>%s：请求数 %d，Quota %d，用户数 %d</li>", html.EscapeString(item.DeptName), item.RequestCount, item.Quota, item.UserCount))
	}
	builder.WriteString("</ul>")
	builder.WriteString("<p>异常高增长部门：</p><ul>")
	if len(snapshot.GrowthDepartments) == 0 {
		builder.WriteString("<li>无</li>")
	} else {
		for _, item := range snapshot.GrowthDepartments {
			builder.WriteString(fmt.Sprintf("<li>%s：请求数 %d（较上期 %+0.0f%%），Quota %d（较上期 %+0.0f%%）</li>", html.EscapeString(item.DeptName), item.RequestCount, item.RequestGrowthRate*100, item.Quota, item.QuotaGrowthRate*100))
		}
	}
	builder.WriteString("</ul>")
	builder.WriteString("</div>")
	return builder.String()
}

func mapUsageReportJob(job entmodel.UsageReportJob) (UsageReportJobResult, error) {
	receivers, err := job.ParsedReceivers()
	if err != nil {
		return UsageReportJobResult{}, err
	}
	snapshot, err := job.ParsedLastSnapshot()
	if err != nil {
		return UsageReportJobResult{}, err
	}
	return UsageReportJobResult{
		Id:              job.Id,
		TenantId:        job.TenantId,
		Receivers:       receivers,
		Frequency:       job.Frequency,
		RangeType:       job.RangeType,
		Enabled:         job.Enabled,
		Status:          job.Status,
		LastRunAt:       job.LastRunAt,
		NextRunAt:       job.NextRunAt,
		LastSuccessAt:   job.LastSuccessAt,
		LastWindowStart: job.LastWindowStart,
		LastWindowEnd:   job.LastWindowEnd,
		RunCount:        job.RunCount,
		FailureCount:    job.FailureCount,
		ErrorReason:     job.ErrorReason,
		LastSnapshot:    snapshot,
		CreatedAt:       job.CreatedAt,
		UpdatedAt:       job.UpdatedAt,
	}, nil
}
