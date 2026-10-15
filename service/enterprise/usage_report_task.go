package enterprise

import (
	"bytes"
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
	usageReportRootDisclaimer  = "注意：企业总览按一级部门完整子树展示，未归属用量仅计入企业总量，部门间数值不可加和"
)

type UsageReportConfigInput struct {
	TenantId           int
	DepartmentId       *int
	IncludeDescendants bool
	Receivers          []string
	Frequency          string
	RangeType          string
	Enabled            *bool
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
	Id                 int
	TenantId           int
	DepartmentId       *int
	ScopeKey           string
	IncludeDescendants bool
	Receivers          []string
	Frequency          string
	RangeType          string
	Enabled            bool
	Status             string
	LastRunAt          int64
	NextRunAt          int64
	LastSuccessAt      int64
	LastWindowStart    int64
	LastWindowEnd      int64
	RunCount           int64
	FailureCount       int64
	ErrorReason        string
	LastSnapshot       *entmodel.UsageReportSnapshot
	CreatedAt          int64
	UpdatedAt          int64
}

type UsageReportDispatchResult struct {
	Processed int
	Failed    int
}

type usageReportClock func() time.Time
type usageReportEmailSender func(subject string, receiver string, content string, attachments []common.EmailAttachment) error

type UsageReportService struct {
	db        *gorm.DB
	now       usageReportClock
	sendEmail usageReportEmailSender
}

