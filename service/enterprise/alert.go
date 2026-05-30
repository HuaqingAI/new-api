package enterprise

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	AlertRiskTypeSensitiveWords = "sensitive_words"
	AlertRiskTypeUnknown        = "unknown"
	AlertActionBlocked          = "blocked"
	AlertActionUnknown          = "unknown"
	maxAlertSummaryLength       = 256
)

type AlertService struct {
	db  *gorm.DB
	now func() time.Time
}

type RecordRiskEventInput struct {
	TenantId        int
	ModelName       string
	RiskType        string
	ActionResult    string
	Summary         string
	SensitiveHits   []string
	EventOccurredAt time.Time
}

type alertEventRow struct {
	DepartmentId   int
	DepartmentName string
	ExternalSource string
	Status         int
}

type AlertEventQuery struct {
	TenantId     int
	DepartmentId *int
	UserId       *int
	Username     string
	ModelName    string
	RiskType     string
	From         *int64
	To           *int64
	Page         int
	PageSize     int
}

type AlertEventItem struct {
	Id                 int
	TenantId           int
	UserId             int
	Username           string
	RequestId          string
	ModelName          string
	RiskType           string
	ActionResult       string
	CreatedAt          int64
	DepartmentSnapshot []entmodel.AlertEventDepartmentSnapshot
	Summary            string
}

type AlertEventListResult struct {
	Items    []AlertEventItem
	Total    int
	Page     int
	PageSize int
}

func NewAlertService(db *gorm.DB) *AlertService {
	if db == nil {
		db = model.DB
	}
	return &AlertService{
		db:  db,
		now: time.Now,
	}
}

func RecordRiskEvent(c *gin.Context, input RecordRiskEventInput) {
	_ = NewAlertService(nil).RecordRiskEvent(c, input)
}

func (s *AlertService) RecordRiskEvent(c *gin.Context, input RecordRiskEventInput) error {
	if s == nil || s.db == nil {
		return nil
	}

	userId := common.GetContextKeyInt(c, constant.ContextKeyUserId)
	username := common.GetContextKeyString(c, constant.ContextKeyUserName)
	requestId := c.GetString(common.RequestIdKey)
	if input.ModelName == "" {
		input.ModelName = common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
	}
	if input.TenantId < 0 {
		input.TenantId = 0
	}
	if input.EventOccurredAt.IsZero() {
		input.EventOccurredAt = s.now()
	}

	snapshot, err := s.loadDepartmentSnapshot(input.TenantId, userId)
	if err != nil {
		logger.LogWarn(c, fmt.Sprintf("enterprise alert event persistence skipped: snapshot_lookup_failed: %s", err.Error()))
		return nil
	}

	event := entmodel.AlertEvent{
		TenantId:     input.TenantId,
		UserId:       userId,
		Username:     username,
		RequestId:    requestId,
		ModelName:    input.ModelName,
		RiskType:     normalizeAlertRiskType(input.RiskType),
		ActionResult: normalizeAlertActionResult(input.ActionResult),
		Summary:      sanitizeRiskSummary(input.RiskType, input.Summary, input.SensitiveHits),
		CreatedAt:    input.EventOccurredAt.Unix(),
		UpdatedAt:    input.EventOccurredAt.Unix(),
	}
	if err := event.SetDepartmentSnapshot(snapshot); err != nil {
		logger.LogWarn(c, fmt.Sprintf("enterprise alert event persistence skipped: snapshot_encode_failed: %s", err.Error()))
		return nil
	}
	if err := s.db.Create(&event).Error; err != nil {
		logger.LogWarn(c, fmt.Sprintf("enterprise alert event persistence skipped: create_failed: %s", err.Error()))
		return nil
	}
	return nil
}