func NewUsageReportService(db *gorm.DB) *UsageReportService {
	if db == nil {
		db = model.DB
	}
	return &UsageReportService{
		db:        db,
		now:       time.Now,
		sendEmail: common.SendEmailWithAttachments,
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

func (s *UsageReportService) GetConfig(tenantId int, departmentId *int) (UsageReportJobResult, error) {
	departmentId = normalizeUsageReportDepartmentId(departmentId)
	scopeKey := entmodel.UsageReportScopeKey(departmentId)
	var job entmodel.UsageReportJob
	if err := s.db.Where("tenant_id = ? AND scope_key = ?", tenantId, scopeKey).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return UsageReportJobResult{
				TenantId:           tenantId,
				DepartmentId:       cloneOptionalInt(departmentId),
				ScopeKey:           scopeKey,
				IncludeDescendants: departmentId != nil,
				Receivers:          []string{},
				Frequency:          entmodel.UsageReportFrequencyDaily,
				RangeType:          entmodel.UsageReportRangeLast7Days,
				Enabled:            false,
				Status:             entmodel.UsageReportStatusPending,
				LastSnapshot:       nil,
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
	scopeKey := entmodel.UsageReportScopeKey(input.DepartmentId)

	var existing entmodel.UsageReportJob
	err := s.db.Where("tenant_id = ? AND scope_key = ?", input.TenantId, scopeKey).First(&existing).Error
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
			TenantId:           input.TenantId,
			DepartmentId:       cloneOptionalInt(input.DepartmentId),
			IncludeDescendants: input.IncludeDescendants,
			Frequency:          input.Frequency,
			RangeType:          input.RangeType,
			Enabled:            enabled,
			Status:             entmodel.UsageReportStatusPending,
			NextRunAt:          nextRunAt,
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
		"department_id":       cloneOptionalInt(input.DepartmentId),
		"include_descendants": input.IncludeDescendants,
		"scope_key":           scopeKey,
		"frequency":           input.Frequency,
		"range_type":          input.RangeType,
		"enabled":             enabled,
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
	if err := s.db.Where("tenant_id = ? AND scope_key = ?", input.TenantId, scopeKey).First(&existing).Error; err != nil {
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

func (s *UsageReportService) SendNow(ctx context.Context, tenantId int, departmentId *int) (UsageReportJobResult, error) {
	departmentId = normalizeUsageReportDepartmentId(departmentId)
	if tenantId < 0 {
		return UsageReportJobResult{}, ErrUsageReportInvalidInput
	}

	scopeKey := entmodel.UsageReportScopeKey(departmentId)
	var job entmodel.UsageReportJob
	if err := s.db.Where("tenant_id = ? AND scope_key = ?", tenantId, scopeKey).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return UsageReportJobResult{}, ErrUsageReportNotConfigured
		}
		return UsageReportJobResult{}, err
	}

	if err := s.dispatchJobReport(ctx, &job, false); err != nil {
		return UsageReportJobResult{}, err
	}
	if err := s.db.Where("tenant_id = ? AND scope_key = ?", tenantId, scopeKey).First(&job).Error; err != nil {
		return UsageReportJobResult{}, err
	}
	return mapUsageReportJob(job)
}

func (s *UsageReportService) runJob(ctx context.Context, job *entmodel.UsageReportJob) error {
	return s.dispatchJobReport(ctx, job, true)
}

func (s *UsageReportService) dispatchJobReport(ctx context.Context, job *entmodel.UsageReportJob, advanceNextRun bool) error {
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
	exportResult, err := NewUsageExportService(s.db).ExportDepartmentUsageCSV(DepartmentUsageExportQuery{
		TenantId:           job.TenantId,
		DepartmentId:       job.DepartmentId,
		From:               windowStart,
		To:                 windowEnd,
		Sort:               UsageUserRankSortByQuota,
		IncludeDescendants: job.IncludeDescendants,
	})
	if err != nil {
		if markErr := s.markJobFailure(job, now, err, advanceNextRun); markErr != nil {
			return markErr
		}
		return err
	}
	var csvBuffer bytes.Buffer
	if err := NewUsageExportService(s.db).WriteDepartmentUsageCSV(&csvBuffer, exportResult); err != nil {
		if markErr := s.markJobFailure(job, now, err, advanceNextRun); markErr != nil {
			return markErr
		}
		return err
	}

	receivers, err := job.ParsedReceivers()
	if err != nil {
		if markErr := s.markJobFailure(job, now, err, advanceNextRun); markErr != nil {
			return markErr
		}
		return err
	}
	if len(receivers) == 0 {
		if markErr := s.markJobFailure(job, now, ErrUsageReportNotConfigured, advanceNextRun); markErr != nil {
			return markErr
		}
		return ErrUsageReportNotConfigured
	}

	previousStart, previousEnd := previousUsageReportWindow(windowStart, windowEnd)
	snapshot := buildUsageReportSnapshotFromExport(exportResult, windowStart, windowEnd, previousStart, previousEnd)
	subject := buildUsageReportSubject(exportResult.ScopeName, job.RangeType, windowStart, windowEnd)
	content := buildUsageReportEmailHTML(job, windowStart, windowEnd, exportResult.FileName, exportResult.ScopeName)
	attachments := []common.EmailAttachment{
		{
			Filename:    exportResult.FileName,
			ContentType: "text/csv; charset=utf-8",
			Content:     append([]byte{}, csvBuffer.Bytes()...),
		},
	}
	if err := s.sendEmail(subject, strings.Join(receivers, ";"), content, attachments); err != nil {
		if markErr := s.markJobFailure(job, now, err, advanceNextRun); markErr != nil {
			return markErr
		}
		return err
	}

	if err := job.SetLastSnapshot(snapshot); err != nil {
		if markErr := s.markJobFailure(job, now, err, advanceNextRun); markErr != nil {
			return markErr
		}
		return err
	}
	job.Status = entmodel.UsageReportStatusSuccess
	job.ErrorReason = ""
	job.LastRunAt = now
	job.LastSuccessAt = now
	job.LastWindowStart = windowStart
	job.LastWindowEnd = windowEnd
	job.RunCount++
	updates := map[string]any{
		"status":            job.Status,
		"error_reason":      job.ErrorReason,
		"last_run_at":       job.LastRunAt,
		"last_success_at":   job.LastSuccessAt,
		"last_window_start": job.LastWindowStart,
		"last_window_end":   job.LastWindowEnd,
		"run_count":         job.RunCount,
		"last_snapshot":     job.LastSnapshot,
	}
	if advanceNextRun {
		job.NextRunAt = computeUsageReportNextRunAt(job.Frequency, now)
		updates["next_run_at"] = job.NextRunAt
	}
	if err := s.db.Model(job).Updates(updates).Error; err != nil {
		return err
	}
	_ = ctx
	return nil
}

func (s *UsageReportService) markJobFailure(job *entmodel.UsageReportJob, now int64, failure error, advanceNextRun bool) error {
	job.Status = entmodel.UsageReportStatusFailed
	job.LastRunAt = now
	job.RunCount++
	job.FailureCount++
	job.ErrorReason = failure.Error()
	updates := map[string]any{
		"status":        job.Status,
		"last_run_at":   job.LastRunAt,
		"run_count":     job.RunCount,
		"failure_count": job.FailureCount,
		"error_reason":  job.ErrorReason,
	}
	if advanceNextRun {
		job.NextRunAt = computeUsageReportNextRunAt(job.Frequency, now)
		updates["next_run_at"] = job.NextRunAt
	}
	if err := s.db.Model(job).Updates(updates).Error; err != nil {
		return err
	}
	return failure
}

func normalizeUsageReportConfigInput(input UsageReportConfigInput) UsageReportConfigInput {
	input.DepartmentId = normalizeUsageReportDepartmentId(input.DepartmentId)
	if input.DepartmentId == nil {
		input.IncludeDescendants = false
	} else {
		input.IncludeDescendants = true
	}
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
	if input.DepartmentId != nil && *input.DepartmentId <= 0 {
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

func normalizeUsageReportDepartmentId(departmentId *int) *int {
	if departmentId == nil || *departmentId <= 0 {
		return nil
	}
	value := *departmentId
	return &value
}

func cloneOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func computeUsageReportNextRunAt(frequency string, now int64) int64 {
	current := time.Unix(now, 0).In(time.Local)
	todayNine := time.Date(current.Year(), current.Month(), current.Day(), 9, 0, 0, 0, current.Location())
	switch frequency {
	case entmodel.UsageReportFrequencyWeekly:
		daysUntilMonday := (int(time.Monday) - int(current.Weekday()) + 7) % 7
		next := todayNine.AddDate(0, 0, daysUntilMonday)
		if !current.Before(next) {
			next = next.AddDate(0, 0, 7)
		}
		return next.Unix()
	case entmodel.UsageReportFrequencyMonthly:
		next := time.Date(current.Year(), current.Month(), 1, 9, 0, 0, 0, current.Location())
		if !current.Before(next) {
			next = next.AddDate(0, 1, 0)
		}
		return next.Unix()
	default:
		next := todayNine
		if !current.Before(next) {
			next = next.AddDate(0, 0, 1)
		}
		return next.Unix()
	}
}

func usageReportWindow(rangeType string, now int64) (int64, int64) {
	current := time.Unix(now, 0).In(time.Local)
	startOfDay := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
	end := startOfDay
	switch rangeType {
	case entmodel.UsageReportRangeToday:
		return end.AddDate(0, 0, -1).Unix(), end.Unix()
	case entmodel.UsageReportRangeLast30Days:
		return end.AddDate(0, 0, -30).Unix(), end.Unix()
	default:
		return end.AddDate(0, 0, -7).Unix(), end.Unix()
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

func buildUsageReportSnapshotFromOverview(current UsageDashboardOverviewResult, previous UsageDashboardOverviewResult, windowStart int64, windowEnd int64, previousStart int64, previousEnd int64) *entmodel.UsageReportSnapshot {
	snapshot := buildUsageReportSnapshot(current.Items, previous.Items, windowStart, windowEnd, previousStart, previousEnd)
	snapshot.DepartmentCount = current.Metrics.DepartmentCount
	snapshot.RequestCount = current.Metrics.RequestCount
	snapshot.PromptTokens = current.Metrics.PromptTokens
	snapshot.CompletionTokens = current.Metrics.CompletionTokens
	snapshot.Quota = current.Metrics.Quota
	snapshot.UserCount = current.Metrics.UserCount
	return snapshot
}

func calculateUsageGrowthRate(previous int64, current int64) float64 {
	if previous <= 0 {
		return 0
	}
	return float64(current-previous) / float64(previous)
}

func buildUsageReportSubject(scopeName string, rangeType string, windowStart int64, windowEnd int64) string {
	return fmt.Sprintf(
		"部门用量报告 %s %s (%s ~ %s)",
		usageReportScopeName(scopeName),
		usageReportRangeLabel(rangeType),
		time.Unix(windowStart, 0).Format("2006-01-02"),
		time.Unix(windowEnd-1, 0).Format("2006-01-02"),
	)
}

func buildUsageReportHTML(snapshot *entmodel.UsageReportSnapshot) string {
	var builder strings.Builder
	builder.WriteString("<div>")
	builder.WriteString(fmt.Sprintf("<p>%s</p>", html.EscapeString(usageReportRootDisclaimer)))
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

func buildUsageReportSnapshotFromExport(result DepartmentUsageExportResult, windowStart int64, windowEnd int64, previousStart int64, previousEnd int64) *entmodel.UsageReportSnapshot {
	snapshot := &entmodel.UsageReportSnapshot{
		WindowStart:         windowStart,
		WindowEnd:           windowEnd,
		PreviousWindowStart: previousStart,
		PreviousWindowEnd:   previousEnd,
		TopDepartments:      []entmodel.UsageReportTopDepartment{},
		GrowthDepartments:   []entmodel.UsageReportGrowthDepartment{},
	}
	users := make(map[int]struct{}, len(result.Rows))
	departments := make(map[string]entmodel.UsageReportTopDepartment)
	for _, row := range result.Rows {
		snapshot.RequestCount += row.RequestCount
		snapshot.PromptTokens += row.PromptTokens
		snapshot.CompletionTokens += row.CompletionTokens
		snapshot.Quota += row.Quota
		users[row.UserId] = struct{}{}

		key := entmodel.UsageReportScopeKey(row.DeptId)
		department := departments[key]
		department.DeptId = row.DeptId
		department.DeptName = row.DeptName
		department.RequestCount += row.RequestCount
		department.Quota += row.Quota
		department.UserCount++
		departments[key] = department
	}
	snapshot.UserCount = int64(len(users))
	snapshot.DepartmentCount = int64(len(departments))
	for _, item := range departments {
		snapshot.TopDepartments = append(snapshot.TopDepartments, item)
	}
	sort.SliceStable(snapshot.TopDepartments, func(i, j int) bool {
		if snapshot.TopDepartments[i].Quota != snapshot.TopDepartments[j].Quota {
			return snapshot.TopDepartments[i].Quota > snapshot.TopDepartments[j].Quota
		}
		return snapshot.TopDepartments[i].RequestCount > snapshot.TopDepartments[j].RequestCount
	})
	if len(snapshot.TopDepartments) > usageReportTopDepartments {
		snapshot.TopDepartments = snapshot.TopDepartments[:usageReportTopDepartments]
	}
	return snapshot
}

func buildUsageReportEmailHTML(job *entmodel.UsageReportJob, windowStart int64, windowEnd int64, fileName string, scopeName string) string {
	scope := usageReportEmailScopeLabel(job, scopeName)
	return fmt.Sprintf(
		"<div><p>部门用量报告已生成。</p><p>范围：%s</p><p>报告范围：%s</p><p>统计周期：%s ~ %s</p><p>CSV 数据见附件：%s</p></div>",
		html.EscapeString(scope),
		html.EscapeString(usageReportRangeLabel(job.RangeType)),
		time.Unix(windowStart, 0).Format("2006-01-02"),
		time.Unix(windowEnd-1, 0).Format("2006-01-02"),
		html.EscapeString(fileName),
	)
}

func usageReportEmailScopeLabel(job *entmodel.UsageReportJob, scopeName string) string {
	scopeName = usageReportScopeName(scopeName)
	if job.DepartmentId != nil && job.IncludeDescendants {
		return scopeName + "及其子组织"
	}
	return scopeName
}

func usageReportScopeName(scopeName string) string {
	scopeName = strings.TrimSpace(scopeName)
	if scopeName == "" {
		return "全公司"
	}
	return scopeName
}

func usageReportRangeLabel(rangeType string) string {
	switch rangeType {
	case entmodel.UsageReportRangeToday:
		return "近一天"
	case entmodel.UsageReportRangeLast30Days:
		return "最近30天"
	default:
		return "最近7天"
	}
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
		Id:                 job.Id,
		TenantId:           job.TenantId,
		DepartmentId:       cloneOptionalInt(job.DepartmentId),
		ScopeKey:           job.ScopeKey,
		IncludeDescendants: job.IncludeDescendants,
		Receivers:          receivers,
		Frequency:          job.Frequency,
		RangeType:          job.RangeType,
		Enabled:            job.Enabled,
		Status:             job.Status,
		LastRunAt:          job.LastRunAt,
		NextRunAt:          job.NextRunAt,
		LastSuccessAt:      job.LastSuccessAt,
		LastWindowStart:    job.LastWindowStart,
		LastWindowEnd:      job.LastWindowEnd,
		RunCount:           job.RunCount,
		FailureCount:       job.FailureCount,
		ErrorReason:        job.ErrorReason,
		LastSnapshot:       snapshot,
		CreatedAt:          job.CreatedAt,
		UpdatedAt:          job.UpdatedAt,
	}, nil
}