func (s *AlertService) ListAlertEvents(query AlertEventQuery) (AlertEventListResult, error) {
	if s == nil || s.db == nil {
		return AlertEventListResult{Items: []AlertEventItem{}}, nil
	}
	if query.TenantId < 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.DepartmentId != nil && *query.DepartmentId <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.UserId != nil && *query.UserId <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.From != nil && *query.From <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.To != nil && *query.To <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.From != nil && query.To != nil && *query.From > *query.To {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}

	page, pageSize := normalizeAlertEventPage(query.Page, query.PageSize)
	db := s.db.Model(&entmodel.AlertEvent{}).Where("tenant_id = ?", query.TenantId)
	db = applyAlertEventFilters(db, query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return AlertEventListResult{Items: []AlertEventItem{}}, err
	}

	var events []entmodel.AlertEvent
	err := db.Order("created_at DESC").
		Order("id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&events).Error
	if err != nil {
		return AlertEventListResult{Items: []AlertEventItem{}}, err
	}

	items := make([]AlertEventItem, 0, len(events))
	for _, event := range events {
		snapshot, err := event.ParsedDepartmentSnapshot()
		if err != nil {
			return AlertEventListResult{}, err
		}
		items = append(items, AlertEventItem{
			Id:                 event.Id,
			TenantId:           event.TenantId,
			UserId:             event.UserId,
			Username:           event.Username,
			RequestId:          event.RequestId,
			ModelName:          event.ModelName,
			RiskType:           event.RiskType,
			ActionResult:       event.ActionResult,
			CreatedAt:          event.CreatedAt,
			DepartmentSnapshot: snapshot,
			Summary:            event.Summary,
		})
	}

	return AlertEventListResult{
		Items:    items,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AlertService) loadDepartmentSnapshot(tenantId int, userId int) ([]entmodel.AlertEventDepartmentSnapshot, error) {
	if userId <= 0 {
		return []entmodel.AlertEventDepartmentSnapshot{}, nil
	}

	var rows []alertEventRow
	err := s.db.Table("enterprise_user_departments AS ud").
		Select("ud.department_id, d.name AS department_name, ud.external_source, ud.status").
		Joins("LEFT JOIN enterprise_departments AS d ON d.id = ud.department_id AND d.tenant_id = ud.tenant_id").
		Where("ud.tenant_id = ? AND ud.user_id = ? AND ud.status = ?", tenantId, userId, constant.EnterpriseMembershipStatusActive).
		Order("ud.department_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	snapshot := make([]entmodel.AlertEventDepartmentSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshot = append(snapshot, entmodel.AlertEventDepartmentSnapshot{
			DepartmentId:   row.DepartmentId,
			DepartmentName: row.DepartmentName,
			ExternalSource: row.ExternalSource,
			Status:         row.Status,
		})
	}
	return snapshot, nil
}

func sanitizeRiskSummary(riskType string, summary string, hits []string) string {
	if normalizeAlertRiskType(riskType) == AlertRiskTypeSensitiveWords {
		if hitCount := countNonEmptyHits(hits); hitCount > 0 {
			return fmt.Sprintf("%d sensitive word hits", hitCount)
		}
		return "sensitive words detected"
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return ""
	}

	for _, hit := range hits {
		hit = strings.TrimSpace(hit)
		if hit == "" {
			continue
		}
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(hit))
		summary = re.ReplaceAllString(summary, "[REDACTED]")
	}
	summary = strings.Join(strings.Fields(summary), " ")
	if len(summary) > maxAlertSummaryLength {
		summary = summary[:maxAlertSummaryLength]
	}
	return summary
}

func applyAlertEventFilters(db *gorm.DB, query AlertEventQuery) *gorm.DB {
	if query.DepartmentId != nil {
		tokenPattern := fmt.Sprintf("%%|%d|%%", *query.DepartmentId)
		snapshotPattern := fmt.Sprintf("%%\"department_id\":%d%%", *query.DepartmentId)
		legacySnapshotPattern := fmt.Sprintf("%%\"department_id\": %d%%", *query.DepartmentId)
		db = db.Where(
			"(department_tokens LIKE ?) OR ((department_tokens = '' OR department_tokens IS NULL) AND (department_snapshot LIKE ? OR department_snapshot LIKE ?))",
			tokenPattern,
			snapshotPattern,
			legacySnapshotPattern,
		)
	}
	if query.UserId != nil {
		db = db.Where("user_id = ?", *query.UserId)
	}
	if username := strings.TrimSpace(query.Username); username != "" {
		db = db.Where("username = ?", username)
	}
	if modelName := strings.TrimSpace(query.ModelName); modelName != "" {
		db = db.Where("model_name = ?", modelName)
	}
	if riskType := strings.TrimSpace(query.RiskType); riskType != "" {
		db = db.Where("risk_type = ?", riskType)
	}
	if query.From != nil {
		db = db.Where("created_at >= ?", *query.From)
	}
	if query.To != nil {
		db = db.Where("created_at <= ?", *query.To)
	}
	return db
}

func normalizeAlertEventPage(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeAlertRiskType(riskType string) string {
	riskType = strings.TrimSpace(riskType)
	if riskType == "" {
		return AlertRiskTypeUnknown
	}
	return riskType
}

func normalizeAlertActionResult(actionResult string) string {
	actionResult = strings.TrimSpace(actionResult)
	if actionResult == "" {
		return AlertActionUnknown
	}
	return actionResult
}

func countNonEmptyHits(hits []string) int {
	count := 0
	for _, hit := range hits {
		if strings.TrimSpace(hit) != "" {
			count++
		}
	}
	return count
}
